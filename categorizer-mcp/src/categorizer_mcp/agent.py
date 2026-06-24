"""Per-product tool-calling agent.

For each (often badly OCR'd) product name the local model is handed read-only
functions it can choose to call to investigate before committing a category — most
importantly ``search_similar_products``, which surfaces how lookalike products were
categorized so the model can correct a misread or a hallucination. When the model
stops calling tools (or the iteration/time budget runs out), a final enum-constrained
``decide_category`` call pins the answer to a valid category.

The reuse cache short-circuits before any of this, and a per-product deadline
(``CATEGORIZE_DEADLINE``, the user's 5–10 min cap) bounds the whole thing.
"""

from __future__ import annotations

import asyncio
import json
import logging
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

from . import ocr

if TYPE_CHECKING:
    from .server import App

log = logging.getLogger("categorizer-mcp.agent")

# Catch-all when the model returns nothing usable or we run out of budget.
DEFAULT_CATEGORY = "Other"

_SYSTEM = """You categorize products from a Hungarian shop receipt. The names are \
OCR-scanned and noisy: letters and digits get swapped (O/0, l/1, 5/S, 8/B), accents \
and characters get dropped, words get truncated. Keep the original Hungarian spelling \
in mind — accents carry meaning.

You have tools to investigate a name before deciding. Prefer search_similar_products: \
if a lookalike product was already categorized, reuse that category. Use ocr_variants \
to reconsider a garbled name. Call tools only as needed, then stop. You will be asked \
for the final category as JSON at the end; it must be one of the allowed categories."""


@dataclass
class Decision:
    name: str  # original, unmodified product name
    category: str
    source: str  # "cache" | "llm" | "timeout" | "error"
    evidence: list[dict] = field(default_factory=list)


def _tool_specs() -> list[dict]:
    """Ollama/OpenAI-style function schemas advertised to the model."""
    name_arg = {
        "type": "object",
        "properties": {"name": {"type": "string", "description": "product name"}},
        "required": ["name"],
    }
    return [
        {
            "type": "function",
            "function": {
                "name": "search_similar_products",
                "description": (
                    "Find already-categorized products whose names look like this "
                    "one despite OCR noise. Returns name, category and match score. "
                    "Best way to clarify a garbled name or check a guess."
                ),
                "parameters": name_arg,
            },
        },
        {
            "type": "function",
            "function": {
                "name": "ocr_variants",
                "description": (
                    "Suggest plausible corrected spellings of an OCR-garbled name by "
                    "undoing common letter/digit swaps."
                ),
                "parameters": name_arg,
            },
        },
        {
            "type": "function",
            "function": {
                "name": "lookup_known_category",
                "description": (
                    "Exact-match lookup: the category this exact product name was "
                    "given before, if any."
                ),
                "parameters": name_arg,
            },
        },
        {
            "type": "function",
            "function": {
                "name": "list_categories",
                "description": "List the allowed categories.",
                "parameters": {"type": "object", "properties": {}},
            },
        },
    ]


async def _exec_tool(
    app: "App", fn: str, args: dict, allowed: list[str], evidence: list[dict]
) -> object:
    """Run one tool call against the DB (off the event loop) and record evidence."""
    name = (args or {}).get("name", "")
    if fn == "search_similar_products":
        hits = await asyncio.to_thread(
            app.db.search_similar_products, name, app.config.similar_limit
        )
        out = [{"name": h.name, "category": h.category, "score": round(h.score, 1)} for h in hits]
        evidence.extend(out)
        return out
    if fn == "ocr_variants":
        return ocr.variants(name)
    if fn == "lookup_known_category":
        hit = await asyncio.to_thread(app.db.find_category_for_name, name)
        return {"category": hit[1]} if hit else {"category": None}
    if fn == "list_categories":
        return allowed
    return {"error": f"unknown tool {fn}"}


async def _run_agent(app: "App", name: str, allowed: list[str]) -> Decision:
    tools = _tool_specs()
    messages: list[dict] = [
        {"role": "system", "content": _SYSTEM},
        {
            "role": "user",
            "content": (
                f"Allowed categories: {', '.join(allowed)}\n"
                f"Categorize this product: {name!r}"
            ),
        },
    ]
    evidence: list[dict] = []

    for _ in range(app.config.agent_max_iters):
        msg = await app.categorizer.chat(messages, tools)
        messages.append(msg)
        calls = msg.get("tool_calls") or []
        if not calls:
            break
        for call in calls:
            fn = (call.get("function") or {}).get("name", "")
            args = (call.get("function") or {}).get("arguments") or {}
            result = await _exec_tool(app, fn, args, allowed, evidence)
            messages.append(
                {"role": "tool", "tool_name": fn, "content": json.dumps(result, ensure_ascii=False)}
            )

    messages.append(
        {"role": "user", "content": "Now give the final category as JSON {\"category\": ...}."}
    )
    category = await app.categorizer.decide_category(messages, allowed)
    if category not in allowed:
        category = DEFAULT_CATEGORY
    return Decision(name=name, category=category, source="llm", evidence=evidence)


async def categorize_product(app: "App", name: str, allowed: list[str]) -> Decision:
    """Categorize a single product: reuse-cache first, else the bounded agent loop,
    all under the per-product deadline. Always returns a valid category."""
    hit = await asyncio.to_thread(app.db.find_category_for_name, name)
    if hit is not None:
        return Decision(name=name, category=hit[1], source="cache")

    try:
        return await asyncio.wait_for(
            _run_agent(app, name, allowed), timeout=app.config.categorize_deadline
        )
    except asyncio.TimeoutError:
        log.warning("categorize timed out for %r after %ss", name, app.config.categorize_deadline)
        return Decision(name=name, category=DEFAULT_CATEGORY, source="timeout")
    except Exception as exc:  # noqa: BLE001 - never let one bad name break a batch
        log.warning("categorize failed for %r: %s", name, exc)
        return Decision(name=name, category=DEFAULT_CATEGORY, source="error")

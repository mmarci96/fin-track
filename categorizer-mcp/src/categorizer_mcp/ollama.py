"""Ollama categorizer client.

Ports the Go ``ollama.Categorizer``: it asks the model to tag product names with a
category drawn from a fixed list, constrained with Ollama structured outputs (a
JSON schema whose ``category`` field is an enum of the allowed names) so the model
cannot invent categories, plus think=false + temperature 0 + a capped token budget
so it neither over- nor under-thinks.
"""

from __future__ import annotations

import json
from dataclasses import dataclass

import httpx

_PROMPT = """You categorize grocery/shop products from a Hungarian receipt.
Assign each product to EXACTLY ONE category from the allowed list. If none fits, use "Other".
Return ONLY JSON of the form {{"items":[{{"name":"...","category":"..."}}]}}, one entry per input product, names copied verbatim.

Allowed categories: {allowed}

Products:
{products}"""


@dataclass(frozen=True)
class ProductCategory:
    name: str
    category: str


class Categorizer:
    def __init__(
        self,
        base_url: str,
        model: str,
        num_predict: int,
        keep_alive: str,
        timeout: float = 300.0,
        chat_model: str | None = None,
        decide_model: str | None = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._model = model
        # Two-model combo: a fast model drives the tool-calling loop, a stronger one
        # makes the final enum-constrained decision (e.g. qwen3:1.7b + qwen3:4b).
        # Both fall back to the primary model so a single-model setup still works.
        self._chat_model = chat_model or model
        self._decide_model = decide_model or model
        self._num_predict = num_predict
        self._keep_alive = keep_alive
        self._client = httpx.AsyncClient(timeout=timeout)

    async def aclose(self) -> None:
        await self._client.aclose()

    async def health(self) -> None:
        resp = await self._client.get(f"{self._base_url}/api/tags")
        resp.raise_for_status()

    async def chat(
        self, messages: list[dict], tools: list[dict] | None = None
    ) -> dict:
        """One turn of an Ollama tool-calling conversation via ``/api/chat``.

        Returns the assistant ``message`` dict, which carries ``content`` and, when
        the model decides to act, a ``tool_calls`` list. The agent loop feeds tool
        results back as ``role: tool`` messages and calls again. think/temperature
        are pinned low so the small model stays decisive."""
        payload = {
            "model": self._chat_model,
            "messages": messages,
            "stream": False,
            "think": False,
            "options": {"temperature": 0.0, "num_predict": self._num_predict},
            "keep_alive": self._keep_alive,
        }
        if tools:
            payload["tools"] = tools

        resp = await self._client.post(f"{self._base_url}/api/chat", json=payload)
        resp.raise_for_status()
        return resp.json().get("message", {}) or {}

    async def categorize(
        self, names: list[str], allowed: list[str]
    ) -> list[ProductCategory]:
        """Return a category for each product name. Categories are guaranteed (by
        the schema enum) to be members of ``allowed``. Names the model drops are
        simply absent from the result; callers default those to "Other"."""
        if not names:
            return []

        prompt = _PROMPT.format(allowed=", ".join(allowed), products="\n".join(names))

        # A tighter budget for the single-item fast path; the schema does the heavy
        # lifting of constraining the answer either way.
        num_predict = self._num_predict
        if len(names) == 1 and num_predict > 64:
            num_predict = 64

        payload = {
            "model": self._model,
            "prompt": prompt,
            "stream": False,
            "think": False,
            "format": _schema(allowed),
            "options": {"temperature": 0.0, "num_predict": num_predict},
            "keep_alive": self._keep_alive,
        }

        resp = await self._client.post(f"{self._base_url}/api/generate", json=payload)
        resp.raise_for_status()
        raw = resp.json().get("response", "")

        try:
            parsed = json.loads(raw)
        except json.JSONDecodeError as exc:
            raise ValueError(f"decode categorize json: {raw!r}") from exc

        return [
            ProductCategory(name=item.get("name", ""), category=item.get("category", ""))
            for item in parsed.get("items", [])
        ]

    async def decide_category(self, messages: list[dict], allowed: list[str]) -> str:
        """Final, enum-constrained commit step for the agent loop: given the full
        conversation (the product plus whatever evidence the model gathered via
        tools), force a single category out of ``allowed`` using structured outputs.
        Guarantees a valid category regardless of how the tool loop went."""
        schema = {
            "type": "object",
            "properties": {"category": {"type": "string", "enum": allowed}},
            "required": ["category"],
        }
        payload = {
            "model": self._decide_model,
            "messages": messages,
            "stream": False,
            "think": False,
            "format": schema,
            "options": {"temperature": 0.0, "num_predict": 64},
            "keep_alive": self._keep_alive,
        }
        resp = await self._client.post(f"{self._base_url}/api/chat", json=payload)
        resp.raise_for_status()
        raw = (resp.json().get("message", {}) or {}).get("content", "")
        try:
            return json.loads(raw).get("category", "")
        except json.JSONDecodeError as exc:
            raise ValueError(f"decode decide_category json: {raw!r}") from exc


def _schema(allowed: list[str]) -> dict:
    """JSON schema pinning the response shape and limiting each item's category to
    the allowed enum."""
    return {
        "type": "object",
        "properties": {
            "items": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "name": {"type": "string"},
                        "category": {"type": "string", "enum": allowed},
                    },
                    "required": ["name", "category"],
                },
            }
        },
        "required": ["items"],
    }

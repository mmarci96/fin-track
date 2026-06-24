"""MCP server wiring: tools an agent (or the backend) calls to categorize products.

Built on FastMCP with the streamable-HTTP transport so it runs as a networked
microservice. The categorization pipeline matches the Go backend: check the
name reuse cache first, ask Ollama only about never-seen names, and (optionally)
commit the assignments to ``product_categories``.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import TYPE_CHECKING

from mcp.server.fastmcp import FastMCP

from . import agent
from .config import Config
from .db import Database
from .ollama import Categorizer

if TYPE_CHECKING:
    from .queue import JobQueue

# The catch-all used when the model returns nothing usable for a product.
DEFAULT_CATEGORY = "Other"


@dataclass
class App:
    config: Config
    db: Database
    categorizer: Categorizer
    # Set by the worker lifespan once the event loop is running.
    queue: "JobQueue | None" = None


def build_server(app: App) -> FastMCP:
    mcp = FastMCP(
        name="fin-track-categorizer",
        instructions=(
            "Categorize fin-track receipt products into a fixed set of grocery "
            "categories. Use list_categories to see the allowed set, "
            "list_uncategorized_products to find work, categorize_names for a "
            "dry run over arbitrary names, and categorize_receipt to categorize "
            "and optionally persist a receipt's products."
        ),
        host=app.config.host,
        port=app.config.port,
    )

    @mcp.tool()
    def list_categories() -> list[dict]:
        """List the fixed categories products may be assigned to."""
        return [{"id": c.id, "name": c.name} for c in app.db.all_categories()]

    @mcp.tool()
    def list_uncategorized_products(receipt_id: int | None = None) -> list[dict]:
        """List products that have no category yet. Pass receipt_id to scope it to
        a single receipt; omit it for everything pending across the database."""
        products = app.db.uncategorized_products(receipt_id)
        return [
            {"id": p.id, "name": p.name, "price": p.price, "receipt_id": p.receipt_id}
            for p in products
        ]

    @mcp.tool()
    def search_similar_products(name: str, limit: int = 5) -> list[dict]:
        """Find already-categorized products whose names look like ``name`` despite
        OCR noise (letter/digit swaps, dropped accents). Returns the original product
        names with their category and an OCR-aware match score — the best way to
        clarify a garbled name or sanity-check a guess."""
        hits = app.db.search_similar_products(name, limit)
        return [{"name": h.name, "category": h.category, "score": round(h.score, 1)} for h in hits]

    @mcp.tool()
    async def categorize_product(name: str) -> dict:
        """Agentic categorization of a single product name: the model may call tools
        (similar-product search, OCR variants, cache lookup) to investigate a noisy
        name before committing, bounded by the per-product deadline. Returns the
        chosen category, its source (cache/llm/timeout/error) and the evidence seen."""
        allowed = [c.name for c in app.db.all_categories()]
        d = await agent.categorize_product(app, name, allowed)
        return {"name": d.name, "category": d.category, "source": d.source, "similar": d.evidence}

    @mcp.tool()
    def submit_categorize_job(products: list[str]) -> dict:
        """Enqueue a categorization job for the always-on worker and return its id
        immediately. Poll get_job to watch progress and collect results."""
        if app.queue is None:
            return {"error": "queue not running"}
        return app.queue.submit(products).to_dict()

    @mcp.tool()
    def get_job(job_id: str) -> dict:
        """Status and results of a queued/running/finished job."""
        if app.queue is None:
            return {"error": "queue not running"}
        job = app.queue.get(job_id)
        return job.to_dict() if job else {"error": "no such job"}

    @mcp.tool()
    def list_jobs() -> list[dict]:
        """All jobs, newest first."""
        if app.queue is None:
            return []
        return [j.to_dict() for j in app.queue.list()]

    @mcp.tool()
    def cancel_job(job_id: str) -> dict:
        """Cancel a queued job, or stop a running one after its current product."""
        if app.queue is None:
            return {"error": "queue not running"}
        return {"cancelled": job_id} if app.queue.cancel(job_id) else {"error": "job unknown or already finished"}

    @mcp.tool()
    async def categorize(names: list[str], allowed: list[str]) -> list[dict]:
        """Pure categorization: map each product name to exactly one category from
        the supplied allowed list, using only the LLM (no database, no cache). The
        backend's Go categorizer calls this in place of talking to Ollama directly.
        Returns [{"name", "category"}] with categories drawn from allowed; names the
        model drops are omitted and should be defaulted to "Other" by the caller."""
        results = await app.categorizer.categorize(names, allowed)
        return [{"name": r.name, "category": r.category} for r in results]

    @mcp.tool()
    async def categorize_names(names: list[str]) -> list[dict]:
        """Dry-run categorization for arbitrary product names without touching the
        database. Returns one proposal per name with its source (cache or llm).
        Useful for previewing before committing."""
        categories = app.db.all_categories()
        proposals = await _categorize(app, names, categories)
        return [_proposal_dict(p) for p in proposals]

    @mcp.tool()
    async def categorize_receipt(receipt_id: int, save: bool = False) -> dict:
        """Categorize every uncategorized product on a receipt. When save is true,
        the assignments are written to product_categories; otherwise the proposals
        are returned for review. Returns a summary plus per-product proposals."""
        products = app.db.uncategorized_products(receipt_id)
        if not products:
            return {"receipt_id": receipt_id, "saved": False, "proposals": [], "message": "nothing to categorize"}

        categories = app.db.all_categories()
        id_by_product = {p.name.lower(): p.id for p in products}
        names = [p.name for p in products]

        proposals = await _categorize(app, names, categories)
        for p in proposals:
            p.product_id = id_by_product.get(p.name.lower(), 0)

        inserted = 0
        if save:
            pairs = [
                (p.product_id, p.category_id)
                for p in proposals
                if p.product_id and p.category_id
            ]
            inserted = app.db.assign_categories(pairs)

        return {
            "receipt_id": receipt_id,
            "saved": save,
            "assigned": inserted,
            "cache_hits": sum(1 for p in proposals if p.source == "cache"),
            "llm": sum(1 for p in proposals if p.source == "llm"),
            "proposals": [_proposal_dict(p) for p in proposals],
        }

    return mcp


@dataclass
class _Proposal:
    name: str
    category_id: int
    category_name: str
    source: str  # "cache" | "llm"
    product_id: int = 0


def _proposal_dict(p: _Proposal) -> dict:
    return {
        "product_id": p.product_id,
        "name": p.name,
        "category_id": p.category_id,
        "category_name": p.category_name,
        "source": p.source,
    }


async def _categorize(app: App, names: list[str], categories) -> list[_Proposal]:
    """Shared pipeline: reuse-cache lookups first, one Ollama call for the misses,
    defaulting anything still unresolved to the catch-all category."""
    id_by_cat = {c.name.lower(): c.id for c in categories}
    default_id = id_by_cat.get(DEFAULT_CATEGORY.lower(), 0)

    proposals: list[_Proposal] = []
    misses: list[str] = []
    for name in names:
        hit = app.db.find_category_for_name(name)
        if hit is not None:
            cat_id, cat_name = hit
            proposals.append(_Proposal(name=name, category_id=cat_id, category_name=cat_name, source="cache"))
        else:
            misses.append(name)

    if misses:
        results = await app.categorizer.categorize(misses, [c.name for c in categories])
        cat_by_name = {r.name.lower(): r.category for r in results}
        for name in misses:
            cat_name = cat_by_name.get(name.lower(), "")
            cat_id = id_by_cat.get(cat_name.lower(), 0)
            if cat_id == 0:  # model returned nothing usable
                cat_name, cat_id = DEFAULT_CATEGORY, default_id
            proposals.append(_Proposal(name=name, category_id=cat_id, category_name=cat_name, source="llm"))

    return proposals

"""Plain HTTP (REST) surface for the categorizer.

Sits alongside the MCP endpoint so you can drive the service without an MCP client:
POST a handful of product names and get categories back. Each name runs through the
tool-calling agent (``agent.categorize_product``), so the same investigation the MCP
tools expose happens here too.
"""

from __future__ import annotations

import asyncio
import logging
from typing import TYPE_CHECKING

from starlette.requests import Request
from starlette.responses import JSONResponse
from starlette.routing import Route

from . import agent

if TYPE_CHECKING:
    from .server import App

log = logging.getLogger("categorizer-mcp.http")


def routes(app: "App") -> list[Route]:
    async def categorize(request: Request) -> JSONResponse:
        try:
            body = await request.json()
        except Exception:
            return JSONResponse({"error": "invalid JSON body"}, status_code=400)

        products = body.get("products")
        if not isinstance(products, list) or not all(isinstance(p, str) for p in products):
            return JSONResponse(
                {"error": "body must be {\"products\": [string, ...]}"}, status_code=400
            )

        allowed = [c.name for c in await asyncio.to_thread(app.db.all_categories)]
        if not allowed:
            return JSONResponse(
                {"error": "no categories in database; seed the categories table"},
                status_code=503,
            )

        # Sequential: a single Ollama instance serves one generation at a time, so
        # concurrency wouldn't help and would just thrash the model.
        results = []
        for name in products:
            d = await agent.categorize_product(app, name, allowed)
            results.append(
                {"name": d.name, "category": d.category, "source": d.source, "similar": d.evidence}
            )
        return JSONResponse(results)

    async def submit_job(request: Request) -> JSONResponse:
        """Enqueue a categorization job; the background worker picks it up. Returns
        the job id immediately so callers can poll GET /jobs/{id}."""
        try:
            body = await request.json()
        except Exception:
            return JSONResponse({"error": "invalid JSON body"}, status_code=400)
        products = body.get("products")
        if not isinstance(products, list) or not all(isinstance(p, str) for p in products):
            return JSONResponse(
                {"error": "body must be {\"products\": [string, ...]}"}, status_code=400
            )
        if app.queue is None:
            return JSONResponse({"error": "queue not running"}, status_code=503)
        job = app.queue.submit(products)
        return JSONResponse(job.to_dict(), status_code=202)

    async def list_jobs(request: Request) -> JSONResponse:
        if app.queue is None:
            return JSONResponse({"error": "queue not running"}, status_code=503)
        return JSONResponse([j.to_dict() for j in app.queue.list()])

    async def get_job(request: Request) -> JSONResponse:
        if app.queue is None:
            return JSONResponse({"error": "queue not running"}, status_code=503)
        job = app.queue.get(request.path_params["job_id"])
        if job is None:
            return JSONResponse({"error": "no such job"}, status_code=404)
        return JSONResponse(job.to_dict())

    async def cancel_job(request: Request) -> JSONResponse:
        if app.queue is None:
            return JSONResponse({"error": "queue not running"}, status_code=503)
        ok = app.queue.cancel(request.path_params["job_id"])
        if not ok:
            return JSONResponse(
                {"error": "job unknown or already finished"}, status_code=409
            )
        return JSONResponse({"cancelled": request.path_params["job_id"]})

    async def health(request: Request) -> JSONResponse:
        ollama_ok, db_ok = True, True
        try:
            await app.categorizer.health()
        except Exception as exc:  # noqa: BLE001
            ollama_ok = False
            log.warning("ollama health failed: %s", exc)
        try:
            await asyncio.to_thread(app.db.ping)
        except Exception as exc:  # noqa: BLE001
            db_ok = False
            log.warning("db health failed: %s", exc)
        code = 200 if (ollama_ok and db_ok) else 503
        return JSONResponse({"ollama": ollama_ok, "db": db_ok}, status_code=code)

    return [
        Route("/categorize", categorize, methods=["POST"]),
        Route("/jobs", submit_job, methods=["POST"]),
        Route("/jobs", list_jobs, methods=["GET"]),
        Route("/jobs/{job_id}", get_job, methods=["GET"]),
        Route("/jobs/{job_id}", cancel_job, methods=["DELETE"]),
        Route("/health", health, methods=["GET"]),
    ]

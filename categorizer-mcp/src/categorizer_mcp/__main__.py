"""Entry point: wires config, DB pool and Ollama client, then serves the REST API
and the MCP endpoint from a single ASGI app under uvicorn.

The plain REST routes (``/categorize``, ``/health``) live at the root; the MCP
streamable-HTTP transport is mounted at ``/mcp``. Both share one ``App`` (DB pool +
Ollama client). Starlette does not run a *mounted* app's lifespan, so the MCP session
manager is entered explicitly in our lifespan.
"""

from __future__ import annotations

import asyncio
import contextlib
import logging

import uvicorn
from starlette.applications import Starlette
from starlette.routing import Mount

from . import http
from .config import Config
from .db import Database
from .ollama import Categorizer
from .queue import JobQueue, worker
from .server import App, build_server


def build_asgi(app: App, mcp) -> Starlette:
    # Serve the MCP transport at the mount root so the external path is exactly /mcp.
    mcp.settings.streamable_http_path = "/"
    mcp_app = mcp.streamable_http_app()

    @contextlib.asynccontextmanager
    async def lifespan(_: Starlette):
        # Create the queue + start the always-on worker now that the loop is running.
        app.queue = JobQueue()
        worker_task = asyncio.create_task(worker(app))
        try:
            async with mcp.session_manager.run():
                yield
        finally:
            worker_task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await worker_task

    return Starlette(
        routes=[*http.routes(app), Mount("/mcp", app=mcp_app)],
        lifespan=lifespan,
    )


def main() -> None:
    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s: %(message)s")
    log = logging.getLogger("categorizer-mcp")

    config = Config.from_env()
    db = Database(config.database_url)
    db.open()
    # DB-backed paths (similar-product search, cache, uncategorized) need Postgres,
    # but the pure categorize tool does not, so a missing DB at startup is a warning.
    try:
        db.ping()
        log.info("connected to postgres")
    except Exception as exc:  # noqa: BLE001 - log and keep serving the pure tool
        log.warning("postgres unreachable at startup; DB-backed tools will fail: %s", exc)

    categorizer = Categorizer(
        base_url=config.ollama_base_url,
        model=config.model,
        num_predict=config.num_predict,
        keep_alive=config.keep_alive,
        chat_model=config.chat_model,
        decide_model=config.decide_model,
    )

    app = App(config=config, db=db, categorizer=categorizer)
    asgi = build_asgi(app, build_server(app))

    log.info(
        "serving REST (/categorize, /health) + MCP (/mcp) on http://%s:%s  chat=%s decide=%s  ollama=%s",
        config.host,
        config.port,
        config.chat_model,
        config.decide_model,
        config.ollama_base_url,
    )
    try:
        uvicorn.run(asgi, host=config.host, port=config.port, log_level="info")
    finally:
        db.close()


if __name__ == "__main__":
    main()

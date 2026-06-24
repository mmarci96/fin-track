"""In-memory job queue with an always-on worker.

The service keeps a queue of categorization jobs that can be added and inspected over
HTTP. A single background worker drains it: whenever a job is waiting it works on it,
categorizing each product through the agent (with the per-product deadline), then goes
back to waiting. This is the "conversation queue, modifiable over HTTP, always working
on a problem if there is one".

The queue is in-memory: simple and fine for a single instance at hobby scale. Jobs are
lost on restart; back it with Postgres if you need durability.
"""

from __future__ import annotations

import asyncio
import logging
import time
import uuid
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

from . import agent

if TYPE_CHECKING:
    from .server import App

log = logging.getLogger("categorizer-mcp.queue")


class JobStatus(str):
    QUEUED = "queued"
    RUNNING = "running"
    DONE = "done"
    FAILED = "failed"
    CANCELLED = "cancelled"


@dataclass
class Job:
    id: str
    products: list[str]
    status: str = JobStatus.QUEUED
    results: list[dict] = field(default_factory=list)
    error: str | None = None
    progress: int = 0  # products categorized so far
    created_at: float = field(default_factory=time.time)
    started_at: float | None = None
    finished_at: float | None = None

    def to_dict(self) -> dict:
        return {
            "id": self.id,
            "status": self.status,
            "progress": self.progress,
            "total": len(self.products),
            "products": self.products,
            "results": self.results,
            "error": self.error,
            "created_at": self.created_at,
            "started_at": self.started_at,
            "finished_at": self.finished_at,
        }


class JobQueue:
    """Holds the pending-id channel and the job registry. Instantiate inside the
    running event loop (the worker lifespan does this)."""

    def __init__(self) -> None:
        self._pending: asyncio.Queue[str] = asyncio.Queue()
        self._jobs: dict[str, Job] = {}

    def submit(self, products: list[str]) -> Job:
        job = Job(id=uuid.uuid4().hex[:12], products=list(products))
        self._jobs[job.id] = job
        self._pending.put_nowait(job.id)
        return job

    def get(self, job_id: str) -> Job | None:
        return self._jobs.get(job_id)

    def list(self) -> list[Job]:
        return sorted(self._jobs.values(), key=lambda j: j.created_at, reverse=True)

    def cancel(self, job_id: str) -> bool:
        """Cancel a job. A queued job is dropped before it runs; a running job is
        flagged so the worker stops after the current product. Returns False if the
        job is unknown or already finished."""
        job = self._jobs.get(job_id)
        if job is None or job.status in (JobStatus.DONE, JobStatus.FAILED, JobStatus.CANCELLED):
            return False
        job.status = JobStatus.CANCELLED
        return True

    async def _next(self) -> str:
        return await self._pending.get()


async def worker(app: "App") -> None:
    """Drain the queue forever: take the next job, categorize each product, repeat.
    Survives per-job failures so one bad job never stops the worker."""
    queue: JobQueue = app.queue  # type: ignore[assignment]
    log.info("job worker started")
    while True:
        job_id = await queue._next()
        job = queue.get(job_id)
        if job is None or job.status == JobStatus.CANCELLED:
            continue

        job.status = JobStatus.RUNNING
        job.started_at = time.time()
        log.info("job %s started (%d products)", job.id, len(job.products))
        try:
            allowed = [c.name for c in await asyncio.to_thread(app.db.all_categories)]
            for name in job.products:
                if job.status == JobStatus.CANCELLED:
                    break
                d = await agent.categorize_product(app, name, allowed)
                job.results.append(
                    {"name": d.name, "category": d.category, "source": d.source, "similar": d.evidence}
                )
                job.progress += 1
            if job.status != JobStatus.CANCELLED:
                job.status = JobStatus.DONE
        except Exception as exc:  # noqa: BLE001 - record and keep the worker alive
            job.status = JobStatus.FAILED
            job.error = str(exc)
            log.warning("job %s failed: %s", job.id, exc)
        finally:
            job.finished_at = time.time()
            log.info("job %s %s (%d/%d)", job.id, job.status, job.progress, len(job.products))

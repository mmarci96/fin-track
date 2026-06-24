"""Runtime configuration, read from the environment.

Env var names line up with the Go backend (``OLLAMA_*``, ``DB_*``,
``CATEGORIZE_*``) so the two services can share a single ``.env`` file.
"""

from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    # HTTP server bind address for the MCP endpoint.
    host: str
    port: int

    # Ollama connection.
    ollama_base_url: str
    model: str
    # Two-model combo: fast model for the tool-calling loop, stronger model for the
    # final decision. Both default to ``model`` (single-model setup).
    chat_model: str
    decide_model: str
    # Token budget for a categorization call; matches the Go CATEGORIZE_NUM_PREDICT.
    num_predict: int
    # How long Ollama keeps the model resident after a call, so single-item
    # follow-ups answer without a cold reload.
    keep_alive: str

    # Postgres connection string (libpq URL).
    database_url: str

    # Agent budgets. Per-product wall-clock cap (the user's 5–10 min ceiling),
    # max tool-calling turns before forcing a decision, and how many lookalike
    # products search_similar_products returns.
    categorize_deadline: float
    agent_max_iters: int
    similar_limit: int

    @staticmethod
    def from_env() -> "Config":
        ollama_host = os.getenv("OLLAMA_HOST", "localhost")
        ollama_port = os.getenv("OLLAMA_PORT", "11434")

        # CATEGORIZE_MODEL wins, falling back to OLLAMA_MODEL, like the Go config.
        model = os.getenv("CATEGORIZE_MODEL") or os.getenv("OLLAMA_MODEL") or "qwen3:1.7b"
        # Optional per-stage overrides for the fast/strong combo; default to `model`.
        chat_model = os.getenv("CATEGORIZE_CHAT_MODEL") or model
        decide_model = os.getenv("CATEGORIZE_DECIDE_MODEL") or model

        return Config(
            host=os.getenv("MCP_HOST", "127.0.0.1"),
            port=int(os.getenv("MCP_PORT", "8090")),
            ollama_base_url=f"http://{ollama_host}:{ollama_port}",
            model=model,
            chat_model=chat_model,
            decide_model=decide_model,
            num_predict=int(os.getenv("CATEGORIZE_NUM_PREDICT", "256")),
            keep_alive=os.getenv("CATEGORIZE_KEEP_ALIVE", "10m"),
            database_url=_database_url(),
            categorize_deadline=float(os.getenv("CATEGORIZE_DEADLINE", "600")),
            agent_max_iters=int(os.getenv("CATEGORIZE_MAX_ITERS", "6")),
            similar_limit=int(os.getenv("CATEGORIZE_SIMILAR_LIMIT", "5")),
        )


def _database_url() -> str:
    """Build the Postgres URL from DATABASE_URL, or assemble it from DB_* parts."""
    if url := os.getenv("DATABASE_URL"):
        return url

    user = os.getenv("DB_USER", "postgres")
    password = os.getenv("DB_PASSWORD", "")
    host = os.getenv("DB_HOST", "localhost")
    port = os.getenv("DB_PORT", "5432")
    name = os.getenv("DB_NAME", "fintrack")
    auth = f"{user}:{password}" if password else user
    return f"postgresql://{auth}@{host}:{port}/{name}?sslmode=disable"

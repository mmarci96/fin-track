# categorizer-mcp

An Ollama-backed **MCP microservice** that categorizes fin-track receipt products.

It exposes the same categorization logic the Go backend has (a fixed category set,
a product-name reuse cache, and Ollama structured outputs) as MCP tools served over
**streamable HTTP**, so an agent — or the backend — can send a few small messages to
categorize items instead of embedding the pipeline.

It also runs a **tool-calling agent**: receipt names are OCR-noisy (letter/digit
swaps, dropped accents, truncation), so per product the model can call read-only
tools — most importantly a fuzzy, OCR-aware search over already-categorized products
— to investigate and self-correct before a final, enum-constrained decision. A
per-product deadline bounds the work.

## HTTP API

Plain REST lives at the root (the MCP transport is at `/mcp`):

| Method & path | Body / result |
| --- | --- |
| `POST /categorize` | `{"products": ["COO Peroni", "EBED", ...]}` → `[{"name","category","source","similar"}]`. Synchronous; runs the agent per name. |
| `POST /jobs` | `{"products": [...]}` → `202` with the job (id + status). Enqueues for the background worker. |
| `GET /jobs` | All jobs, newest first. |
| `GET /jobs/{id}` | One job: `status` (queued/running/done/failed/cancelled), `progress`/`total`, `results`. |
| `DELETE /jobs/{id}` | Cancel a queued job (or stop a running one after its current product). |
| `GET /health` | `{"ollama": bool, "db": bool}` (503 if either is down). |

`/categorize` blocks until done (good for a quick batch); `/jobs` returns immediately
and an **always-on worker** drains the queue one job at a time — submit, then poll
`GET /jobs/{id}`. The queue is in-memory (lost on restart).

```sh
curl -s -X POST localhost:8090/categorize -H 'content-type: application/json' \
  -d '{"products":["COO Peroni","EBED","TEJ 2.8%"]}' | jq
```

## How it fits

The Go backend OCRs receipts and stores `products`. This service reads those rows,
categorizes their names, and (optionally) writes `product_categories` against the
same Postgres database and the same fixed `categories` set. The model is constrained
by a JSON-schema enum so it can only return an allowed category.

## Tools

| Tool | What it does |
| --- | --- |
| `list_categories()` | The fixed category set products may be assigned to. |
| `list_uncategorized_products(receipt_id?)` | Products with no category yet, optionally one receipt. |
| `search_similar_products(name, limit=5)` | OCR-aware fuzzy search over already-categorized products; returns `{name, category, score}`. |
| `categorize_product(name)` | Agentic single-name categorization (the model may call tools to investigate). Returns `{name, category, source, similar}`. |
| `categorize_names(names)` | Dry-run categorization for arbitrary names. No DB writes. Returns proposals with `source` = `cache` \| `llm`. |
| `categorize_receipt(receipt_id, save=false)` | Categorize a receipt's uncategorized products; when `save` is true, commit them to `product_categories`. |
| `submit_categorize_job(products)` / `get_job(id)` / `list_jobs()` / `cancel_job(id)` | Queue a job for the background worker and track it (same queue as `POST /jobs`). |

The pipeline for `categorize_names` / `categorize_receipt`: check the reuse cache
(another product of the same name already categorized) first, ask Ollama only about
never-seen names in a single call, and default anything unresolved to `Other`.

## Run

```sh
cp .env.example .env        # then edit DB_* / OLLAMA_* to match your setup
uv sync                     # or: pip install -e .
uv run categorizer-mcp      # serves http://127.0.0.1:8090/mcp
```

Requires a running Ollama (`OLLAMA_MODEL` pulled) and the fin-track Postgres schema.

## Deploy with Docker (portable bundle)

The service is stateless — it only needs an external Postgres and an Ollama. The
bundle ships its own Ollama with a one-shot model puller, so a fresh server needs
nothing but Docker and a reachable Postgres.

```sh
cp .env.example .env        # set DATABASE_URL; tweak the models if you like
make up                     # starts ollama, pulls the combo models, starts the API
curl localhost:8090/health  # {"ollama":true,"db":true}
make logs                   # follow the categorizer
make down                   # stop (keeps the model volume)
```

By default the bundle runs the **1.7b/4b combo**: `qwen3:1.7b` drives the tool-calling
loop and `qwen3:4b` makes the final decision (`CATEGORIZE_CHAT_MODEL` /
`CATEGORIZE_DECIDE_MODEL`). The puller fetches `CATEGORIZE_PULL_MODELS`.

### Publish to Docker Hub

```sh
make login                  # once
make publish                # build + tag :<version> and :latest, push both
make publish TAG=1.2.0 DOCKER_USER=otheruser   # override
```

The image defaults to `mmarci96/categorizer-mcp`; `TAG` defaults to the version in
`pyproject.toml`.

### Run on another server

Either copy `docker-compose.yml` + `.env` and `docker compose up -d` (it pulls the
published image), or run the image alone against an existing Ollama + Postgres:

```sh
make run DATABASE_URL=postgres://user:pass@db-host:5432/fin-track-db?sslmode=disable \
         OLLAMA_HOST=ollama-host
# or plain docker:
docker run -d -p 8090:8090 \
  -e MCP_HOST=0.0.0.0 \
  -e OLLAMA_HOST=ollama-host -e OLLAMA_PORT=11434 \
  -e CATEGORIZE_CHAT_MODEL=qwen3:1.7b -e CATEGORIZE_DECIDE_MODEL=qwen3:4b \
  -e DATABASE_URL=postgres://user:pass@db-host:5432/fin-track-db?sslmode=disable \
  mmarci96/categorizer-mcp:latest
```

`DATABASE_URL` must be reachable from inside the container (not `localhost`). If your
Postgres lives on another compose project's network, attach this service to it — see
the commented `networks:` block in `docker-compose.yml`.

## Connecting a client

Point any streamable-HTTP MCP client at `http://127.0.0.1:8090/mcp`. For Claude
Code:

```sh
claude mcp add --transport http categorizer http://127.0.0.1:8090/mcp
```

## Configuration

All via environment (see `.env.example`); names line up with the Go backend so a
shared `.env` works. Key ones: `MCP_HOST` / `MCP_PORT`, `OLLAMA_HOST` / `OLLAMA_PORT`
/ `OLLAMA_MODEL` (or `CATEGORIZE_MODEL`), the combo overrides `CATEGORIZE_CHAT_MODEL`
/ `CATEGORIZE_DECIDE_MODEL`, the agent budgets `CATEGORIZE_DEADLINE` /
`CATEGORIZE_MAX_ITERS` / `CATEGORIZE_SIMILAR_LIMIT`, and `DATABASE_URL` (or the
`DB_*` parts).

## Backend integration

The Go backend reads two env vars (already added to its config + Makefile) so it can
point at this service:

| Env var | Meaning | Example |
| --- | --- | --- |
| `CATEGORIZER_URL` | Base URL of this service | `http://categorizer-mcp:8090` |
| `CATEGORIZER_ENABLED` | Delegate categorization here instead of calling Ollama directly | `true` |

Set them on the backend (e.g. `make -C backend start CATEGORIZER_ENABLED=true`). The
backend then calls `POST {CATEGORIZER_URL}/categorize` with
`{"products":[...]}`. Config plumbing is wired; swapping the handlers to actually call
it is the remaining backend step.

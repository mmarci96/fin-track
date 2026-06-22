# fin-track

App to track finances during the month. Snap a photo of a receipt, OCR + LLM
parse it into line items, review/edit, and store it.

## Monorepo layout

```
fin-track/
├── backend/     Go API (Gin) — OCR, receipt parsing, persistence
│   ├── cmd/         entrypoint (cmd/fin-track)
│   ├── internal/    handlers, router, services, repository
│   ├── pkg/         shared packages (logger)
│   ├── sql/         schema + drop
│   ├── data/        sample receipt images for testing
│   ├── Makefile     docker-based dev/build targets
│   └── go.mod
├── frontend/    React client + Traefik edge (mobile-first)
├── Makefile     root orchestrator (delegates to backend/ and frontend/)
└── README.md
```

## Quickstart (Docker)

```sh
# database + schema (also creates the shared `fin-track` network)
make -C backend db-start
make -C backend db-schema
# optional: LLM runtime for the OCR parse fallback
make -C backend ollama-start

# build + run backend, UI, and the Traefik edge on the same network
make start
```

Open **http://localhost** (Traefik dashboard at http://localhost:8088).
Traefik routes `/api` → Go backend and everything else → the React UI, so the
browser sees a single origin. Tear down with `make stop`.

## Backend

```sh
cd backend
make dev-start   # docker dev stack with live reload (air)
```

API is served at `http://localhost:8080/api`. See `backend/Makefile` for the
full set of targets (db, ollama, ocr test fixtures).

## Frontend

Mobile-first React + Vite + Tailwind client (cropping-led receipt capture).

```sh
make -C frontend start   # build the UI image + run UI and Traefik
make -C frontend dev     # local Vite dev server (needs host Node)
make e2e                 # Playwright DOM tests against the live edge
```

The full design and runbook live in `.claude/plan.md`.

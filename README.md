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
├── frontend/    React client (planned — see .claude/plan.md)
└── README.md
```

## Backend

```sh
cd backend
make dev-start   # docker dev stack with live reload (air)
```

API is served at `http://localhost:8080/api`. See `backend/Makefile` for the
full set of targets (db, ollama, ocr test fixtures).

## Frontend

Not built yet. The implementation plan lives in `.claude/plan.md`.

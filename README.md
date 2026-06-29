# fin-track

App to track finances during the month. Snap a photo of a receipt, OCR + LLM
parse it into line items, review/edit, and store it.

## Monorepo layout

```
fin-track/
├── backend/        Go API (Gin) — OCR, receipt parsing, persistence
│   ├── cmd/            entrypoint (cmd/fin-track)
│   ├── internal/       handlers, router, services, repository
│   ├── pkg/            shared packages (logger)
│   ├── sql/            schema + drop
│   ├── data/           sample receipt images for testing
│   ├── Makefile        docker-based dev/build + swarm targets
│   └── go.mod
├── auth-service/   Stateless JWT auth (login / logout / verify)
├── frontend/       React client (nginx image), mobile-first
├── traefik/        Traefik edge + docker-socket-proxy + auth plugin
├── categorizer-mcp/  Optional LLM categorizer (published separately)
├── .env            single config file (PUBLIC_HOSTNAME, ENV)   [gitignored]
├── .secrets/       single secrets folder (db / auth / jwt)     [gitignored]
├── Makefile        root orchestrator — start / stop / reset / create-user
└── README.md
```

## Running the stack (Docker Swarm)

The app runs as a **Docker Swarm** stack — Postgres, the Go backend, the auth
service, the Traefik edge and the React UI — on an overlay network, pinned to
the manager node. It is all driven from the root `Makefile`.

### Prerequisites

1. Docker with Swarm mode enabled (one-time, on the manager node):
   ```sh
   docker swarm init
   ```
2. A single `.env` at the repo root:
   ```sh
   cat > .env <<'EOF'
   PUBLIC_HOSTNAME=localhost
   ENV=prod
   EOF
   ```
   `PUBLIC_HOSTNAME` is the host the edge is served on (used to build the auth
   login redirect). `ENV` selects the auth database/role (e.g. `auth_prod`).

Secrets are generated on first run into the gitignored `.secrets/` folder
(`db_password`, `auth_db_password`, `jwt_secret`) and registered as Docker
secrets — no manual step needed.

### Bring it up

```sh
make start      # secrets + network + db + auth + backend + UI + edge
```

Open **http://localhost** (Traefik dashboard at http://localhost:8088).
Traefik routes `/login`,`/logout`,`/verify` → auth-service, `/api/*` → backend,
and everything else → the React UI, so the browser sees a single origin.
Requests without a valid session cookie are redirected to `/login`.

### Create a user

The app requires a login. Add one interactively — this creates the fin-track
application user **and** the linked auth login in one step:

```sh
make create-user      # prompts for email, username, password
```

### Operations

```sh
make stop       # remove every service (keeps data, secrets, networks)
make start      # bring them back — reuses the db volume (non-destructive)
make restart    # stop + start
make reset      # DESTRUCTIVE: wipe the db volume and rebuild from zero
make ps         # list running services
make logs SVC=auth-service
```

> The Cloudflare tunnel (public ingress) is managed separately under
> `~/cloudflare` and is intentionally **not** part of `make start` / `make stop`.

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
make -C frontend start   # build the UI image + deploy the UI swarm service
make -C frontend dev     # local Vite dev server (needs host Node)
make e2e                 # Playwright DOM tests against the live edge
```

The full design and runbook live in `.claude/plan.md`.

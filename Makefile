# fin-track — central orchestrator for the Docker Swarm stack.
#
# One place to stand the whole application up or tear it down, and to add users.
# The Cloudflare tunnel is intentionally NOT managed here — it lives under
# ~/cloudflare and is brought up separately (docker-compose).
#
#   make start        # bring the whole stack up (reuses the existing db volume)
#   make stop         # remove every service (keeps data, secrets, networks)
#   make restart      # stop + start (data preserved)
#   make reset        # DESTRUCTIVE: wipe the db volume and rebuild from zero
#   make create-user  # interactively add a login (app user + auth user)
#   make ps           # list the running services
#
# Configuration lives in ONE place each:
#   - secrets:     $(SECRETS_DIR)/  (gitignored raw values; shared by every service)
#   - non-secrets: ./.env           (PUBLIC_HOSTNAME, ENV)
# The .env is read here for our own targets; sub-makes read it themselves, so we
# deliberately do NOT `export` (that would leak SECRETS_DIR etc. and override the
# sub-makefile defaults).

-include .env

# Single source of truth for raw secret values, shared by all services.
SECRETS_DIR ?= .secrets

# Overlay networks. `homelab` carries all app traffic; `cloudflare` is shared
# with the tunnel so the edge is reachable from it.
NETWORK            ?= homelab
CLOUDFLARE_NETWORK ?= cloudflare

# Postgres coordinates (kept in sync with backend/Makefile). Used by create-user
# to insert the fin-track application user before the auth login is created.
DB_SERVICE ?= fin-track-db
DB_NAME    ?= fin_track_db
DB_USER    ?= fin-track
# Resolve the postgres swarm task container on this (manager) node. Swarm names
# tasks <service>.<n>.<id>, so the bare service name can't be `docker exec`'d.
DB_CID = $(shell docker ps -qf name=$(DB_SERVICE) | head -n1)

# Auth login coordinates. ENV (from .env, e.g. prod) selects the dedicated auth
# database/role — must match auth-service/Makefile + sql/provision.sql.
AUTH_ENV     ?= $(ENV)
AUTH_IMAGE   ?= auth-service:latest
AUTH_DB_NAME ?= auth_$(AUTH_ENV)
AUTH_DB_USER ?= auth_$(AUTH_ENV)

.PHONY: start up stop down restart reset ps logs \
        secrets network db provision backend-up auth-up edge \
        create-user e2e backend frontend

## start: bring the whole stack up (except cloudflare). Non-destructive — an
## existing postgres data volume is reused. Run as the first bring-up (after the
## secrets exist) or after `make stop`.
start: secrets network db provision backend-up auth-up edge
	@echo
	@echo "stack is up:"
	@$(MAKE) --no-print-directory ps
up: start

## stop: remove every service (frontend, edge, auth, backend, ollama, db).
## Keeps the db volume, the secret files/objects, and the overlay networks.
stop:
	-$(MAKE) -C frontend kill
	-$(MAKE) -C traefik kill
	-$(MAKE) -C traefik socket-proxy-kill
	-$(MAKE) -C auth-service kill
	-$(MAKE) -C backend app-kill
	-$(MAKE) -C backend ollama-kill
	-$(MAKE) -C backend db-kill
down: stop

## restart: stop then start (data preserved).
restart: stop start

## reset: DESTRUCTIVE. Tear everything down, wipe the postgres data volume, and
## rebuild from zero. Loses ALL application + auth data.
reset: stop secrets network
	$(MAKE) -C backend db-fresh
	$(MAKE) provision
	$(MAKE) backend-up auth-up edge
	@echo
	@echo "stack rebuilt from zero:"
	@$(MAKE) --no-print-directory ps

# --- bring-up phases (used by start/reset) ---------------------------------

## secrets: ensure the shared docker secrets exist (generated on first run).
secrets:
	$(MAKE) -C backend secrets
	$(MAKE) -C auth-service secrets

## network: ensure both overlay networks exist (no-op if already created).
network:
	-docker network create --driver overlay --attachable $(NETWORK)
	-docker network create --driver overlay --attachable $(CLOUDFLARE_NETWORK)

## db: bring postgres up (reusing its volume) and load the fin-track schema.
db:
	$(MAKE) -C backend db-start
	$(MAKE) -C backend db-wait
	$(MAKE) -C backend db-schema

## provision: create the auth roles + databases (password aligned to the secret)
## and load the auth schema.
provision:
	$(MAKE) -C auth-service provision
	$(MAKE) -C auth-service schema

## backend-up: build + run the backend service and its ollama sidecar.
backend-up:
	$(MAKE) -C backend image
	$(MAKE) -C backend ollama-start
	$(MAKE) -C backend app-start

## auth-up: build + run the auth-service.
auth-up:
	$(MAKE) -C auth-service image
	$(MAKE) -C auth-service start

## edge: bring up the Traefik edge (+ socket-proxy) and the UI.
edge:
	$(MAKE) -C traefik start
	$(MAKE) -C frontend start

# --- user administration ---------------------------------------------------

## create-user: interactively add a user. Prompts for email, username and
## password, inserts the fin-track application user (users table) to obtain its
## id, then creates the matching auth login (auth_users) linked via app_user_id.
## The password is only ever held in a shell variable (never a make variable).
create-user:
	@test -n "$(DB_CID)" || { echo "postgres '$(DB_SERVICE)' is not running — run 'make start' first" >&2; exit 1; }
	@test -s $(SECRETS_DIR)/jwt_secret || { echo "missing $(SECRETS_DIR)/jwt_secret — run 'make secrets'" >&2; exit 1; }
	@test -s $(SECRETS_DIR)/auth_db_password || { echo "missing $(SECRETS_DIR)/auth_db_password — run 'make secrets'" >&2; exit 1; }
	@$(MAKE) --no-print-directory -C auth-service image
	@printf 'Email: '; read email; \
	 printf 'Username: '; read username; \
	 printf 'Password: '; stty -echo 2>/dev/null; read password; stty echo 2>/dev/null; printf '\n'; \
	 if [ -z "$$email" ] || [ -z "$$username" ] || [ -z "$$password" ]; then \
		echo "email, username and password are all required" >&2; exit 1; fi; \
	 echo "==> fin-track app user (users table)"; \
	 appid=$$(printf '%s\n' "INSERT INTO users (name, email) VALUES (:'un', :'em') ON CONFLICT (email) DO UPDATE SET name = EXCLUDED.name RETURNING id;" \
		| docker exec -i $(DB_CID) psql -U $(DB_USER) -d $(DB_NAME) -v ON_ERROR_STOP=1 -qtA \
		-v em="$$email" -v un="$$username" -f -); \
	 if [ -z "$$appid" ]; then echo "failed to create/resolve the app user" >&2; exit 1; fi; \
	 echo "    app_user_id = $$appid"; \
	 echo "==> auth login (auth_users)"; \
	 docker run --rm --network $(NETWORK) \
		-e JWT_SECRET="$$(cat $(SECRETS_DIR)/jwt_secret)" \
		-e DB_HOST=$(DB_SERVICE) -e DB_PORT=5432 \
		-e DB_NAME=$(AUTH_DB_NAME) -e DB_USER=$(AUTH_DB_USER) \
		-e DB_PASSWORD="$$(cat $(SECRETS_DIR)/auth_db_password)" \
		$(AUTH_IMAGE) \
		-createuser -email "$$email" -password "$$password" -app-user-id "$$appid"; \
	 echo "==> done: $$email can now log in (app_user_id=$$appid)"

# --- misc ------------------------------------------------------------------

## ps: show the running swarm services.
ps:
	@docker service ls

## logs: follow a service's logs, e.g. make logs SVC=auth-service
logs:
	@docker service logs --tail 80 -f $(SVC)

## e2e: run the frontend Playwright DOM tests.
e2e:
	$(MAKE) -C frontend e2e

## backend / frontend: pass an arbitrary target through, e.g.
##   make backend CMD=db-schema
##   make frontend CMD=build
backend:
	$(MAKE) -C backend $(CMD)
frontend:
	$(MAKE) -C frontend $(CMD)

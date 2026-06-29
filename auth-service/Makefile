# auth-service — stateless JWT auth microservice fronting the Traefik edge.
-include ../.env
#
# Shares the same postgres container as fin-track but uses its own dedicated
# database (auth_uat / auth_dev). Bring up the shared infra first with
# `make -C ../backend infra-start`.
#
#   make ENV=uat up        # provision + schema + build + run (default ENV=uat)
#   make create-user EMAIL=me@x.com PASSWORD=secret APP_USER_ID=1
#   make kill

APP_NAME     ?= auth-service
NETWORK      ?= homelab
IMAGE        ?= auth-service:latest
HOST_PORT    ?= 8082
PORT         ?= 8000

# ENV selects the dedicated auth database: auth_uat (default) or auth_dev.
ENV          ?= uat

# Shared postgres container. The cluster superuser only PROVISIONS the auth
# role + database (over the local socket); the service itself connects over TCP
# as the dedicated, least-privilege auth role.
DB_CONTAINER ?= fin-track-db
DB_HOST      ?= $(DB_CONTAINER)
DB_PORT      ?= 5432
DB_SUPERUSER ?= fin-track
# Resolve the postgres swarm task container on THIS node (run provisioning where
# the db task is placed — the manager). The bare service name can't be exec'd
# because swarm names tasks <service>.<n>.<id>.
DB_CID = $(shell docker ps -qf name=$(DB_CONTAINER) | head -n1)
# Gitignored dir holding raw secret values; single source of truth for both the
# docker secret and the matching postgres role password. Centralised at the repo
# root so every service shares one set of secret files.
SECRETS_DIR ?= ../.secrets

# Dedicated auth credentials — must match sql/provision.sql.
AUTH_DB_NAME     ?= auth_$(ENV)
AUTH_DB_USER     ?= auth_$(ENV)
AUTH_DB_PASSWORD ?= auth_$(ENV)_password

# Shared with the Traefik traefikauth plugin (dynamic.yml jwtSecret) — keep in
# sync and rotate for anything beyond local/UAT. Read from the central secrets
# dir; only used by one-shot targets (create-user) that need the raw value.
JWT_SECRET    ?= $(shell cat $(SECRETS_DIR)/jwt_secret 2>/dev/null)
JWT_SECRET_PATH ?= /run/secrets/jwt_secret
AUTH_DB_PASSWORD_PATH ?= /run/secrets/auth_db_password
COOKIE_SECURE ?= true
COOKIE_DOMAIN ?=
# Session/JWT lifetime. Must be > 0 or every issued token expires immediately.
TOKEN_TTL     ?= 24h

.PHONY: up image provision schema run kill create-user

## up: provision the auth db, load schema, build the image, and run the service
up: provision schema image start

## bootstrap: from-zero auth on the manager node — secrets, provision (roles +
## dbs, with the role password aligned to the secret), schema, build, and run.
bootstrap: secrets provision schema image start

image:
	docker build -t $(IMAGE) .

## secrets: create the jwt + auth-db docker secrets from gitignored files,
## generating random values the first time. Idempotent (an existing secret is
## kept; remove it to rotate).
secrets:
	@mkdir -p $(SECRETS_DIR)
	@test -s $(SECRETS_DIR)/auth_db_password || { umask 077; openssl rand -base64 24 | tr -d '\n' > $(SECRETS_DIR)/auth_db_password; echo "generated $(SECRETS_DIR)/auth_db_password"; }
	@test -s $(SECRETS_DIR)/jwt_secret       || { umask 077; openssl rand -base64 32 | tr -d '\n' > $(SECRETS_DIR)/jwt_secret;       echo "generated $(SECRETS_DIR)/jwt_secret"; }
	@docker secret inspect auth_db_password >/dev/null 2>&1 || docker secret create auth_db_password $(SECRETS_DIR)/auth_db_password
	@docker secret inspect jwt_secret       >/dev/null 2>&1 || docker secret create jwt_secret       $(SECRETS_DIR)/jwt_secret

## provision: create the auth roles + databases (idempotent), then align the
## active env's role password with the auth_db_password secret so the service can
## log in. Runs against the maintenance `postgres` db over the local socket.
provision:
	docker exec -i $(DB_CID) \
		psql -U $(DB_SUPERUSER) -d postgres -v ON_ERROR_STOP=1 \
		-f - < sql/provision.sql
	docker exec -i $(DB_CID) \
		psql -U $(DB_SUPERUSER) -d postgres -v ON_ERROR_STOP=1 \
		-c "ALTER ROLE $(AUTH_DB_USER) PASSWORD '$$(cat $(SECRETS_DIR)/auth_db_password)';"

## schema: create the auth_users table in the selected auth database
schema:
	docker exec -i $(DB_CID) \
		psql -U $(AUTH_DB_USER) -d $(AUTH_DB_NAME) -v ON_ERROR_STOP=1 \
		-f - < sql/schema.sql

start: kill
	docker service create -d \
		--name $(APP_NAME) \
		--network $(NETWORK) \
		--constraint 'node.role==manager' \
		-p $(HOST_PORT):$(PORT) \
		--label 'traefik.enable=true' \
		--label 'traefik.http.routers.auth-service.rule=PathPrefix(`/login`) || PathPrefix(`/logout`) || PathPrefix(`/verify`) || PathPrefix(`/healthz`)' \
		--label 'traefik.http.routers.auth-service.entrypoints=web' \
		--label 'traefik.http.routers.auth-service.priority=200' \
		--label 'traefik.http.routers.auth-service.service=auth-service' \
		--label 'traefik.http.services.auth-service.loadbalancer.server.port=$(PORT)' \
		--secret jwt_secret \
		--secret auth_db_password \
		-e AUTH_PORT=$(PORT) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_NAME=$(AUTH_DB_NAME) \
		-e DB_USER=$(AUTH_DB_USER) \
		-e DB_PASSWORD_PATH=$(AUTH_DB_PASSWORD_PATH) \
		-e JWT_SECRET_PATH=$(JWT_SECRET_PATH) \
		-e COOKIE_SECURE=$(COOKIE_SECURE) \
		-e COOKIE_DOMAIN=$(COOKIE_DOMAIN) \
		-e TOKEN_TTL=$(TOKEN_TTL) \
		$(IMAGE)

kill:
	-docker service rm $(APP_NAME)

## create-user: hash + insert a user. EMAIL, PASSWORD, APP_USER_ID required.
create-user: image
	@test -n "$(EMAIL)"       || { echo "EMAIL is required";       exit 1; }
	@test -n "$(PASSWORD)"    || { echo "PASSWORD is required";    exit 1; }
	@test -n "$(APP_USER_ID)" || { echo "APP_USER_ID is required"; exit 1; }
	docker run --rm \
		--network $(NETWORK) \
		-e JWT_SECRET=$$(cat $(SECRETS_DIR)/jwt_secret) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_NAME=$(AUTH_DB_NAME) \
		-e DB_USER=$(AUTH_DB_USER) \
		-e DB_PASSWORD=$$(cat $(SECRETS_DIR)/auth_db_password) \
		$(IMAGE) \
		-createuser -email "$(EMAIL)" -password "$(PASSWORD)" -app-user-id $(APP_USER_ID)

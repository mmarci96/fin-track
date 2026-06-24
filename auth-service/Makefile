# auth-service — stateless JWT auth microservice fronting the Traefik edge.
#
# Shares the same postgres container as fin-track but uses its own dedicated
# database (auth_uat / auth_dev). Bring up the shared infra first with
# `make -C ../backend infra-start`.
#
#   make ENV=uat up        # provision + schema + build + run (default ENV=uat)
#   make create-user EMAIL=me@x.com PASSWORD=secret APP_USER_ID=1
#   make kill

APP_NAME     ?= auth-service
NETWORK      ?= fin-track
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

# Dedicated auth credentials — must match sql/provision.sql.
AUTH_DB_NAME     ?= auth_$(ENV)
AUTH_DB_USER     ?= auth_$(ENV)
AUTH_DB_PASSWORD ?= auth_$(ENV)_password

# Shared with the Traefik traefikauth plugin (dynamic.yml jwtSecret) — keep in
# sync and rotate for anything beyond local/UAT.
JWT_SECRET    ?= $(shell cat $(shell pwd)/.secret)
COOKIE_SECURE ?= true
COOKIE_DOMAIN ?=
# Session/JWT lifetime. Must be > 0 or every issued token expires immediately.
TOKEN_TTL     ?= 24h

.PHONY: up image provision schema run kill create-user

## up: provision the auth db, load schema, build the image, and run the service
up: provision schema image start 

image:
	docker build -t $(IMAGE) .

## provision: create the auth_uat / auth_dev roles + databases (idempotent)
provision:
	docker exec -i $(DB_CONTAINER) \
		psql -U $(DB_SUPERUSER) -d fin-track-db -v ON_ERROR_STOP=1 \
		-f - < sql/provision.sql

## schema: create the auth_users table in the selected auth database
schema:
	docker exec -i $(DB_CONTAINER) \
		psql -U $(AUTH_DB_USER) -d $(AUTH_DB_NAME) -v ON_ERROR_STOP=1 \
		-f - < sql/schema.sql

start: kill
	docker run -d \
		--name $(APP_NAME) \
		--network $(NETWORK) \
		-p $(HOST_PORT):$(PORT) \
		--label 'traefik.enable=true' \
		--label 'traefik.http.routers.auth-service.rule=PathPrefix(`/login`) || PathPrefix(`/logout`) || PathPrefix(`/verify`) || PathPrefix(`/healthz`)' \
		--label 'traefik.http.routers.auth-service.entrypoints=web' \
		--label 'traefik.http.routers.auth-service.priority=200' \
		--label 'traefik.http.routers.auth-service.service=auth-service' \
		--label 'traefik.http.services.auth-service.loadbalancer.server.port=$(PORT)' \
		-e AUTH_PORT=$(PORT) \
		-e JWT_SECRET=$(JWT_SECRET) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_NAME=$(AUTH_DB_NAME) \
		-e DB_USER=$(AUTH_DB_USER) \
		-e DB_PASSWORD=$(AUTH_DB_PASSWORD) \
		-e COOKIE_SECURE=$(COOKIE_SECURE) \
		-e COOKIE_DOMAIN=$(COOKIE_DOMAIN) \
		-e TOKEN_TTL=$(TOKEN_TTL) \
		$(IMAGE)

kill:
	-docker rm -f $(APP_NAME)

## create-user: hash + insert a user. EMAIL, PASSWORD, APP_USER_ID required.
create-user: image
	@test -n "$(EMAIL)"       || { echo "EMAIL is required";       exit 1; }
	@test -n "$(PASSWORD)"    || { echo "PASSWORD is required";    exit 1; }
	@test -n "$(APP_USER_ID)" || { echo "APP_USER_ID is required"; exit 1; }
	docker run --rm \
		--network $(NETWORK) \
		-e JWT_SECRET=$(JWT_SECRET) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_NAME=$(AUTH_DB_NAME) \
		-e DB_USER=$(AUTH_DB_USER) \
		-e DB_PASSWORD=$(AUTH_DB_PASSWORD) \
		$(IMAGE) \
		-createuser -email "$(EMAIL)" -password "$(PASSWORD)" -app-user-id $(APP_USER_ID)

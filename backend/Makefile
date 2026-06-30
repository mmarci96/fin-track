# App specific configs
APP_NAME ?= fin-track
APP_VERSION ?= 0.0.0
HOST ?= 0.0.0.0
PORT ?= 8080
NETWORK ?= homelab

# Auth posture. The plain `start`/app-start container runs behind the Traefik
# auth edge, so it must reject requests without a verified X-User-ID. dev-start
# overrides this to false for frictionless local testing.
REQUIRE_USER_ID ?= true

# Debug image store: a named docker volume mounted into the backend. The debug
# upload endpoint writes original images here so they can be viewed next to
# their transcript on the UI.
IMAGE_STORE_DIR ?= /data/receipt-images
IMAGE_VOLUME    ?= receipt_images

# Destructive DB targets (reset / db-nuke) refuse to run unless RUNTIME_ENV is
# one of these, so they can't wipe a UAT/prod database by accident.
DEV_ENVS := dev DEBUG debug development

# Public hostname for the cloudflare smoke-test target. Sourced from the single
# gitignored repo-root .env so the real hostname never lands in version control.
# Override ad hoc with: make test-all-img-cloudflare PUBLIC_HOSTNAME=host
-include ../.env

# Ollama env
OLLAMA_APP_NAME ?= ollama-app
OLLAMA_HOST ?= $(OLLAMA_APP_NAME)
OLLAMA_PORT ?= 11434
LOG_LEVEL ?= DEBUG
RUNTIME_ENV ?= DEBUG
OLLAMA_IMAGE ?= ollama/ollama
OLLAMA_VERSION ?= latest

#DB
DB_IMAGE ?= postgres
DB_VERSION ?= 17
# DB_SERVICE: docker swarm service name + DNS hostname of postgres (kept
# hyphenated — that's the running service apps resolve by name).
# DB_NAME: the postgres database name (underscored; hyphens force SQL quoting).
DB_SERVICE ?= fin-track-db
DB_NAME ?= fin_track_db
DB_HOST ?= $(DB_SERVICE)
DB_PORT ?= 5432
DB_USER ?= $(APP_NAME)
DB_PASSWORD ?= secure_password
DB_PASSWORD_PATH ?= /run/secrets/db_password
DB_DATA_DOCKER_VOL ?= postgres_data
# Resolve the postgres swarm task container on THIS node. Swarm names tasks
# <service>.<n>.<id>, so `docker exec` the bare service name fails — exec targets
# must run on the node where the db task is placed (the manager, since db-start
# pins it there). Evaluated lazily per (sub-)make invocation.
DB_CID = $(shell docker ps -qf name=$(DB_SERVICE) | head -n1)
# Gitignored dir holding the raw secret values (single source of truth for both
# the docker secret and the postgres role password). Centralised at the repo
# root so every service shares one set of secret files.
SECRETS_DIR ?= ../.secrets

# Testing
IMG_PATH ?= @$(shell pwd)/data
IMGS = IMG_0955.jpeg IMG_0961.jpeg IMG_0962.jpeg IMG_0963.jpeg IMG_0964.jpeg IMG_0967.jpeg IMG_0971.jpeg
OTPR_IMGS = IMG_0975.jpeg IMG_0974.jpeg IMG_0973.jpeg IMG_0972.jpeg
RECEIPT_ID = 5

PHONY:$(MAKECMDGOALS)

start: kill db-start ollama-start db-wait db-schema image app-start

kill: ollama-kill db-kill app-kill

## reset: dev-only soft reset — clears scanned data (receipts/products/images)
## but preserves users, merchants, categories, currencies. See sql/drop.sql.
reset: guard-dev db-drop kill start

## guard-dev: refuse to run a destructive target outside a dev RUNTIME_ENV.
guard-dev:
	@case " $(DEV_ENVS) " in \
		*" $(RUNTIME_ENV) "*) : ;; \
		*) echo "refusing destructive op: RUNTIME_ENV='$(RUNTIME_ENV)' is not a dev env ($(DEV_ENVS))" >&2; exit 1 ;; \
	esac

## db-nuke: dev-only FULL wipe (includes users + reference data). Use only when
## changing the schema of the reference tables themselves. See sql/nuke.sql.
db-nuke: guard-dev
	docker exec -i $(DB_CID) \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME) \
		-f - < sql/nuke.sql

test-categorize:
	@echo "Testing categorize with receipt id $(RECEIPT_ID)"
	curl -X POST http://localhost:8080/api/receipts/$(RECEIPT_ID)/categorize;

test-default:
	@echo "Testing with sample image IMG_0955.jpeg"
	curl -X POST -F "image=$(IMG_PATH)/IMG_0955.jpeg" http://localhost:8080/api/receipts/image;

## test-debug-img: send a sample image to the developer debug endpoint, which
## also persists the image + transcript. Sends X-User-ID since the secured
## container (app-start) requires it; harmless against dev-start.
test-debug-img:
	@echo "Sending sample image to the debug endpoint..."
	curl -X POST -H "X-User-ID: 1" \
		-F "image=$(IMG_PATH)/IMG_0955.jpeg" \
		http://localhost:8080/api/receipts/image/debug;

test-otpr-imgs:
	@echo "Sending otp restaurant receipts..."
	@for img in $(OTPR_IMGS); do \
		echo "Sending $$img..."; \
		curl -X POST \
			-F "image=$(IMG_PATH)/$$img" \
			http://localhost:8080/api/receipts/image; \
		done

test-all-img:
	@echo "Sending all images..."
	@for img in $(IMGS); do \
		echo "Sending $$img..."; \
		curl -X POST \
			-F "image=$(IMG_PATH)/$$img" \
			http://localhost:8080/api/receipts/image; \
		done

test-all-img-cloudflare:
	@echo "Sending all images..."
	@for img in $(IMGS); do \
		echo "Sending $$img..."; \
		curl -X POST \
			-F "image=$(IMG_PATH)/$$img" \
			https://$(PUBLIC_HOSTNAME)/api/receipts/image; \
		done

test-ocr:
	-curl -X POST \
		-F "image=$(IMG_PATH)/IMG_0955.jpeg" \
		http://localhost:8080/api/receipts/image
	-curl -X POST \
		-F "image=$(IMG_PATH)/IMG_0963.jpeg" \
		http://localhost:8080/api/receipts/image

dev-image:
	docker build -f Dockerfile.dev -t $(APP_NAME)-dev:latest .

DEV_NETWORK = fin-track-dev

dev-start: kill dev-image
	docker run  -it --rm \
		--name $(APP_NAME) \
		-p $(PORT):$(PORT) \
		-p 8081:$(PORT) \
		--network $(DEV_NETWORK) \
		-v $(shell pwd):/tmp/fin-track-src \
		-v go-mod-cache:/go/pkg/mod \
		-e HOST=$(HOST) \
		-e PORT=$(PORT) \
		-e OLLAMA_HOST=$(OLLAMA_HOST) \
		-e OLLAMA_PORT=$(OLLAMA_PORT) \
		-e DB_NAME=$(DB_NAME) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_USER=$(DB_USER) \
		-e DB_PASSWORD=$(DB_PASSWORD) \
		-e REQUIRE_USER_ID=false \
		-e IMAGE_STORE_DIR=$(IMAGE_STORE_DIR) \
		-v $(IMAGE_VOLUME):$(IMAGE_STORE_DIR) \
		--entrypoint=air \
		$(APP_NAME)-dev:latest \
		 -c /tmp/fin-track-src/.air.toml

dev-net:
	-docker network create $(DEV_NETWORK)

dev-db: dev-net 
	-docker rm -f $(DB_SERVICE)
	docker run -d \
		--name $(DB_SERVICE) \
		--network $(DEV_NETWORK) \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_DB=$(DB_NAME) \
		--mount type=volume,source=$(DB_DATA_DOCKER_VOL),target=/var/lib/postgresql/data \
		$(DB_IMAGE):$(DB_VERSION)


image:
	docker build -t $(APP_NAME):$(APP_VERSION) .

app-start: app-kill
	docker service create -d \
		--name $(APP_NAME) \
		--network $(NETWORK) \
		--constraint 'node.role==manager' \
		-p $(PORT):$(PORT) \
		--label 'traefik.enable=true' \
		--label 'traefik.http.routers.api.rule=PathPrefix(`/api`)' \
		--label 'traefik.http.routers.api.entrypoints=web' \
		--label 'traefik.http.routers.api.priority=100' \
		--label 'traefik.http.routers.api.middlewares=auth-middleware@file' \
		--label 'traefik.http.routers.api.service=backend' \
		--label 'traefik.http.services.backend.loadbalancer.server.port=$(PORT)' \
		--secret db_password \
		--mount type=volume,source=$(IMAGE_VOLUME),target=$(IMAGE_STORE_DIR) \
		-e HOST=$(HOST) \
		-e PORT=$(PORT) \
		-e OLLAMA_HOST=$(OLLAMA_HOST) \
		-e OLLAMA_PORT=$(OLLAMA_PORT) \
		-e DB_NAME=$(DB_NAME) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_USER=$(DB_USER) \
		-e DB_PASSWORD_PATH=$(DB_PASSWORD_PATH) \
		-e REQUIRE_USER_ID=$(REQUIRE_USER_ID) \
		-e IMAGE_STORE_DIR=$(IMAGE_STORE_DIR) \
		$(APP_NAME):$(APP_VERSION)

app-kill:
	-docker service rm $(APP_NAME)


#-p $(OLLAMA_PORT):$(OLLAMA_PORT)
ollama-start: create-network
	docker run --rm -d \
		--name $(OLLAMA_APP_NAME) \
		--network $(NETWORK) \
		-v ollama_models:/root/.ollama \
		-e OLLAMA_HOST=$(OLLAMA_HOST) \
		-e OLLAMA_PORT=$(OLLAMA_PORT) \
		$(OLLAMA_IMAGE):$(OLLAMA_VERSION)

ollama-kill:
	-docker kill $(OLLAMA_APP_NAME)
	-docker rm -f $(OLLAMA_APP_NAME)


## secrets: create docker secrets from gitignored files in $(SECRETS_DIR),
## generating a random value the first time a file is missing. Idempotent — an
## existing swarm secret is kept (remove it first to rotate).
secrets:
	@mkdir -p $(SECRETS_DIR)
	@test -s $(SECRETS_DIR)/db_password || { umask 077; openssl rand -base64 24 | tr -d '\n' > $(SECRETS_DIR)/db_password; echo "generated $(SECRETS_DIR)/db_password"; }
	@docker secret inspect db_password >/dev/null 2>&1 || docker secret create db_password $(SECRETS_DIR)/db_password

## bootstrap: full from-zero backend stack on the manager node — secrets, a fresh
## (wiped) db, schema, image build, and the backend service. Single-node by
## design (everything pinned to the manager) so the overlay never crosses hosts.
bootstrap: secrets create-network db-fresh image app-start

## db-fresh: DESTRUCTIVE. Remove the postgres service AND its data volume, then
## recreate + wait + load schema. Loses ALL data. Run `make secrets` first.
db-fresh: app-kill
	-docker service rm $(DB_SERVICE)
	@echo "waiting for the old task + volume lock to release..."; sleep 6
	-docker volume rm $(DB_DATA_DOCKER_VOL)
	$(MAKE) db-start
	$(MAKE) db-wait
	$(MAKE) db-schema

#-p "$(DB_PORT):5432"
db-start:
	docker service create -d \
		--name $(DB_SERVICE) \
		--network $(NETWORK) \
		--constraint 'node.role==manager' \
		--secret source=db_password,target=$(DB_PASSWORD_PATH) \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_PASSWORD_FILE=$(DB_PASSWORD_PATH) \
		-e POSTGRES_DB=$(DB_NAME) \
		--mount type=volume,source=$(DB_DATA_DOCKER_VOL),target=/var/lib/postgresql/data \
		$(DB_IMAGE):$(DB_VERSION)

db-kill:
	-docker service rm $(DB_SERVICE)

## db-wait: block until the TARGET database actually answers a query. We probe
## with a real `SELECT 1`, NOT pg_isready: on a fresh volume postgres runs a
## temporary socket-only server during initdb, so pg_isready reports "ready"
## *before* POSTGRES_DB is created — db-schema would then fail with
## "database does not exist". A real query only succeeds once the final server
## is up and $(DB_NAME) exists.
db-wait:
	@echo "waiting for postgres database '$(DB_NAME)' to accept queries..."
	@for i in $$(seq 1 60); do \
		cid=$$(docker ps -qf name=$(DB_SERVICE) | head -n1); \
		if [ -n "$$cid" ] && docker exec $$cid psql -U $(DB_USER) -d $(DB_NAME) -tAc 'select 1' >/dev/null 2>&1; then \
			echo "postgres ready"; exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "postgres database '$(DB_NAME)' did not become ready in 60s" >&2; exit 1

db-schema:	
	docker exec -i $(DB_CID) \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME) \
		-f - < sql/schema.sql

db-drop:
	docker exec -i $(DB_CID) \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME) \
		-f - < sql/drop.sql

create-network:
	-docker network create --driver overlay --attachable $(NETWORK)



# App specific configs
APP_NAME ?= fin-track
APP_VERSION ?= 0.0.0-dev
HOST ?= 0.0.0.0
PORT ?= 8080
NETWORK ?= fin-track

# Public hostname for the cloudflare smoke-test target. Sourced from the
# gitignored traefik/.env so the real hostname never lands in version control.
# Override ad hoc with: make test-all-img-cloudflare PUBLIC_HOSTNAME=host
-include ../traefik/.env

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
DB_NAME ?= fin-track-db
DB_HOST ?= $(DB_NAME)
DB_PORT ?= 5432
DB_USER ?= $(APP_NAME)
DB_PASSWORD ?= secure_password
DB_DATA_DOCKER_VOL ?= postgres_data

# Testing
IMG_PATH ?= @$(shell pwd)/data
IMGS = IMG_0955.jpeg IMG_0961.jpeg IMG_0962.jpeg IMG_0963.jpeg IMG_0964.jpeg IMG_0967.jpeg IMG_0971.jpeg
OTPR_IMGS = IMG_0975.jpeg IMG_0974.jpeg IMG_0973.jpeg IMG_0972.jpeg
RECEIPT_ID = 5

PHONY:$(MAKECMDGOALS)

start: ollama-kill db-kill kill db-start ollama-start  db-schema image app-start

kill: ollama-kill db-kill app-kill

reset: db-drop kill start 

test-categorize:
	@echo "Testing categorize with receipt id $(RECEIPT_ID)"
	curl -X POST http://localhost:8080/api/receipts/$(RECEIPT_ID)/categorize;

test-default:
	@echo "Testing with sample image IMG_0955.jpeg"
	curl -X POST -F "image=$(IMG_PATH)/IMG_0955.jpeg" http://localhost:8080/api/receipts/image;

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

dev-start: kill dev-image
	docker run  -it --rm \
		--name $(APP_NAME) \
		-p $(PORT):$(PORT) \
		--network $(NETWORK) \
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
		--entrypoint=air \
		$(APP_NAME)-dev:latest \
		 -c /tmp/fin-track-src/.air.toml

image:
	docker build -t $(APP_NAME):$(APP_VERSION) .

app-start: create-network app-kill
	docker run  -d \
		--name $(APP_NAME) \
		-p $(PORT):$(PORT) \
		--network $(NETWORK) \
		--label 'traefik.enable=true' \
		--label 'traefik.http.routers.api.rule=PathPrefix(`/api`)' \
		--label 'traefik.http.routers.api.entrypoints=web' \
		--label 'traefik.http.routers.api.priority=100' \
		--label 'traefik.http.routers.api.middlewares=auth-middleware@file' \
		--label 'traefik.http.routers.api.service=backend' \
		--label 'traefik.http.services.backend.loadbalancer.server.port=$(PORT)' \
		-e HOST=$(HOST) \
		-e PORT=$(PORT) \
		-e OLLAMA_HOST=$(OLLAMA_HOST) \
		-e OLLAMA_PORT=$(OLLAMA_PORT) \
		-e DB_NAME=$(DB_NAME) \
		-e DB_HOST=$(DB_HOST) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_USER=$(DB_USER) \
		-e DB_PASSWORD=$(DB_PASSWORD) \
		$(APP_NAME):$(APP_VERSION)

app-kill:
	-docker rm -f $(APP_NAME)


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


#-p "$(DB_PORT):5432" 
db-start: create-network
	docker run -d --rm \
		--network $(NETWORK) \
		--name $(DB_NAME) \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_DB=$(DB_NAME) \
		-v $(DB_DATA_DOCKER_VOL)=/var/lib/postgresql \
		$(DB_IMAGE):$(DB_VERSION)

db-kill:
	-docker kill $(DB_NAME)

db-schema:	
	-docker exec -i $(DB_NAME) \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME) \
		-f - < sql/schema.sql

db-drop:
	-docker exec -i $(DB_NAME) \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME) \
		-f - < sql/drop.sql

create-network:
	-docker network create $(NETWORK)


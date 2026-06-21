# fin-track app specific configs
APP_NAME ?= fin-track
APP_VERSION ?= 0.0.0-dev
HOST ?= 0.0.0.0
PORT ?= 8080
NETWORK ?= fin-track

# Ollama env
OLLAMA_APP_NAME ?= ollama-app 
OLLAMA_HOST ?= $(OLLAMA_APP_NAME)
OLLAMA_PORT ?= 11434
LOG_LEVEL ?= DEBUG
RUNTIME_ENV ?= DEBUG

#DB
DB_NAME ?= fin-track-db
DB_HOST ?= $(DB_NAME)
DB_PORT ?= 5432
DB_USER ?= $(APP_NAME)
DB_PASSWORD ?= secure_password
DB_DATA_DOCKER_VOL ?= postgres_data

# Testing
IMG_PATH ?= @$(shell pwd)/data
# IMG_0955.jpeg
PHONY:$(MAKECMDGOALS)

all: ollama-kill db-kill db-start ollama-start image start

	# docker cp ./schema.sql $(DB_NAME):/tmp/schema.sql
db-shcema:	
	docker exec -i $(DB_NAME) \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME) \
		-f - < sql/schema.sql

db-drop:
	docker exec -i $(DB_NAME) \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME) \
		-f - < sql/drop.sql

# db-reset: db-drop db-shcema
IMGS = IMG_0955.jpeg IMG_0961.jpeg IMG_0962.jpeg IMG_0963.jpeg IMG_0964.jpeg IMG_0967.jpeg IMG_0971.jpeg

test-all-img:
	@echo "Sending all images..."
	@for img in $(IMGS); do \
		echo "Sending $$img..."; \
		curl -X POST \
			-F "image=$(IMG_PATH)/$$img" \
			http://localhost:8080/api/receipts/image; \
		done

test-bad-img-to-ocr:
	@echo "Sending poor quality images..."
	-curl -X POST \
		-F "image=$(IMG_PATH)/IMG_0961.jpeg" \
		http://localhost:8080/api/receipts/image 
	-curl -X POST \
		-F "image=$(IMG_PATH)/IMG_0962.jpeg" \
		http://localhost:8080/api/receipts/image

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
		--network $(NETWORK) \
		-v $(shell pwd):/tmp/fin-track-src \
		-v go-mod-cache:/go/pkg/mod \
		-p $(PORT):$(PORT) \
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


start: create-network kill
	docker run  -d \
		--name $(APP_NAME) \
		--network $(NETWORK) \
		-p $(PORT):$(PORT) \
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

kill:
	-docker rm -f $(APP_NAME)

create-network:
	-docker network create $(NETWORK)

ollama-start: create-network
	docker run --rm -d \
		--name $(OLLAMA_APP_NAME) \
		--network $(NETWORK) \
		-p $(OLLAMA_PORT):$(OLLAMA_PORT) \
		-v ollama_models:/root/.ollama \
		-e OLLAMA_HOST=$(OLLAMA_HOST) \
		-e OLLAMA_PORT=$(OLLAMA_PORT) \
		ollama/ollama:latest


ollama-kill:
	-docker kill $(OLLAMA_APP_NAME)
	-docker rm -f $(OLLAMA_APP_NAME)


db-start: create-network
	docker run -d --rm \
		--network $(NETWORK) \
		-p "$(DB_PORT):5432" \
		--name $(DB_NAME) \
		-e POSTGRES_USER=$(DB_USER) \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_DB=$(DB_NAME) \
		-v $(DB_DATA_DOCKER_VOL)=/var/lib/postgresql \
		postgres:17 


db-kill:
	-docker kill $(DB_NAME)


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
DB_USER ?= local_user
DB_PASSWORD ?= secure_password
DB_DATA_DOCKER_VOL ?= postgres_data

.PHONY:$(MAKECMDGOALS)

image:
	docker build -t $(APP_NAME):$(APP_VERSION) .


start: create-network
	docker run  -d \
		--name $(APP_NAME) \
		--network $(NETWORK) \
		-p "8080:8080" \
		-e HOST=$(HOST) \
		-e PORT=$(PORT) \
		-e OLLAMA_HOST=$(OLLAMA_HOST) \
		-e OLLAMA_PORT=$(OLLAMA_PORT) \
		-e DB_NAME=$(DB_NAME) \
		-e DB_HOST=$(DB_NAME) \
		-e DB_PORT=$(DB_PORT) \
		-e DB_USER=$(DB_HOST) \
		-e DB_PASSWORD=secure_password \
		$(APP_NAME):$(APP_VERSION)

kill:
	docker rm -f $(APP_NAME)

create-network:
	-docker network create $(NETWORK)

ollama-start: create-network
	docker run --rm -d \
		--name $(OLLAMA_APP_NAME) \
		--network $(NETWORK) \
		-p "11434:11434" \
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
		-e POSTGRES_USER=$(DB_HOST) \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-e POSTGRES_DB=$(DB_NAME) \
		-v $(DB_DATA_DOCKER_VOL)=/var/lib/postgresql \
		postgres:17 


db-kill:
	docker kill $(DB_NAME)


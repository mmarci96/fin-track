# Build / publish / run the categorizer service.
#
# Publish to Docker Hub:   make publish              (build + tag :TAG and :latest, push both)
# Run the full bundle:     make up                   (ollama + model puller + categorizer)
# Override anything:       make publish TAG=1.2.0 DOCKER_USER=otheruser

DOCKER_USER ?= mmarci96
IMAGE       ?= $(DOCKER_USER)/categorizer-mcp
# Default tag tracks the package version in pyproject.toml.
TAG         ?= $(shell sed -n 's/^version = "\(.*\)"/\1/p' pyproject.toml)

export IMAGE TAG

.PHONY: build push publish up down restart logs ps pull-models run login clean

## build: build the image as IMAGE:TAG and IMAGE:latest
build:
	docker build -t $(IMAGE):$(TAG) -t $(IMAGE):latest .

## login: authenticate to Docker Hub (prompts for credentials)
login:
	docker login

## push: push both tags to the registry
push:
	docker push $(IMAGE):$(TAG)
	docker push $(IMAGE):latest

## publish: build, tag and push in one step
publish: build push
	@echo "published $(IMAGE):$(TAG) and $(IMAGE):latest"

## up: start the full bundle (ollama + model puller + categorizer) in the background
up:
	docker compose up -d

## down: stop the bundle (keeps the ollama_models volume)
down:
	docker compose down

## restart: recreate just the categorizer (e.g. after pulling a new image)
restart:
	docker compose up -d --no-deps --force-recreate categorizer

## logs: follow the categorizer logs
logs:
	docker compose logs -f categorizer

## ps: show bundle status
ps:
	docker compose ps

## pull-models: (re)pull the CATEGORIZE_PULL_MODELS into the ollama volume
pull-models:
	docker compose run --rm model-puller

## run: standalone container against an EXTERNAL ollama + postgres (no bundle)
##   make run DATABASE_URL=... OLLAMA_HOST=...
run:
	docker run --rm -p $(or $(MCP_PORT),8090):8090 \
		-e MCP_HOST=0.0.0.0 \
		-e OLLAMA_HOST=$(or $(OLLAMA_HOST),host.docker.internal) \
		-e OLLAMA_PORT=$(or $(OLLAMA_PORT),11434) \
		-e CATEGORIZE_CHAT_MODEL=$(or $(CATEGORIZE_CHAT_MODEL),qwen3:1.7b) \
		-e CATEGORIZE_DECIDE_MODEL=$(or $(CATEGORIZE_DECIDE_MODEL),qwen3:4b) \
		-e DATABASE_URL=$(DATABASE_URL) \
		$(IMAGE):$(TAG)

## clean: remove the bundle and the ollama_models volume
clean:
	docker compose down -v

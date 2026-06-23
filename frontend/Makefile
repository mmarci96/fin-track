# fin-track frontend + Traefik edge — docker targets.
# Runs on the SAME network as the backend (default `fin-track`) so Traefik can
# reach both containers by name. No docker-compose.

APP_NAME        ?= fin-track-ui
TRAEFIK_NAME    ?= fin-track-traefik
NETWORK         ?= fin-track
IMAGE           ?= $(APP_NAME):latest
WEB_PORT        ?= 80
DASHBOARD_PORT  ?= 8088
PLAYWRIGHT_IMAGE ?= mcr.microsoft.com/playwright:v1.48.0-noble

.PHONY: start image ui-start traefik-start network kill kill-ui kill-traefik clean dev install build e2e

## start: build the UI image and run UI + Traefik on the backend network
start: network image ui-start traefik-start
	@echo ""
	@echo "  UI:                http://localhost:$(WEB_PORT)"
	@echo "  Traefik dashboard: http://localhost:$(DASHBOARD_PORT)"

reset: kill start

image:
	docker build -t $(IMAGE) .

network:
	-docker network create $(NETWORK)

ui-start: kill-ui
	docker run -d \
		--name $(APP_NAME) \
		--network $(NETWORK) \
		$(IMAGE)

traefik-start: kill-traefik
	docker run -d \
		--name $(TRAEFIK_NAME) \
		--network $(NETWORK) \
		--network cloudflare \
		-p $(WEB_PORT):80 \
		-p $(DASHBOARD_PORT):8080 \
		-v $(CURDIR)/traefik/dynamic.yml:/etc/traefik/dynamic.yml:ro,Z \
		-v $(CURDIR)/traefik/traefik.yml:/etc/traefik/traefik.yml:Z \
		-v $(PWD)/traefik/traefikauth:/plugins-local/src/github.com/mmarci96/traefikauth:Z \
		traefik:v3 \
		--entrypoints.web.address=:80 \
		--api.dashboard=true \
		--api.insecure=true \
		--providers.file.filename=/etc/traefik/dynamic.yml \
		--providers.file.watch=true

kill: kill-traefik kill-ui

kill-ui:
	-docker rm -f $(APP_NAME)

kill-traefik:
	-docker rm -f $(TRAEFIK_NAME)

clean: kill

## dev: run Vite locally (needs node); proxies /api -> localhost:8080
dev:
	npm install && npm run dev

install:
	npm install

build:
	npm run build

## e2e: run Playwright DOM tests against the running edge, via docker
e2e:
	docker run --rm \
		--network $(NETWORK) \
		-e PW_BASE_URL=http://$(TRAEFIK_NAME) \
		-e CI=1 \
		-v $(CURDIR):/work \
		-w /work \
		$(PLAYWRIGHT_IMAGE) \
		sh -c "npm install --no-audit --no-fund && npx playwright test"

auth-service-kill:
	-docker rm -f auth-service

auth-service-run: auth-service-kill
	docker run -d \
		--name auth-service \
		--network $(NETWORK) \
		-p 8081:8000 \
		-v $(PWD)/traefik/auth-service:/app \
		-w /app \
		python:3.11-slim \
		sh -c "pip install -r requirements.txt && uvicorn main:app --host 0.0.0.0 --port 8000"



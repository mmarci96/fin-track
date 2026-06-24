# fin-track frontend + Traefik edge — docker targets.
# Runs on the SAME network as the backend (default `fin-track`) so Traefik can
# reach both containers by name. No docker-compose.
#
#   UAT client = the built nginx image fronted by Traefik (`make start`).
#   DEV client = the Vite dev server proxying /api to the air backend (`make dev`).

APP_NAME        ?= fin-track-ui
NETWORK         ?= fin-track
IMAGE           ?= $(APP_NAME):latest
WEB_PORT        ?= 80
PLAYWRIGHT_IMAGE ?= mcr.microsoft.com/playwright:v1.48.0-noble

.PHONY: start image ui-start traefik-start network kill kill-ui kill-traefik clean dev install build e2e

## start: build the UI image and run UI + Traefik on the backend network
start: image ui-start

reset: kill start

image:
	docker build -t $(IMAGE) .

ui-start: kill
	docker run -d \
		--name $(APP_NAME) \
		--network $(NETWORK) \
		--label 'traefik.enable=true' \
		--label 'traefik.http.routers.web.rule=PathPrefix(`/`)' \
		--label 'traefik.http.routers.web.entrypoints=web' \
		--label 'traefik.http.routers.web.priority=1' \
		--label 'traefik.http.routers.web.middlewares=auth-middleware@file' \
		--label 'traefik.http.routers.web.service=frontend' \
		--label 'traefik.http.services.frontend.loadbalancer.server.port=$(WEB_PORT)' \
		$(IMAGE)

kill:
	-docker rm -f $(APP_NAME)

## dev: run the Vite dev server (needs node), proxying /api to the DEV backend.
## Pair with `make -C ../backend dev` (air hot-reload on host port $(DEV_BACKEND_PORT)).
DEV_BACKEND_PORT ?= 8081
dev:
	npm install && VITE_API_TARGET=http://localhost:$(DEV_BACKEND_PORT) npm run dev

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


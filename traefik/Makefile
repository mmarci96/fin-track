# fin-track Traefik edge — docker targets. Runs on the SAME network as the
# backend (default `fin-track`) so Traefik can reach the app containers by name,
# and reads their `traefik.*` labels via the docker-socket-proxy sidecar.
#
# Secrets and the public hostname come from `.env` (gitignored). Copy the
# template first:  cp dotenv.example .env && $EDITOR .env

# Load real values (JWT_SECRET, PUBLIC_HOSTNAME) and export them to the recipe
# environment / docker run. `-include` so `make` doesn't fail if .env is absent.
-include .env
export

TRAEFIK_NAME       ?= fin-track-traefik
PROXY_NAME         ?= docker-socket-proxy
FIN_TRACK_NETWORK  ?= fin-track
CLOUDFLARE_NETWORK ?= cloudflare
DASHBOARD_PORT     ?= 8088
WEB_PORT           ?= 80
TRAEFIK_DIR        := $(shell pwd)

.PHONY: start kill reset socket-proxy socket-proxy-kill

reset: kill start

## start: bring up the read-only Docker API proxy, then the Traefik edge
start: kill socket-proxy
	docker run -d \
		--name $(TRAEFIK_NAME) \
		--network $(FIN_TRACK_NETWORK) \
		--network $(CLOUDFLARE_NETWORK) \
		-p $(WEB_PORT):80 \
		-p $(DASHBOARD_PORT):8080 \
		-v $(TRAEFIK_DIR)/dynamic.yml:/etc/traefik/dynamic.yml:ro,Z \
		-v $(TRAEFIK_DIR)/traefik.yml:/etc/traefik/traefik.yml:Z \
		-v $(TRAEFIK_DIR)/traefikauth:/plugins-local/src/github.com/mmarci96/traefikauth:Z \
		-e JWT_SECRET=$(JWT_SECRET) \
		-e PUBLIC_HOSTNAME=$(PUBLIC_HOSTNAME) \
		traefik:v3 \
		--entrypoints.web.address=:80 \
		--api.dashboard=true \
		--api.insecure=true \
		--providers.file.filename=/etc/traefik/dynamic.yml \
		--providers.file.watch=true

## socket-proxy: the ONLY container that touches /var/run/docker.sock. It exposes
## a minimal, read-only slice of the Docker API on tcp://docker-socket-proxy:2375
## (containers + networks + events) so Traefik can discover labels without the
## internet-facing edge getting root-equivalent host access. No published port —
## it's reachable only on the internal fin-track network.
socket-proxy: socket-proxy-kill
	docker run -d \
		--name $(PROXY_NAME) \
		--network $(FIN_TRACK_NETWORK) \
		-v /var/run/docker.sock:/var/run/docker.sock:ro \
		-e CONTAINERS=1 \
		-e NETWORKS=1 \
		-e EVENTS=1 \
		tecnativa/docker-socket-proxy

socket-proxy-kill:
	-docker rm -f $(PROXY_NAME)

kill: socket-proxy-kill
	-docker rm -f $(TRAEFIK_NAME)


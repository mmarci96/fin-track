# fin-track Traefik edge — docker targets. Runs on the SAME network as the
# backend (default `fin-track`) so Traefik can reach the app containers by name,
# and reads their `traefik.*` labels via the docker-socket-proxy sidecar.
#
# Secrets and the public hostname come from `.env` (gitignored). Copy the
# template first:  cp dotenv.example .env && $EDITOR .env

# Load real values (JWT_SECRET, PUBLIC_HOSTNAME) and export them to the recipe
# environment / docker run. `-include` so `make` doesn't fail if .env is absent.
-include ../.env
export

TRAEFIK_NAME       ?= homelab-traefik
PROXY_NAME         ?= docker-socket-proxy
FIN_TRACK_NETWORK  ?= homelab
CLOUDFLARE_NETWORK ?= cloudflare
DASHBOARD_PORT     ?= 8088
WEB_PORT           ?= 80
TRAEFIK_DIR        := $(shell pwd)

.PHONY: start kill reset socket-proxy socket-proxy-kill

## start: bring up the read-only Docker API proxy, then the Traefik edge
start: kill socket-proxy
	docker service create -d \
		--name $(TRAEFIK_NAME) \
		--network $(FIN_TRACK_NETWORK) \
		--network $(CLOUDFLARE_NETWORK) \
		--constraint 'node.role==manager' \
		-p $(WEB_PORT):80 \
		-p $(DASHBOARD_PORT):8080 \
		--mount type=bind,source=$(TRAEFIK_DIR)/dynamic.yml,destination=/etc/traefik/dynamic.yml\
		--mount type=bind,source=$(TRAEFIK_DIR)/traefik.yml,destination=/etc/traefik/traefik.yml \
		--mount type=bind,source=$(TRAEFIK_DIR)/traefikauth,destination=/plugins-local/src/github.com/mmarci96/traefikauth \
		--secret=jwt_secret \
		-e PUBLIC_HOSTNAME=$(PUBLIC_HOSTNAME) \
		-e NETWORK=$(FIN_TRACK_NETWORK) \
		traefik:v3

## socket-proxy: the ONLY container that touches /var/run/docker.sock. It exposes
## a minimal, read-only slice of the Docker API on tcp://docker-socket-proxy:2375
## (containers + networks + events) so Traefik can discover labels without the
## internet-facing edge getting root-equivalent host access. No published port —
## it's reachable only on the internal fin-track network.
socket-proxy: socket-proxy-kill
	docker service create -d \
		--name $(PROXY_NAME) \
		--network $(FIN_TRACK_NETWORK) \
		--constraint 'node.role==manager' \
		--mount type=bind,source=/var/run/docker.sock,destination=/var/run/docker.sock \
		-e SERVICES=1 \
		-e TASKS=1 \
		-e NODES=1 \
		-e CONTAINERS=1 \
		-e NETWORKS=1 \
		-e EVENTS=1 \
		tecnativa/docker-socket-proxy

socket-proxy-kill:
	-docker service rm $(PROXY_NAME)

kill: 	
	-docker service rm $(TRAEFIK_NAME)


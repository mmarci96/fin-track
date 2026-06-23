# fin-track monorepo root — delegates to backend/ and frontend/ sub-makefiles
# via `make -C <dir> <target>` so the same command names stay in sync.

.PHONY: start stop clean e2e backend frontend

reset:	
	$(MAKE) -C backend reset 
	$(MAKE) -C frontend reset 

## start: bring up the backend, then the UI + Traefik edge (shared network)
start:
	$(MAKE) -C backend start 
	$(MAKE) -C frontend start

## stop: tear down the UI/edge and the backend app container
stop:
	-$(MAKE) -C frontend kill
	-$(MAKE) -C backend kill 

clean: stop

## e2e: run the frontend Playwright DOM tests
e2e:
	$(MAKE) -C frontend e2e

backend-test:
	$(MAKE) -C backend test-all-img 

## backend / frontend: pass an arbitrary target through, e.g.
##   make backend CMD=db-schema
##   make frontend CMD=build
backend:
	$(MAKE) -C backend $(CMD)

frontend:
	$(MAKE) -C frontend $(CMD)

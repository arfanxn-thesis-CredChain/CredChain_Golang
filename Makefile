# CredChain — orchestrator + backend
# Run everything from here. Prod = full Docker. Local = hybrid (infra in Docker,
# Go via air + React via vite on host).

ENV_FILE ?= .env
# NOTE: do not `include`+`export` .env here. setup-contracts.py rewrites contract
# addresses into .env mid-run; make's export would freeze the pre-deploy values in
# each recipe's env, and godotenv.Load (config.go) won't override them, so `go run
# ... --env .env` would read stale addresses. Recipes load .env themselves via --env.

.PHONY: help local-up build-push prod-up down local-fresh prod-fresh logs backup restore \
	dev-up dev-fresh dev test lint get-google-id-token \
	credential-extraction-benchmark wait-golang

help:
	@echo "CredChain — make targets"
	@echo ""
	@echo "Stack (Docker):"
	@echo "  local-up   Full stack in Docker: build amd64+arm64, push to Docker Hub, pull back and run (Mac smoke test + release)"
	@echo "  prod-up    Full stack in Docker, pulling prebuilt images (VPS; builds nothing)"
	@echo "  down       Stop all containers"
	@echo "  local-fresh Wipe volumes + uploads, then local-up"
	@echo "  prod-fresh  Wipe volumes + uploads, then prod-up (recover fresh/partial chain)"
	@echo "  logs       Follow backend/infra logs"
	@echo "  backup    Dump Postgres + Mongo + uploads      (BACKUP=<ts>)"
	@echo "  restore   Restore from a backup                 (BACKUP=<ts>)"
	@echo ""
	@echo "Local (hybrid):"
	@echo "  dev-up    Infra + Anvil + Python in Docker, deploy contracts, migrate, seed"
	@echo "  dev-fresh Wipe local infra data + uploads, then dev-up (clean chain + reseed)"
	@echo "  dev       Run Go backend hot-reload (air). Then run 'make dev' in CredChain_React."
	@echo ""
	@echo "Backend:"
	@echo "  test      go test ./..."
	@echo "  lint      go vet + gofmt"
	@echo "  get-google-id-token          Obtain Google ID token (Postman)"
	@echo "  credential-extraction-benchmark   Run extraction benchmark"

# ---------------------------------------------------------------- full stack (Docker)

local-up:
	-docker network create credchain
	docker compose up -d anvil postgres mongo
	python3 scripts/setup-contracts.py
	docker buildx build --platform linux/amd64,linux/arm64 -t arfanxn/credchain-golang:latest --push .
	docker compose pull golang
	docker compose run --rm golang ./server migrate up
	-docker compose run --rm golang ./server migrate-mongo up
	docker compose up -d --no-build golang
	@$(MAKE) wait-golang
	-docker compose exec golang ./server init-super-admin
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg VITE_GOOGLE_CLIENT_ID="$$(grep -E '^VITE_GOOGLE_CLIENT_ID=' ../CredChain_React/.env.docker | cut -d= -f2-)" \
		--build-arg VITE_APP_ENV=production \
		--build-arg VITE_SUPPORT_EMAIL="$$(grep -E '^VITE_SUPPORT_EMAIL=' ../CredChain_React/.env.docker | cut -d= -f2-)" \
		-t arfanxn/credchain-react:latest --push ../CredChain_React
	cd ../CredChain_React && docker compose --env-file .env.docker pull && docker compose --env-file .env.docker up -d --no-build
	docker compose up -d nginx
	docker buildx build --platform linux/amd64,linux/arm64 -t arfanxn/credchain-python:latest --push ../CredChain_Python
	cd ../CredChain_Python && docker compose pull && docker compose up -d --no-build
	-docker image prune -f
	@echo "local-up: all services built (amd64+arm64), pushed, pulled, and started"

build-push:
	docker buildx build --platform linux/amd64,linux/arm64 -t arfanxn/credchain-golang:latest --push .
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg VITE_GOOGLE_CLIENT_ID="$$(grep -E '^VITE_GOOGLE_CLIENT_ID=' ../CredChain_React/.env.docker | cut -d= -f2-)" \
		--build-arg VITE_APP_ENV=production \
		--build-arg VITE_SUPPORT_EMAIL="$$(grep -E '^VITE_SUPPORT_EMAIL=' ../CredChain_React/.env.docker | cut -d= -f2-)" \
		-t arfanxn/credchain-react:latest --push ../CredChain_React
	docker buildx build --platform linux/amd64,linux/arm64 -t arfanxn/credchain-python:latest --push ../CredChain_Python
	@echo "build-push: all 3 images built (amd64+arm64) and pushed"

prod-up:
	-docker network create credchain
	docker compose pull anvil postgres mongo
	docker compose up -d anvil postgres mongo
	python3 scripts/setup-contracts.py
	docker compose pull golang
	docker compose run --rm golang ./server migrate up
	-docker compose run --rm golang ./server migrate-mongo up
	docker compose up -d --no-build golang
	@$(MAKE) wait-golang
	-docker compose exec golang ./server init-super-admin
	cd ../CredChain_React && docker compose --env-file .env.docker pull && docker compose --env-file .env.docker up -d --no-build
	docker compose pull nginx
	docker compose up -d nginx
	cd ../CredChain_Python && docker compose pull && docker compose up -d --no-build
	-docker image prune -f
	@echo "prod-up: all services started (pulled images)"

down:
	-cd ../CredChain_React && docker compose down
	-cd ../CredChain_Python && docker compose down
	docker compose down

local-fresh:
	@$(MAKE) down
	rm -rf docker/postgres/data/* docker/mongo/data/* docker/anvil/data/* uploads/*
	@$(MAKE) local-up

prod-fresh:
	@$(MAKE) down
	rm -rf docker/postgres/data/* docker/mongo/data/* docker/anvil/data/* uploads/*
	@$(MAKE) prod-up

logs:
	docker compose logs -f

# ---------------------------------------------------------------- local (hybrid)

dev-up:
	-docker network create credchain
	docker compose up --wait --wait-timeout 120 anvil postgres mongo
	cd ../CredChain_Python && docker compose up -d --build
	ENV_FILE=.env python3 scripts/setup-contracts.py
	go run main.go migrate up --env .env
	-go run main.go migrate-mongo up --env .env
	go run main.go seed --env .env
	go run main.go seed-chain --env .env
	-docker image prune -f
	@echo "dev-up ready. Run 'make dev' here, and 'make dev' in CredChain_React."

dev-fresh:
	-cd ../CredChain_Python && docker compose down
	docker compose down
	rm -rf docker/postgres/data/* docker/mongo/data/* docker/anvil/data/* uploads/*
	@$(MAKE) dev-up

dev:
	@command -v air >/dev/null 2>&1 || { echo "error: air not installed. Run: go install github.com/air-verse/air@latest"; exit 1; }
	@PORT=$$(grep -E '^GIN_PORT=' $(ENV_FILE) | cut -d= -f2); \
	if [ -n "$$PORT" ]; then echo "freeing port $$PORT"; kill $$(lsof -ti tcp:$$PORT) 2>/dev/null || true; fi
	air -c .air.toml

# ---------------------------------------------------------------- backend tasks

test:
	go test ./...

lint:
	go vet ./... && gofmt -l .

get-google-id-token:
	go run main.go get-google-id-token --env $(ENV_FILE)

CREDENTIAL_EXTRACTION_BENCH_COUNT ?= 3
CREDENTIAL_EXTRACTION_BENCH_DIRECTORY ?= benchmarks/credential-extraction
CREDENTIAL_EXTRACTION_BENCH_CSV ?=
CREDENTIAL_EXTRACTION_BENCH_PROFILE ?=

credential-extraction-benchmark:
	mkdir -p $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)
	@FLAGS="" && \
	if [ -n "$(CREDENTIAL_EXTRACTION_BENCH_CSV)" ]; then FLAGS="$$FLAGS --output $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/results.csv"; fi && \
	if [ -n "$(CREDENTIAL_EXTRACTION_BENCH_PROFILE)" ]; then FLAGS="$$FLAGS --cpuprofile $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/cpu.prof --memprofile $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/mem.prof --trace $(CREDENTIAL_EXTRACTION_BENCH_DIRECTORY)/trace.out"; fi && \
	go run main.go credential-extraction-benchmark \
		--count $(CREDENTIAL_EXTRACTION_BENCH_COUNT) \
		$$FLAGS \
		--env $(ENV_FILE)

# ---------------------------------------------------------------- backup / restore
# Creds read from each container's own env (.env.docker), so no Makefile var juggling.

BACKUP ?= $(shell date +%Y%m%d_%H%M%S)

backup:
	@mkdir -p docker/backups
	docker compose exec -T postgres sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_dump -Fc -U $$POSTGRES_USER $$POSTGRES_DB' > docker/backups/pg_$(BACKUP).dump
	docker compose exec -T mongo sh -c 'mongodump --archive -u $$MONGO_INITDB_ROOT_USERNAME -p $$MONGO_INITDB_ROOT_PASSWORD --authenticationDatabase admin' > docker/backups/mongo_$(BACKUP).archive
	tar czf docker/backups/uploads_$(BACKUP).tar.gz uploads
	@echo "backup $(BACKUP)"

restore:
	@test -n "$(BACKUP)" || { echo "error: set BACKUP=<timestamp>, e.g. make restore BACKUP=20260725_120000"; exit 1; }
	docker compose exec -T postgres sh -c 'PGPASSWORD=$$POSTGRES_PASSWORD pg_restore -U $$POSTGRES_USER -d $$POSTGRES_DB --clean' < docker/backups/pg_$(BACKUP).dump
	docker compose exec -T mongo sh -c 'mongorestore --archive --drop -u $$MONGO_INITDB_ROOT_USERNAME -p $$MONGO_INITDB_ROOT_PASSWORD --authenticationDatabase admin' < docker/backups/mongo_$(BACKUP).archive
	tar xzf docker/backups/uploads_$(BACKUP).tar.gz
	@echo "restore $(BACKUP) done"

# ---------------------------------------------------------------- helpers

wait-golang:
	@echo "waiting for golang to be healthy..."
	@for i in 1 2 3 4 5 6 7 8 9 10; do \
		if docker compose ps golang | grep -q "(healthy)"; then echo "golang is healthy"; exit 0; fi; \
		echo "waiting... ($$i/10)"; sleep 3; \
	done; \
	echo "error: golang did not become healthy in time"; exit 1

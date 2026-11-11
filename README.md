# CredChain — Go Backend API

Production backend for the CredChain decentralized credential platform.

## Stack

Go 1.25 · Gin v1.12 · GORM v1.31 (PostgreSQL 15 + SQLite for tests) · MongoDB 8.0 · go-ethereum v1.17 · Uber FX · Cobra CLI · River job workers

## Quick Start

```bash
cp .env.example .env
# Fill in required vars (see .env.example for full list)
make dev-up    # infra + Anvil + Python in Docker, deploy contracts, migrate, seed
make dev       # run the API on host with hot reload
```

The API starts on port 8080. `make dev-up` brings up the PostgreSQL, MongoDB, and Anvil containers it depends on. For the full-Docker stack instead, use `make local-up`.

## Project Structure

```
CredChain_Golang/
├── cmd/              # Cobra CLI commands
├── config/           # Env config via godotenv + startup validation
├── domain/           # Domain entities, error codes, query types
├── feature/          # Business logic per domain (auth, user, credential, notification, overview, meta)
├── infrastructure/   # Database, HTTP router, chain bindings, AI client, jobs, i18n, logging
├── locales/          # en.json + id.json
├── scripts/          # setup-contracts.py (required by Makefile contract deploy)
├── tests/            # Test helpers, fixtures, gintest, mocks
├── benchmarks/        # credential extraction benchmark harness
├── docs/              # project documentation records
└── docker/           # Compose bind-mount data and configuration: postgres, mongo, anvil, nginx
```

## Key Commands

This repo is the **orchestrator** for the whole monorepo. Run `make help` for the full list.

| Command | Purpose |
|---|---|
| `make local-up` | Build and push multi-platform images, pull and run the Docker stack, deploy contracts, migrate, initialize super-admin |
| `make prod-up` | Full stack from prebuilt images (VPS deploy; builds nothing) |
| `make prod-fresh` | Wipe volumes + uploads, then `prod-up` |
| `make down` | Stop all containers |
| `make local-fresh` | Wipe persisted database/chain data + uploads, then `local-up` |
| `make logs` | Follow backend/infra logs |
| `make dev-up` | Local hybrid: infra + Anvil + Python in Docker, deploy contracts, migrate, seed |
| `make dev-fresh` | `dev-up` after wiping local infra data + chain (clean reset + reseed) |
| `make dev` | Run API on host with hot reload (requires air) |
| `make test` | Run all tests |
| `make lint` | `go vet` + `gofmt` |
| `make backup` / `make restore` | Dump / restore Postgres + Mongo + uploads (`BACKUP=<ts>`) |

Migrate and super-admin initialization run inside `local-up`/`prod-up`; `dev-up` runs migrate, seed, and seed-chain. For a standalone run, call the CLI: `go run main.go <cmd> --env .env`.

## Deployment

Documented deployment target: **https://credchain.web.id** (Tencent VPS); current availability is not established by this checkout. `prod-up` pulls service images, but contract compilation/deployment can run on the host for a fresh chain. Existing databases require the separately approved transition described in [DATABASE.md](DATABASE.md).

```bash
# On an authorized VPS with compatible schema: pull images, deploy contracts if needed,
# migrate, initialize super-admin, and start services
make prod-up

# Redeploy after code changes: build+push from your Mac, then pull+run on the VPS
make build-push                        # Mac
git pull && make prod-up               # VPS
```

Full step-by-step runbook (VPS bootstrap, DNS, TLS/certbot, and hard-won gotchas) → **[DEPLOYMENT.md](DEPLOYMENT.md)**.

## Related Docs

- [DEPLOYMENT.md](DEPLOYMENT.md) — VPS deployment runbook + production gotchas
- [AGENTS.md](AGENTS.md) — Full architecture, patterns, conventions, deployment
- [ROLE.md](ROLE.md) — Role hierarchy, authorization matrix, user policy rules
- [CREDENTIAL.md](CREDENTIAL.md) — Credential lifecycle, verification pipeline, storage
- [DATABASE.md](DATABASE.md): PostgreSQL and MongoDB schemas, migrations, indexes

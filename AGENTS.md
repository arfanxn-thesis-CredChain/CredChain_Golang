# CredChain Golang - Agent Instructions

Backend API for the CredChain decentralized credential platform. Cobra CLI + Gin HTTP server using Domain-Driven Design (DDD). Persists to PostgreSQL (GORM) and MongoDB. Manages on-chain roles and credentials via go-ethereum bindings. Wired with Uber FX dependency injection. Internationalized responses via go-i18n. Handles Google OAuth authentication via httpOnly cookies.

This file is the authoritative reference for AI assistants and engineers working in `CredChain_Golang/`.

## Repo Position

Sibling to `CredChain_Solidity/` (smart contracts), `CredChain_React/` (frontend), and `CredChain_Python/` (AI service).

- **`CredChain_Solidity/`:** abigen-generated contract bindings live at `infrastructure/chain/contracts/{authority.go,registry.go}`. These are artifacts — **never hand-edit**. Go-side `chain.AuthorityBinding` / `chain.RegistryBinding` interfaces are satisfied structurally by the abigen output.
- **`CredChain_React/`:** sole HTTP consumer of this API. All routes live under `/api`. Frontend communicates via httpOnly cookies set by `/api/auth/google` and `/api/auth/refresh`.
- **`CredChain_Python/`:** AI service called over HTTP for OCR, extraction, similarity. The Go backend serializes requests to it (single-worker on the Python side).

## Documentation Map

| Doc | Purpose |
|---|---|
| **AGENTS.md** (this file) | Project overview, commands, architecture, patterns, deployment |
| [**ROLE.md**](ROLE.md) | Role hierarchy, authorization matrix, user policy rules |
| [**CREDENTIAL.md**](CREDENTIAL.md) | Credential lifecycle, verification pipeline, credential policies |

## Critical Commands

```bash
cd CredChain_Golang

# This repo is the orchestrator for the whole monorepo.

# Full stack (Docker / prod)
make local-up                             # network, contracts, migrate, super-admin, all services
make prod-up                        # full stack from prebuilt images (VPS; builds nothing)
make down                           # Stop all containers (React + Python + Golang/infra)
make local-fresh                          # Wipe volumes + uploads, then up
make logs                           # Follow backend/infra logs
make backup BACKUP=<ts>             # Dump Postgres + Mongo + uploads
make restore BACKUP=<ts>            # Restore from a backup

# Local hybrid (infra in Docker, Go on host)
make dev-up                         # Infra + Anvil + Python in Docker, deploy contracts, migrate, seed
make dev-fresh                      # dev-up after wiping local infra data + chain (clean reset + reseed)
make dev                            # Hot reload via Air (requires `go install github.com/air-verse/air@latest`)

# Backend tasks
make test                           # go test ./...
make lint                           # go vet + gofmt
make get-google-id-token            # Obtain Google ID token via OAuth (for Postman)
make credential-extraction-benchmark

# migrate / seed / seed-chain / init-super-admin are NOT standalone targets —
# they run automatically inside `make local-up` (init-super-admin) and `make dev-up`
# (seed + seed-chain). For the rare standalone need, call the CLI directly:
go run main.go migrate up --env .env          # (also provisions River tables)
go run main.go migrate-mongo up --env .env    # MongoDB index migrations
go run main.go seed --env .env
go run main.go seed-chain --env .env          # register seeded roles on-chain
go run main.go init-super-admin --env .env

go test ./...                       # Run all tests
go test ./domain/... -v             # Run single package tests
go test ./feature/user/... -v       # Run feature package tests
go test -race ./...                 # Race detector
go test -cover ./...                # Coverage summary
go test -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out    # Coverage HTML report
go vet ./...                        # Static analysis (must produce zero output)
gofmt -l .                          # Format check (must produce zero output)
git log --oneline -5                # Recent commits
```

Hot reload via Air: config in `.air.toml` — watches `**/*.go`, excludes `vendor/`, `docker/`, `tmp/`, `bin/`, 1s debounce.

All CLI commands use `go run main.go <command> --env <path>` (the `--env`/`-e` flag is required).

No CI pipeline is configured. `make lint` = `go vet ./... && gofmt -l .`; there is no separate typecheck step.

## Environment Setup

**Local hybrid run** (infra in Docker, Go on host — uses `.env`):

```bash
cd CredChain_Golang
cp .env.example .env                    # create .env from template (first time)
make dev-up                             # infra + Anvil + Python, deploy contracts, migrate, seed
make dev                                # run Go app locally with hot reload
```

**Full Docker run** (prod — uses `.env.docker`, hostnames `postgres`, `mongo`, `anvil`):

```bash
make local-up                                 # brings the whole stack up
make local-fresh                              # same, but wipes persisted data first
```

**Note:** `init-super-admin` and `seed` both create a SuperAdmin (`arfan2173@gmail.com`) and are **mutually exclusive** — but the two modes keep them apart automatically:
- **`make local-up`** (prod bootstrap) runs `init-super-admin` (one SuperAdmin).
- **`make dev-up`** (dev seeding) runs `seed` → `seed-chain` (15 test users + on-chain roles).

**Required env vars** (app exits at startup if empty or invalid):

| Var | Constraint |
|---|---|
| `GIN_PORT` | HTTP port (default 8080) |
| `JWT_SECRET` | fatal if empty |
| `WALLET_ENCRYPTION_KEY` | **must be exactly 32 bytes** (AES-256); fatal if wrong length |
| `FILE_ENCRYPTION_KEY` | **must be exactly 32 bytes** (AES-256); fatal if wrong length — for credential file at-rest encryption |
| `GEMINI_API_KEY` | no fatal check, but AI features break |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret |
| `GOOGLE_REDIRECT_URI` | default `http://localhost:3000/google/callback` |
| `GIN_CORS_*` | 6 vars (origins, methods, headers, credentials, max_age, expose_headers) |
| `COOKIE_DOMAIN`, `COOKIE_SECURE`, `COOKIE_SAMESITE` | httpOnly cookie config |

**Super admin init env vars** (optional; used only by `init-super-admin`):

| Var | Purpose |
|---|---|
| `INITIAL_SUPER_ADMIN_EMAIL` | required for init-super-admin |
| `INITIAL_SUPER_ADMIN_PRIVATE_KEY` | 64-char hex with 0x prefix; required for init-super-admin |
| `INITIAL_SUPER_ADMIN_NAME`, `_NUMBER`, `_PHONE_NUMBER`, `_BIRTH_DATE`, `_GENDER`, `_META` | optional profile fields |

CLI flags take priority over env vars when both are supplied.

## Project Architecture

```
CredChain_Golang/
  main.go               → entry: calls cmd.Execute(), log.Fatal on error
  cmd/                  → Cobra CLI commands
    root.go             → registers --env/-e flag
    context.go          → ConfigFromCmd, NewConfigFromCmd helpers
    server.go           → FX app wiring for HTTP server
    migrate.go          → inline migrateUp / migrateDown (private helpers)
    init_super_admin.go → initSuperAdmin* helpers, FX lifecycle, explicit shutdown
    get_google_id_token.go → CLI to obtain ID token via Authorization Code Flow
  config/               → env loading (godotenv), required-field validation (pointer types)
  domain/               → entities, value objects, interfaces, error codes
    codes.go            → 6-digit AABBCC response codes
    errors.go           → domain error wrapper + metadata
    user.go             → User entity, Role enum (None/Holder/Issuer/Admin/SuperAdmin)
    user_unit.go        → UserUnit entity (self-referencing org tree)
    credential.go       → Credential entity, Status()/ExtractState() derivations
    credential_type.go  → CredentialType taxonomy entity
    credential_issuer_organization.go → CredentialIssuerOrganization taxonomy entity
    competency.go       → Competency taxonomy entity
    competency_credential.go → Competency↔Credential join entity
    credential_extraction.go    → Mongo extraction document
    credential_verification.go  → Mongo verification audit document
    wallet.go           → Wallet struct + WalletFromUser()
    uow.go              → UnitOfWork interface
    query.go + query/   → query DSL + parser for filters/sorts/pagination
  feature/              → business logic per domain (DDD)
    auth/               → auth_handler.go, auth_service.go, auth_request.go (+ tests)
    user/               → user_handler.go, user_service.go, user_policy.go,
                          user_request.go, gorm_user_repository.go,
                          gorm_user_token_repository.go,
                          user_unit_handler.go, user_unit_service.go,
                          user_unit_request.go, gorm_user_unit_repository.go (+ tests)
    credential/         → credential_handler.go, credential_service.go,
                          gorm_credential_repository.go, credential_policy.go,
                          credential_request.go, mock_credential_service_test.go (+ tests, 67% coverage)
                          taxonomy CRUD: credential_type_*.go, issuer_organization_*.go,
                          competency_*.go + their gorm repositories,
                          gorm_competency_credential_repository.go
                          mongo_credential_extraction_repository.go,
                          mongo_credential_verification_repository.go
    meta/               → meta_handler.go, meta_service.go (+ tests)
    overview/           → overview_handler.go, overview_service.go,
                          gorm_overview_repository.go (+ tests)
  infrastructure/
    ai/pyai/client.go   → Python AI service HTTP client (no tests)
    jobs/               → River workers for async credential extraction
    chain/              → go-ethereum chain infrastructure
      contracts/        → abigen-generated bindings (DO NOT EDIT)
      bindings.go       → AuthorityBinding + RegistryBinding interfaces
      authority_service.go → AuthorityService interface + implementation
      client.go         → RPC + contract client facade
      address.go        → mustHexToAddress, validateHexToAddress
      signer.go         → signature packing utilities
    crypto/             → AES-256 encryption (encryption.go), random hex (token.go)
    database/gorm/      → GORM setup, UnitOfWork, gorm_context
      helpers.go        → shared filter/sort/pagination/CASE update helpers (ApplyFilters, ApplySorts, ApplyPagination, BuildCaseColumnSQL, BuildBatchUpdateSQL)
      model/            → GORM structs: User, UserToken, UserUnit, Credential,
                          CredentialType, CredentialIssuerOrganization,
                          Competency, CompetencyCredential
    database/migrations → 000001_initial_schema.up.sql / .down.sql
    database/seeder/    → seeder.go (Seeder interface + Registry runner),
                          user_unit_seeder.go, user_seeder.go (15 users),
                          credential_type_seeder.go, credential_issuer_organization_seeder.go,
                          competency_seeder.go, credential_seeder.go,
                          pdf.go (deterministic PDF bytes), ulid.go (deterministic ULIDs)
    http/               → router.go + sub-packages
      context/          → auth user injection helpers
      middleware/        → AuthMiddleware, ErrorLoggerMiddleware,
                           I18nMiddleware, RateLimitMiddleware (4 types)
      request/query/    → query DSL parser (HTTP layer)
      responder/        → mapper.go (code→message key), responder.go
      response/         → Auth, User, Pagination DTOs + factory methods
    i18n/bundle.go      → go-i18n bundle factory
    logger/logger.go    → Zap logger
    oauth/google.go     → GoogleOAuthClient interface + impl
    security/jwt.go     → JWT generate/verify
    storage/            → local.go, ipfs.go (no tests). Storage holds *Config, methods
                          use all-in-one `path` argument: SaveBytes/SaveFile/ReadBytes/Delete(path).
                          Base directory from STORAGE_PATH env (default "uploads").
  locales/en.json, locales/id.json   → tracked i18n locale files
  tests/              → test-only helpers (never imported by production)
    db/sqlite.go      → in-memory SQLite via glebarez (pure Go, no CGO)
    fixtures/         → NewDomainUser, NewModelUser, NewDomainUserToken, NewWallet, TestWalletEncryptionKey
    gintest/          → NewContext, LoadTestI18nBundle
    mocks/            → testify mocks: repos, UoW, AuthorityService, UserPolicy, GoogleOAuthClient, bindings
  Makefile              → 22+ targets
  Dockerfile            → multi-stage Go 1.25-alpine → alpine:3.19
  docker-compose.yml    → backend + nginx + postgres:16 + mongo:8.0
  .air.toml             → hot reload config
  go.mod                → module CredChain_Golang (underscore), Go 1.25.1
  CredChain_postman_collection.json → API testing collection
```

## Key Patterns & Conventions

### Cobra CLI Entry Point

`cmd/root.go` registers a required `--env`/`-e` flag. All subcommands obtain config via `ConfigFromCmd(cmd)` or `NewConfigFromCmd(cmd)` (defined in `cmd/context.go`). `main.go` is a thin wrapper that calls `cmd.Execute()` and uses `log.Fatal` for startup errors.

**Helper function naming:** all helper functions in `cmd/` use the command name as prefix (e.g., `initSuperAdmin*` in `init_super_admin.go`, `getGoogleIdToken*` in `get_google_id_token.go`).

**Config helper functions:** `getEnv`, `getIntEnv`, `getBoolEnv`, `getStringSliceEnv` for standard types; `getTimeEnv` (ISO 8601), `getJSONEnv` (JSON object, returns nil if empty) for super admin initialization.

**Config pointer types:** all `Config` struct fields are pointers (`*string`, `*int`, `*bool`, `*[]string`); `nil` = not provided, non-nil = provided (from env or default); consumers dereference with `*cfg.Field`. CORS slice fields (`GinCorsAllowOrigins`, etc.) are `[]string` (not pointers) since slices are already nilable.

### FX Dependency Injection

Uber FX wires every component. `cmd/server.go` declares the full provider list. CLI subcommands instantiate trimmed FX apps with explicit shutdown (e.g., `init-super-admin` exits after initialization completes rather than blocking).

**Logger:** all CLI commands use `infraLogger.NewZapLogger()` via FX injection with `ZapLoggerParams`. Never use `zap.NewProduction()` directly. Configurable via `LOG_LEVEL` (debug/info/warn/error, default info) and `LOG_OUTPUT` (stdout or file path, default stdout).

**FX-injectable middlewares:** every middleware (`ErrorLoggerMiddleware`, `I18nMiddleware`, rate limiters, role middlewares) uses a named wrapper type + `Params` struct with `fx.In` + exported factory function. They are injected into `RouteParams` in `infrastructure/http/router.go`.

### Feature Internal Structure

Each domain folder under `feature/` follows the same five-file pattern:

- `<feature>_handler.go` — Gin HTTP handlers (exported interface + unexported impl)
- `<feature>_service.go` — business logic (exported interface + unexported impl)
- `<feature>_policy.go` — authorization / role rules (exported interface + unexported impl)
- `<feature>_request.go` — request DTOs with Ozzo validation (`Validate()` method)
- `<feature>_request_test.go` — request validation tests
- `gorm_<entity>_repository.go` — GORM repository implementations (unexported struct, exported factory)

### Naming Convention (Interface + Impl + Factory)

- Interfaces are **exported** (`UserService`, `AuthorityService`, `GoogleOAuthClient`).
- Implementations are **unexported** (`userService`, `authorityService`, `googleOAuthClient`).
- Factories are **exported** (`NewUserService`, `NewAuthorityService`, `NewGoogleOAuthClient`).
- Request DTOs are **exported with feature prefix** (`AuthGoogleLoginRequest`, `UserStoreRequest`, `UserUpdateSelfProfileRequest`).

### GORM Model Mapping

`infrastructure/database/gorm/model/` contains GORM-specific structs (`User`, `UserToken`, `Credential`) that map domain entities to database tables.

- `model.User.Meta` uses `gorm:"type:jsonb;serializer:json"` — Postgres stores native JSONB, SQLite stores as TEXT, GORM handles round-tripping.
- `model.UserToken.Type` uses Postgres ENUM `user_token_type` (stored as TEXT in SQLite).
- `model.User` has `gorm.DeletedAt` field for soft delete.
- All models expose `ToDomain()` and `FromDomain()` mapping methods.

### Shared GORM Helpers

`infrastructure/database/gorm/helpers.go` provides five exported functions shared by all GORM repositories:

| Helper | Purpose |
|--------|---------|
| `ApplyFilters(db, filters, allowedColumns, columnPrefix)` | Maps all 14 `domainQuery.Filter` operators to GORM Where clauses, gated by a column allowlist with optional table prefix |
| `ApplySorts(db, query, allowedColumns, defaultSort, mapper, tiebreaker)` | Applies sort clauses with column allowlist, falls back to `defaultSort` when no sorts or all filtered out; supports column name translation via `mapper`; appends `tiebreaker` (e.g., `"id ASC"`) for deterministic pagination |
| `ApplyPagination(db, query)` | Applies `LIMIT`/`OFFSET` when query has pagination |
| `BuildCaseColumnSQL(idColumn, col, pairs)` | Generates `col = CASE idColumn WHEN ? THEN ? ELSE col END` for batch partial UPDATEs |
| `BuildBatchUpdateSQL(table, idColumn, clauses, args, ids, extra...)` | Assembles complete `UPDATE table SET ... WHERE idColumn IN (?)` batch statement with optional extra set clauses (e.g., `updated_at = CURRENT_TIMESTAMP`) |

Repos import via `gormhelpers "CredChain_Golang/infrastructure/database/gorm"` to avoid naming conflict with `gorm.io/gorm`. Tests for helpers live in `helpers_test.go`. Import cycle in `uow_test.go` was resolved by moving it to external test package (`package gorm_test`).

### Unit of Work

`domain.UnitOfWork` wraps multi-repo transactions. `GormUnitOfWork.Execute(ctx, fn)` creates transaction-scoped repositories via factory functions and passes a fresh UoW into `fn`. Roll back on error; commit on nil return.

Mock variant `mocks.PropagatingUnitOfWork` (wraps `MockUnitOfWork`) actually runs the inner function and propagates its error — needed for testing chain-failure rollback paths where the default `RunUnitOfWorkFn` would swallow inner errors.

### Request DTOs & Ozzo Validation

DTOs validate with Ozzo (`Validate()` method). The `ToDomain()` pattern converts them into domain entities — see `UserStoreInput` and `UserStoreRequest`.

**Date fields use `*string` + `validation.Date("2006-01-02")`** rather than `*time.Time` (default `time.Time` JSON unmarshalling only accepts RFC3339). `ToDomain()` then parses the string into `*time.Time`.

**DB-constraint validation:** request DTOs (`UserStoreInput`, `UserUpdateInput`, `UserUpdateSelfEmailRequest`) enforce DB column maxima — name (1–256), number (0–256), unit_id (0–26), email (1–256 + `is.Email`), gender (`validation.In("male", "female")`). Defined in `feature/user/user_request.go`.

### Response Envelope & 6-Digit Codes

Every HTTP response uses `{code, message, data?}` envelope. Helpers:

- `responder.Send(c, code, data)` — success
- `responder.SendPagination(c, code, items, total, page, perPage)` — paginated success
- `responder.SendError(c, err)` — error (auto-resolves code → message via i18n)
- `responder.SendValidationError(c, errors)` — Ozzo validation errors
- **Service-level validation.Errors dispatch:** Services can return `validation.Errors` directly for business-logic validation beyond request-DTO validation (e.g., `userService.storeValidateEmails` checks for duplicate emails across a batch, `credentialService.validateIssueCredential` checks for duplicate file hashes). Handlers catch these via `if verrs, ok := err.(validation.Errors); ok { responder.SendValidationError(c, verrs) }`. This produces the `{code, message, errors: {...}}` response envelope with per-field error messages, HTTP 400. See `feature/user/user_handler.go` and `feature/credential/credential_handler.go` for handler dispatch patterns.

Response codes follow a 6-digit `AABBCC` format defined in `domain/codes.go`. Categories: `10` (system), `20` (auth), `30` (user), `40` (credential — sub-categories 01=fetch, 02=issue, 03=revoke, 04=verify, 05=reextract, 06=file download). The `50` category is owned by `CredChain_Python` (AI service) and propagates through this backend untouched.

`SendError` detects malformed-body errors (`io.EOF`, `io.ErrUnexpectedEOF`, `*json.SyntaxError`, `*json.UnmarshalTypeError` from `c.ShouldBindJSON`) via `isMalformedBodyError` helper and returns `CodeSystemValidation` (400) instead of falling through to `CodeSystemInternal` (500).

**Response DTOs** live in `infrastructure/http/response/`:
- `response.User` — mirrors `domain.User` minus `EncryptedWalletPrivateKey`; `Name` is `*string`; includes `Number`, `UnitID`, `JoinedYear`, `BirthDate`, `Gender`, `Meta`, `DeletedAt` (`response/user.go`).
- `response.Auth` — embeds `response.User` with `json:",inline"` + `AccessToken`, `RefreshToken`, `AccessTokenExpiresIn`, `RefreshTokenExpiresIn`, `TokenType`.
- Factory: `response.NewAuth(user, accessToken, refreshToken, accessExpirySec, refreshExpirySec)` — `TokenType` hardcoded to `"Bearer"`.

### i18n & Locale Key Enforcement

`responder.Send` / `SendError` resolves messages via `go-i18n` from `locales/{en,id}.json`. Locales directory configurable via `I18N_LOCALES_DIR` env var (default `./locales`). Bundle factory is `NewI18nBundle`; localizer helpers are `GetI18nLocalizer` / `SetI18nLocalizer`.

**`infrastructure/http/responder/locale_keys_test.go` enforces two invariants via AST parsing:**

1. Every `CodeToMessageKey` value must exist in both `locales/en.json` and `locales/id.json`.
2. Every `{{.X}}` placeholder must be either auto-injected (`field`/`min`/`max`/`values`) or backed by a `WithMetadata("X", ...)` call somewhere in Go source.

Adding a new `domain.Code*` constant requires updating: `mapper.go` (both `CodeToMessageKey` and `HttpCodes`), both locale files, and the hardcoded `allDomainCodes` list in `mapper_test.go`. Tests catch the drift.

### Token Management & Cookie Auth

- **Access tokens:** stateless JWTs; cannot be server-revoked; expire naturally.
- **Refresh tokens:** stored in DB; can be revoked individually or in bulk via `UserTokenRepository.RevokeByUserIdAndType(ctx, userId, tokenType)`. Type-safe — prevents accidentally revoking the wrong token type.
- **Token rotation:** uses `Update` (single query sets both `last_used_at` and `revoked_at`).
- **Login flow:** revokes all existing refresh tokens before issuing new ones.
- **Token generation:** `crypto.MustGenerateRandomHex32()` returns a 32-byte (64-char hex) cryptographically secure random string.

**Cookie auth:** `POST /api/auth/google` and `POST /api/auth/refresh` set two httpOnly cookies — `access_token` (Path=/api) and `refresh_token` (Path=/api/auth). `POST /api/auth/logout` clears both. `AuthMiddleware` reads `access_token` cookie first, falls back to `Authorization: Bearer` header (for CLI/Postman).

`GIN_CORS_ALLOW_ORIGINS` must NOT be `*` when `GIN_CORS_ALLOW_CREDENTIALS=true` — router panics at startup if misconfigured. Cookie env vars: `COOKIE_DOMAIN`, `COOKIE_SECURE` (must be `true` in prod), `COOKIE_SAMESITE` (strict/lax/none), `COOKIE_ACCESS_PATH` (default `/api`), `COOKIE_REFRESH_PATH` (default `/api/auth`).

### Soft Delete & Trashed Scoping

`model.User` has `gorm.DeletedAt`. `gormUserRepository.Delete` performs a soft delete. Migration includes `deleted_at TIMESTAMP WITH TIME ZONE` + `idx_users_deleted_at` index.

**Scoping rules (admins must be able to see trashed users):**

- `Get` is **always unscoped** — returns soft-deleted rows alongside live ones. To exclude trashed users, clients must explicitly filter `deleted_at_` (IS NULL).
- `Find` / `FindByIds` / `FindByEmails` / `FindByRole` are **unscoped** — return soft-deleted rows so admins can inspect, list, and recover trashed users.

**Mutation paths protect against acting on trashed rows:**

- `AuthMiddleware` rejects trashed users with `CodeAuthUnauthorized` (401) so a valid JWT cannot reauthenticate a trashed user.
- `userPolicy.UpdatePostFetch` returns `CodeUserUpdateTrashedForbidden` (300846, 403).
- `userPolicy.UpdateRolePostFetch` returns `CodeUserRoleTrashedForbidden` (300547, 403).
- `userService.Delete` is idempotent on already-trashed rows (skips on-chain `RoleNone` sync, returns `deleted_count` of newly-deleted rows only).
- `cmd/init_super_admin` filters out trashed entries from the `FindByRole(SuperAdmin)` existence check so a system whose previous SuperAdmin was soft-deleted can be re-initialized.
- `userService.Restore` unsets `deleted_at` and re-syncs the preserved DB role to the blockchain via `syncBlockchainRoles`. Admin can restore Admin peers but NOT SuperAdmin targets (`CodeUserRestoreSuperAdminTargetForbidden` 300943). Live (non-trashed) targets are rejected (`CodeUserRestoreNotTrashedForbidden` 300944) — strict BatchOption B.

`domain.User.DeletedAt` (`*time.Time`) is GORM-agnostic; mapped from `gorm.DeletedAt.Time` when `Valid=true`. Propagated through `response.User.DeletedAt`.

**Trashed-user pagination:** `GET /api/users` exposes trashed via the `deleted_at` column allowlist on filters and sorts. Operator vocabulary: `deleted_at!_` (only trashed), `deleted_at_` (only live, explicit), `deleted_at..a and b` (trashed in date range), `-deleted_at` (sort desc, mixes live+trashed).

### Policy Layer

CredChain uses a two-method policy split: `PreFetch` methods validate before gorm operations, `PostFetch` methods validate after.

**User policies:** See [ROLE.md §Policy Rules](ROLE.md) for the complete user policy interface.

**Credential policies:** See [CREDENTIAL.md §Policy Rules](CREDENTIAL.md) for credential-specific policy methods.

### Chain Infrastructure

`infrastructure/chain/` wraps go-ethereum:

- `contracts/` — abigen-generated bindings (`authority.go`, `registry.go`) — **DO NOT EDIT**
- `bindings.go` — `AuthorityBinding` + `RegistryBinding` interfaces that abstract abigen types for testability. Hand-written code holds these interfaces, not concrete pointers. `AuthorityBinding` includes `UserToRole`, `UserToNonce`, `BatchUpdateUserRoleWithSignature`. `RegistryBinding` includes `UserToNonce` (separate nonce for credential flows).
- `authority_service.go` — `AuthorityService` interface + implementation
- `client.go` — facade for RPC + contract bindings
- `address.go` — `mustHexToAddress` (trusted) and `validateHexToAddress` (validates)
- `signer.go` — signature packing utilities

**AuthorityService rules:**
- Accepts hex string addresses with `0x` prefix (42 chars total)
- Uses `domain.Wallet` for signer authentication
- Methods return raw errors (feature layer translates to domain codes)
- `FindNonce` reads from `Authority.UserToNonce` (NOT Registry — they maintain separate nonces)
- `UpdateUserRole` calls `bind.WaitMined` after the relayer tx and verifies `receipt.Status == 1` before returning, ensuring on-chain nonce has incremented before the next call
- `waitMined` is a struct field (`waitMinedFunc` type) initialized in `NewAuthorityService` to `bind.WaitMined`; tests can substitute their own stub via type assertion to `*authorityService`

**Verify verdicts (400401-400413):** authentic, revoked, integrity_warning, tampered, suspicious, low_similarity, not_similar, no_identifiers, no_match, holder_disabled, issuer_disabled, party_disabled, expired. `integrity_warning` returns HTTP 409, all others 200.

**Verify cache:** the MongoDB `credential_verifications` cache (24h TTL) stores only the credential-level verdict snapshot. Holder/issuer deleted status and `expires_at` are re-checked live on every call (including cache hits), so the party-disabled (400410–400412) and expired (400413) verdicts are always fresh. All four only override an otherwise-`Authentic` result; revocation is checked first (`credential_service.go:1721-1736`).

**RoleNone:** `domain.RoleNone` (Solidity enum value 0) is the on-chain revocation target. Used by `userService.Delete` to revoke roles via `AuthorityService.UpdateUserRole`. **Never persisted to the Postgres `role` ENUM** — in-memory only for chain calls.

**`syncBlockchainRoles` helper:** shared helper used by `Store`, `UpdateRole`, `Delete`, `Restore`, and `Update` flows. Calls `authorityService.UpdateUserRole(ctx, wallet, users)` and translates raw chain errors into a caller-supplied domain code. Caller is responsible for transactional context.

### Rate Limiting Middlewares

Four FX-injectable middlewares with named wrapper types:

| Middleware | Scope | Limit (dev defaults) |
|---|---|---|
| `LoginRateLimitMiddleware` | per IP | 300/min, burst 50 |
| `RefreshRateLimitMiddleware` | per IP | 200/min, burst 30 |
| `LogoutRateLimitMiddleware` | per user | 100/min, burst 10 |
| `ApiRateLimitMiddleware` | per user ID or IP fallback | 1200/min, burst 100 |

`ApiRateLimitMiddleware` is applied globally on the `/api` group before any endpoint-specific limiter (global → specific; more restrictive wins).

Production should re-tighten these (suggested: login 30/burst 5, refresh 60/burst 10, logout 30/burst 5, API 600/burst 50).

### Google OAuth

`oauth.GoogleOAuthClient` is an exported interface (single method `Validate(ctx, idToken, audience)`). Concrete impl is unexported `googleOAuthClient` embedding `*idtoken.Validator`.

Backend validates **ID tokens only** (not the full OAuth flow). Use `make get-google-id-token` CLI to obtain ID tokens via OAuth 2.0 Authorization Code Flow for Postman testing.

**UpdateEmail Google reauth:** `userService.UpdateEmail(ctx, id, email, idToken)` validates the ID token via `oauth.GoogleOAuthClient.Validate(ctx, idToken, *cfg.GoogleClientID)`, asserts the token's email claim matches the requested email, checks for cross-user email conflicts, then updates email + revokes all refresh tokens (`UserTokenRepository.RevokeByUserIdAndType`) inside the same UoW. Codes: `CodeUserEmailInvalidIdToken`, `CodeUserEmailMismatchedIdToken`, `CodeUserEmailConflict`.

### API Routes

> **For full authorization details** (exact role requirements, on-chain vs DB role sources, denied operations), see [ROLE.md §Route Authorization](ROLE.md).

All under `/api` prefix. Middleware order: `ErrorLoggerMiddleware` → `I18nMiddleware`.

| Method | Path | Auth | Notes |
|---|---|---|---|
| GET | `/api/health` | None | Health check |
| GET | `/api/meta` | None | Public metadata endpoint (issuing organization name, authority/registry contract addresses, chain ID, last block) |
| POST | `/api/auth/google` | None | Google OAuth login |
| POST | `/api/auth/refresh` | None | Refresh token (rotates) |
| POST | `/api/auth/logout` | Authenticated | Revoke all refresh tokens |
| GET | `/api/users` | Issuer+ | Paginated user list (handler `Paginate`); read-only for Issuer/Admin |
| GET | `/api/users/self` | Authenticated | Current user (handler `Self`) |
| PUT | `/api/users/self/email` | Authenticated | Update own email; requires fresh Google ID token |
| POST | `/api/users/self/transfer-super-admin` | SuperAdmin | Transfer SuperAdmin role (handler `TransferSuperAdmin`); caller → Admin, target → SuperAdmin, both refresh tokens revoked |
| GET | `/api/users/self/credentials` | Authenticated | Paginated list of own credentials (handler `SelfPaginate`); same query DSL as `GET /api/credentials`, scoped to `holder_user_id == auth_user.id` |
| GET | `/api/users/self/credentials/:id` | Authenticated | Single own credential (handler `SelfFind`); returns 404 (`CodeCredentialFetchNotFound`) when not found OR not owned — never leaks IDs across holders |
| GET | `/api/users/:id` | Issuer+ | Single user lookup (handler `Find`); read-only for Issuer/Admin |
| POST | `/api/users/batch` | Admin+ | Batch create users (optional: number, unit_id, birth_date, gender, meta) |
| PUT | `/api/users/batch` | Admin+ | Batch update users (handler `Update`); same-role updates silently skipped; email changes revoke target's refresh tokens; role changes sync to blockchain |
| PUT | `/api/users/batch/role` | Admin+ | Batch role update (handler `UpdateRole`); syncs DB and on-chain |
| DELETE | `/api/users/batch` | Admin+ | Soft delete users (handler `Delete`); on-chain role revoked to `RoleNone` |
| PUT | `/api/users/batch/restore` | Admin+ | Restore trashed users (handler `Restore`); clears soft-delete, re-syncs DB role to chain |
| GET | `/api/credentials` | Issuer+ | List credentials |
| GET | `/api/credentials/:id` | Issuer+ | Single credential |
| GET | `/api/credentials/:id/file` | Authenticated (no role gate) | Download decrypted file; authorization via policy (holder OR Issuer+) |
| PUT | `/api/credentials/:id/competencies` | Issuer+ | Link competencies to a credential |
| GET | `/api/credentials/:id/metadata/suggestions` | Issuer+ | Suggest taxonomy rows for staged free text |
| PUT | `/api/credentials/:id/metadata` | Issuer+ | Resolve staged free text to taxonomy IDs (pending rows only) |
| POST | `/api/credentials/batch/issue` | Issuer+ | Direct issuance — born approved, type/org required as IDs |
| POST | `/api/credentials/batch/submit` | Authenticated (no role gate) | Holder self-submission — awaits review; type/org may be staged free text |
| POST | `/api/credentials/batch/approve` | Issuer+ | Approve pending submissions; mints on chain, stamps `issuer_user_id` with the approving officer |
| POST | `/api/credentials/batch/reject` | Issuer+ | Reject pending submissions with a reason |
| PUT | `/api/credentials/batch` | Issuer+ | Batch update credentials (pending only) |
| POST | `/api/credentials/batch/revoke` | Issuer+ | Revoke credentials (approved only) |
| POST | `/api/credentials/batch/reextract` | Issuer+ | Re-extract failed credentials |
| POST | `/api/credentials/verify` | None (public) | Returns verdict code (400401-400413) + locale description; used by external verifiers (HR, employers) — no auth required |
| GET | `/api/credential-types` | Authenticated | List credential types (a holder needs these to submit) |
| POST/PUT/DELETE | `/api/credential-types[/:id]` | Issuer+ | Credential type CRUD |
| GET | `/api/issuer-organizations` | Authenticated | List issuer organizations |
| POST/PUT/DELETE | `/api/issuer-organizations[/:id]` | Issuer+ | Issuer organization CRUD |
| GET | `/api/competencies` | Authenticated | List competencies |
| POST/PUT/DELETE | `/api/competencies[/:id]` | Issuer+ | Competency CRUD |
| GET | `/api/user-units` | Authenticated | List organizational units |
| POST/PUT/DELETE | `/api/user-units[/:id]` | Admin+ | User unit CRUD |
| GET | `/api/overview` | Authenticated (no role gate) | Role-conditional dashboard: credential counts, user counts, recents, chain details. Optional `?limit=N` controls recent items per category (default 5). Issuer+ get system-wide data; Holder get own only. |

### Database

**PostgreSQL** (via GORM + golang-migrate):

- Primary store for `users`, `credentials`, `user_tokens`
- Migration files in `infrastructure/database/migrations/` (currently only `000001_initial_schema`)
- Migration logic is inlined in `cmd/migrate.go` (no separate `infrastructure/database/migrate.go`)
- DSN: `POSTGRES_DSN=postgres://...?sslmode=disable`
- Repository search uses dialect-agnostic `LOWER(name) LIKE LOWER(?)` (not Postgres-only `ILIKE`); works on both Postgres and SQLite
- `pg_trgm` extension enabled (`000001_initial_schema.up.sql`); trigram GIN indexes on taxonomy `name` columns back the reviewer's metadata suggestions (`similarity(name, ?)`)

**SQLite (testing only):** GORM repository tests run against in-memory SQLite via `github.com/glebarez/sqlite` (pure Go, no CGO). Not used in production.

**Credentials schema note:** the `credentials.embeddings` column is removed. Extraction data (text, ids, embedding) now lives in MongoDB `credential_extractions`.

**Credential metadata staging:** a submitted credential may carry free-text `submitted_type_name` / `submitted_issuer_organization_name` / `submitted_competencies` names with NULL FKs — the unmatched text is staged on the row, not silently created, and a reviewer resolves it via `PUT /api/credentials/:id/metadata` before approval (approval is blocked while unresolved, `CodeCredentialApproveUnresolvedMetadata` 401546).

**MongoDB:** Actively used. Two collections: `credential_extractions` (text, ids, embedding, file_hash) and `credential_verifications` (TTL-bounded verify cache keyed by uploaded_file_hash). Migration runs inside `make local-up` / `make dev-up`; standalone via `go run main.go migrate-mongo up` / `migrate-mongo down --env .env`. `infrastructure/database/mongo/client.go` provides FX providers.

**Repository Store error translation:** `gormUserRepository.Store` catches Postgres `*pgconn.PgError` with code `23505` (unique_violation) and translates to `CodeUserStoreEmailDuplicateInDatabase` with input emails as metadata. Handles concurrent batch creates that race past the pre-check in `userService.storeValidateEmails`.

**Repository Find error translation:** `userService.Find` translates `gorm.ErrRecordNotFound` to `domain.NewError(CodeUserFetchNotFound, WithMetadata("user_id", id))` so `GET /users/:id` returns HTTP 404 instead of leaking 500.

**Repository Update:** variadic `Update(ctx, users ...domain.User) ([]domain.User, error)` — single and batch callers compile naturally. Uses a single batch CASE statement (`updateBatchCase` helper) to eliminate N+1: one UPDATE + one SELECT regardless of batch size. Per-column CASE branches are emitted only for users that provide a non-nil value; users that don't touch a column fall through to `ELSE column` (preserves existing value). `updated_at` stamped to `CURRENT_TIMESTAMP` explicitly (raw `db.Exec` bypasses GORM autoUpdateTime). `meta` marshalled via `json.Marshal` before binding. The CASE SQL generation is delegated to shared helpers `BuildCaseColumnSQL` and `BuildBatchUpdateSQL` in `infrastructure/database/gorm/helpers.go`. Credential repository uses the same helpers.

**Repository UpdateRole:** batch CASE statement with `WHERE id IN (?)` — pass IDs as a single `[]interface{}` slice (not spread) so GORM can expand the placeholder. Also uses shared `BuildCaseColumnSQL` + `BuildBatchUpdateSQL` helpers.

**Repository Get (pagination):** uses shared `ApplyFilters`, `ApplySorts`, and `ApplyPagination` helpers from `infrastructure/database/gorm/helpers.go`. Default sort for users: `updated_at DESC`. Default sort for credentials: `credentials.issued_at DESC`.

**Repository nil-safety (Get):** `Get` and the shared helpers `ApplySorts` / `ApplyPagination` accept a nil `*Query` — nil queries skip search, filters, and pagination, returning all rows. The default sort is still applied.

**Database Seeder:** `infrastructure/database/seeder/` implements a `Seeder` interface with a `Registry` runner accepting variadic `--names` flags, run inside `make dev-up`; standalone via `go run main.go seed` and `go run main.go seed-chain --env .env`. Seeders run in dependency order: `UserUnitSeeder` → `UserSeeder` → `CredentialTypeSeeder` → `CredentialIssuerOrganizationSeeder` → `CompetencySeeder` → `CredentialSeeder`. Shared helpers live in `pdf.go` (deterministic PDF bytes) and `ulid.go` (`deterministicULID`, the only order-independent handle on a seeded row — `userRepo.Get` sorts by `updated_at DESC, id ASC`, so slice position is not stable). The `UserSeeder` creates 15 users (5 defined + 10 randomised Indonesian names, 60/40 Holder/Issuer tilt) with wallet keys derived from the standard Hardhat mnemonic via BIP44 (`DeriveKeyFromMnemonic`). All users receive an employee number (NIP, 18-digit `YYYYMMDDYYYYMMXNNN`) for Issuer+ roles or a student number (NIM, `2209XXXX`) for Holder roles. Issuers are assigned a root faculty unit and Holders a leaf study program unit (both round-robin, independent counters); Admin and SuperAdmin keep `unit_id` unset. Half the users receive random `{"key":"...}` metadata. Five users are soft-deleted (Anna Sorokin at index 4 + 4 users at indices 10-13). All timestamps (created_at, updated_at, deleted_at) are deterministically generated from a seeded RNG. Chain roles are registered via the `seed-chain` CLI, which reads the database with a nil query and signs batch `UpdateUserRole` transactions in chunks of ≤100 (respecting `MAX_BATCH_ROLE=100` limit in the CredentialAuthority contract) with the SuperAdmin wallet (Hardhat node #1). SuperAdmin and users whose target role is `RoleNone` on a fresh deploy are skipped to avoid contract reverts (`SuperAdminRoleNotUpdatableError`, `SameRoleUpdateError`). The seeded SuperAdmin still holds `Role.SuperAdmin` on chain despite being skipped, because `INITIAL_SUPER_ADMIN_WALLET_ADDRESS` grants it at contract initialization. Soft-deleted users are created with `DeletedAt` pre-set; the `seed-chain` CLI detects these via `DeletedAt != nil` and assigns `RoleNone` on-chain.

> `seed-chain` is not idempotent against a chain that already holds the seeded roles — a re-run reverts with `SameRoleUpdateError` (selector `0x538b255d`). Reset the chain first (`docker compose stop anvil && rm -rf docker/anvil/data/* && docker compose up -d anvil`), then redeploy with `ENV_FILE=.env python3 scripts/setup-contracts.py` so the new contract addresses land in `.env`.

**Credential Seeder:** `credential_seeder.go` builds every row from four orthogonal enums — `credWorkflow` (issue/submit), `credOutcome` (pending/approved/rejected/revoked), `credExtract` (unextracted/pending/succeeded/failed) and `credMetadata` (resolved/staged/resolved-from-staged) — so each row is a shape one of the two real service paths could have produced. A fixed A/B/C matrix covers the named scenarios (both workflows, all four acting roles, a past `expires_at`, a soft-deleted holder, and one file hash reused after rejection); filler rows repeat those shapes across the randomised pool. The seeder never sets `token_id` — `seed-chain` writes it back after minting.

### Docker

`.env` uses `localhost` hostnames; `.env.docker` uses Docker network hostnames (`postgres`, `mongo`, `anvil`). The `docker-compose.yml` includes nginx reverse proxy on ports 80/443.

**Note:** `docker-compose.yml` healthcheck hardcodes port 8080 (`wget -qO- http://localhost:8080/api/health`). If you change `GIN_PORT`, update the healthcheck too.

**Stale data issue:** `make local-fresh` wipes `docker/postgres/data/*` and `docker/mongo/data/*` (and Anvil data + uploads) before rebooting, so a clean reset needs no manual deletion.

**Chain persistence:** The `anvil` service persists chain state to
`./docker/anvil/data/state.json` via bind mount. State survives
container restarts and `docker compose down`. For local development,
the Go backend runs natively (no Docker build) and connects to Anvil
via `RPC_URL=http://127.0.0.1:8545` in `.env`.

Setup (one command does infra + deploy + migrate + seed, then run Go on the host):
```bash
make dev-up                                         # anvil/postgres/mongo, deploy contracts → .env, migrate, seed, seed-chain
make dev                                            # air hot-reload
```

`make dev-up` seeds test users (dev path). For a single SuperAdmin instead of seed data,
use the full-Docker path `make local-up`, which runs `init-super-admin`. The two paths are mutually
exclusive. Rare standalone steps: `go run main.go <migrate|seed|seed-chain|init-super-admin> --env .env`.

### Postman Collection

Postman collection for API testing: `CredChain_postman_collection.json` (in `CredChain_Golang/` root).

**Structure:** `CredChain` → `API` → [Health, Auth, User > [Self, Admin], Credential]

Updated 2026-06-14: Credential endpoints (List, Get, Issue, Revoke, ReExtract) no longer stubs — real DTOs and bodies. Verify endpoint renamed from "Verify Hash" to "Verify". Fixed Login/Logout variable mismatch (USER_ID/EMAIL → AUTH_USER_ID/EMAIL). Added folder-level variables to API group. Removed unused TARGET_ADMIN_ID and TEST_NAME.

**Usage:**

1. Obtain ID token: `make get-google-id-token`
2. Paste into `ID_TOKEN` collection variable
3. Run Google Login → tokens auto-captured into `ACCESS_TOKEN` + `REFRESH_TOKEN`
4. Subsequent requests use Bearer auth inherited from API folder

**Collection variables:** `BASE_URL`, `ID_TOKEN`, `ACCESS_TOKEN`, `REFRESH_TOKEN`, `AUTH_USER_ID`, `AUTH_USER_EMAIL`, `AUTH_USER_NAME`, `TARGET_USER_ID`, `TARGET_CREDENTIAL_ID`, `TEST_EMAIL`, `TEST_PHONE`, `TEST_BIRTH_DATE`.

Every endpoint has response examples for all domain codes it can return (success + all error paths). Auth responses include `deleted_at` field in User DTO. List Users and Find User by ID include "trashed user" success variants (non-null `deleted_at`); Batch Update Users / Update Roles include `300846` / `300547` (trashed-target forbidden, 403) examples.

## Configuration / Env Vars

All Config fields are pointers (`*T`); `nil` = not provided, non-nil = provided (env or default). Application exits fast at startup for required fields.

| Var | Required | Default | Purpose |
|---|---|---|---|
| `GIN_PORT` | no | `8080` | HTTP port |
| `JWT_SECRET` | **yes** | — | JWT signing key (fatal if empty) |
| `WALLET_ENCRYPTION_KEY` | **yes** | — | exactly 32 bytes (AES-256); fatal otherwise |
| `FILE_ENCRYPTION_KEY` | **yes** | — | exactly 32 bytes (AES-256); for credential file at-rest encryption; fatal otherwise |
| `GEMINI_API_KEY` | recommended | — | AI features break without it |
| `GOOGLE_CLIENT_ID` | **yes** | — | OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | **yes** | — | OAuth client secret |
| `GOOGLE_REDIRECT_URI` | no | `http://localhost:3000/google/callback` | OAuth callback URL |
| `POSTGRES_DSN` | **yes** | — | `postgres://...?sslmode=disable` |
| `MONGO_URI` | yes | — | MongoDB connection string |
| `RPC_URL` | **yes** | — | Ethereum/Polygon RPC endpoint |
| `AUTHORITY_CONTRACT` | **yes** | — | CredentialAuthority contract address |
| `REGISTRY_CONTRACT` | **yes** | — | CredentialRegistry contract address |
| `GIN_CORS_ALLOW_ORIGINS` | no | `*` | must NOT be `*` if credentials=true |
| `GIN_CORS_ALLOW_METHODS` | no | all | |
| `GIN_CORS_ALLOW_HEADERS` | no | incl. Authorization | |
| `GIN_CORS_ALLOW_CREDENTIALS` | no | `false` | set `true` for cookie auth |
| `GIN_CORS_MAX_AGE` | no | | seconds |
| `GIN_CORS_EXPOSE_HEADERS` | no | | |
| `COOKIE_DOMAIN` | no | empty | |
| `COOKIE_SECURE` | recommended | `false` | **must be `true` in production** |
| `COOKIE_SAMESITE` | no | `lax` | `strict`/`lax`/`none` |
| `COOKIE_ACCESS_PATH` | no | `/api` | |
| `COOKIE_REFRESH_PATH` | no | `/api/auth` | |
| `LOG_LEVEL` | no | `info` | `debug`/`info`/`warn`/`error` |
| `LOG_OUTPUT` | no | `stdout` | `stdout` or file path |
| `I18N_LOCALES_DIR` | no | `./locales` | locale JSON files directory |
| `INITIAL_SUPER_ADMIN_EMAIL` | only for init | — | required for `init-super-admin` |
| `INITIAL_SUPER_ADMIN_PRIVATE_KEY` | only for init | — | 64-char hex with `0x` prefix |
| `INITIAL_SUPER_ADMIN_NAME` | no | — | optional profile field |
| `INITIAL_SUPER_ADMIN_NUMBER` | no | — | employee/student number |
| `INITIAL_SUPER_ADMIN_PHONE_NUMBER` | no | — | E.164 format |
| `INITIAL_SUPER_ADMIN_BIRTH_DATE` | no | — | ISO 8601 `YYYY-MM-DD` |
| `INITIAL_SUPER_ADMIN_GENDER` | no | — | `male` or `female` only (Postgres ENUM, `migration:12-15`) |
| `INITIAL_SUPER_ADMIN_META` | no | — | JSON object string |
| `PYTHON_AI_API_KEY` | no | — | Python AI service API key (empty = auth disabled) |
| `MONGO_DATABASE` | no | `credchain` | MongoDB database name |
| `AI_VERIFICATION_CACHE_TTL_HOURS` | no | `24` | Verify cache TTL hours before auto-expiry |
| `RIVER_MAX_WORKERS` | no | `10` | River job worker pool size |
| `STORAGE_PATH` | no | `uploads` | Base directory for file storage |
| `CREDENTIAL_FILE_STORAGE_PATH` | no | `credentials` | Subdirectory under STORAGE_PATH for credential files |
| `ISSUING_ORGANIZATION_NAME` | no | — | Organization name displayed on issued credentials |
| `RELAYER_PRIVATE_KEY` | **yes** | — | Relayer wallet private key for on-chain transactions |
| `HARDHAT_MNEMONIC` | no | — | Hardhat development mnemonic (local dev only) |
| `PYTHON_AI_BASE_URL` | no | — | Python AI service base URL |
| `PYTHON_AI_TIMEOUT_SECONDS` | no | — | Python AI request timeout |
| `POSTGRES_USER` | **yes** | — | PostgreSQL user |
| `POSTGRES_PASSWORD` | **yes** | — | PostgreSQL password |
| `POSTGRES_DB` | **yes** | — | PostgreSQL database name |
| `DB_MAX_OPEN_CONNS` | no | — | Max open DB connections |
| `DB_MAX_IDLE_CONNS` | no | — | Max idle DB connections |
| `DB_CONN_MAX_LIFETIME` | no | — | Max connection lifetime |
| `DB_CONN_MAX_IDLE_TIME` | no | — | Max idle connection time |
| `MONGO_INIT_DB_USERNAME` | no | — | MongoDB init username |
| `MONGO_INITDB_ROOT_PASSWORD` | no | — | MongoDB root password |
| `JWT_ACCESS_EXPIRY_MINUTES` | no | — | JWT access token expiry in minutes |
| `JWT_REFRESH_EXPIRY_HOURS` | no | — | JWT refresh token expiry in hours |
| `CREDENTIAL_EXTRACT_WORKER_COUNT` | no | — | Credential extract worker count |
| `CREDENTIAL_EXTRACT_WORKER_POLL_SECONDS` | no | — | Credential extract poll interval |
| `CREDENTIAL_EXTRACT_WORKER_MAX_ATTEMPTS` | no | — | Credential extract max attempts |

## Testing

- **Framework:** `stretchr/testify/assert` (assertions) + `stretchr/testify/mock` (mocks)
- **Style:** white-box, in-package (same package as source)
- **Count:** 61+ test files
- **Database tests:** in-memory SQLite via `github.com/glebarez/sqlite` (pure Go, no CGO). Postgres-specific JSONB operators are not exercised — `Meta` round-trips via `serializer:json`. `UserTokenType` Postgres ENUM stores as TEXT in SQLite.
- **No integration tests** against real Postgres / MongoDB / IPFS / RPC.

**Coverage** (from `go test -cover ./...`):

| Range | Packages |
|---|---|
| 100% | `config`, `infrastructure/http/context` |
| 90%+ | `response` (97%), `security` (93%), `middleware` (92%), `request/query` (92%), `i18n` (90%) |
| 60–85% | `crypto` (83%), `domain` (79%), `domain/query` (69%), `chain` (69%), `responder` (67%), `model` (67%), `feature/credential` (67%), `feature/user` (67%) |
| 30–55% | `feature/auth` (54%), `oauth` (50%), `database/gorm` (32%) |
| 0% (intentional) | `cmd`, `ai`, `chain/contracts`, `http`, `logger`, `storage` |

**Test Infrastructure** (`tests/`, test-only, never imported by production):

- `tests/db/sqlite.go` — `OpenInMemorySQLite(t)` opens in-memory SQLite, auto-migrates `model.User`, `model.UserToken`, `model.Credential`; parallel-safe via random per-call shared-cache name
- `tests/fixtures/` — entity builders with options pattern: `NewDomainUser(opts...)`, `NewModelUser(opts...)`, `NewDomainUserToken(opts...)`, `NewWallet(t)`, `TestWalletEncryptionKey()`
- `tests/gintest/` — `NewContext(t, opts...)` builds gin context with i18n/user injection; `LoadTestI18nBundle(t)` loads real `locales/` (cached via `runtime.Caller`)
- `tests/mocks/` — `stretchr/testify/mock` implementations: `MockUserRepository`, `MockUserTokenRepository`, `MockCredentialRepository`, `MockUnitOfWork` + `RunUnitOfWorkFn(uow, inner)` helper, `PropagatingUnitOfWork` (for chain-failure rollback testing), `MockAuthorityService`, `MockUserPolicy`, `MockGoogleOAuthClient`, `MockAuthorityBinding`, `MockRegistryBinding`
- `chain/authority_service_test.go` defines `localAuthorityBinding` / `localRegistryBinding` inline to avoid import cycle (would create `tests/mocks` → `chain` cycle)

**Locale coverage** (`infrastructure/http/responder/locale_keys_test.go`):

1. Every `CodeToMessageKey` value must exist in both `locales/en.json` and `locales/id.json`
2. Every `{{.X}}` placeholder must be either auto-injected (`field`/`min`/`max`/`values`) or backed by a `WithMetadata("X", ...)` call somewhere in Go source (parsed via `go/ast`)

When adding a new endpoint or service method, add at least: one happy-path test, one validation-failure test, one repository-error test, and update locale files + `mapper.go` if new codes are introduced.

## Tech Stack

| Layer | Tool | Version |
|---|---|---|
| Language | Go | 1.25.1 |
| Module path | `CredChain_Golang` (underscore, not URL) | |
| Web framework | gin-gonic/gin | v1.12 |
| CORS | gin-contrib/cors | latest |
| JWT | golang-jwt/jwt | v5 |
| Validation | go-ozzo/ozzo-validation | v4 |
| CLI | spf13/cobra | v1.10 |
| DI | uber/fx | v1.24 |
| Logger | uber/zap | v1.27 |
| ORM | gorm.io/gorm | v1.31 |
| Postgres driver | gorm.io/driver/postgres + pgx/v5 | latest |
| SQLite (test only) | glebarez/sqlite | latest (pure Go) |
| Migrations | golang-migrate/migrate/v4 | latest |
| Mongo | go.mongodb.org/mongo-driver/v2 | latest |
| Chain | ethereum/go-ethereum | v1.17 |
| Env loader | joho/godotenv | latest |
| i18n | nicksnyder/go-i18n/v2 | latest |
| IDs | oklog/ulid/v2 | latest |
| AI | google/generative-ai-go | latest |
| Testing | stretchr/testify | latest |
| Google ID token | google.golang.org/api/idtoken | latest |

## Cross-Repo Integration

- **`../CredChain_Solidity/AGENTS.md`** — contracts consumed via abigen bindings at `infrastructure/chain/contracts/`. When contract ABI changes, regenerate bindings and update `chain.AuthorityBinding` / `chain.RegistryBinding` interfaces in `chain/bindings.go` to mirror new methods.
- **`../CredChain_React/AGENTS.md`** — sole HTTP consumer. Frontend mirrors `domain.Code*` constants in `@shared/api/codes.ts`. Locale files in `locales/{en,id}.json` are mirrored in `src/shared/i18n/` and verified by `npm run check-locales`.
- **`../CredChain_Python/AGENTS.md`** — AI downstream over HTTP. Response envelope `{code, message, data, errors}` matches Go's format. Python owns category `50` (AI). Python errors propagate through this backend with their original `50xxxx` code intact so the frontend can resolve the i18n key.

**Role enum (mirrored in all four repos):** `None(0) → Holder(1) → Issuer(2) → Admin(3) → SuperAdmin(4)`. Same enum in Solidity (`CredentialAuthority.Role`), Go (`domain.Role`), and React (`@shared/auth/role.ts`).

**Response code format (mirrored in all four repos):** 6-digit `AABBCC`. Categories: `10` (system), `20` (auth), `30` (user), `40` (credential), `50` (AI service).

## Key Constraints

- **Go module path** is `CredChain_Golang` (with underscore) — imports use this, not a URL path.
- **`WALLET_ENCRYPTION_KEY`** must be exactly 32 bytes (AES-256 key). Validated at startup in `config.NewConfig` — app fails fast with clear error if length is wrong.
- **`FILE_ENCRYPTION_KEY`** must be exactly 32 bytes (AES-256). Same validation — credential files encrypted at rest with this key.
- **SuperAdmin** can only be created via the `init-super-admin` CLI (run automatically by `make local-up`, or `go run main.go init-super-admin --env .env`), never via API.
- **Transfer Super Admin**: Only the current SuperAdmin can transfer their role via `POST /api/users/self/transfer-super-admin`. Caller is downgraded to Admin, target promoted to SuperAdmin. Refresh tokens for both users are revoked.
- **Self-profile lockdown**: there is no self-profile route. Name, number, unit_id, birth_date, gender, and meta are admin-managed via `PUT /api/users/batch`. **SuperAdmin** may include their own ID in `PUT /api/users/batch` to self-edit profile fields (other roles cannot self-target via batch — `CodeUserUpdateSelfForbidden`). However, **no role can change their own email via batch** (`CodeUserUpdateSelfEmailForbidden` 300847, 403) — email changes must go through `PUT /api/users/self/email` which requires a fresh Google ID token. This prevents accidentally locking the account out with an inaccessible email.
- **`init-super-admin`** validates wallet has SuperAdmin role on-chain before database initialization; checks for existing SuperAdmin by role using `FindByRole`, not by email. Filters out trashed users from the existence check.
- **Auth is Google OAuth only** — no email/password login exists.
- **Soulbound tokens:** `CredentialRegistry._update()` blocks all transfers and burns (reverts with `CredentialTransferError`) — Solidity-side enforcement; backend should not attempt transfer flows.

- **Policy checks** happen in `feature/*/policy.go`, not in middleware or domain.
- **`migrate-down`** rolls back only ONE migration step, not all.
- **Environment files:** `.env` and `.env.docker` contain credentials and are NOT tracked by git; use `.env.example` as template.
- **`GoogleOAuthClient` is an interface** (not concrete pointer) — `*oauth.GoogleOAuthClient` does not exist; use `oauth.GoogleOAuthClient` directly.
- **Chain bindings:** hand-written code holds `chain.AuthorityBinding` / `chain.RegistryBinding`, not concrete `*contracts.Authority` / `*contracts.Registry`.
- **Test infrastructure** lives at `tests/` — never imported by production code.
- **Repository search** is dialect-agnostic via `LOWER(...) LIKE LOWER(...)` — works on Postgres and SQLite.
- **`model.Meta`** uses `serializer:json` — required for SQLite test compatibility.
- **Locale message templates:** any `{{.X}}` placeholder must be backed by either an auto-injected key or a `WithMetadata("X", ...)` call in Go source (enforced by `locale_keys_test.go`).

### Strict NO-N+1 Rule

All batch repository operations (Postgres and MongoDB) MUST execute a single
query/aggregation regardless of batch size. NEVER issue queries inside a loop
over input items.

- Postgres batch updates use CASE statements via shared `BuildCaseColumnSQL` + `BuildBatchUpdateSQL` helpers (`infrastructure/database/gorm/helpers.go`).
- Mongo id-search uses ONE aggregation pipeline (`FindRankedByIds`).
- Relation/tie-break lookups use a single `IN`-clause / `$in` query.
- Verify fuzzy path: 1 extract-ids call + 1 aggregation + 1 verify call + 1 cache write.

Reviewers MUST reject any code path that issues queries inside a loop.

## Deployment

**Push to master branch only when build succeeds. Do not create feature branches, bugfix branches, or any other branch types — commit directly to master.**

Before pushing, run the repo's canonical verification command and confirm it passes:

- `CredChain_Golang`: `make test && make lint` (lint = `go vet ./... && gofmt -l .`; the `gofmt` list must be empty)
- `CredChain_Solidity`: `make compile && make test`
- `CredChain_Python`: `make check`
- `CredChain_React`: `npm run lint && npm run build && npm run test && npm run check-locales`

### Python AI Service API Key

The AI service requires an `API_KEY` for authentication. After generating it
in the Python project (`make generate-api-key`), copy the key:

1. Read `API_KEY=...` from `CredChain_Python/.env.docker`
2. Set `PYTHON_AI_API_KEY=<that-value>` in `.env.docker`

Mismatched keys cause all AI requests to fail with 401.

## See Also

- `ROLE.md` — comprehensive role system reference
- `CREDENTIAL.md` — comprehensive credential system reference
- `Makefile` — canonical commands
- `CredChain_postman_collection.json` — endpoint testing collection
- `.air.toml` — hot reload config
- `docker-compose.yml` — full stack (backend + nginx + postgres + mongo)
- `infrastructure/database/migrations/` — schema migrations
- `locales/{en,id}.json` — i18n message catalog
- `../AGENTS.md` (workspace root, uncommitted) — multi-repo reference
- `../CredChain_Solidity/AGENTS.md` — smart contract reference
- `../CredChain_React/AGENTS.md` — frontend consumer reference
- `../CredChain_Python/AGENTS.md` — AI service reference

# CredChain Database Design

## Overview

CredChain uses a **dual-database architecture**: PostgreSQL for relational/core data and MongoDB for AI/analytics payloads.

| Database | Purpose | Engine |
|---|---|---|
| **PostgreSQL** | User accounts, credential metadata, auth tokens, job queue (River) | PG 15+ |
| **MongoDB** | AI extraction results, verification cache | Mongo 7+ |

---

## Why Two Databases?

The `credentials` table in Postgres holds only structural metadata (50 bytes per row excluding JSONB). The heavy AI payload — full OCR text, extracted identifiers, float64 embedding vector — lives in MongoDB's `credential_extractions` collection. This keeps:
- **Postgres rows small** — faster scans, cheaper partial indexes
- **Mongo flexible** — no schema migrations for evolving extraction output
- **Scaling independent** — AI read-path throughput doesn't contend with OLTP

The `credential_verifications` cache uses MongoDB's native TTL index for automatic expiry (24h default), which would require a background job in Postgres.

---

## Entity Relationships

```
┌─────────┐       ┌──────────────┐       ┌──────────────┐
│  users  │──FK──→│ credentials  │──ref──→│ credential_  │ (Mongo)
│ (PG)    │       │ (PG)         │  logical│ extractions  │
└─────────┘       └──────────────┘       └──────────────┘
     │                                      (by id)
     │FK
     ↓
┌─────────┐       ┌──────────────┐       ┌──────────────┐
│  user_  │       │  credential_ │──ref──→│  credential_ │ (Mongo)
│  tokens │       │  verifications│ logical│  extractions  │
│ (PG)    │       │ (Mongo)      │  lookup│               │
└─────────┘       └──────────────┘       └──────────────┘
                   (by file_hash)         (by ids.value)
```

### Cross-Database References

| From | To | Key | Nature |
|---|---|---|---|
| `credentials.id` (PG) | `credential_extractions.credential_id` (Mongo) | ULID string | Logical 1:1 |
| `credentials.file_hash` (PG) | `credential_verifications.uploaded_file_hash` (Mongo) | keccak256 hex | Logical 1:N |
| `credentials.file_hash` (PG) | `credential_extractions.file_hash` (Mongo) | keccak256 hex | Logical 1:1 |

MongoDB has no foreign key constraints. References are enforced at the application layer.

---

## PostgreSQL Schema

### Application Tables

#### `users`
Central identity table. Supports soft-delete via `deleted_at`. Four roles (super_admin, admin, issuer, holder) controlled by the `role` ENUM. Wallet address is unique — one wallet per user.

#### `credentials`
Core credential record. Tracks lifecycle: issue → extract → (optional) revoke. The `file_hash` has a **partial unique index** (`WHERE revoked_at IS NULL`) so re-issuing a credential with the same file after revocation is allowed. Only the extraction status is stored here — the actual extracted data (text, identifiers, embedding) lives in MongoDB.

#### `notifications`
Recipient-owned inbox history with exactly nine columns in this order:

| Column | Definition |
|---|---|
| `id` | ULID primary key |
| `notified_user_id` | Required recipient FK to `users` |
| `subject_type` | Nullable discriminator: `credential`, `credential_type`, `credential_issuer_organization`, or `competency` |
| `subject_id` | Nullable subject ULID |
| `type` | Required prefixed historical event type |
| `data` | Nullable JSONB; mapped with GORM `serializer:json` |
| `created_at` | Required creation timestamp |
| `updated_at` | Nullable update timestamp, initially NULL |
| `read_at` | Nullable recipient read timestamp |

CHECKs require both subject fields to be NULL or both present, allowlist the seven notification types, and enforce subject/type compatibility. Types are `credential_registered`, `credential_approved`, `credential_rejected`, `credential_revoked`, `credential_type_submitted`, `credential_issuer_organization_submitted`, and `competency_submitted`.

Unique `(notified_user_id, subject_type, subject_id, type)` makes subject-backed insertion replay-safe without changing IDs, payload, creation time, or read state. Ordinary NULL uniqueness intentionally does not deduplicate subjectless rows. Feed index: `(notified_user_id, created_at DESC, id DESC)`; partial unread index: `(notified_user_id) WHERE read_at IS NULL`. First mark-read atomically sets `read_at` and `updated_at` to the same database timestamp; repeated reads preserve both.

The recipient FK remains. Polymorphic subjects have no ordinary subject FK, including no credential FK/delete restriction. Trusted producers validate subjects; unavailable targets are handled without leaking details. Current authorized summaries use one allowlisted UNION ALL batch query. Credential summaries are holder-only; reference alerts require live DB and chain reviewer permission before pagination, totals, and unread counts. `data` is recipient-visible optional structured content, never a permission grant, secret store, trusted HTML, or arbitrary navigation URL.

Reference submission and reviewer alerts commit together. Approval/rejection preserves historical alerts and independent read state. Live pending-review workload is queried from subject tables, never counted from inbox rows. Credential lifecycle emails remain separate River jobs. Development notification fixtures derive from existing seeded credential histories; see [CREDENTIAL.md](CREDENTIAL.md).

#### `user_tokens`
Refresh token store. Single ENUM value (`refresh`). Supports revocation and expiry. Tokens are hashed before storage.

### River Queue Tables

River provides PostgreSQL-backed job queue tables for async processing. The application uses it for credential extraction jobs.

| Table | Purpose | Noteworthy |
|---|---|---|
| `river_job` | Job queue | BIGSERIAL PK, JSONB args, GIN indexes, unique jobs via unique_key |
| `river_leader` | Leader election | UNLOGGED — no WAL overhead |
| `river_queue` | Queue metadata | Named queues (default, etc.) |
| `river_client` | Worker registration | UNLOGGED |
| `river_client_queue` | Per-client queue subscriptions | UNLOGGED; FK → river_client ON DELETE CASCADE |
| `river_migration` | Migration tracking | PK on (line, version) |

---

## MongoDB Schema

#### `credential_extractions`
One document per credential. Upserted by `credential_id`. Contains:
- **`text`**: Full OCR-extracted text
- **`ids`**: Array of `{type, value}` pairs (e.g., name, student ID, email) — the `ids.value` multikey index enables the fuzzy-matching rank pipeline
- **`embedding`**: Float64 vector from Python AI service

#### `credential_verifications`
Verify-result cache. Keyed by `uploaded_file_hash`. TTL index on `created_at` expires entries after 24 hours (configurable). Sliding expiry: re-verifying the same file resets the TTL window.

---

## Key Design Patterns

### Cross-Database Atomicity
Postgres and MongoDB cannot share a distributed transaction. The extraction flow uses idempotent upserts: Mongo upsert first, then Postgres update. If Postgres fails, River retries the job, and the Mongo upsert safely overwrites.

### Re-Issuable File Hashes
The partial unique index on `credentials.file_hash WHERE revoked_at IS NULL` allows a credential to be revoked and then re-issued with the same file (same hash) — the old row's `revoked_at` is set, removing it from the uniqueness scope.

### Sliding TTL Cache
Unlike a fixed-expiry cache, re-verifying the same file resets `created_at` in MongoDB, keeping recently-verified files warm while still bounding total storage.

### Offloaded Extraction Payload
Keeping heavy AI results in MongoDB means Postgres indexes (including the partial unique on `file_hash`) stay small and fast. The extraction collection can be indexed or schema-changed independently.

---

## Migration Strategy

| Component | Tool | Command |
|---|---|---|
| Postgres (app) | `golang-migrate` | `go run main.go migrate up --env .env` |
| Postgres (River) | `rivermigrate` | Applied independently by the same migrate-up CLI |
| MongoDB indexes | Custom Go runner | `go run main.go migrate-mongo up --env .env` |

Application DDL lives only in `000001_initial_schema.up.sql` / `.down.sql`, including generic notifications. The up creates notifications after users and credentials; the down drops notifications before those tables. River owns its own migration history and is unchanged. The notification schema/type change is incompatible with existing credential-only rows. Deployment against existing data remains blocked until its handling is separately approved; no conversion scripts, ALTER statements, or additional schema files accompany this change.

**Existing database warning:** rewriting migration 1 affects fresh installs only. A database already at version 1 will not replay it, and a database at version 2 cannot find its migration source after the separate notification migration pair is removed. Such installations require a separately authorized transition or development reset before using this consolidated layout. Never silently force migration versions or change migration bookkeeping.

**Rollback warning:** one application migration-down step from version 1 now drops the entire application schema and its data, not only notifications. It does not roll back River migrations. Do not run down against existing application data as a verification step. Automated SQLite notification-DDL tests do not verify a full PostgreSQL initial up/down round-trip.

MongoDB collections are created on first write; the migration only creates indexes (idempotent). Re-running after changing `AI_VERIFICATION_CACHE_TTL_HOURS` requires dropping and recreating the TTL index.

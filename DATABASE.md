# CredChain Database Design

## Overview

CredChain uses a **dual-database architecture**: PostgreSQL for relational/core data and MongoDB for AI/analytics payloads.

| Database | Purpose | Engine |
|---|---|---|
| **PostgreSQL** | User accounts, credential metadata, auth tokens, job queue (River) | PG 15+ |
| **MongoDB** | AI extraction results, verification cache | Mongo 8.0 |

---

## Why Two Databases?

The `credentials` table in Postgres holds structural metadata and lifecycle timestamps; no measured per-row size is established here. The heavy AI payload (full OCR text, extracted identifiers, float64 embedding vector) lives in MongoDB's `credential_extractions` collection. This keeps:
- **Postgres rows small** — faster scans, cheaper partial indexes
- **Mongo flexible** — no schema migrations for evolving extraction output
- **Scaling independent** — AI read-path throughput doesn't contend with OLTP

The `credential_verifications` cache uses MongoDB's native TTL index for automatic expiry (24h default), which would require a background job in Postgres.

---

## Entity Relationships

Partial sketch of the core cross-database references (the schema also holds `user_units`, `credential_types`, `credential_issuer_organizations`, `competencies`, `competency_credential`, and `notifications`; see Application Tables below):

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
Core credential record. Review lifecycle and extraction state are independent and timestamp-derived. The `file_hash` has a **partial unique index** (`WHERE revoked_at IS NULL AND rejected_at IS NULL`), allowing reuse after revocation or rejection, but not expiry alone. Type and organization IDs are required; references must be approved and active when selected. Extracted text, identifiers, and embeddings live in MongoDB.

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
Refresh token store. Single ENUM value (`refresh`). Supports single-use rotation, revocation, and expiry. **Requirement conflict:** tokens should be hashed before storage, but current issuance and repository lookup persist and compare the raw refresh-token value. Hash-at-rest protection is not implemented.

#### `user_units`
Self-referencing org/faculty/program tree. `parent_id` FK → `user_units(id)` (NULL root), `name` VARCHAR(256), `active` BOOLEAN default true. `users.unit_id` references this table; Issuer seed rows get root faculty units, Holders get leaf study programs.

#### `credential_types`
Credential taxonomy rows with a submit/approve/reject lifecycle: `submitter_user_id`, `approver_user_id`, `rejecter_user_id` FKs to `users`, `name`, `active` BOOLEAN default false, `rejection_reason`, `activated_at`, `rejected_at`. CHECK constraint `chk_credential_types_activated_xor_rejected` prevents both lifecycle timestamps from being non-NULL; both may be NULL for pending rows.

#### `credential_issuer_organizations`
Same submit/approve/reject lifecycle shape as `credential_types` (submitter/approver/rejecter FKs, `active` default false, `rejection_reason`, a CHECK preventing both `activated_at` and `rejected_at` from being non-NULL). Referenced by `credentials.issuer_organization_id`.

#### `competencies`
Same submit/approve/reject lifecycle shape as `credential_types` (submitter/approver/rejecter FKs, `active` default false, `rejection_reason`, a CHECK preventing both `activated_at` and `rejected_at` from being non-NULL). Linked to credentials through `competency_credential`.

#### `competency_credential`
Join table: composite PK (`competency_id`, `credential_id`), FKs to `competencies(id)` and `credentials(id)`.

### River Queue Tables

River provides PostgreSQL-backed job queue tables for async processing. The application uses it for credential extraction and lifecycle email jobs.

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
Verify-result cache. Keyed by `uploaded_file_hash`. TTL index on `created_at` expires entries after 24 hours (configurable). `Store` resets `created_at` on an upsert; cache-hit reads do not renew it. MongoDB TTL cleanup is asynchronous, and lookup does not independently reject an overdue document.

---

## Key Design Patterns

### Cross-Database Atomicity
Postgres and MongoDB cannot share a distributed transaction. The extraction flow uses idempotent upserts: Mongo upsert first, then Postgres update. If Postgres fails, River retries the job, and the Mongo upsert safely overwrites.

### Re-Issuable File Hashes
The partial unique index on `credentials.file_hash WHERE revoked_at IS NULL AND rejected_at IS NULL` removes revoked and rejected rows from uniqueness scope. Expiry alone does not release the hash; the chain's Issued status also remains until revocation.

### Write-Reset TTL Cache
A new cache result upsert resets `created_at`. Returning an existing cached verdict does not call `Store`, so repeated cache hits do not create sliding read expiry. Exact-hash outcomes bypass this cache and always repeat the live correspondence check.

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

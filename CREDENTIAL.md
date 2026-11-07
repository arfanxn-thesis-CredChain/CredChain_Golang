# CredChain Credential System

> **Related docs:** For the role hierarchy and general authorization rules, see [ROLE.md](ROLE.md).

## Entity Definitions

### `domain.Credential` (`domain/credential.go:69-123`)

| Field | DB Type | Purpose |
|-------|---------|---------|
| `ID` | `CHAR(26)` PK | ULID primary key |
| `HolderUserID` | `CHAR(26)` FK → `users.id`, NOT NULL | Credential owner |
| `SubmitterUserID` | `CHAR(26)` FK → `users.id`, NOT NULL | Who submitted it (self-submission: submitter == holder; direct issuance: submitter == issuer) |
| `IssuerUserID` | `CHAR(26)` FK → `users.id`, nullable | Who minted it on-chain (stamped at approval for self-submissions; NULL while pending/rejected) |
| `SubmittedIssuerOrganizationName` | `VARCHAR(256)`, nullable | Free-text issuer org name staged when no taxonomy row matched at submit |
| `IssuerOrganizationID` | `CHAR(26)` FK → `credential_issuer_organizations.id`, nullable | Resolved org; nil until a reviewer resolves the staged name |
| `SubmittedTypeName` | `VARCHAR(256)`, nullable | Free-text credential-type name staged when no taxonomy row matched at submit |
| `TypeID` | `CHAR(26)` FK → `credential_types.id`, nullable | Resolved type; nil until a reviewer resolves the staged name |
| `Number` | `VARCHAR(256)`, nullable | Credential number; unique within an issuer organization (partial unique index) |
| `RevokerUserID` | `CHAR(26)` FK → `users.id`, nullable | Who revoked it |
| `Name` | `VARCHAR(256)`, NOT NULL | Human-readable label (e.g. "Bachelor's Degree") |
| `Meta` | `JSONB` | Arbitrary metadata (institution, grade, etc.) |
| `SubmittedCompetencies` | `JSONB` | Staged competency names, `[{name, resolved_id}]`; the name survives resolution for audit |
| `TokenID` | `VARCHAR(256)`, unique, nullable | On-chain ERC-721 token ID (decimal string) |
| `FileHash` | `CHAR(66)`, NOT NULL | `0x`-prefixed keccak256 of raw file bytes |
| `FileURI` | `TEXT`, nullable | Storage path (e.g. `local:///uploads/...`) |
| `ExtractEnqueuedAt` | `TIMESTAMP`, nullable | When OCR extract job was enqueued |
| `ExtractFailedAt` | `TIMESTAMP`, nullable | When OCR extract job permanently failed |
| `ExtractError` | `TEXT`, nullable | Error message if extraction failed |
| `ExtractedAt` | `TIMESTAMP`, nullable | When extraction completed successfully |
| `IssuedAt` | `TIMESTAMP`, NOT NULL | Date printed on the physical credential (entered via form; defaults to now if omitted) |
| `RevokedAt` | `TIMESTAMP`, nullable | When credential was revoked |
| `ExpiresAt` | `TIMESTAMP`, nullable | Expiry; evaluated by `Credential.Status()`, Overview lifecycle counts and verification |
| `ActivatedAt` | `TIMESTAMP`, nullable | When approved or directly registered (mint time); drives `approved` lifecycle |
| `RejecterUserID` | `CHAR(26)` FK → `users.id`, nullable | Who rejected it |
| `RejectedAt` | `TIMESTAMP`, nullable | When rejected; drives `rejected` lifecycle |
| `RejectionReason` | `TEXT`, nullable | Why rejected |
| `CreatedAt` | `TIMESTAMP`, NOT NULL | Row creation time |
| `UpdatedAt` | `TIMESTAMP`, nullable | Row update time |

**Sources:** Go `domain/credential.go:69-123`, GORM model `infrastructure/database/gorm/model/credential.go:27-70`, Postgres migration `000001_initial_schema.up.sql:100-163`.

Embeds `Holder`, `Issuer`, `Revoker` (`*domain.User`, `gorm:"-" json:"-"`) populated by GORM Preload when query.Includes contains the corresponding key, plus `Competencies` (`[]domain.Competency`) from the `competency_credential` join table and `Type` / `IssuerOrganization` (`*domain.CredentialType` / `*domain.CredentialIssuerOrganization`) — all keyed off `Includes` (`"competencies"`, `"type"`, `"issuer_organization"`). None are serialized to JSON — the response DTO maps them explicitly.

### `response.Credential` (`response/credential.go:15-33`)

Mirrors domain entity minus embeddings. `Holder`, `Issuer`, `Revoker` are `*response.User` with `omitempty` for null-safe JSON output. Factory: `response.FromDomainCredential(c domain.Credential)`.

### `response.CredentialVerify` (`response/credential.go:74-80`)

| Field | Type | Purpose |
|-------|------|---------|
| `VerdictCode` | `int` | 6-digit domain code (400401-400413) |
| `SimilarityScore` | `*float64` | Fuzzy match score (0-1), non-nil only on fuzzy path |
| `SimilarityPercent` | `*string` | Human-readable percentage, non-nil only on fuzzy path |
| `Description` | `string` | Localized verdict description resolved via the request's i18n localizer |
| `Credential` | `*VerificationCredential` | Public matched facts, if any; omitted when absent |

The base `credential` summary uses an explicit eleven-field allowlist: `id`, `name`, `status`, `is_submission`, `issuer_organization_id`, `issuer_organization_name`, `credential_type_name`, `number`, `file_hash`, `issued_at`, `expires_at`. Organization name, credential type name, number and expiry remain nullable. Exact, cache and text-comparison results expose the actual linked organization and type names equally to anonymous and authenticated visitors. A candidate's identifying fields describe the compared credential, not established facts about the uploaded document.

`exact_match_confirmed` is always emitted. It is true only after the uploaded bytes match a registered hash and the existing live registry correspondence checks succeed. The named internal `CredentialVerificationResult` carries this evidence separately from verdict and similarity. Cached verdicts, AI results (including score1.0), integrity warnings and inconclusive searches never establish this evidence; operational failures return no verification payload. Legacy cached authentic-like codes therefore do not imply a fresh exact check.

Only confirmed exact results with a matched credential include `public_details`. Its complete nineteen-field allowlist is the eleven summary fields plus `revoked_at`, `holder`, `competencies`, `meta`, `token_id`, `created_at`, `activated_at`, `rejected_at`. Reference names and nullable dates remain nullable. Names come from linked organization/type relations loaded during verification, never the global `/meta` organization label. A missing relation yields null, not an invented name.

`holder` is null when unavailable, otherwise an explicit eleven-field displayed-account DTO: `name`, `number`, `unit_name`, `joined_year`, `email`, `gender`, `role`, `wallet_address`, `created_at`, `updated_at`, `deleted_at`. It is not the unrestricted User DTO. Unit name is resolved once for the selected live exact winner, never per candidate or on fuzzy/cache branches. Missing units yield null; unexpected unit lookup failures return an operational error without exact evidence or public details. Inactive units retain their stored name. Holder account fields and reference names reflect current database data, not immutable issuance-time identity or independent issuer confirmation.

`competencies` contains name-only objects (an empty array when absent); `meta` is the credential's existing Additional Information object or null. These fields are intentionally public after live exact confirmation. User metadata, extraction output and taxonomy workflow data are not included. The approved footer exposes Credential ID, File Hash, Token ID and applicable lifecycle dates using the original Credential Details presentation.

Anonymous and authenticated verifiers receive the same public contract. React opens these details from the same POST response, in memory; there is no public lookup by credential ID or shareable detail URL. Reload requires re-upload/reverification. Revoked, expired and disabled-party exact outcomes retain file-identity evidence and allow public inspection, but do not imply current validity. No immutable recipient-name snapshot exists; account names/OCR must not be presented as verified recipient identity.

The ordinary summary adds only the nullable linked organization and credential type names; no expanded candidate details are exposed. The expanded public object excludes raw user/unit IDs, profile links, birth date, language, user metadata, secrets, storage URI, extraction fields, other lifecycle-party identities and private rejection/revocation reasons. Public opening previews only the uploaded file; it grants no stored-file download rights. Existing authorized credential/detail/download endpoints remain protected and unchanged. Own Holders and Issuer+ can open the existing private modal through those endpoints; a text-comparison candidate uses its protected stored file, never the unmatched upload. AI similarity100% grants no public expansion or additional permissions.

### `CredentialIssuance` (service-layer input, `credential_service.go:69-82`)

Direct issuance is the Issuer+ path. Type and organization are **IDs, not names** — an officer issuing directly must pick existing taxonomy rows, so a directly issued credential can never carry staged free text.

| Field | Type | Source |
|-------|------|--------|
| `HolderUserID` | `string` | Multipart form field — required, and unlike submit the holder is someone else |
| `TypeID` | `string` | Multipart form field — required, must be an existing `credential_types` row |
| `IssuerOrganizationID` | `string` | Multipart form field — required, must be an existing `credential_issuer_organizations` row |
| `Number` | `*string` | Multipart form field — optional; unique within the organization |
| `IssuedAt` | `*time.Time` | Multipart form field — optional; defaults to now |
| `ExpiresAt` | `*time.Time` | Multipart form field — optional |
| `Name` | `string` | Multipart form field |
| `Meta` | `map[string]any` | Multipart form field (JSON string) |
| `CompetencyIDs` | `[]string` | Multipart form field — existing competency rows, linked at creation |
| `Filename` | `string` | Uploaded file name |
| `MIMEType` | `string` | Content-Type header (validated against allowlist) |
| `FileBytes` | `[]byte` | Uploaded file contents (max 10 MB) |

### `CredentialSubmission` (service-layer input, `credential_service.go:94-110`)

The self-submission counterpart. No `HolderUserID` — the submitter is always the holder. Metadata is **id-or-name**: for type and organization exactly one of each pair must be set, and a name matching nothing is staged on the row (`submitted_type_name` / `submitted_issuer_organization_name`) for a reviewer to resolve before approval. Competencies work the same way per entry via `CompetencyIDs` / `SubmittedCompetencyNames`.

### `CredentialIssueInput` / `CredentialIssueRequest` (`credential_request.go:21-68`)

Validator-enforced constraints: `HolderUserID` required, `Name` 1-256 chars, batch size 1-100 items. Gin does not support nested multipart structs; handler parses manually via `buildIssueItems()`.

### `CredentialRevokeRequest` / `CredentialReExtractRequest` (`credential_request.go:71-93`)

Plain JSON `{"ids": [...]}`, 1-100 items per batch.

---

## Extract State Lifecycle (timestamp-derived)

Extraction status is derived from three timestamps (`extract_enqueued_at`, `extracted_at`, `extract_failed_at`) via `Credential.ExtractState()`. No extraction enum column exists in the database.

```
unextracted ──→ pending ──→ succeeded
                │
                └──→ failed ──→ pending (via ReExtract)
```

| State | Go Constant | Meaning | Condition |
|-------|-------------|---------|-----------|
| `pending` | `ExtractStatePending` | Awaiting OCR by Python River worker | `extract_enqueued_at IS NOT NULL` |
| `succeeded` | `ExtractStateSucceeded` | Text/IDs/embedding extracted and stored in Mongo | `extracted_at IS NOT NULL` |
| `failed` | `ExtractStateFailed` | Extraction failed; retryable via ReExtract | `extract_failed_at IS NOT NULL` |
| `unextracted` | `ExtractStateUnextracted` | No extraction performed — self-submitted default | All three extract timestamps NULL |

*Precedence:* `failed` wins over `succeeded` so a failed re-extract is not masked by a historical `extracted_at`.

**Sources:** `domain/credential.go:19-41` (constants), `domain/credential.go:150-165` (`ExtractState()`), migration `000001_initial_schema.up.sql`.

Extraction is enqueued at the moment the credential becomes approved, which is a different moment per workflow. Directly issued credentials are born approved, so `extract_enqueued_at` is stamped at creation (`credential_service.go:490`) and the job is enqueued in the same call — there is no separate approval step to wait for. Self-submitted credentials start with all extract timestamps nil (`unextracted`); `extract_enqueued_at` is stamped only when an officer approves (`credential_service.go:1272`). On-chain issuance is synchronous (keccak256 computed immediately), but extraction (text, IDs, embedding — needed by verify's fuzzy path) requires a slow Python OCR+EmbeddingGemma round-trip via River async worker.

**ReExtract flow** (`credential_service.go`):
1. Validates all targets exist, are in `ExtractStateFailed`, and have `file_uri`
2. Resets via `ClearExtractOutcome` (stamps fresh `extract_enqueued_at`, sets `extracted_at = NULL`, `extract_failed_at = NULL`, `extract_error = NULL`)
3. Enqueues River jobs
4. If enqueue fails, **compensates**: stamps `extract_failed_at` with `"reenqueue failed"` error

---

## On-Chain Registration

### Token ID Derivation

**Token ID derivation** (`chain/registry_service.go:254-258`):

packed := append(issuer.Bytes(), common.LeftPadBytes(nonce.Bytes(), 32)...)
packed = append(packed, holder.Bytes()...)
packed = append(packed, []byte(hash)...)
return new(big.Int).SetBytes(crypto.Keccak256(packed))

Token ID = `uint256(keccak256(issuer || zeroPadLeft(nonce, 32) || holder || hash))`

This matches the Solidity side in `CredentialRegistry.sol:160-169`.

### On-Chain Storage

**Mapping** on CredentialRegistry:

```solidity
enum CredentialStatus { None, Issued, Revoked }

mapping(bytes32 => CredentialStatus) public credentialHashToStatus;

function getCredentialHashStatuses(
    bytes32[] calldata hashes
) external view returns (CredentialStatus[] memory);
```

- `None (0)`: No credential issued for this hash
- `Issued (1)`: Active credential on-chain
- `Revoked (2)`: Credential was revoked

**Sources:** `CredentialRegistry.sol`.

### Issue Flow (On-Chain)

RegistryService.IssueCredentials (planned):
1. Fetches nonce from CredentialRegistry for signer
2. Packs: `issuer || pad32(nonce) || (holder || hash || uri)[]`
3. Signs EIP-191 digest with signer's encrypted private key
4. Calls `batchIssueCredentialsWithSignature(params)` via relayer
5. Contract mints ERC-721 tokens and sets `credentialHashToStatus[hash] = Issued`
6. Returns token IDs (derived from issuer+nonce+holder+hash)

### Revoke Flow (On-Chain)

RegistryService.RevokeCredentials:
1. Fetches nonce from CredentialRegistry for signer
2. Packs: `revoker || pad32(nonce) || pad32(tokenId)[]`
3. Signs EIP-191 digest
4. Calls `batchRevokeCredentialsWithSignature(params)` via relayer
5. Contract sets `credentialHashToStatus[hash] = Revoked`
6. Token remains soulbound (no transfer/burn allowed — `_update()` reverts)

**Known revocation gap:** `CredentialRegistry.sol:205` (`batchRevokeCredentialsWithSignature`) gates revocation only on the revoker holding the Issuer role (`onlyRoleOrAbove(params.revoker, CredentialAuthority.Role.Issuer)`, line 209) — `_revokeCredential` (line 375) never checks that the revoker is the credential's issuing organization. Any address holding the Issuer role can revoke any credential, not only ones it issued. Accepted within the current single-institution deployment; revisit before multi-tenant.

### Exact-Hash Verify Path

There is no `FindCredentialByHash` and no `tokenIdFromHash` — the exact-hash path is assembled in the service from two primitives:

1. `repo.FindByFileHashes(ctx, []string{uploadedHash}, verifyQuery)` (`credential_service.go:1745`) returns matching **approved** Postgres rows
2. Their `token_id`s feed `registryService.GetCredentialsByIds` (`chain/registry_service.go:114`) for the on-chain cross-check
3. A row that matches in Postgres but has no valid on-chain counterpart yields `IntegrityWarning` (400403) rather than a match

**Sources:** `chain/registry_service.go:106` (`FindNonce`), `:114` (`GetCredentialsByIds`), `:122` (`GetCredentialHashStatuses`), `:130` (`IssueCredentials`), `:214` (`RevokeCredentials`).

---

## Credential Status (timestamp-derived)

There is no separate DB status column. `Credential.Status()` (`domain/credential.go`) derives the workflow lifecycle purely from timestamps, in this precedence order:

| Status | Condition | Meaning |
|--------|-----------|---------|
| `pending` | no approval/rejection/revocation/past-expiry timestamp | Submitted, awaiting Issuer review |
| `approved` | `activated_at IS NOT NULL` (not revoked, rejected, or expired) | Minted on-chain (approved via review if submitted; directly registered if issued by officer) |
| `expired` | `expires_at IS NOT NULL AND expires_at <= NOW()` (not revoked or rejected) | Past expiration timestamp |
| `rejected` | `rejected_at IS NOT NULL` (not revoked) | Reviewed and refused |
| `revoked` | `revoked_at IS NOT NULL` | Previously approved, later invalidated |

Notes:

- `chk_credentials_activated_xor_rejected` makes approve/reject mutually exclusive at the DB level.
- Directly-issued credentials are stamped `activated_at` at creation (never `pending`). In the presentation layer, active credentials with `submitter_user_id != holder_user_id` are labeled Registered ("Didaftarkan"), while those with `submitter_user_id == holder_user_id` are labeled Approved ("Disetujui"). Filter controls use Approved & Registered ("Disetujui & Didaftarkan") to encompass both origins.
- Expiry is derived when `expires_at <= now`, including an unactivated submission; revocation and rejection take precedence. `GET /api/credentials?filters=status=approved` returns only active (approved/registered), unexpired, non-rejected, non-revoked records; the predicate is applied before pagination.
- The on-chain `CredentialStatus` enum (None/Issued/Revoked) is separate and reflects the chain state rather than DB state.

---

## API Routes

**Source:** `infrastructure/http/router.go:91-166`

| Route | Method | Auth | Handler | Notes |
|---|---|---|---|---|
| `/api/credentials/verify` | POST | None (public) | `Verify` | Rate-limited by global ApiRateLimitMiddleware |
| `/api/credentials` | GET | Issuer+ (on-chain) | `Paginate` | Search, filters, sorts, includes (holder/issuer/revoker) |
| `/api/credentials/:id` | GET | Issuer+ (on-chain) | `Find` | Single credential with optional Preload |
| `/api/credentials/:id/competencies` | PUT | Issuer+ (on-chain) | `LinkCompetencies` | Replace-set of resolved competency links |
| `/api/credentials/:id/metadata/suggestions` | GET | Issuer+ (on-chain) | `SuggestMetadata` | Top-5 pg_trgm similarity suggestions per staged name |
| `/api/credentials/:id/metadata` | PUT | Issuer+ (on-chain) | `ResolveMetadata` | Link existing or create new taxonomy rows for staged names |
| `/api/credentials/batch/issue` | POST | Issuer+ (on-chain) | `Issue` | Multipart form, 1-100 items; stamped approved at creation |
| `/api/credentials/batch/submit` | POST | Authenticated (no role gate) | `Submit` | Self-submission; id-or-name, unmatched names staged |
| `/api/credentials/batch/approve` | POST | Issuer+ (on-chain) | `Approve` | JSON body `{"ids": [...]}`; blocked while metadata unresolved |
| `/api/credentials/batch/reject` | POST | Issuer+ (on-chain) | `Reject` | JSON body `{"ids": [...]}`; keeps unresolved names as audit |
| `/api/credentials/batch` | PUT | Issuer+ (on-chain) | `Update` | Batch update |
| `/api/credentials/batch/revoke` | POST | Issuer+ (on-chain) | `Revoke` | JSON body `{"ids": [...]}` |
| `/api/credentials/batch/reextract` | POST | Issuer+ (on-chain) | `ReExtract` | JSON body `{"ids": [...]}` |
| `/api/credentials/:id/file` | GET | Authenticated (no role gate) | `DownloadFile` | Download decrypted credential file; authorization via policy (holder OR Issuer+) |
| `/api/credential-types` | GET | Authenticated (no role gate) | `Paginate` | Taxonomy lookup — open to any authenticated user |
| `/api/credential-types` | POST | Issuer+ (on-chain) | `Store` | Create a credential type |
| `/api/credential-types/:id` | PUT | Issuer+ (on-chain) | `Update` | Rename / deactivate a credential type |
| `/api/credential-types/:id` | DELETE | Issuer+ (on-chain) | `Destroy` | Delete a credential type |
| `/api/issuer-organizations` | GET | Authenticated (no role gate) | `Paginate` | Taxonomy lookup — open to any authenticated user |
| `/api/issuer-organizations` | POST | Issuer+ (on-chain) | `Store` | Create an issuer organization |
| `/api/issuer-organizations/:id` | PUT | Issuer+ (on-chain) | `Update` | Rename / deactivate an issuer organization |
| `/api/issuer-organizations/:id` | DELETE | Issuer+ (on-chain) | `Destroy` | Delete an issuer organization |
| `/api/competencies` | GET | Authenticated (no role gate) | `Paginate` | Taxonomy lookup — open to any authenticated user |
| `/api/competencies` | POST | Issuer+ (on-chain) | `Store` | Create a competency |
| `/api/competencies/:id` | PUT | Issuer+ (on-chain) | `Update` | Rename / deactivate a competency |
| `/api/competencies/:id` | DELETE | Issuer+ (on-chain) | `Destroy` | Delete a competency |
| `/api/users/self/credentials` | GET | Authenticated | `SelfPaginate` | Scoped to `holder_user_id == auth_user.id` |
| `/api/users/self/credentials/:id` | GET | Authenticated | `SelfFind` | 404 if not owned (no ID leak) |
| `/api/notifications` | GET | Authenticated (no role gate) | `List` | Self-only notifications list with keyset pagination (`?status=all\|unread&limit=N&cursor=...`) |
| `/api/notifications/unread-count` | GET | Authenticated (no role gate) | `UnreadCount` | Self-only unread notifications count |
| `/api/notifications/read-all` | PUT | Authenticated (no role gate) | `MarkAllRead` | Self-only mark all unread notifications as read |
| `/api/notifications/:id/read` | PUT | Authenticated (no role gate) | `MarkRead` | Self-only mark single notification as read |
| `/api/overview` | GET | Authenticated (no role gate) | `Get` | Role-conditional dashboard: credential_counts + recents (Holder: own, Issuer+: system-wide). Optional `?limit=N` controls recent items per category (default 5). |

**Route middleware chain:** `ErrorLoggerMiddleware` → `I18nMiddleware` → `ApiRateLimitMiddleware` → `AuthMiddleware` → `IssuerRoleMiddleware` (for credential management and taxonomy-write routes).

Credential policy checks use **DB-stored role rank**, not on-chain.

---

## Policy Rules

### Credential Policy Rules

The credential policy interface lives at `feature/credential/credential_policy.go:16-20`. It defines three methods:

| Method | Line | Purpose |
|---|---|---|
| `IssuePostFetch` | 30 | `return nil` — a stub. Role enforcement is via `IssuerRoleMiddleware`, on-chain |
| `RevokePostFetch` | 34 | `return nil` — a stub. Role enforcement is via `IssuerRoleMiddleware`, on-chain |
| `DownloadFilePreFetch` | 38 | The holder of the credential first, then any Issuer+; everyone else gets `CodeCredentialFileDownloadForbidden` |

`DownloadFilePreFetch` checks the **holder** before the role (`credential_policy.go:39-47`), which is what makes `/api/credentials/:id/file` usable by a plain Holder for their own document while staying closed to a Holder reaching for someone else's.

Role enforcement for issue/revoke/reextract is done at the **route level** by `IssuerRoleMiddleware` (on-chain check), not by credential policy. The route-level guard is applied in `router.go:114-160`.

There is no `IssuePreFetch`, `RevokePreFetch`, `VerifyPreFetch`, or `ReExtractPreFetch` method. The verify route is public (no auth middleware — `router.go:91`).

> **Known gap:** `IssuePostFetch` and `RevokePostFetch` are stubs, and nothing anywhere prevents an Issuer+ from submitting a credential for themselves and then approving it. Both are accepted in the current single-institution deployment.

---

## Duplicate Hash Rules

### Global File Hash Uniqueness

| Scenario | Rule |
|---|---|
| Same hash, any holder, active (neither revoked nor rejected) | **Blocked** |
| Same hash, earlier row revoked | **Allowed** — re-issue after revocation (any holder) |
| Same hash, earlier row rejected | **Allowed** — a refused document may be resubmitted |

"Active" is defined identically in three places, and all three must agree:

1. **Submit-time count** — `repo.CountActiveByFileHashes` (`gorm_credential_repository.go:692`) counts rows matching the hash `WHERE revoked_at IS NULL AND rejected_at IS NULL`. One count per batch, no N+1 (`credential_service.go:817`). A hit returns validation error `validation_issue_duplicate_file_hash`.
2. **DB partial unique index** — `idx_credentials_file_hash_active` on `file_hash WHERE revoked_at IS NULL AND rejected_at IS NULL` (`000001_initial_schema.up.sql`). This is the final authority under concurrency; the count above is only a fast pre-check.
3. **On-chain** — `batchIssueCredentialsWithSignature` reverts `IssuedCredentialError` if `credentialHashToStatus[hash] == Issued`. `mintCredentials` maps that revert to `CodeCredentialIssueDuplicateFileHash` (400242) (`credential_service.go:2154-2157`).

Note the asymmetry: a **rejected** row frees the hash in Postgres but was never minted, so the chain never held it. A **revoked** row frees it on both sides. Both are legal resubmission paths; the credential seeder exercises the rejected case (one rejected row and one pending row sharing a file hash).

---

## Batch Flows

### Issue (`credential_service.go:187-342`)

Architecture: sync chain, async embeddings.

1. Role enforcement via `IssuerRoleMiddleware` (route-level, on-chain check) — no policy gate
2. `issueValidate` — pre-computed holder lookup + on-chain `GetCredentialHashStatuses` batch → holder existence + duplicate file hash checks
3. `issuePrepareCredentials` — encrypt files, persist to storage, build domain entities with `ExtractEnqueuedAt: &now`
4. `issueCommit` (within UoW): `Store` → check `file_uri` invariant → `syncBlockchainIssue` → `Update` token IDs → enqueue River extraction jobs
5. Chain failure rolls back DB transaction; orphan files cleaned up via `issueCleanupOrphanFiles`

**All-or-nothing:** Any per-item failure (validation, chain sync, storage, hash computation) aborts the entire batch inside the UoW — no partial results are returned. The handler returns a single error code via `responder.SendError`.

**File cleanup:** `cleanupOrphanFiles` deletes stored files on validation/chain failure — best-effort (log-and-continue).

**Invariant:** Every stored credential with `file_uri == nil` triggers `CodeCredentialIssueStorageFailed` (400245) BEFORE on-chain mint — prevents orphaned NFT.

### Revoke (`credential_service.go:362-427`)

1. Role enforcement via `IssuerRoleMiddleware` (route-level, on-chain check) — no policy gate
2. UoW: `FindByIds` targets → validate all found → check none already revoked → `RevokePostFetch` → CASE batch UPDATE (revoked_at, revoker_user_id) → `syncBlockchainRevoke` with decimal token IDs
3. Already-revoked check: `CodeCredentialRevokeAlreadyRevoked` (400342)
4. Missing targets: `CodeCredentialRevokeNotFound` (400341)
5. Token IDs with non-nil value collected for on-chain sync

### ReExtract (`credential_service.go:1950-2027`)

1. Role enforcement via `IssuerRoleMiddleware` (route-level, on-chain check) — no policy gate
2. UoW: `FindByIds` targets → validate all exist + are failed + have file_uri → `ClearExtractOutcome` (sets extract_enqueued_at = now, clears extracted_at, extract_failed_at, extract_error) → enqueue River jobs
3. On enqueue failure: **compensate** — stamp back to failed with `"reenqueue failed"` error

### Submit / Resolve / Approve (`credential_service.go:649-1544`)

Self-submission pipeline: a holder submits a credential (possibly naming taxonomy rows that do not exist yet), a reviewer resolves the staged names, then approves (mints). Direct issue (`/batch/issue`) bypasses this — those rows are stamped approved at creation.

**Submit** (`Submit`, `credential_service.go:649`) — `POST /api/credentials/batch/submit`, any authenticated user:

1. Type and issuer organization are **id-or-name**; competency IDs are strict, competency names follow the same name rules.
2. An unknown ID is a **client bug** — it errors (the row must exist and be active).
3. A name matching an existing **active** row resolves immediately (the row's ID is stamped).
4. A name matching an **inactive** row errors — a reviewer could only link it to a deliberately retired row.
5. A name matching nothing is **staged** on the credential row (`submitted_type_name` / `submitted_issuer_organization_name` / `submitted_competencies` JSONB) with the paired FK left NULL. Nothing is silently created.
6. Name lookups are batched: exactly one query per taxonomy table per batch (`resolveSubmissionNames`, `credential_service.go:722`) — no N+1.
7. Number uniqueness is **skipped** for a staged (unresolved) organization at submit — there is no org scope to be unique within yet; it is re-checked at resolution time.
8. Audit trail: `submitter_user_id` is set to the authenticated submitter (the holder), while `issuer_user_id` remains NULL until approval.

**Resolve metadata** (`ResolveMetadata`, `credential_service.go:1286`) — `PUT /api/credentials/:id/metadata`:

1. Pending-only: a non-pending credential returns `CodeCredentialMetadataResolveNotPending` (401541, HTTP 422) — the same immutability rule as batch Update.
2. Idempotent upsert-by-name creation: creating a taxonomy row by name routes through the taxonomy service's `Store`, so two concurrent reviewers racing on the same name converge on one row (backstopped by the `LOWER(name)` unique index).
3. A resolved competency becomes a real `competency_credential` join row; the submitted name's `resolved_id` is stamped in place.
4. Staged names are **never erased** — the free text survives resolution (and rejection) as an audit trail.
5. Number uniqueness is re-checked once the credential finally has an org scope (`CodeCredentialMetadataResolveNumberDuplicate` 401545, HTTP 409).

**Suggestions** (`SuggestMetadataMatches`, `credential_service.go:1489`) — `GET /api/credentials/:id/metadata/suggestions`:

- Top 5 per staged name (`metadataSuggestionLimit = 5`), ranked by Postgres `pg_trgm` trigram similarity (`similarity(name, ?) > 0.1`).

**Approve** (`Approve`, `credential_service.go:1213`) — `POST /api/credentials/batch/approve`:

1. Refuses any credential with unresolved staged metadata — `CodeCredentialApproveUnresolvedMetadata` (401546, HTTP 422).
2. The DB CHECK `chk_credentials_activated_metadata_resolved` backstops the type + organization half; the service enforces the JSONB competency half (`UnresolvedMetadata`, `domain/credential.go:143-158`) so the error is usable either way.
3. Refuses any credential whose resolved type, organization, or competency has since been deactivated — `CodeCredentialApproveInactiveMetadata` (401547, HTTP 422). Resolution and approval are separate calls, potentially days apart; a taxonomy row can be retired in between, and the mint is a permanent soulbound NFT so this must be caught before minting, not after. The fetch preloads `type`/`issuer_organization`/`competencies` (no extra query) and `InactiveMetadata` (`domain/credential.go:184-199`) checks their `Active` flag; a nil relation is treated as fine since the unresolved check above already covers nil FKs.
4. Approval and on-chain mint run in the **same unit of work** — a failed mint rolls the approval back and the rows stay pending.
5. Stamps `issuer_user_id` with the approving officer ID, sets `activated_at` to now, and enqueues background OCR extraction.

**Reject** (`Reject`, `credential_service.go:1554`) — `POST /api/credentials/batch/reject`:

- No unresolved-metadata guard: rejected rows keep their unresolved names as an audit trail.

---

## Verify Pipeline (`credential_service.go`, `Verify`)

Three-stage pipeline: **Exact Hash → Cache → Fuzzy**. Hash is Keccak256 of the original uploaded bytes, not encrypted storage bytes.

### Stage 1: Exact Hash Path
- `FindByFileHashes([uploadedHash])` selects activated Postgres credentials; pending/rejected submissions are not public candidates.
- The implemented Postgres bridge parses positive `token_id`s and calls `registryService.GetCredentialsByIds` on-chain.
- Compare token ID, uploaded/stored/chain hash, URI credential ID, holder/issuer wallet relationships, physical issue date, expiry, credential number and revocation presence.
- A completed matching read returns `Authentic` (400401) or `Revoked` (400402). Authentic results are then checked for expiry (400413), then disabled parties (400410-400412).
- Missing/invalid token IDs or a completed read without corresponding facts return `IntegrityWarning` (400403, HTTP409).
- Failed chain calls return operational `ChainUnavailable` (400447, HTTP503), with no credential verdict, AI fallback or cache write. Unclassified contract errors are inconclusive, not proof of mismatch.
- Exact outcomes are never cached; every activated exact upload retries the live correspondence check before consulting any old cache entry. No similarity percentage is produced.

### Stage 2: Cache Lookup
- Only runs when no activated exact-hash row exists.
- Query MongoDB `credential_verifications` by `uploaded_file_hash`; fetch the matched verifiable credential fresh when present.
- Cached `Authentic` alone receives fresh DB revocation, expiry and party checks; missing credential becomes integrity warning. Other cached codes retain their verdict, even if the accompanying credential status has changed. No live chain read occurs here.
- Cache TTL: configurable via `AI_VERIFICATION_CACHE_TTL_HOURS` (default24h).

### Stage 3: Fuzzy Path
- Call Python AI `/extract-ids` for identifiers in the uploaded document.
- No identifiers produces `NoIdentifiers` (400408); no ranked extracted candidates produces `NoMatch` (400409). Both are cached without a semantic score.
- `FindRankedByIds` ranks Mongo extractions by ID intersection count (cap10); `verifyPickBestMatch` prefers non-revoked candidates and newer `IssuedAt` when tied.
- Call Python `/verify` with the best candidate's stored embedding. No live chain correspondence check occurs on this path.
- Legacy wire codes/thresholds remain: `tampered`400404 (high text similarity), `suspicious`400405 (similar text), `low_similarity`400406 (limited similarity), otherwise `not_similar`400407 (no close text match).
- Current Python verdict logic never returns `authentic`; Go retains compatibility handling for that value, including party/expiry overrides.
- Cache the result with matched ID and similarity score/percent. Stored credential lifecycle is independent of the uploaded-file comparison.

### Meaning and Party Overrides

Text similarity does not establish authenticity or prove alteration. Rescans/reexports can change bytes without fraud. Use the original registered file or contact the issuing organization. The service does not independently confirm credential numbers with external issuing organizations.

Party-disabled overrides apply only to `Authentic`:

| Condition | Override Code |
|-----------|---------------|
| Holder soft-deleted, Issuer live | `HolderDisabled` (400410) |
| Issuer soft-deleted, Holder live | `IssuerDisabled` (400411) |
| Both soft-deleted | `PartyDisabled` (400412) |

Exact and cached-authentic paths check expiry before disabled parties. Compatibility handling for a future AI `authentic` checks parties before expiry. Existing non-authentic cached codes are not recomputed into every lifecycle override.

### Verdict and Operational HTTP Status Codes

| Outcome | HTTP Status |
|---------|-------------|
| All verdicts except IntegrityWarning | 200 |
| `IntegrityWarning` (400403), structured verdict data | 409 |
| `ChainUnavailable` (400447), no verdict data | 503 |

---

## Storage Architecture

### PostgreSQL (GORM) — `credentials` table

Primary store for credential metadata, file hash, token ID, status flags. Full migration at `000001_initial_schema.up.sql:39-63`.

```sql
CREATE INDEX idx_credentials_holder_user_id ON credentials(holder_user_id);
CREATE INDEX idx_credentials_issuer_user_id ON credentials(issuer_user_id);
CREATE INDEX idx_credentials_revoked_at     ON credentials(revoked_at);
CREATE INDEX idx_credentials_extract_enqueued_at ON credentials(extract_enqueued_at) WHERE extract_enqueued_at IS NOT NULL;
CREATE INDEX idx_credentials_extract_failed_at   ON credentials(extract_failed_at)   WHERE extract_failed_at IS NOT NULL;
CREATE INDEX idx_credentials_file_hash      ON credentials(file_hash);
```

**Repository methods** (`gorm_credential_repository.go`):
- `Get`: pagination with search (name, meta TEXT, id, token_id, file_hash, number, holder/issuer/revoker name/email/number, joined type name, joined issuer organization name, competency name via EXISTS), filters (name/issued_at/revoked_at/holder_user_id), sorts (name/issued_at/revoked_at + holder_name/email/number/phone), Preload (holder/issuer/revoker). Type/organization joins are gated on `HasSearch()` and are 1:1 LEFT JOINs so they never inflate the paginated total; staged (`submitted_*`) names are intentionally excluded since reviewers reach unresolved credentials through the review queue, not free-text search.
- `Find`: single row by ID with optional Preload
- `FindByIds`: batch lookup, single IN-clause
- `FindByHolderId`: scoped to one holder
- `FindByFileHashes`: batch dup-check, single IN-clause
- `Store`: batch insert with ULID generation
- `Update`: CASE-based batch UPDATE (single SQL statement regardless of batch size)

### MongoDB — `credential_extractions`

Lightweight collection: `credential_id` (unique), `file_hash`, text, `ids[]`, `embedding[]` (768 floats), `created_at`, `updated_at`.

Used by fuzzy verify path only. Repository: `domain.CredentialExtractionRepository` (`credential_extraction.go:32-42`).

Searchable by `ids.value` via aggregation pipeline (`FindRankedByIds`).

### MongoDB — `credential_verifications`

TTL-bounded verify result cache. Keyed by `uploaded_file_hash`. Fields: `verdict_code`, `matched_credential_id`, `similarity_score`, `similarity_percent`, `created_at`.

TTL enforced via `created_at` MongoDB TTL index (default 24h, configurable via `AI_VERIFICATION_CACHE_TTL_HOURS`).

Repository: `domain.CredentialVerificationRepository` (`credential_verification.go:22-28`).

### Local File Storage (IPFS-compatible interface)

Persisted via `storage.Storage`. Storage base path configured by `STORAGE_PATH` (default `"uploads"`). Credential files stored under `{STORAGE_PATH}/{CREDENTIAL_FILE_STORAGE_PATH}/{filename}` where `CREDENTIAL_FILE_STORAGE_PATH` defaults to `"credentials"`. The DB `file_uri` field stores only the filename — subdirectory is reconstructed at read time.

Files are encrypted at rest with AES-256-GCM using `FILE_ENCRYPTION_KEY`. The file hash (keccak256) is computed from the **original plaintext** before encryption, so on-chain fingerprints always represent the original document.

**File URI format:** `ULID.ext` (e.g. `01JQNXYZ...pdf`). Full on-disk path: `uploads/credentials/01JQNXYZ...pdf`.

Extendable to IPFS via `storage.Storage` interface.

---

## Error Codes

### Credential Fetch (40-01)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400100 | `CodeCredentialFetchSuccess` | 200 | Credential(s) fetched |
| 400140 | `CodeCredentialFetchNotFound` | 404 | Credential not found (or not owned, self-find) |
| 400141 | `CodeCredentialFetchValidation` | 400 | Invalid query parameters |

### Credential Issue (40-02)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400200 | `CodeCredentialIssueSuccess` | 200 | All credentials issued |
| 400241 | `CodeCredentialIssueValidation` | 400 | Validation error (invalid MIME, file too large) |
| 400242 | `CodeCredentialIssueDuplicateFileHash` | 409 | Duplicate file hash (holder already has active credential) |
| 400243 | `CodeCredentialIssueHolderNotFound` | 400 | Target holder user not found |
| 400244 | `CodeCredentialIssueBlockchainSyncFailed` | 500 | On-chain mint failed |
| 400245 | `CodeCredentialIssueStorageFailed` | 500 | File storage failed (or empty path, or missing file_uri) |
| 400246 | `CodeCredentialIssueHashFailed` | 500 | Hash computation failed |

Direct issuance resolves type, organization and competencies by ID at request time, so it owns a second block of codes for taxonomy failures the submit path never hits (a submission stages free text and defers resolution to review):

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400247 | `CodeCredentialIssueTypeNotFound` | 400 | `type_id` does not resolve |
| 400248 | `CodeCredentialIssueTypeInactive` | 400 | Credential type is deactivated |
| 400249 | `CodeCredentialIssueOrganizationNotFound` | 400 | `issuer_organization_id` does not resolve |
| 400250 | `CodeCredentialIssueNumberDuplicate` | 409 | `number` already used within that organization (`uq_credentials_issuer_org_number`) |
| 400251 | `CodeCredentialIssueCompetencyNotFound` | 400 | A referenced competency does not resolve |
| 400252 | `CodeCredentialIssueOrganizationInactive` | 400 | Issuer organization is deactivated |
| 400253 | `CodeCredentialIssueCompetencyInactive` | 400 | A referenced competency is deactivated |

### Credential Revoke (40-03)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400300 | `CodeCredentialRevokeSuccess` | 200 | Credentials revoked |
| 400340 | `CodeCredentialRevokeFailed` | 500 | General revoke failure |
| 400341 | `CodeCredentialRevokeNotFound` | 404 | One or more credential IDs not found |
| 400342 | `CodeCredentialRevokeAlreadyRevoked` | 409 | One or more credentials already revoked |
| 400343 | `CodeCredentialRevokeBlockchainSyncFailed` | 500 | On-chain revocation failed |
| 400344 | `CodeCredentialRevokeNotApproved` | 422 | Target is not in `approved` status — only an approved credential can be revoked |

### Credential Verify (40-04)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400400 | `CodeCredentialVerifySuccess` | 200 | Verify succeeded (unused — verdict codes used instead) |
| 400440 | `CodeCredentialVerifyFailed` | 500 | General verify failure |
| 400441 | `CodeCredentialVerifyValidation` | 400 | Invalid input (bad file, wrong MIME, too large) |
| 400442 | `CodeCredentialVerifyExtractNotReady` | 503 | Extraction not yet complete |
| 400443 | `CodeCredentialVerifyExtractFailed` | 500 | Extraction previously failed |
| 400444 | `CodeCredentialVerifyAiServiceFailed` | 502 | Python AI service unreachable/errored |
| 400445 | `CodeCredentialVerifyCredentialNotFound` | 404 | Matched credential not found in DB |
| 400446 | `CodeCredentialVerifyDocumentUnreadable` | 422 | Uploaded file could not be read as a document |
| 400447 | `CodeCredentialVerifyChainUnavailable` | 503 | Chain read could not complete; no verdict |

#### Verdict Codes (400401-400413)

| Code | Constant | HTTP | Stage | Meaning |
|------|----------|------|-------|---------|
| 400401 | `CodeCredentialVerifyAuthentic` | 200 | Exact | Exact bytes and registered chain facts correspond; Python currently cannot emit authentic |
| 400402 | `CodeCredentialVerifyRevoked` | 200 | Exact | Hash matched but revoked on-chain |
| 400403 | `CodeCredentialVerifyIntegrityWarning` | 409 | Exact | Missing/invalid token bridge or completed chain read disagrees with stored facts |
| 400404 | `CodeCredentialVerifyTampered` | 200 | Fuzzy | High text similarity; not proof of alteration |
| 400405 | `CodeCredentialVerifySuspicious` | 200 | Fuzzy | Similar text; not an authenticity verdict |
| 400406 | `CodeCredentialVerifyLowSimilarity` | 200 | Fuzzy | Similar but below threshold |
| 400407 | `CodeCredentialVerifyNotSimilar` | 200 | Fuzzy | No fuzzy match |
| 400408 | `CodeCredentialVerifyNoIdentifiers` | 200 | Fuzzy | AI could not extract any identifiers |
| 400409 | `CodeCredentialVerifyNoMatch` | 200 | Fuzzy | Identifiers extracted but no database match |
| 400410 | `CodeCredentialVerifyHolderDisabled` | 200 | Override | Authentic but holder soft-deleted |
| 400411 | `CodeCredentialVerifyIssuerDisabled` | 200 | Override | Authentic but issuer soft-deleted |
| 400412 | `CodeCredentialVerifyPartyDisabled` | 200 | Override | Authentic but both parties soft-deleted |
| 400413 | `CodeCredentialVerifyExpired` | 200 | Override | Authentic but `expires_at` has passed |

Verdict codes (400401-400413) deliberately avoid CC 01-13 for other credential codes — these are success outcomes, not errors (`domain/codes.go:128-131`).

### Credential Re-Extract (40-05)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400500 | `CodeCredentialReExtractSuccess` | 200 | Re-extraction queued |
| 400540 | `CodeCredentialReExtractNotFound` | 404 | One or more credential IDs not found |
| 400541 | `CodeCredentialReExtractNotEligible` | 409 | One or more credentials not in failed state (or missing file_uri) |

### Credential File Download (40-06)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400600 | `CodeCredentialFileDownloadSuccess` | 200 | File downloaded |
| 400640 | `CodeCredentialFileDownloadNotFound` | 404 | Credential not found |
| 400641 | `CodeCredentialFileDownloadForbidden` | 403 | Not authorized (not holder, not Issuer+) |
| 400642 | `CodeCredentialFileDownloadDecryptionFailed` | 500 | File decryption error |
| 400643 | `CodeCredentialFileDownloadNoFile` | 404 | Credential has no stored file |

### Credential Submission (40-11)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 401100 | `CodeCredentialSubmitSuccess` | 200 | Submission accepted, awaiting review |
| 401141 | `CodeCredentialSubmitStorageFailed` | 500 | File encryption or storage failed (`credential_service.go:1015,1021`) |

Submission reuses the Issue group's validation and duplicate-hash codes (400241/400242); only storage failure gets its own code.

### Credential Review (40-12)

Approve, Reject and the review-path revoke all report through this group.

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 401200 | `CodeCredentialReviewSuccess` | 200 | Review action applied |
| 401240 | `CodeCredentialReviewNotFound` | 404 | One or more credential IDs not found |
| 401241 | `CodeCredentialReviewAlreadyApproved` | 409 | Target already approved — review is single-shot |
| 401242 | `CodeCredentialReviewAlreadyRejected` | 409 | Target already rejected |
| 401243 | `CodeCredentialReviewAlreadyRevoked` | 409 | Target already revoked |
| 401244 | `CodeCredentialReviewBlockchainSyncFailed` | 500 | Approval minted nothing — on-chain write failed |

### Credential Competency Link (40-13)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 401300 | `CodeCredentialCompetencyLinkSuccess` | 200 | Competencies linked |
| 401340 | `CodeCredentialCompetencyLinkCredentialNotFound` | 404 | Credential not found |
| 401341 | `CodeCredentialCompetencyLinkCompetencyNotFound` | 400 | One or more competency IDs do not resolve (`credential_service.go:2103`) |

### Credential Update (40-14)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 401400 | `CodeCredentialUpdateSuccess` | 200 | Credential updated |
| 401440 | `CodeCredentialUpdateNotFound` | 404 | Credential not found |
| 401441 | `CodeCredentialUpdateNotPending` | 409 | Credential is no longer pending — an approved, rejected or revoked row is immutable (`credential_service.go:1104`) |

### Credential Metadata Resolve / Approve (40-15)

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 401500 | `CodeCredentialMetadataResolveSuccess` | 200 | Metadata resolved |
| 401501 | `CodeCredentialMetadataSuggestSuccess` | 200 | Suggestions fetched |
| 401540 | `CodeCredentialMetadataResolveNotFound` | 404 | Credential not found |
| 401541 | `CodeCredentialMetadataResolveNotPending` | 422 | Credential not pending (immutable) |
| 401542 | `CodeCredentialMetadataResolveNothingStaged` | 422 | Nothing staged to resolve |
| 401543 | `CodeCredentialMetadataResolveTargetNotFound` | 404 | Target taxonomy row not found |
| 401544 | `CodeCredentialMetadataResolveTargetInactive` | 422 | Target taxonomy row inactive |
| 401545 | `CodeCredentialMetadataResolveNumberDuplicate` | 409 | Credential number already used within the org |
| 401546 | `CodeCredentialApproveUnresolvedMetadata` | 422 | Approval blocked — unresolved staged metadata |
| 401547 | `CodeCredentialApproveInactiveMetadata` | 422 | Approval blocked — resolved type, organization, or competency has been deactivated |

### Taxonomy CRUD (40-08, 40-09, 40-10)

The three lookup tables a credential resolves against share one code shape. Success codes are fetch/store/update/destroy in order.

| Domain | Success | Not found | Name duplicate |
|--------|---------|-----------|----------------|
| Credential Type | 400800-400803 | 400840 (404) | 400841 (409) |
| Issuer Organization | 400900-400903 | 400940 (404) | 400941 (409) |
| Competency | 401000-401003 | 401040 (404) | 401041 (409) |

Hard deletion is guarded — a row still referenced by a credential cannot be destroyed. Deactivating (`active = false`) is the everyday path; these codes are raised by the services and again by the repository's 23503 translation backstop (`domain/codes.go:158-166`):

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 400741 | `CodeCredentialTypeDestroyInUse` | 409 | Credential type is referenced by a credential |
| 400742 | `CodeCredentialIssuerOrganizationDestroyInUse` | 409 | Issuer organization is referenced by a credential |
| 400743 | `CodeCompetencyDestroyInUse` | 409 | Competency is referenced by a credential |

### Notifications (40-16)

Generic inbox codes retain their existing numeric values for coordinated Go/React compatibility.

| Code | Constant | HTTP | Meaning |
|------|----------|------|---------|
| 401600 | `CodeNotificationFetchSuccess` | 200 | Notifications retrieved |
| 401601 | `CodeNotificationUnreadCountSuccess` | 200 | Unread count retrieved |
| 401602 | `CodeNotificationMarkReadSuccess` | 200 | Single notification marked as read |
| 401603 | `CodeNotificationMarkAllReadSuccess` | 200 | All notifications marked as read |
| 401640 | `CodeNotificationNotFound` | 404 | Notification not found (or not owned by user) |

---

## Allowed MIME Types

Defined in `credential_request.go:10-16`:

```go
var allowedMIMETypes = map[string]bool{
    "application/pdf": true,
    "image/jpeg":      true,
    "image/png":       true,
    "image/webp":      true,
    "image/tiff":      true,
}
```

Max file size: 10 MB (`maxFileBytes = 10 * 1024 * 1024`). Validation enforced at handler level for both issue and verify.

---

## Architecture: Async Extraction Model

```
Issue (sync chain, async embeddings)
  │
  ├── 1. Compute hash (Go, sync)
  ├── 2. Store file (Go, sync)
  ├── 3. DB INSERT (Go, sync)
  ├── 4. On-chain mint (Go → Registry, sync)
  ├── 5. DB UPDATE token_id (Go, sync)
  └── 6. Enqueue River job (Go → River/PG, async)
         │
         └── River worker (async)
              ├── 1. Call Python /extract (text, IDs, embedding)
              ├── 2. Store in Mongo credential_extractions
              └── 3. DB UPDATE extracted_at = now (or extract_failed_at on terminal failure)
```

River jobs live in Postgres (`river_jobs` table) but use a separate `pgx` connection pool from GORM's (`database/sql` + `pgx`). They cannot share the GORM transaction. Mitigation: credentials stay in `pending` and ReExtract recovers failures.

---

## Credential Lifecycle Notifications

Holders receive asynchronous email notifications for four lifecycle events:

1. **`approved`**: Credential submission approved by an Issuer+.
2. **`rejected`**: Credential submission rejected by an Issuer+.
3. **`registered`**: Credential issued directly by an Issuer+ (bypassing submission).
4. **`revoked`**: Credential revoked by an Issuer+.

### Architecture & Trigger Points

```
Lifecycle Mutation (Issue / Approve / Reject / Revoke)
  │
  ├── 1. On-Chain Transaction / DB Mutation (Go, sync)
  ├── 2. Commit DB Transaction (Go, sync - GORM UoW or issueCommit)
  └── 3. Post-Commit Enqueue (Go, async - context.WithoutCancel with 5s timeout)
         │
         └── River Queue (Postgres river_jobs)
              │
              └── CredentialNotificationWorker (async)
                   ├── 1. Fetch current holder user via UserRepository.Find
                   ├── 2. Render HTML/text in recipient's current saved language (id or en)
                   └── 3. Deliver via bounded SMTP (net/smtp + STARTTLS)
```

- **Post-commit execution**: Notifications are enqueued strictly after database commits and on-chain transactions succeed. If a transaction fails or rolls back, no notification is queued.
- **Delivery limits & fault tolerance**:
  - Enqueue failures are logged and swallowed; mutation endpoints always return success.
  - Job uniqueness is keyed by `(credential_id, event)` with up to 5 retry attempts.
  - Non-retryable conditions (deleted user, missing user, empty email) cancel the job (`river.JobCancel`).
  - Transport failures (SMTP network/auth) trigger exponential backoff retries.
  - Commit-to-enqueue crashes may drop emails; SMTP acceptance races may cause duplicate emails. In-app rows are stored separately in `notifications` within the lifecycle transaction; they are not an email outbox. The River job kind `credential_notification`, worker, payload, and email event values remain credential-specific.
- **Security & Privacy**: Email payloads contain no sensitive attributes, file paths, or rejection reasons. Emails link directly to authenticated `/credentials/:id` route.

---

### Persisted inbox and reference review

`feature/notification` owns the recipient-scoped inbox at `/api/notifications`. Each row has nullable paired `subject_type` / `subject_id`, nullable JSON `data`, and nullable `updated_at`. Subject summaries are current authorized data, not snapshots. Credential summaries remain holder-only. Missing targets produce unavailable entries without navigation; subjectless entries retain localized fallback copy and read actions.

Persisted credential types are `credential_registered`, `credential_approved`, `credential_rejected`, and `credential_revoked`. These prefixes do not change the separate email events above. First mark-read sets `read_at` and `updated_at` to the same database timestamp. Repeated reads and replay-safe inserts preserve timestamps, payload, and independent recipient read state.

Submitting a credential type, issuer organization, or competency creates a pending reference and one alert per eligible reviewer in the same transaction. Recipients are live DB Issuer/Admin/SuperAdmin users except the submitter, selected once and inserted as one batch. Zero reviewers is valid; notification failure rolls back the proposal. Direct approved creation sends no alert. Types are `credential_type_submitted`, `credential_issuer_organization_submitted`, and `competency_submitted`. Approval/rejection preserves that historical type and does not read or delete alerts. No reference outcome emails or alerts, retrospective fan-out, or credential-submission reviewer alerts are generated.

Reviewer alerts, totals, unread counts, and summaries require a current live DB Issuer+ role and one on-chain permission check per request. Chain denial or lookup failure hides reviewer alerts while retaining personal credential alerts. The recipient snapshot never grants review permission.

The bell counts visible unread inbox rows. Review indicators count live pending subject rows (`activated_at IS NULL AND rejected_at IS NULL`) independently of notifications and read state. Reference counts use their list endpoint with `limit=1`, `filters=status=pending`, and pagination `total`. History links use `status=all` and current-name search; queue links use `status=pending`.

This is a breaking API/schema cutover coordinated with React. Existing databases need a separately approved data transition; editing the initial schema does not upgrade them. No live migration, conversion, or reset is authorized by these application changes. See [DATABASE.md](DATABASE.md).

### Notification seed data

`CredentialNotificationSeeder` runs after `CredentialSeeder` and derives notifications only from the existing credential fixture IDs. It performs one batch read and one batch insert, without invoking lifecycle services, email jobs, or blockchain operations. It does not backfill production credentials.

- Direct issue: `credential_registered` at `CreatedAt`; self-submission: no event until `credential_approved` at `ActivatedAt`.
- `RejectedAt` and `RevokedAt` add `credential_rejected` and `credential_revoked`. Revoked and expired credentials retain earlier registration/approval history; expiry and extraction create no events.
- Recipients are always credential holders, including soft-deleted fixture holders. Issuers/reviewers receive no rows merely for acting on credentials.
- IDs and mixed read/unread examples are deterministic per fixture slot and event. Composite conflict handling preserves existing IDs, event timestamps, and user read state on replay.
- Missing individual fixtures are skipped. No seeded credentials returns a prerequisite error; pending-only fixtures succeed without inserts.

For an already seeded development database, the standalone selection is:

```bash
go run main.go seed --env .env --names credential-notification
```

Named seeders run only requested names in caller order. If requesting both, use `--names credential --names credential-notification` (after other prerequisites exist). **Only notification seeding is replay-safe; this does not make the existing credential/full seed idempotent.** CLI startup still constructs MongoDB and storage dependencies even for notification-only selection. Run these commands only against an authorized development database with the consolidated schema in place; see [DATABASE.md](DATABASE.md) before migrating an existing installation.

Current fixtures produce 36 notifications from 41 credentials: 16 registered, 10 approved, 5 rejected, and 5 revoked; 10 pending credentials produce none. Existing fixtures give Issuer/Admin/SuperAdmin accounts no owned credentials, so empty personal feeds are expected. Their pending-review workload card remains a live query, not seeded reviewer notifications.

## Cross-Repo Integration

- **`../CredChain_Solidity/CredentialRegistry.sol`** — ERC-721 soulbound token, `credentialHashToStatus` mapping, CredentialStatus enum. Used by `chain.RegistryService` via abigen bindings.
- **`../CredChain_Python/`** — AI service called via HTTP for `/extract` and `/verify` endpoints. Response envelope matches Go's `{code, message, data, errors}`. Python owns error category `50`.
- **`../CredChain_React/`** — sole HTTP consumer. Mirrors `domain.Code*` verdict constants in `@shared/api/codes.ts`. Locale keys mirrored in `src/shared/i18n/`.

**Response code format:** 6-digit `AABBCC`. `40` credential category shared across all repos. Python AI errors propagate with original `50xxxx` code.

# CredChain Role System

> **Related docs:** For the credential system (issuance, verification, storage), see [CREDENTIAL.md](CREDENTIAL.md). For project-level commands and architecture, see [AGENTS.md](AGENTS.md).

## Role Definitions

Five roles form strict hierarchy (0=lowest, 4=highest):

| Rank | Go Constant | Go String | Solidity Name | Solidity Value | Postgres ENUM | Description |
|------|-------------|-----------|---------------|----------------|---------------|-------------|
| 0 | `RoleNone` | `"none"` | `None` | 0 | *(not persisted)* | Chain-only revocation target |
| 1 | `RoleHolder` | `"holder"` | `Holder` | 1 | `holder` | Basic credential holder |
| 2 | `RoleIssuer` | `"issuer"` | `Issuer` | 2 | `issuer` | Can issue/revoke credentials |
| 3 | `RoleAdmin` | `"admin"` | `Admin` | 3 | `admin` | User management (limited) |
| 4 | `RoleSuperAdmin` | `"super_admin"` | `SuperAdmin` | 4 | `super_admin` | Full system control |

**Sources:**
- Go: `domain/user.go:14-18` (constants), `domain/user.go:45-60` (`ToUint8`), `domain/user.go:63-78` (`RoleFromUint8`)
- Solidity: `CredentialAuthority.sol:19-25` (enum)
- Postgres: `infrastructure/database/migrations/000001_initial_schema.up.sql:1-6`

`RoleNone` is never persisted in the database — it exists only in Go domain code for on-chain revocation calls.

---

## Role Hierarchy

**Go** (`domain/user.go:22-37`):
```go
func (r Role) Rank() int {
    switch r {
    case RoleSuperAdmin: return 4
    case RoleAdmin:      return 3
    case RoleIssuer:     return 2
    case RoleHolder:     return 1
    case RoleNone:       return 0
    default:             return 0
    }
}
```

**Solidity** (`CredentialAuthority.sol:118-120`):
```solidity
function hasRoleOrAbove(address user, Role minimumRole) public view returns (bool) {
    return userToRole[user] >= minimumRole;
}
```

Both leverage Solidity enum natural ordering, with Go mirroring via `Rank()`.

---

## API Route Authorization

**Source:** `infrastructure/http/router.go:78-166`

Middleware chain: `ErrorLoggerMiddleware` → `I18nMiddleware` → `ApiRateLimitMiddleware` → `AuthMiddleware` → `RoleMiddleware` (if applicable)

| Route | Method | Auth | Min Role (on-chain) |
|-------|--------|------|---------------------|
| `/api/health` | GET | None | None |
| `/api/meta` | GET | None | — | Public metadata: issuing organization name, authority/registry contract addresses, chain ID, last block |
| `/api/auth/google` | POST | None | None |
| `/api/auth/refresh` | POST | None | None |
| `/api/auth/logout` | POST | Authenticated | Any |
| `/api/users/self` | GET | Authenticated | Any |
| `/api/users/self/email` | PUT | Authenticated | Any |
| `/api/users/self/transfer-super-admin` | POST | Authenticated | SuperAdmin |
| `/api/users/self/credentials` | GET | Authenticated | Any — lists own credentials (handler `SelfPaginate`) |
| `/api/users/self/credentials/:id` | GET | Authenticated | Any — fetch own credential; 404 if not owned (handler `SelfFind`) |
| `/api/credentials/:id/file` | GET | Authenticated | Any — download own credential or any if Issuer+ (policy-checked) |
| `/api/users` | GET | Authenticated | **Issuer+** (read-only) |
| `/api/users/:id` | GET | Authenticated | **Issuer+** (read-only) |
| `/api/users/batch` | POST | Authenticated | Admin+ |
| `/api/users/batch` | PUT | Authenticated | Admin+ |
| `/api/users/batch/role` | PUT | Authenticated | Admin+ |
| `/api/users/batch` | DELETE | Authenticated | Admin+ |
| `/api/users/batch/restore` | PUT | Authenticated | Admin+ |
| `/api/credentials` | GET | Authenticated | Issuer+ |
| `/api/credentials/:id` | GET | Authenticated | Issuer+ |
| `/api/credentials/:id/competencies` | PUT | Authenticated | Issuer+ — replace-set of resolved competency links |
| `/api/credentials/:id/metadata/suggestions` | GET | Authenticated | Issuer+ — top-5 pg_trgm suggestions per staged name |
| `/api/credentials/:id/metadata` | PUT | Authenticated | Issuer+ — resolve staged names (link existing or create new) |
| `/api/credentials/batch/issue` | POST | Authenticated | Issuer+ |
| `/api/credentials/batch/submit` | POST | Authenticated | Any — self-submit; unmatched names staged, not created |
| `/api/credentials/batch/approve` | POST | Authenticated | Issuer+ — blocked while metadata unresolved |
| `/api/credentials/batch/reject` | POST | Authenticated | Issuer+ |
| `/api/credentials/batch` | PUT | Authenticated | Issuer+ |
| `/api/credentials/batch/revoke` | POST | Authenticated | Issuer+ |
| `/api/credentials/batch/reextract` | POST | Authenticated | Issuer+ |
| `/api/credentials/verify` | POST | **None (public)** | None — used by external verifiers (HR, employers); returns verdict 400401-400413 including party-disabled (400410-400412) for trashed holder/issuer and expired (400413) |
| `/api/credential-types` | GET | Authenticated | Any — taxonomy lookup |
| `/api/credential-types` | POST | Authenticated | Issuer+ |
| `/api/credential-types/:id` | PUT | Authenticated | Issuer+ |
| `/api/credential-types/:id` | DELETE | Authenticated | Issuer+ |
| `/api/issuer-organizations` | GET | Authenticated | Any — taxonomy lookup |
| `/api/issuer-organizations` | POST | Authenticated | Issuer+ |
| `/api/issuer-organizations/:id` | PUT | Authenticated | Issuer+ |
| `/api/issuer-organizations/:id` | DELETE | Authenticated | Issuer+ |
| `/api/competencies` | GET | Authenticated | Any — taxonomy lookup |
| `/api/competencies` | POST | Authenticated | Issuer+ |
| `/api/competencies/:id` | PUT | Authenticated | Issuer+ |
| `/api/competencies/:id` | DELETE | Authenticated | Issuer+ |
| `/api/user-units` | GET | Authenticated | Any — organizational unit lookup |
| `/api/user-units` | POST | Authenticated | Admin+ |
| `/api/user-units/:id` | PUT | Authenticated | Admin+ |
| `/api/user-units/:id` | DELETE | Authenticated | Admin+ |
| `/api/overview` | GET | Authenticated | Any — role-conditional response (Holder: own data; Issuer+: system-wide) |
| `/api/overview/credential-recap` | GET | Authenticated | Issuer+ — current-holder grouped snapshot; middleware and service role guard |

**Role middlewares** (`infrastructure/http/middleware/auth.go:94-146`):
- `AdminRoleMiddleware` → checks `HasRoleOrAbove(ctx, wallet, RoleAdmin)` on-chain
- `IssuerRoleMiddleware` → checks `HasRoleOrAbove(ctx, wallet, RoleIssuer)` on-chain
- `SuperAdminRoleMiddleware` → checks `HasRoleOrAbove(ctx, wallet, RoleSuperAdmin)` on-chain

All check the **on-chain** role from CredentialAuthority contract, not the DB role.

---

## Policy Layer Rules

### User Policy (`feature/user/user_policy.go`)

#### Store (Admin+)

| Rule | Condition | Code |
|------|-----------|------|
| Cannot create SuperAdmin | RoleSuperAdmin in payload | `CodeUserStoreSuperAdminForbidden` (300245) |
| Admin cannot create Admin | Signer=Admin, target role=Admin | `CodeUserStoreAdminCreateAdminForbidden` (300246) |

Allowed combos: Admin creates Holder/Issuer; SuperAdmin creates anything except SuperAdmin.

#### UpdatePreFetch (put batch)

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | Below Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Cannot update self via batch | Signer ID in target list **AND signer is not SuperAdmin** | `CodeUserUpdateSelfForbidden` (300844) |

> **SuperAdmin self-edit carve-out:** SuperAdmin may include their own ID in the batch payload (Admin still blocked). This lets SuperAdmin manage their own profile fields. See email guard in UpdatePostFetch.

#### UpdatePostFetch

| Rule | Condition | Code |
|------|-----------|------|
| Cannot change own email via batch | Target ID == signer ID AND payload email non-nil | `CodeUserUpdateSelfEmailForbidden` (300847) |
| Cannot update trashed user | Target has non-nil DeletedAt | `CodeUserUpdateTrashedForbidden` (300846) |
| Cannot update SuperAdmin | Target role is SuperAdmin AND target is not self | `CodeUserUpdateSuperAdminForbidden` (300843) |
| Admin cannot update Admin | Signer=Admin, target role=Admin+ | `CodeUserUpdatePeerAdminForbidden` (300842) |
| Cannot assign SuperAdmin | Payload includes RoleSuperAdmin | `CodeUserRoleSuperAdminBatchForbidden` (300544) |
| Admin cannot promote to Admin | Signer=Admin, payload role=Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |

> **Self-email guard:** Even SuperAdmin (the only role allowed to self-edit via batch) cannot change their own email via batch — they must use `PUT /api/users/self/email` (which requires a fresh Google ID token). This prevents a self-update from setting an inaccessible email and locking the account out.

#### UpdateRolePreFetch

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | Below Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Cannot assign SuperAdmin | Target role is SuperAdmin | `CodeUserRoleSuperAdminBatchForbidden` (300544) |
| Cannot target self | Signer ID in target list | `CodeUserRoleSelfTargetForbidden` (300546) |

#### UpdateRolePostFetch

| Rule | Condition | Code |
|------|-----------|------|
| Target must exist | Not found in DB | generic not-found |
| Cannot update trashed target | Target has non-nil DeletedAt | `CodeUserRoleTrashedForbidden` (300547) |
| Cannot update to same role | Target role == payload role | `CodeUserRoleSameRoleUpdateForbidden` (300543) |
| Admin cannot update Admin target | Signer=Admin, target=Admin+ | `CodeUserRoleAdminUpdatePeerForbidden` (300541) |
| Admin cannot promote to Admin | Signer=Admin, payload=Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |

#### DeletePreFetch

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | Below Admin | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Cannot self-delete | Signer ID in target list | `CodeUserDeleteSelfTargetForbidden` (300743) |

#### DeletePostFetch

| Rule | Condition | Code |
|------|-----------|------|
| Admin cannot delete Admin/SuperAdmin | Signer=Admin, target=Admin+ | `CodeUserDeleteAdminForbidden` (300741) |

#### RestorePreFetch

| Rule | Condition | Code |
|------|-----------|------|
| Signer must be Admin+ | Below Admin | `CodeUserRestoreSignerAdminRequiredForbidden` (300941) |
| Cannot self-restore | Signer ID in target list | `CodeUserRestoreSelfTargetForbidden` (300942) |

#### RestorePostFetch

| Rule | Condition | Code |
|------|-----------|------|
| Cannot restore SuperAdmin | Target role is SuperAdmin | `CodeUserRestoreSuperAdminTargetForbidden` (300943) |
| Cannot restore non-trashed user | Target has nil DeletedAt | `CodeUserRestoreNotTrashedForbidden` (300944) |

#### TransferSuperAdminPreFetch

| Rule | Condition | Code |
|------|-----------|------|
| Cannot transfer to self | Target ID == signer ID | `CodeUserTransferSuperAdminSelfTargetForbidden` (300641) |

### Credential Policy Rules

Credential-specific policies are documented in [CREDENTIAL.md](CREDENTIAL.md). Role enforcement for credential operations (issue, revoke, re-extract) is at the route level via `IssuerRoleMiddleware` (on-chain check at `router.go:112-116`).

---

## Solidity Contract Access Control

| Function | Min Role | Contract | Location |
|----------|----------|----------|----------|
| `initialize()` | deployer only (`_requireDeployer`) | All | All contracts |
| `batchUpdateUserRoleWithSignature(params)` | Signer Admin+ | CredentialAuthority | `CredentialAuthority.sol:140-161` |
| `transferSuperAdminWithSignature(params)` | Signer SuperAdmin | CredentialAuthority | `CredentialAuthority.sol:174-192` |
| `batchIssueCredentialsWithSignature(params)` | Issuer+ | CredentialRegistry | `CredentialRegistry.sol:134-166` |
| `batchRevokeCredentialsWithSignature(params)` | Issuer+ | CredentialRegistry | `CredentialRegistry.sol:178-201` |

Solidity `_enforceUserRoleUpdateHierarchy` (`CredentialAuthority.sol:246-260`):
- Signer below Admin → revert `RoleBelowAdminError`
- Admin cannot modify Admin+ targets → revert `AdminUpdatePeerAdminRoleError`
- Admin cannot assign Admin+ roles → revert `RoleBelowAdminError`

Additional checks in `batchUpdateUserRoleWithSignature` (`CredentialAuthority.sol:154`):
- Any SuperAdmin assignment → revert `SuperAdminRoleNotUpdatableError`

Additional check in `_updateUserRole` (`CredentialAuthority.sol:222`):
- Same-role update → revert `SameRoleUpdateError`

---

## Per-Role Capability Matrix

```
Capability                      None   Holder  Issuer  Admin   SuperAdmin
──                             ────   ──────  ──────  ─────   ──────────
View overview dashboard            —      ✓       ✓       ✓       ✓
Google Login                    —      ✓       ✓       ✓       ✓
Refresh Token                   —      ✓       ✓       ✓       ✓
Logout                          —      ✓       ✓       ✓       ✓
View own profile                —      ✓       ✓       ✓       ✓
Update own email                —      ✓       ✓       ✓       ✓
List own credentials             —      ✓       ✓       ✓       ✓
Find own credential              —      ✓       ✓       ✓       ✓
Download own credential file     —      ✓       ✓       ✓       ✓
──
Transfer SuperAdmin             —      —       —       —       ✓
──
List users (read-only)          —      —       ✓       ✓       ✓
Find user by ID (read-only)     —      —       ✓       ✓       ✓
Create users (batch)            —      —       —       ✓¹      ✓²
Update users (batch)            —      —       —       ✓³      ✓⁶
Update user roles               —      —       —       ✓⁴      ✓²
Delete users                    —      —       —       ✓⁵      ✓
Restore users                   —      —       —       ✓       ✓
──
List credentials                —      —       ✓       ✓       ✓
Find credential                 —      —       ✓       ✓       ✓
Issue credentials               —      —       ✓       ✓       ✓
Revoke credentials              —      —       ✓       ✓       ✓
Re-extract credential           —      —       ✓       ✓       ✓
Submit own credentials          —      ✓       ✓       ✓       ✓
Approve/reject submissions      —      —       ✓       ✓       ✓
Resolve staged metadata         —      —       ✓       ✓       ✓
Read taxonomy lookups           —      ✓       ✓       ✓       ✓
Create/update taxonomy rows     —      —       ✓       ✓       ✓
Verify credential (public)      ✓      ✓       ✓       ✓       ✓
Read/manage own notifications   —      ✓       ✓       ✓       ✓
```

**Key:**
- ✓ = Allowed
- — = Denied

**Notes:**
1. Admin can create Holder, Issuer only; **cannot** create Admin or SuperAdmin
2. SuperAdmin cannot create/assign SuperAdmin via batch — only via Transfer SuperAdmin flow
3. Admin can update Holder, Issuer; **cannot** update Admin, SuperAdmin, self (via batch), or trashed users; **cannot** promote to Admin
4. Admin can update Holder/Issuer among Holder/Issuer roles; **cannot** target self, Admin peers, SuperAdmin; **cannot** assign Admin role
5. Admin can delete Holder, Issuer; **cannot** delete Admin, SuperAdmin, or self
6. **SuperAdmin can update self via batch (profile fields only — name, number, birth_date, gender, meta, role).** Email cannot be changed via batch even by SuperAdmin — must use `PUT /api/users/self/email` (Google reauth required) to prevent locking out with an inaccessible email.

---

## Denied Operations Quick Reference

| Operation | Who Can't Do It | Why / Error Code |
|-----------|----------------|------------------|
| Create SuperAdmin via API | Everyone (incl. SuperAdmin) | `CodeUserStoreSuperAdminForbidden` (300245) — CLI only |
| Assign SuperAdmin via batch | Everyone | `CodeUserRoleSuperAdminBatchForbidden` (300544) |
| Admin creates another Admin | Admin signer | `CodeUserStoreAdminCreateAdminForbidden` (300246) |
| Admin updates another Admin | Admin signer | `CodeUserUpdatePeerAdminForbidden` (300842) |
| Admin promotes to Admin | Admin signer | `CodeUserRoleSignerAdminRequiredForbidden` (300542) |
| Admin deletes Admin/SuperAdmin | Admin signer | `CodeUserDeleteAdminForbidden` (300741) |
| Update self via batch | Admin (SuperAdmin allowed) | `CodeUserUpdateSelfForbidden` (300844) |
| Change own email via batch | Anyone (incl. SuperAdmin) | `CodeUserUpdateSelfEmailForbidden` (300847) — use `/users/self/email` |
| Self-delete | Anyone | `CodeUserDeleteSelfTargetForbidden` (300743) |
| Self-target role update | Anyone | `CodeUserRoleSelfTargetForbidden` (300546) |
| Restore SuperAdmin | Admin+ | `CodeUserRestoreSuperAdminTargetForbidden` (300943) |
| Restore live (non-trashed) user | Admin+ | `CodeUserRestoreNotTrashedForbidden` (300944) |
| Restore self | Anyone | `CodeUserRestoreSelfTargetForbidden` (300942) |
| Transfer SuperAdmin to self | SuperAdmin | `CodeUserTransferSuperAdminSelfTargetForbidden` (300641) |
| Transfer SuperAdmin to trashed user | SuperAdmin | `CodeUserTransferSuperAdminTrashedForbidden` (300643) |
| Update trashed user | Admin+ | `CodeUserUpdateTrashedForbidden` (300846) |
| Update trashed user's role | Admin+ | `CodeUserRoleTrashedForbidden` (300547) |
| Update SuperAdmin's profile | Admin+ (SuperAdmin can self-edit; cannot change own email via batch) | `CodeUserUpdateSuperAdminForbidden` (300843) |
| Same-role update | Admin+ | `CodeUserRoleSameRoleUpdateForbidden` (300543) |
| Below-Admin accesses Admin+ endpoint | Holder, Issuer | `CodeUserRoleSignerAdminRequiredForbidden` (300542) / `CodeAuthForbidden` (200142) |
| Create a taxonomy row (credential type / issuer org / competency) | Holder | `CodeAuthForbidden` (200142) — submit a free-text name instead; a reviewer resolves it |
| Approve a submission with unresolved metadata | Everyone (incl. Issuer+) | `CodeCredentialApproveUnresolvedMetadata` (401546) — resolve staged names first |
| Resolve metadata on a non-pending credential | Everyone | `CodeCredentialMetadataResolveNotPending` (401541) |

**Taxonomy writes are Issuer+ by design, not Admin+.** The metadata resolver is an Issuer, so a reviewer who could create a taxonomy row but not rename a typo'd one would be stuck. `POST` was previously unguarded — any authenticated holder could create globally-visible taxonomy rows — so the write role was deliberately lowered from Admin+ to Issuer+ (still above Holder) rather than left open.

### Reference Read Visibility (Taxonomy Lookups)

`GET /api/credential-types`, `GET /api/issuer-organizations`, `GET /api/competencies` share identical endpoints across all authenticated roles without client-facing visibility query parameters:

- **Service-Owned Policy:** The HTTP handler binds `requestQuery.QueryRequest` into `domainQuery.Query`. Reference services (`credential_type_service.go`, `issuer_organization_service.go`, `competency_service.go`) inspect the authenticated user (`httpContext.GetUser(ctx)`).
- **Holder Scoping:** When caller rank is below Issuer (`u.Role.Rank() < RoleIssuer.Rank()`), the service sanitizes any client-supplied `holder_user_id` and injects `domainQuery.NewFilter("holder_user_id", domainQuery.OperatorEqual, u.Id)`.
- **Repository Execution:** Repositories remain pure and decoupled from HTTP context. In their filter processing loops, they apply the table-qualified visibility predicate before `Count(&total)` and pagination:
  ```sql
  ((<table_name>.activated_at IS NOT NULL AND <table_name>.rejected_at IS NULL AND <table_name>.active = ?) OR
   (<table_name>.submitter_user_id = ? AND (<table_name>.activated_at IS NULL OR <table_name>.rejected_at IS NOT NULL)))
  ```
- **Visibility Matrix:**
  - **Holder:** Receives approved active records (`activated_at IS NOT NULL AND rejected_at IS NULL AND active = true`) and own pending or rejected proposals (`submitter_user_id = user.id`). Approved inactive records (`active = false`) are excluded, even if originally submitted by that Holder. Other users' pending/rejected proposals remain hidden.
  - **Issuer, Admin, SuperAdmin:** Full visibility across all lifecycle and activation states (active, inactive, pending, rejected).
- **Historical Credentials:** Existing issued credentials retain historical inactive reference names through embedded relations; reference scoping restricts reference collection lookups only.

---

## Special Flows

### SuperAdmin Creation

**Only via CLI.** Source: `cmd/init_super_admin.go:322`

```bash
go run main.go init-super-admin --env .env             # host
docker compose exec golang ./server init-super-admin   # in-container (what local-up/prod-up run)
```

There is no dedicated Makefile target — `local-up` and `prod-up` invoke the in-container form as one of their steps.

Pre-conditions:
1. Wallet must have `SuperAdmin` role on-chain in CredentialAuthority contract
2. No existing (live) SuperAdmin in database (`userRepo.FindByRole(ctx, RoleSuperAdmin)`, filtering out soft-deleted)

### Transfer SuperAdmin

**Only endpoint that changes SuperAdmin role.** Source: `feature/user/user_service.go:420-471`

Endpoint: `POST /api/users/self/transfer-super-admin`

Flow:
1. SuperAdmin calls endpoint targeting another user
2. Policy blocks self-transfer (`CodeUserTransferSuperAdminSelfTargetForbidden`, step 5 in `TransferSuperAdminPreFetch`)
3. The trashed-target check (`CodeUserTransferSuperAdminTrashedForbidden`, 300643) happens in the **service layer** (`user_service.go:443-446`), not in `TransferSuperAdminPreFetch`. The policy method only checks that the caller is not transferring to themselves.
4. On-chain: `authorityService.TransferSuperAdmin()` — atomic swap (signer→Admin, target→SuperAdmin)
5. DB: both users' roles updated
6. Refresh tokens revoked for both users

### On-Chain Role Sync

**`syncBlockchainRoles`** (`feature/user/user_service.go:153-160`) called by all mutation paths:
- `Store` → `storeUsersAndSyncBlockchainRoles`
- `Update` (when role changes) → inside single `s.uow.Execute`
- `UpdateRole` → inside single `s.uow.Execute`
- `Delete` → inside single `s.uow.Execute` (sends `RoleNone`, the chain-only revocation target)
- `Restore` → inside single `s.uow.Execute` (re-syncs preserved DB role to chain)

All five flows call `syncBlockchainRoles` directly inside a single `s.uow.Execute` closure, matching the `Update` pattern. The helper delegates to `authorityService.UpdateUserRole`, which converts domain users into on-chain `UserRoleUpdation` structs, signs with the auth user's wallet, calls `batchUpdateUserRoleWithSignature` on CredentialAuthority, waits for receipt via `bind.WaitMined`, and verifies receipt status. Chain failure rolls back the DB transaction.

### RoleNone (On-Chain Revocation Only)

`RoleNone` (`domain/user.go:14`, value `"none"` → Solidity `0`) is **never persisted** in Postgres (the `role` ENUM excludes `none`). It exists solely as the Solidity revocation target used by `Delete` to revoke the user's on-chain role.

### Restore (PUT /api/users/batch/restore)

**Only endpoint that un-does a soft-delete.** Source: `feature/user/user_service.go`

Endpoint: `PUT /api/users/batch/restore`

Flow:
1. Admin submits a batch of trashed user IDs
2. Policy blocks: below-Admin signers (300941), self-target (300942), SuperAdmin targets (300943), live non-trashed targets (300944)
3. Single DB transaction: clears `deleted_at` via `uow.User().Restore()` repo method
4. On-chain: re-syncs the preserved DB role via `syncBlockchainRoles`
5. If chain sync fails, DB restore rolls back (single UoW)

Constraints:
- Admin can restore Admin peers — restore is an undo, not role escalation
- SuperAdmin target restore is unconditionally blocked (300943)
- Non-trashed target restore is strictly rejected (300944 — Option B)

### Credential Verify: Party-Disabled and Expired Verdicts

When `/api/credentials/verify` matches a credential whose holder or issuer has been soft-deleted, the verdict is overridden to indicate manual review is needed. Three codes: `holder_disabled` (400410), `issuer_disabled` (400411), `party_disabled` (400412 — both deleted). A fourth override, `expired` (400413), fires when the credential's `expires_at` has passed (`verifyApplyExpiry`, `credential_service.go:1860-1866`).

All four only override `Authentic` (400401); non-authentic verification outcomes persist unchanged. Stored lifecycle is separate: `Credential.Status()` uses revoked, rejected, expired, approved, then pending precedence. Expiry therefore appears outside verification too, including an unactivated submission whose expiry has passed.

Exact-hash and cached-authentic paths apply expiry first, so an expired credential with a disabled holder returns400413. Go's compatibility path for an AI `authentic` applies party checks first. Current Python scoring never emits `authentic`; AI similarity results must not be interpreted as lifecycle or authenticity verdicts.

Activated exact-hash credentials receive live chain correspondence checks before cache lookup and exact outcomes are not cached. Chain read failures return inconclusive HTTP503/code400447, not integrity mismatch. Other uploads can use Mongo's24h cache without a chain read. Only cached `Authentic` receives fresh DB revocation, expiry and party overrides; other cached codes retain their verdict while the matched record is fetched fresh. Do not assume all cached outcomes receive every live override.

The ordinary eleven-field public verification summary includes nullable `issuer_organization_name` and `credential_type_name` from the linked organization and type for exact, cache and text-comparison results, equally for anonymous and authenticated visitors. Only those names are newly disclosed; candidate Holder information, expanded details and stored files remain protected. Candidate fields describe the compared credential rather than proving the upload's issuance.

Public verification emits `exact_match_confirmed` separately from the verdict. Only successful live byte-hash/registry correspondence checks set it true; cache results and AI similarity (even100%) cannot unlock expanded public details. The same upload response may then include the nineteen-field `public_details` allowlist documented in CREDENTIAL.md, equally for anonymous and authenticated visitors, including revoked/expired/disabled exact matches with their warnings. This explicitly permits the displayed Holder account fields (including email, wallet, role, account dates and resolved unit name), name-only competencies, credential Additional Information and the existing technical/lifecycle footer. It does not expose raw user/unit IDs, user metadata, secrets, staff identities, storage URI, extraction or private workflow reasons. Holder account/reference information is current database data, not immutable issuance-time identity or independent issuer confirmation.

No public GET-by-ID, profile directory, shareable details URL or stored-file download access is introduced. Public details preview only the submitted upload. Own Holders use `/users/self/credentials/:id`; Issuer/Admin/SuperAdmin use the existing protected credential endpoint to inspect applicable lifecycle parties and stored files. An unrelated Holder retains only confirmed public details, even if authenticated. Authorized text-comparison candidates can open the private compared credential; anonymous or unrelated viewers cannot. AI100% changes neither authorization nor evidence. Private detail/file query caches are scoped by account and role; session changes close open verification details.

**Seed coverage:** the credential seeder covers `holder_disabled` (an approved credential held by soft-deleted Anna Sorokin) and `expired`. It deliberately does **not** seed a soft-deleted *issuer*, so `issuer_disabled` and `party_disabled` are untested by seed data: `seed-chain` signs each mint batch with the wallet named by `issuer_user_id`, and a soft-deleted user is deregistered on-chain (`RoleNone`), so `batchIssueCredentialsWithSignature` would revert `RoleBelowIssuerError`. A disabled *holder* is chain-safe because `_issueCredential` only requires a non-zero holder address.

---

## Architecture: Dual Role Sources

Roles are stored in two places, kept in sync:

| Check Location | Role Source | Used By |
|---------------|-------------|---------|
| API middleware (`middleware/auth.go`) | **On-chain** (Authority contract `userToRole`) | Route-level access control |
| Policy layer (`feature/*/policy.go`) | **Database** (Postgres `role` column) | Business rule enforcement |
| Credential policy (`credential_policy.go`) | **Database** | Store-level checks |

The middleware checks the on-chain role (via `authorityService.HasRoleOrAbove`), while policy layers check the DB-stored role rank. The `syncBlockchainRoles` utility keeps both sources in sync on every mutation. If diverged (e.g., due to a failed chain tx that DB rolled back), the middleware enforces the stricter (on-chain) role.

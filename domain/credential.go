package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// ExtractState is the lifecycle of the asynchronous Python /extract job
// attached to a credential. On-chain issuance is synchronous (Go computes
// keccak256 of raw file bytes immediately), but extraction (text, ids, embedding —
// needed by /api/credentials/verify) requires a slow Python OCR+EmbeddingGemma
// round-trip and is computed asynchronously via the River worker. Results are
// stored in MongoDB (credential_extractions collection), not Postgres.
//
// ExtractState is derived, not a column (see Credential.ExtractState) — it is
// computed from ExtractEnqueuedAt / ExtractedAt / ExtractFailedAt.
type ExtractState string

const (
	ExtractStatePending   ExtractState = "pending"
	ExtractStateSucceeded ExtractState = "succeeded"
	ExtractStateFailed    ExtractState = "failed"
	// ExtractStateUnextracted marks rows for which no extraction has been
	// performed (submitted rows, and rejected rows that never got approved).
	// It is review-agnostic: the job is enqueued at approval, when the row
	// flips to pending.
	ExtractStateUnextracted ExtractState = "unextracted"
)

// CredentialStatus is the workflow lifecycle of a credential, derived purely
// from timestamps (see Credential.Status). It is NOT a database column.
type CredentialStatus string

const (
	CredentialStatusPending  CredentialStatus = "pending"
	CredentialStatusApproved CredentialStatus = "approved"
	CredentialStatusRejected CredentialStatus = "rejected"
	CredentialStatusRevoked  CredentialStatus = "revoked"
)

// SubmittedCompetency is one entry of credentials.submitted_competencies.
// Name is what the submitter typed; ResolvedID is stamped by a reviewer once
// the name is linked to (or created as) a competencies row.
type SubmittedCompetency struct {
	Name       string  `json:"name"`
	ResolvedID *string `json:"resolved_id"`
}

// SubmittedCompetencies is the JSONB column type. Serialization is handled by
// the GORM model layer (serializer:json, same as Meta) — this type carries no
// Scan/Value of its own.
type SubmittedCompetencies []SubmittedCompetency

// Credential represents a row in the credentials table.
//
// FileHash is the keccak256 of the raw file bytes, used by the on-chain
// CredentialRegistry contract to derive the ERC-721 token ID:
//
//	id = uint256(keccak256(abi.encodePacked(hash)))
//
// FileURI points at the persisted upload (e.g. local:///uploads/...).
// Embeddings is populated asynchronously by the extract worker.
//
// Holder, Issuer, and Revoker are optional preloaded User references.
// They are populated by the repository when the caller's query includes
// "holder", "issuer", or "revoker" in its Includes slice.
type Credential struct {
	ID              string  `json:"id"`
	HolderUserID    string  `json:"holder_user_id"`
	SubmitterUserID string  `json:"submitter_user_id"`
	IssuerUserID    *string `json:"issuer_user_id"`
	// SubmittedIssuerOrganizationName / SubmittedTypeName hold the free text a
	// submitter typed when no taxonomy row matched. The paired ID stays nil
	// until a reviewer resolves it; an approved credential always has both IDs.
	SubmittedIssuerOrganizationName *string        `json:"submitted_issuer_organization_name"`
	IssuerOrganizationID            *string        `json:"issuer_organization_id"`
	SubmittedTypeName               *string        `json:"submitted_type_name"`
	TypeID                          *string        `json:"type_id"`
	Number                          *string        `json:"number"`
	RevokerUserID                   *string        `json:"revoker_user_id"`
	Name                            string         `json:"name"`
	Meta                            map[string]any `json:"meta"`
	// SubmittedCompetencies stages competency names that had no row at submit
	// time. ResolvedID is stamped in place by the reviewer; the name survives
	// for audit even after resolution, and on rejected rows stays unresolved.
	SubmittedCompetencies SubmittedCompetencies `json:"submitted_competencies"`
	TokenID               *string               `json:"token_id"`
	FileHash              string                `json:"file_hash"`
	FileURI               *string               `json:"file_uri"`
	ExtractEnqueuedAt     *time.Time            `json:"extract_enqueued_at"`
	ExtractedAt           *time.Time            `json:"extracted_at"`
	ExtractFailedAt       *time.Time            `json:"extract_failed_at"`
	ExtractError          *string               `json:"extract_error"`
	IssuedAt              time.Time             `json:"issued_at"`
	RevokedAt             *time.Time            `json:"revoked_at"`
	ExpiresAt             *time.Time            `json:"expires_at"`
	ApprovedAt            *time.Time            `json:"approved_at"`
	RejecterUserID        *string               `json:"rejecter_user_id"`
	RejectedAt            *time.Time            `json:"rejected_at"`
	RejectionReason       *string               `json:"rejection_reason"`
	CreatedAt             time.Time             `json:"created_at"`
	UpdatedAt             *time.Time            `json:"updated_at"`

	// Preloaded relations (populated by repository when query.Includes contains
	// "holder", "issuer", "revoker", or "rejecter"). json:"-" so they never leak through
	// the API envelope; the response DTO maps them explicitly.
	Holder   *User `gorm:"-" json:"-"`
	Issuer   *User `gorm:"-" json:"-"`
	Revoker  *User `gorm:"-" json:"-"`
	Rejecter *User `gorm:"-" json:"-"`

	// Competencies are the resolved competency rows linked through the
	// competency_credential join table. Populated by the repository when the
	// query's Includes contains "competencies".
	Competencies []Competency `gorm:"-" json:"-"`

	// Type / IssuerOrganization are populated by the repository when the
	// query's Includes contains "type" / "issuer_organization".
	Type               *CredentialType               `gorm:"-" json:"-"`
	IssuerOrganization *CredentialIssuerOrganization `gorm:"-" json:"-"`
}

// Status derives the workflow lifecycle from timestamps only:
//
//	revoked  when RevokedAt is set
//	rejected when RejectedAt is set (and not revoked)
//	approved when ApprovedAt is set (and neither revoked nor rejected)
//	pending  otherwise
//
// Expiry is deliberately NOT part of this derivation — expiry is evaluated
// only on the verification path (Task C1).
func (c *Credential) Status() CredentialStatus {
	if c.RevokedAt != nil {
		return CredentialStatusRevoked
	}
	if c.RejectedAt != nil {
		return CredentialStatusRejected
	}
	if c.ApprovedAt != nil {
		return CredentialStatusApproved
	}
	return CredentialStatusPending
}

// ExtractState derives the extraction lifecycle from timestamps only.
// Failure wins over success so a re-extract that fails again is not masked
// by a stale ExtractedAt.
func (c *Credential) ExtractState() ExtractState {
	if c.ExtractFailedAt != nil {
		return ExtractStateFailed
	}
	if c.ExtractedAt != nil {
		return ExtractStateSucceeded
	}
	if c.ExtractEnqueuedAt != nil {
		return ExtractStatePending
	}
	return ExtractStateUnextracted
}

// UnresolvedMetadata returns the metadata kinds still awaiting reviewer
// resolution, in a stable order: "type", "issuer_organization", "competency".
// An empty result means the credential is safe to approve. Competencies are
// unresolved as a group — one unresolved entry blocks the whole set.
func (c *Credential) UnresolvedMetadata() []string {
	var out []string
	if c.TypeID == nil {
		out = append(out, "type")
	}
	if c.IssuerOrganizationID == nil {
		out = append(out, "issuer_organization")
	}
	for _, sc := range c.SubmittedCompetencies {
		if sc.ResolvedID == nil {
			out = append(out, "competency")
			break
		}
	}
	return out
}

// InactiveMetadata returns the metadata kinds whose resolved taxonomy row is no
// longer active, in a stable order: "type", "issuer_organization", "competency".
// A row may be retired between resolution and approval; approving would mint a
// permanent on-chain credential against a deactivated row. Relations must be
// preloaded (Includes: type, issuer_organization, competencies) — a nil relation
// is reported as resolved-and-fine, since UnresolvedMetadata covers nil FKs.
func (c *Credential) InactiveMetadata() []string {
	var out []string
	if c.Type != nil && !c.Type.Active {
		out = append(out, "type")
	}
	if c.IssuerOrganization != nil && !c.IssuerOrganization.Active {
		out = append(out, "issuer_organization")
	}
	for _, comp := range c.Competencies {
		if !comp.Active {
			out = append(out, "competency")
			break
		}
	}
	return out
}

// CredentialRepository defines the database contract for the credential domain.
type CredentialRepository interface {
	// Pagination-aware retrieval with filters, sorts, search, and includes.
	// When query.Includes contains "holder", "issuer", or "revoker" the
	// corresponding GORM Preload runs (single batch IN-clause; no N+1).
	Get(ctx context.Context, query *domainQuery.Query) ([]Credential, int, error)

	// Find retrieves a single credential by ID. query.Includes may request
	// Preload of holder/issuer/revoker user relations.
	Find(ctx context.Context, id string, query *domainQuery.Query) (*Credential, error)

	// FindByIds retrieves credentials by ID list (batch lookup). When query is
	// non-nil it may carry Includes for preloading holder/issuer/revoker relations.
	FindByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]Credential, error)

	// FindVerifiableById retrieves a single approved credential by ID.
	// Verification path only: rows with approved_at IS NULL are invisible.
	FindVerifiableById(ctx context.Context, id string, query *domainQuery.Query) (*Credential, error)

	// FindVerifiableByIds retrieves approved credentials by ID list.
	// Verification path only: rows with approved_at IS NULL are invisible.
	FindVerifiableByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]Credential, error)

	// FindByHolderId retrieves all credentials owned by a given holder. When
	// query is non-nil it may carry Includes for preloading relations.
	FindByHolderId(ctx context.Context, holderID string, query *domainQuery.Query) ([]Credential, error)

	// FindByFileHashes retrieves approved credentials whose file_hash matches
	// any of the given hashes. approved_at IS NOT NULL is enforced; sole
	// consumer is the public verify path. When query is non-nil it may carry
	// Includes for preloading relations.
	FindByFileHashes(ctx context.Context, hashes []string, query *domainQuery.Query) ([]Credential, error)

	// Store batch-inserts credentials. Generates ULIDs for any missing IDs.
	Store(ctx context.Context, credentials ...Credential) ([]Credential, error)

	// Update partially updates one or more credentials using a single batched
	// UPDATE with per-column CASE expressions (mirrors user repository pattern).
	// Only non-nil / non-zero fields are touched; unspecified columns fall
	// through to ELSE column (preserving existing value).
	Update(ctx context.Context, credentials ...Credential) ([]Credential, error)

	// ClearExtractOutcome stamps a fresh extract attempt and clears the prior
	// result. Separate from Update because Update's nil-means-skip semantics
	// cannot write NULL (needed to clear extracted_at / extract_failed_at /
	// extract_error on re-extract).
	ClearExtractOutcome(ctx context.Context, enqueuedAt time.Time, ids ...string) error

	// CountByTypeIds counts credentials referencing any of the given
	// credential_type ids. Pure read primitive for the step-3 deletion guard.
	CountByTypeIds(ctx context.Context, typeIds ...string) (int64, error)

	// CountByIssuerOrganizationIds counts credentials referencing any of the
	// given issuer_organization ids. Pure read primitive for the step-3
	// deletion guard.
	CountByIssuerOrganizationIds(ctx context.Context, organizationIds ...string) (int64, error)

	// CountActiveByFileHashes counts credentials whose file_hash matches any
	// of the given hashes AND which are neither revoked nor rejected (mirrors
	// the partial unique index). Pure read primitive for submit-time duplicate
	// detection — pending rows are invisible to the approved-gated finders.
	CountActiveByFileHashes(ctx context.Context, hashes ...string) (int64, error)
}

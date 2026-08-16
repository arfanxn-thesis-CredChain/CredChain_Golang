package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// ExtractStatus is the lifecycle of the asynchronous Python /extract job
// attached to a credential. On-chain issuance is synchronous (Go computes
// keccak256 of raw file bytes immediately), but extraction (text, ids, embedding —
// needed by /api/credentials/verify) requires a slow Python OCR+EmbeddingGemma
// round-trip and is computed asynchronously via the River worker. Results are
// stored in MongoDB (credential_extractions collection), not Postgres.
type ExtractStatus string

const (
	ExtractStatusPending   ExtractStatus = "pending"
	ExtractStatusSucceeded ExtractStatus = "succeeded"
	ExtractStatusFailed    ExtractStatus = "failed"
)

// CredentialStatus is the workflow status of a credential, derived purely
// from timestamps (see Credential.Status). It is NOT a database column.
type CredentialStatus string

const (
	CredentialStatusPending  CredentialStatus = "pending"
	CredentialStatusApproved CredentialStatus = "approved"
	CredentialStatusRejected CredentialStatus = "rejected"
	CredentialStatusRevoked  CredentialStatus = "revoked"
)

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
	ID                   string         `db:"id"              json:"id"`
	HolderUserID         string         `db:"holder_user_id"  json:"holder_user_id"`
	SubmitterUserID      string         `db:"submitter_user_id"       json:"submitter_user_id"`
	IssuerUserID         string         `db:"issuer_user_id"  json:"issuer_user_id"`
	IssuerOrganizationID string         `db:"issuer_organization_id" json:"issuer_organization_id"`
	TypeID               string         `db:"type_id"                json:"type_id"`
	Number               *string        `db:"number"                 json:"number"`
	RevokerUserID        *string        `db:"revoker_user_id" json:"revoker_user_id"`
	Name                 string         `db:"name"            json:"name"`
	Meta                 map[string]any `db:"meta"            json:"meta"`
	TokenID              *string        `db:"token_id"        json:"token_id"`
	FileHash             string         `db:"file_hash"       json:"file_hash"`
	FileURI              *string        `db:"file_uri"        json:"file_uri"`
	ExtractStatus        ExtractStatus  `db:"extract_status"  json:"extract_status"`
	ExtractError         *string        `db:"extract_error"   json:"extract_error"`
	ExtractedAt          *time.Time     `db:"extracted_at"    json:"extracted_at"`
	IssuedAt             time.Time      `db:"issued_at"       json:"issued_at"`
	RevokedAt            *time.Time     `db:"revoked_at"      json:"revoked_at"`
	ExpiresAt            *time.Time     `db:"expires_at"        json:"expires_at"`
	ApproverUserID       *string        `db:"approver_user_id"  json:"approver_user_id"`
	ApprovedAt           *time.Time     `db:"approved_at"       json:"approved_at"`
	RejecterUserID       *string        `db:"rejecter_user_id"  json:"rejecter_user_id"`
	RejectedAt           *time.Time     `db:"rejected_at"       json:"rejected_at"`
	RejectionReason      *string        `db:"rejection_reason"  json:"rejection_reason"`
	CreatedAt            time.Time      `db:"created_at"        json:"created_at"`
	UpdatedAt            *time.Time     `db:"updated_at"        json:"updated_at"`

	// Preloaded relations (populated by repository when query.Includes contains
	// "holder", "issuer", or "revoker"). json:"-" so they never leak through
	// the API envelope; the response DTO maps them explicitly.
	Holder  *User `gorm:"-" json:"-"`
	Issuer  *User `gorm:"-" json:"-"`
	Revoker *User `gorm:"-" json:"-"`
}

// Status derives the workflow status from timestamps only:
//
//	revoked  when RevokedAt is set
//	rejected when RejectedAt is set (and not revoked)
//	approved when ApprovedAt is set (and neither revoked nor rejected)
//	pending  otherwise
//
// Expiry is deliberately NOT part of this derivation — expiry is evaluated
// only on the verification path in a later step.
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

	// CountByTypeIds counts credentials referencing any of the given
	// credential_type ids. Pure read primitive for the step-3 deletion guard.
	CountByTypeIds(ctx context.Context, typeIds ...string) (int64, error)

	// CountByIssuerOrganizationIds counts credentials referencing any of the
	// given issuer_organization ids. Pure read primitive for the step-3
	// deletion guard.
	CountByIssuerOrganizationIds(ctx context.Context, organizationIds ...string) (int64, error)
}

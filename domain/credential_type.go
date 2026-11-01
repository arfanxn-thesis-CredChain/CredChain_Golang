package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// CredentialType represents a row in the credential_types table.
// Active flags whether the type may still be used for new credentials.
type CredentialType struct {
	Id              string     `json:"id"`
	SubmitterUserId *string    `json:"submitter_user_id"`
	ApproverUserId  *string    `json:"approver_user_id"`
	RejecterUserId  *string    `json:"rejecter_user_id"`
	Name            string     `json:"name"`
	Active          bool       `json:"active"`
	RejectionReason *string    `json:"rejection_reason"`
	ActivatedAt     *time.Time `json:"activated_at"`
	RejectedAt      *time.Time `json:"rejected_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`

	Submitter *User `gorm:"-" json:"-"`
	Approver  *User `gorm:"-" json:"-"`
	Rejecter  *User `gorm:"-" json:"-"`
}

func (t *CredentialType) Status() ReferenceStatus {
	if t.RejectedAt != nil {
		return ReferenceStatusRejected
	}
	if t.ActivatedAt != nil {
		return ReferenceStatusApproved
	}
	return ReferenceStatusPending
}

// CredentialTypeRepository defines the database contract for credential_types.
type CredentialTypeRepository interface {
	Store(ctx context.Context, types ...CredentialType) ([]CredentialType, error)
	Find(ctx context.Context, id string) (*CredentialType, error)
	FindByIds(ctx context.Context, ids ...string) ([]CredentialType, error)
	// FindByNames resolves names to rows in one query, matching
	// case-insensitively on the trimmed name (same semantics as the
	// uq_credential_types_lower_name index). Names with no row are simply
	// absent from the result — the caller decides whether that is an error.
	FindByNames(ctx context.Context, names ...string) ([]CredentialType, error)

	// SuggestByName returns up to limit rows ordered by descending trigram
	// similarity to name, backed by idx_credential_types_name_trgm. It powers
	// the reviewer's "did you mean" list when resolving a submitted free-text
	// name.
	SuggestByName(ctx context.Context, name string, limit int) ([]CredentialType, error)
	// Get lists credential types, returning the page and the total number of
	// rows matching the query before pagination.
	Get(ctx context.Context, query *domainQuery.Query) ([]CredentialType, int, error)
	Update(ctx context.Context, types ...CredentialType) ([]CredentialType, error)

	// Destroy hard-deletes rows by ID (batch). Rows referenced by credentials
	// are protected by the FK constraint — reference pre-checks belong to the
	// service layer (step 3).
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

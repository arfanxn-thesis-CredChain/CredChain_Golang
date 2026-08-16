package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// CredentialType represents a row in the credential_types table.
// Active flags whether the type may still be used for new credentials.
type CredentialType struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CredentialTypeRepository defines the database contract for credential_types.
type CredentialTypeRepository interface {
	Store(ctx context.Context, types ...CredentialType) ([]CredentialType, error)
	Find(ctx context.Context, id string) (*CredentialType, error)
	FindByIds(ctx context.Context, ids ...string) ([]CredentialType, error)
	Get(ctx context.Context, query *domainQuery.Query) ([]CredentialType, error)
	Update(ctx context.Context, types ...CredentialType) ([]CredentialType, error)

	// Delete hard-deletes rows by ID (batch). Rows referenced by credentials
	// are protected by the FK constraint — reference pre-checks belong to the
	// service layer (step 3).
	Delete(ctx context.Context, ids ...string) (int64, error)
}

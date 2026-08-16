package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// CredentialIssuerOrganization represents a row in the
// credential_issuer_organizations table.
type CredentialIssuerOrganization struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CredentialIssuerOrganizationRepository defines the database contract for
// credential_issuer_organizations.
type CredentialIssuerOrganizationRepository interface {
	Store(ctx context.Context, orgs ...CredentialIssuerOrganization) ([]CredentialIssuerOrganization, error)
	Find(ctx context.Context, id string) (*CredentialIssuerOrganization, error)
	FindByIds(ctx context.Context, ids ...string) ([]CredentialIssuerOrganization, error)
	Get(ctx context.Context, query *domainQuery.Query) ([]CredentialIssuerOrganization, error)
	Update(ctx context.Context, orgs ...CredentialIssuerOrganization) ([]CredentialIssuerOrganization, error)

	// Delete hard-deletes rows by ID (batch). Rows referenced by credentials
	// are protected by the FK constraint — reference pre-checks belong to the
	// service layer (step 3).
	Delete(ctx context.Context, ids ...string) (int64, error)
}

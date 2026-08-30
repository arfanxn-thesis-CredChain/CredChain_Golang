package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// CredentialIssuerOrganization represents a row in the
// credential_issuer_organizations table.
// Active flags whether the organization may still be used for new credentials.
type CredentialIssuerOrganization struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CredentialIssuerOrganizationRepository defines the database contract for
// credential_issuer_organizations.
type CredentialIssuerOrganizationRepository interface {
	Store(ctx context.Context, orgs ...CredentialIssuerOrganization) ([]CredentialIssuerOrganization, error)
	Find(ctx context.Context, id string) (*CredentialIssuerOrganization, error)
	FindByIds(ctx context.Context, ids ...string) ([]CredentialIssuerOrganization, error)
	// FindByNames resolves names to rows in one query, matching
	// case-insensitively on the trimmed name (same semantics as the
	// uq_credential_issuer_organizations_lower_name index). Names with no row
	// are simply absent from the result — the caller decides whether that is
	// an error.
	FindByNames(ctx context.Context, names ...string) ([]CredentialIssuerOrganization, error)

	// SuggestByName returns up to limit rows ordered by descending trigram
	// similarity to name, backed by idx_credential_issuer_organizations_name_trgm.
	// It powers the reviewer's "did you mean" list when resolving a submitted
	// free-text name.
	SuggestByName(ctx context.Context, name string, limit int) ([]CredentialIssuerOrganization, error)
	// Get lists issuer organizations, returning the page and the total number of
	// rows matching the query before pagination.
	Get(ctx context.Context, query *domainQuery.Query) ([]CredentialIssuerOrganization, int, error)
	Update(ctx context.Context, orgs ...CredentialIssuerOrganization) ([]CredentialIssuerOrganization, error)

	// Destroy hard-deletes rows by ID (batch). Rows referenced by credentials
	// are protected by the FK constraint — reference pre-checks belong to the
	// service layer (step 3).
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

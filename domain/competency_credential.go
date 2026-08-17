package domain

import "context"

// CompetencyCredential is the composite-PK join row between competencies and
// credentials (table competency_credential).
type CompetencyCredential struct {
	CompetencyId string `json:"competency_id"`
	CredentialId string `json:"credential_id"`
}

// CompetencyCredentialRepository defines the database contract for the
// competency_credential join table.
type CompetencyCredentialRepository interface {
	// Store batch-inserts join rows (composite PK: competency_id, credential_id).
	Store(ctx context.Context, links ...CompetencyCredential) ([]CompetencyCredential, error)

	// Destroy removes join rows (composite PK pairs).
	Destroy(ctx context.Context, links ...CompetencyCredential) (int64, error)

	// DestroyByCredentialId removes all join rows for one credential.
	DestroyByCredentialId(ctx context.Context, credentialId string) (int64, error)

	// FindByCredentialId lists all join rows for one credential.
	FindByCredentialId(ctx context.Context, credentialId string) ([]CompetencyCredential, error)

	// FindByCompetencyId lists all join rows for one competency.
	FindByCompetencyId(ctx context.Context, competencyId string) ([]CompetencyCredential, error)

	// CountByCompetencyIds counts join rows referencing any of the given
	// competency ids. Pure read primitive for the step-3 deletion guard.
	CountByCompetencyIds(ctx context.Context, competencyIds ...string) (int64, error)
}

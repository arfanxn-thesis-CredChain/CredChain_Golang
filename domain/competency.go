package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// Competency represents a row in the competencies table.
type Competency struct {
	Id        string     `json:"id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// CompetencyRepository defines the database contract for competencies.
type CompetencyRepository interface {
	Store(ctx context.Context, competencies ...Competency) ([]Competency, error)
	Find(ctx context.Context, id string) (*Competency, error)
	FindByIds(ctx context.Context, ids ...string) ([]Competency, error)
	Get(ctx context.Context, query *domainQuery.Query) ([]Competency, error)
	Update(ctx context.Context, competencies ...Competency) ([]Competency, error)

	// Delete hard-deletes rows by ID (batch). Rows referenced by credentials
	// are protected by the FK constraint — reference pre-checks belong to the
	// service layer (step 3).
	Delete(ctx context.Context, ids ...string) (int64, error)
}

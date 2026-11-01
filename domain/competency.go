package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// Competency represents a row in the competencies table.
// Active flags whether the competency may still be used for new credentials.
type Competency struct {
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

func (c *Competency) Status() ReferenceStatus {
	if c.RejectedAt != nil {
		return ReferenceStatusRejected
	}
	if c.ActivatedAt != nil {
		return ReferenceStatusApproved
	}
	return ReferenceStatusPending
}

// CompetencyRepository defines the database contract for competencies.
type CompetencyRepository interface {
	Store(ctx context.Context, competencies ...Competency) ([]Competency, error)
	Find(ctx context.Context, id string) (*Competency, error)
	FindByIds(ctx context.Context, ids ...string) ([]Competency, error)
	// FindByNames resolves names to rows in one query, matching
	// case-insensitively on the trimmed name (same semantics as the
	// uq_competencies_lower_name index). Names with no row are simply absent
	// from the result — the caller decides whether that is an error.
	FindByNames(ctx context.Context, names ...string) ([]Competency, error)

	// SuggestByName returns up to limit rows ordered by descending trigram
	// similarity to name, backed by idx_competencies_name_trgm. It powers the
	// reviewer's "did you mean" list when resolving a submitted free-text name.
	SuggestByName(ctx context.Context, name string, limit int) ([]Competency, error)
	// Get lists competencies, returning the page and the total number of rows
	// matching the query before pagination.
	Get(ctx context.Context, query *domainQuery.Query) ([]Competency, int, error)
	Update(ctx context.Context, competencies ...Competency) ([]Competency, error)

	// Destroy hard-deletes rows by ID (batch). Rows referenced by credentials
	// are protected by the FK constraint — reference pre-checks belong to the
	// service layer (step 3).
	Destroy(ctx context.Context, ids ...string) (int64, error)
}

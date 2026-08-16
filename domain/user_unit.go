package domain

import (
	"context"
	"time"

	domainQuery "CredChain_Golang/domain/query"
)

// UserUnit represents a row in the user_units table — an organizational unit
// (faculty / study program / department) in a self-referential tree.
type UserUnit struct {
	Id        string     `json:"id"`
	ParentId  *string    `json:"parent_id"`
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// UserUnitRepository defines the database contract for the user_units table.
type UserUnitRepository interface {
	// Store batch-inserts units. Generates ULIDs for any missing IDs.
	Store(ctx context.Context, units ...UserUnit) ([]UserUnit, error)

	// Find retrieves a single unit by ID.
	Find(ctx context.Context, id string) (*UserUnit, error)

	// FindByIds retrieves units by ID list (batch lookup).
	FindByIds(ctx context.Context, ids ...string) ([]UserUnit, error)

	// Get lists units. Accepts a nil *Query. Default sort: name ASC.
	Get(ctx context.Context, query *domainQuery.Query) ([]UserUnit, error)

	// Update partially updates units using a single batched CASE statement.
	// Only non-nil / non-zero fields are touched.
	Update(ctx context.Context, units ...UserUnit) ([]UserUnit, error)

	// FindWithDescendants returns the unit with the given id plus all of its
	// descendants in the tree (single WITH RECURSIVE query; includes self).
	FindWithDescendants(ctx context.Context, id string) ([]UserUnit, error)
}

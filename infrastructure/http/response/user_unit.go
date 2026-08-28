package response

import (
	"time"

	"CredChain_Golang/domain"
)

// UserUnit is the response DTO for user_units rows.
type UserUnit struct {
	ID        string     `json:"id"`
	ParentID  *string    `json:"parent_id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// FromDomainUserUnit converts a domain entity to its response DTO.
func FromDomainUserUnit(u domain.UserUnit) UserUnit {
	return UserUnit{
		ID:        u.Id,
		ParentID:  u.ParentId,
		Name:      u.Name,
		Active:    u.Active,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

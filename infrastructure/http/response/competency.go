package response

import (
	"time"

	"CredChain_Golang/domain"
)

// Competency is the response DTO for competencies rows.
type Competency struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// FromDomainCompetency converts a domain entity to its response DTO.
func FromDomainCompetency(c domain.Competency) Competency {
	return Competency{
		ID:        c.Id,
		Name:      c.Name,
		Active:    c.Active,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

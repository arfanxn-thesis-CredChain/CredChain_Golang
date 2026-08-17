package response

import (
	"time"

	"CredChain_Golang/domain"
)

// CredentialType is the response DTO for credential_types rows.
type CredentialType struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// FromDomainCredentialType converts a domain entity to its response DTO.
func FromDomainCredentialType(t domain.CredentialType) CredentialType {
	return CredentialType{
		ID:        t.Id,
		Name:      t.Name,
		Active:    t.Active,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

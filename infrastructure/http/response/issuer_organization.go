package response

import (
	"time"

	"CredChain_Golang/domain"
)

// IssuerOrganization is the response DTO for credential_issuer_organizations rows.
type IssuerOrganization struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Active    bool       `json:"active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// FromDomainIssuerOrganization converts a domain entity to its response DTO.
func FromDomainIssuerOrganization(o domain.CredentialIssuerOrganization) IssuerOrganization {
	return IssuerOrganization{
		ID:        o.Id,
		Name:      o.Name,
		Active:    o.Active,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

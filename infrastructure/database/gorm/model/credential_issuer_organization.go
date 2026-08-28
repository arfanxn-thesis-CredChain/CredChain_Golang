package model

import (
	"time"

	"CredChain_Golang/domain"
)

// CredentialIssuerOrganization maps to credential_issuer_organizations. Note:
// the column is `active` (NOT `is_active`). The GORM tag intentionally omits
// `default:true` so GORM inserts an explicit value for Active=false instead of
// letting the DB default silently override it.
type CredentialIssuerOrganization struct {
	Id        string     `gorm:"primaryKey;type:char(26);column:id"`
	Name      string     `gorm:"type:varchar(256);column:name;not null;uniqueIndex"`
	Active    bool       `gorm:"type:boolean;column:active;not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime;column:updated_at"`
}

func (CredentialIssuerOrganization) TableName() string { return "credential_issuer_organizations" }

func (m CredentialIssuerOrganization) ToDomain() domain.CredentialIssuerOrganization {
	return domain.CredentialIssuerOrganization{
		Id:        m.Id,
		Name:      m.Name,
		Active:    m.Active,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromDomainCredentialIssuerOrganization(o domain.CredentialIssuerOrganization) CredentialIssuerOrganization {
	return CredentialIssuerOrganization{
		Id:        o.Id,
		Name:      o.Name,
		Active:    o.Active,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

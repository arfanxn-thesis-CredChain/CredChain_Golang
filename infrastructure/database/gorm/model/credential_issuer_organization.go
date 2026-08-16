package model

import (
	"time"

	"CredChain_Golang/domain"
)

type CredentialIssuerOrganization struct {
	Id        string     `gorm:"primaryKey;type:char(26);column:id"`
	Name      string     `gorm:"type:varchar(256);column:name;not null;uniqueIndex"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime;column:updated_at"`
}

func (CredentialIssuerOrganization) TableName() string { return "credential_issuer_organizations" }

func (m CredentialIssuerOrganization) ToDomain() domain.CredentialIssuerOrganization {
	return domain.CredentialIssuerOrganization{
		Id:        m.Id,
		Name:      m.Name,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromDomainCredentialIssuerOrganization(o domain.CredentialIssuerOrganization) CredentialIssuerOrganization {
	return CredentialIssuerOrganization{
		Id:        o.Id,
		Name:      o.Name,
		CreatedAt: o.CreatedAt,
		UpdatedAt: o.UpdatedAt,
	}
}

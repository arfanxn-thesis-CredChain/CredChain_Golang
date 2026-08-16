package model

import (
	"time"

	"CredChain_Golang/domain"
)

// CredentialType maps to credential_types. Note: the column is `active`
// (NOT `is_active`). The GORM tag intentionally omits `default:true` so GORM
// inserts an explicit value for Active=false instead of letting the DB default
// silently override it.
type CredentialType struct {
	Id        string     `gorm:"primaryKey;type:char(26);column:id"`
	Name      string     `gorm:"type:varchar(256);column:name;not null;uniqueIndex"`
	Active    bool       `gorm:"type:boolean;column:active;not null"`
	CreatedAt time.Time  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt *time.Time `gorm:"autoUpdateTime;column:updated_at"`
}

func (CredentialType) TableName() string { return "credential_types" }

func (m CredentialType) ToDomain() domain.CredentialType {
	return domain.CredentialType{
		Id:        m.Id,
		Name:      m.Name,
		Active:    m.Active,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func FromDomainCredentialType(t domain.CredentialType) CredentialType {
	return CredentialType{
		Id:        t.Id,
		Name:      t.Name,
		Active:    t.Active,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

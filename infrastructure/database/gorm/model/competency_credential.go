package model

import "CredChain_Golang/domain"

// CompetencyCredential maps to the competency_credential join table
// (composite PK: competency_id, credential_id).
type CompetencyCredential struct {
	CompetencyId string `gorm:"type:char(26);column:competency_id;primaryKey"`
	CredentialId string `gorm:"type:char(26);column:credential_id;primaryKey"`
}

func (CompetencyCredential) TableName() string { return "competency_credential" }

func (m CompetencyCredential) ToDomain() domain.CompetencyCredential {
	return domain.CompetencyCredential{
		CompetencyId: m.CompetencyId,
		CredentialId: m.CredentialId,
	}
}

func FromDomainCompetencyCredential(l domain.CompetencyCredential) CompetencyCredential {
	return CompetencyCredential{
		CompetencyId: l.CompetencyId,
		CredentialId: l.CredentialId,
	}
}

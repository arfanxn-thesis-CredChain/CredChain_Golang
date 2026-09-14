package model

import (
	"time"

	"CredChain_Golang/domain"
)

// Credential is the GORM model for the credentials table.
//
// Meta uses serializer:json so SQLite tests round-trip JSONB columns through
// TEXT (matching model.User.Meta). Extraction data (text, ids, embedding)
// lives in MongoDB (credential_extractions), not on this row.
// Extract state is derived from ExtractEnqueuedAt / ExtractedAt /
// ExtractFailedAt (see domain.Credential.ExtractState) — not a column.
//
// The partial unique index on file_hash mirrors the migration:
// WHERE revoked_at IS NULL AND rejected_at IS NULL.
// The composite unique index (issuer_organization_id, number) mirrors
// uq_credentials_issuer_org_number, which is partial in Postgres
// (WHERE issuer_organization_id IS NOT NULL) because the FK is nullable
// while a submitted organization name awaits reviewer resolution.
//
// HolderUser / IssuerUser / RevokerUser / RejecterUser are GORM relationship fields.
// They are populated by Preload in the repository when the query's Includes
// contains the corresponding key.
type Credential struct {
	Id                              string                       `gorm:"primaryKey;type:char(26);column:id"`
	HolderUserId                    string                       `gorm:"type:char(26);column:holder_user_id;index;not null"`
	SubmitterUserId                 string                       `gorm:"type:char(26);column:submitter_user_id;not null"`
	IssuerUserId                    *string                      `gorm:"type:char(26);column:issuer_user_id;index"`
	SubmittedIssuerOrganizationName *string                      `gorm:"type:varchar(256);column:submitted_issuer_organization_name"`
	IssuerOrganizationId            *string                      `gorm:"type:char(26);column:issuer_organization_id;uniqueIndex:uq_credentials_issuer_org_number"`
	SubmittedTypeName               *string                      `gorm:"type:varchar(256);column:submitted_type_name"`
	TypeId                          *string                      `gorm:"type:char(26);column:type_id;index"`
	Number                          *string                      `gorm:"type:varchar(256);column:number;uniqueIndex:uq_credentials_issuer_org_number"`
	Name                            string                       `gorm:"type:varchar(256);column:name;not null"`
	Meta                            map[string]any               `gorm:"type:jsonb;serializer:json;column:meta"`
	SubmittedCompetencies           domain.SubmittedCompetencies `gorm:"type:jsonb;serializer:json;column:submitted_competencies"`
	TokenID                         *string                      `gorm:"type:varchar(256);column:token_id;uniqueIndex"`
	FileHash                        string                       `gorm:"type:char(66);column:file_hash;index:idx_credentials_file_hash_active,unique,where:revoked_at IS NULL AND rejected_at IS NULL"`
	FileURI                         *string                      `gorm:"type:text;column:file_uri"`
	ExtractEnqueuedAt               *time.Time                   `gorm:"column:extract_enqueued_at"`
	ExtractFailedAt                 *time.Time                   `gorm:"column:extract_failed_at"`
	ExtractError                    *string                      `gorm:"type:text;column:extract_error"`
	RejecterUserId                  *string                      `gorm:"type:char(26);column:rejecter_user_id"`
	RevokerUserId                   *string                      `gorm:"type:char(26);column:revoker_user_id"`
	RejectionReason                 *string                      `gorm:"type:text;column:rejection_reason"`
	IssuedAt                        time.Time                    `gorm:"column:issued_at;not null"`
	ExpiresAt                       *time.Time                   `gorm:"column:expires_at;index"`
	ApprovedAt                      *time.Time                   `gorm:"column:approved_at"`
	RejectedAt                      *time.Time                   `gorm:"column:rejected_at"`
	RevokedAt                       *time.Time                   `gorm:"column:revoked_at;index"`
	ExtractedAt                     *time.Time                   `gorm:"column:extracted_at"`
	CreatedAt                       time.Time                    `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt                       *time.Time                   `gorm:"autoUpdateTime;column:updated_at"`

	// GORM relations — populated by db.Preload("HolderUser") etc. from the
	// repository layer when the caller requests includes of "holder",
	// "issuer", "revoker", or "rejecter". A single batch IN-clause query runs per
	// Preload regardless of result size (no N+1).
	HolderUser   User `gorm:"foreignKey:Id;references:HolderUserId"`
	IssuerUser   User `gorm:"foreignKey:Id;references:IssuerUserId"`
	RevokerUser  User `gorm:"foreignKey:Id;references:RevokerUserId"`
	RejecterUser User `gorm:"foreignKey:Id;references:RejecterUserId"`

	// Competencies are the competency rows linked through the
	// competency_credential join table. Populated by Preload("Competencies")
	// when the caller requests the "competencies" include.
	Competencies []Competency `gorm:"many2many:competency_credential;"`

	// Type / IssuerOrganization are pointer relations since TypeId /
	// IssuerOrganizationId are nullable FKs: an un-preloaded or unresolved
	// relation stays nil, no value-type zero-row guard needed. Populated by
	// Preload("Type") / Preload("IssuerOrganization") when the caller
	// requests the "type" / "issuer_organization" include.
	Type               *CredentialType               `gorm:"foreignKey:Id;references:TypeId"`
	IssuerOrganization *CredentialIssuerOrganization `gorm:"foreignKey:Id;references:IssuerOrganizationId"`
}

func (Credential) TableName() string { return "credentials" }

// ToDomain converts a GORM credential to its domain entity. Preloaded
// user relations (HolderUser, IssuerUser, RevokerUser) are mapped to the
// corresponding domain.User pointers.
func (m Credential) ToDomain() domain.Credential {
	c := domain.Credential{
		ID:                              m.Id,
		HolderUserID:                    m.HolderUserId,
		SubmitterUserID:                 m.SubmitterUserId,
		IssuerUserID:                    m.IssuerUserId,
		SubmittedIssuerOrganizationName: m.SubmittedIssuerOrganizationName,
		IssuerOrganizationID:            m.IssuerOrganizationId,
		SubmittedTypeName:               m.SubmittedTypeName,
		TypeID:                          m.TypeId,
		Number:                          m.Number,
		Name:                            m.Name,
		Meta:                            m.Meta,
		SubmittedCompetencies:           m.SubmittedCompetencies,
		TokenID:                         m.TokenID,
		FileHash:                        m.FileHash,
		FileURI:                         m.FileURI,
		ExtractEnqueuedAt:               m.ExtractEnqueuedAt,
		ExtractFailedAt:                 m.ExtractFailedAt,
		ExtractError:                    m.ExtractError,
		RejecterUserID:                  m.RejecterUserId,
		RevokerUserID:                   m.RevokerUserId,
		RejectionReason:                 m.RejectionReason,
		IssuedAt:                        m.IssuedAt,
		ExpiresAt:                       m.ExpiresAt,
		ApprovedAt:                      m.ApprovedAt,
		RejectedAt:                      m.RejectedAt,
		RevokedAt:                       m.RevokedAt,
		ExtractedAt:                     m.ExtractedAt,
		CreatedAt:                       m.CreatedAt,
		UpdatedAt:                       m.UpdatedAt,
	}
	// Guard on the preloaded row's own PK, not the foreign key: the *User
	// association fields are value types, so an un-preloaded relation is a
	// zero User{} (empty Id). Keying off the FK would fabricate an empty
	// user whenever the FK is set but the row wasn't preloaded.
	if m.HolderUser.Id != "" {
		u := m.HolderUser.ToDomain()
		c.Holder = &u
	}
	if m.IssuerUser.Id != "" {
		u := m.IssuerUser.ToDomain()
		c.Issuer = &u
	}
	if m.RevokerUser.Id != "" {
		u := m.RevokerUser.ToDomain()
		c.Revoker = &u
	}
	if m.RejecterUser.Id != "" {
		u := m.RejecterUser.ToDomain()
		c.Rejecter = &u
	}
	for _, comp := range m.Competencies {
		c.Competencies = append(c.Competencies, comp.ToDomain())
	}
	if m.Type != nil {
		t := m.Type.ToDomain()
		c.Type = &t
	}
	if m.IssuerOrganization != nil {
		o := m.IssuerOrganization.ToDomain()
		c.IssuerOrganization = &o
	}
	return c
}

// FromDomainCredential converts a domain.Credential into a GORM model.
func FromDomainCredential(c domain.Credential) Credential {
	m := Credential{
		Id:                              c.ID,
		HolderUserId:                    c.HolderUserID,
		SubmitterUserId:                 c.SubmitterUserID,
		IssuerUserId:                    c.IssuerUserID,
		SubmittedIssuerOrganizationName: c.SubmittedIssuerOrganizationName,
		IssuerOrganizationId:            c.IssuerOrganizationID,
		SubmittedTypeName:               c.SubmittedTypeName,
		TypeId:                          c.TypeID,
		Number:                          c.Number,
		Name:                            c.Name,
		Meta:                            c.Meta,
		SubmittedCompetencies:           c.SubmittedCompetencies,
		TokenID:                         c.TokenID,
		FileHash:                        c.FileHash,
		FileURI:                         c.FileURI,
		ExtractEnqueuedAt:               c.ExtractEnqueuedAt,
		ExtractFailedAt:                 c.ExtractFailedAt,
		ExtractError:                    c.ExtractError,
		RejecterUserId:                  c.RejecterUserID,
		RevokerUserId:                   c.RevokerUserID,
		RejectionReason:                 c.RejectionReason,
		IssuedAt:                        c.IssuedAt,
		ExpiresAt:                       c.ExpiresAt,
		ApprovedAt:                      c.ApprovedAt,
		RejectedAt:                      c.RejectedAt,
		RevokedAt:                       c.RevokedAt,
		ExtractedAt:                     c.ExtractedAt,
		CreatedAt:                       c.CreatedAt,
		UpdatedAt:                       c.UpdatedAt,
	}
	for _, comp := range c.Competencies {
		m.Competencies = append(m.Competencies, FromDomainCompetency(comp))
	}
	return m
}

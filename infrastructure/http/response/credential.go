package response

import (
	"time"

	"CredChain_Golang/domain"
)

// Credential is the response DTO for credential data. Embeddings are excluded
// to keep payloads small (a 768-float vector per credential adds substantial
// bytes).
//
// Holder, Issuer, and Revoker are optional user expansions loaded from the
// preloaded domain.Credential entity via FromDomainCredential.
type Credential struct {
	ID                   string                  `json:"id"`
	HolderUserID         string                  `json:"holder_user_id"`
	SubmitterUserID      string                  `json:"submitter_user_id"`
	IssuerUserID         *string                 `json:"issuer_user_id"`
	IssuerOrganizationID string                  `json:"issuer_organization_id"`
	TypeID               string                  `json:"type_id"`
	Number               *string                 `json:"number"`
	RevokerUserID        *string                 `json:"revoker_user_id"`
	Name                 string                  `json:"name"`
	Meta                 map[string]any          `json:"meta"`
	Competencies         []Competency            `json:"competencies"`
	TokenID              *string                 `json:"token_id"`
	FileHash             string                  `json:"file_hash"`
	FileURI              *string                 `json:"file_uri"`
	ExtractState         domain.ExtractState     `json:"extract_state"`
	ExtractEnqueuedAt    *time.Time              `json:"extract_enqueued_at"`
	ExtractFailedAt      *time.Time              `json:"extract_failed_at"`
	ExtractError         *string                 `json:"extract_error"`
	ExtractedAt          *time.Time              `json:"extracted_at"`
	IssuedAt             time.Time               `json:"issued_at"`
	RevokedAt            *time.Time              `json:"revoked_at"`
	ExpiresAt            *time.Time              `json:"expires_at"`
	Status               domain.CredentialStatus `json:"status"`
	ActivatedAt          *time.Time              `json:"activated_at"`
	RejecterUserID       *string                 `json:"rejecter_user_id"`
	RejectedAt           *time.Time              `json:"rejected_at"`
	RejectionReason      *string                 `json:"rejection_reason"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            *time.Time              `json:"updated_at"`
	Holder               *User                   `json:"holder,omitempty"`
	Issuer               *User                   `json:"issuer,omitempty"`
	Revoker              *User                   `json:"revoker,omitempty"`
	Rejecter             *User                   `json:"rejecter,omitempty"`
	Type                 *CredentialType         `json:"type,omitempty"`
	IssuerOrganization   *IssuerOrganization     `json:"issuer_organization,omitempty"`
}

// FromDomainCredential converts a domain Credential entity to a response DTO.
// Preloaded holder/issuer/revoker users are read directly from the domain
// entity (populated by the repository's GORM Preload).
func FromDomainCredential(c domain.Credential) Credential {
	out := Credential{
		ID:                   c.ID,
		HolderUserID:         c.HolderUserID,
		SubmitterUserID:      c.SubmitterUserID,
		IssuerUserID:         c.IssuerUserID,
		IssuerOrganizationID: c.IssuerOrganizationID,
		TypeID:               c.TypeID,
		Number:               c.Number,
		RevokerUserID:        c.RevokerUserID,
		Name:                 c.Name,
		Meta:                 c.Meta,
		TokenID:              c.TokenID,
		FileHash:             c.FileHash,
		FileURI:              c.FileURI,
		ExtractState:         c.ExtractState(),
		ExtractEnqueuedAt:    c.ExtractEnqueuedAt,
		ExtractFailedAt:      c.ExtractFailedAt,
		ExtractError:         c.ExtractError,
		ExtractedAt:          c.ExtractedAt,
		IssuedAt:             c.IssuedAt,
		RevokedAt:            c.RevokedAt,
		ExpiresAt:            c.ExpiresAt,
		Status:               c.Status(),
		ActivatedAt:          c.ActivatedAt,
		RejecterUserID:       c.RejecterUserID,
		RejectedAt:           c.RejectedAt,
		RejectionReason:      c.RejectionReason,
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
	}
	if c.Holder != nil {
		h := FromDomainUser(*c.Holder)
		out.Holder = &h
	}
	if c.Issuer != nil {
		i := FromDomainUser(*c.Issuer)
		out.Issuer = &i
	}
	if c.Revoker != nil {
		r := FromDomainUser(*c.Revoker)
		out.Revoker = &r
	}
	if c.Rejecter != nil {
		rej := FromDomainUser(*c.Rejecter)
		out.Rejecter = &rej
	}
	if c.Type != nil {
		ty := FromDomainCredentialType(*c.Type)
		out.Type = &ty
	}
	if c.IssuerOrganization != nil {
		o := FromDomainIssuerOrganization(*c.IssuerOrganization)
		out.IssuerOrganization = &o
	}
	for _, comp := range c.Competencies {
		out.Competencies = append(out.Competencies, FromDomainCompetency(comp))
	}
	return out
}

// VerificationCredential contains only facts intentionally exposed by public verification.
type VerificationCredential struct {
	IssuerOrganizationName *string                 `json:"issuer_organization_name"`
	CredentialTypeName     *string                 `json:"credential_type_name"`
	ID                     string                  `json:"id"`
	Name                   string                  `json:"name"`
	Status                 domain.CredentialStatus `json:"status"`
	IsSubmission           bool                    `json:"is_submission"`
	IssuerOrganizationID   string                  `json:"issuer_organization_id"`
	Number                 *string                 `json:"number"`
	FileHash               string                  `json:"file_hash"`
	IssuedAt               time.Time               `json:"issued_at"`
	ExpiresAt              *time.Time              `json:"expires_at"`
}

func FromDomainVerificationCredential(c domain.Credential) VerificationCredential {
	out := VerificationCredential{
		ID: c.ID, Name: c.Name, Status: c.Status(),
		IsSubmission:         c.SubmitterUserID == c.HolderUserID,
		IssuerOrganizationID: c.IssuerOrganizationID, Number: c.Number,
		FileHash: c.FileHash, IssuedAt: c.IssuedAt, ExpiresAt: c.ExpiresAt,
	}
	if c.IssuerOrganization != nil {
		name := c.IssuerOrganization.Name
		out.IssuerOrganizationName = &name
	}
	if c.Type != nil {
		name := c.Type.Name
		out.CredentialTypeName = &name
	}
	return out
}

// VerificationPublicDetails is released only after live exact-file verification.
// Reference names are current database labels, not independently attested facts.
type VerificationPublicDetails struct {
	VerificationCredential
	RevokedAt    *time.Time               `json:"revoked_at"`
	Holder       *VerificationHolder      `json:"holder"`
	Competencies []VerificationCompetency `json:"competencies"`
	Meta         map[string]any           `json:"meta"`
	TokenID      *string                  `json:"token_id"`
	CreatedAt    time.Time                `json:"created_at"`
	ActivatedAt  *time.Time               `json:"activated_at"`
	RejectedAt   *time.Time               `json:"rejected_at"`
}

type VerificationHolder struct {
	Name          *string        `json:"name"`
	Number        *string        `json:"number"`
	UnitName      *string        `json:"unit_name"`
	JoinedYear    *int           `json:"joined_year"`
	Email         string         `json:"email"`
	Gender        *domain.Gender `json:"gender"`
	Role          domain.Role    `json:"role"`
	WalletAddress string         `json:"wallet_address"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     *time.Time     `json:"updated_at"`
	DeletedAt     *time.Time     `json:"deleted_at"`
}

type VerificationCompetency struct {
	Name string `json:"name"`
}

func FromDomainVerificationPublicDetails(c domain.Credential, holderUnitName *string) VerificationPublicDetails {
	out := VerificationPublicDetails{
		VerificationCredential: FromDomainVerificationCredential(c),
		RevokedAt:              c.RevokedAt,
		Competencies:           make([]VerificationCompetency, 0, len(c.Competencies)),
		Meta:                   c.Meta, TokenID: c.TokenID, CreatedAt: c.CreatedAt,
		ActivatedAt: c.ActivatedAt, RejectedAt: c.RejectedAt,
	}
	if h := c.Holder; h != nil {
		out.Holder = &VerificationHolder{
			Name: h.Name, Number: h.Number, UnitName: holderUnitName, JoinedYear: h.JoinedYear,
			Email: h.Email, Gender: h.Gender, Role: h.Role, WalletAddress: h.WalletAddress,
			CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt, DeletedAt: h.DeletedAt,
		}
	}
	for _, competency := range c.Competencies {
		out.Competencies = append(out.Competencies, VerificationCompetency{Name: competency.Name})
	}
	return out
}

// CredentialVerify is the response payload for POST /api/credentials/verify.
// VerdictCode is the 6-digit domain code for the verify outcome; Description
// is its localized message, resolved by the handler via the request's i18n
// localizer (Accept-Language). Credential is the matched credential, if any.
type CredentialVerify struct {
	ExactMatchConfirmed bool                       `json:"exact_match_confirmed"`
	PublicDetails       *VerificationPublicDetails `json:"public_details,omitempty"`
	VerdictCode         int                        `json:"verdict_code"`
	SimilarityScore     *float64                   `json:"similarity_score,omitempty"`
	SimilarityPercent   *string                    `json:"similarity_percent,omitempty"`
	Description         string                     `json:"description"`
	Credential          *VerificationCredential    `json:"credential,omitempty"`
}

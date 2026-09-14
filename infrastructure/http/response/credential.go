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
	ID                              string                  `json:"id"`
	HolderUserID                    string                  `json:"holder_user_id"`
	SubmitterUserID                 string                  `json:"submitter_user_id"`
	IssuerUserID                    *string                 `json:"issuer_user_id"`
	SubmittedIssuerOrganizationName *string                 `json:"submitted_issuer_organization_name"`
	IssuerOrganizationID            *string                 `json:"issuer_organization_id"`
	SubmittedTypeName               *string                 `json:"submitted_type_name"`
	TypeID                          *string                 `json:"type_id"`
	Number                          *string                 `json:"number"`
	RevokerUserID                   *string                 `json:"revoker_user_id"`
	Name                            string                  `json:"name"`
	Meta                            map[string]any          `json:"meta"`
	// SubmittedCompetencies mirrors the staged names; Competencies carries the
	// resolved rows the reviewer linked. A UI shows staged entries whose
	// resolved_id is null as pending review.
	SubmittedCompetencies []domain.SubmittedCompetency `json:"submitted_competencies"`
	Competencies          []Competency                 `json:"competencies"`
	// UnresolvedMetadata lists the metadata kinds blocking approval; empty
	// means approvable. Mirrors domain.Credential.UnresolvedMetadata.
	UnresolvedMetadata []string                `json:"unresolved_metadata"`
	TokenID            *string                 `json:"token_id"`
	FileHash           string                  `json:"file_hash"`
	FileURI            *string                 `json:"file_uri"`
	ExtractState       domain.ExtractState     `json:"extract_state"`
	ExtractEnqueuedAt  *time.Time              `json:"extract_enqueued_at"`
	ExtractFailedAt    *time.Time              `json:"extract_failed_at"`
	ExtractError       *string                 `json:"extract_error"`
	ExtractedAt        *time.Time              `json:"extracted_at"`
	IssuedAt           time.Time               `json:"issued_at"`
	RevokedAt          *time.Time              `json:"revoked_at"`
	ExpiresAt          *time.Time              `json:"expires_at"`
	Status             domain.CredentialStatus `json:"status"`
	ApprovedAt         *time.Time              `json:"approved_at"`
	RejecterUserID     *string                 `json:"rejecter_user_id"`
	RejectedAt         *time.Time              `json:"rejected_at"`
	RejectionReason    *string                 `json:"rejection_reason"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          *time.Time              `json:"updated_at"`
	Holder             *User                   `json:"holder,omitempty"`
	Issuer             *User                   `json:"issuer,omitempty"`
	Revoker            *User                   `json:"revoker,omitempty"`
	Rejecter           *User                   `json:"rejecter,omitempty"`
	Type               *CredentialType         `json:"type,omitempty"`
	IssuerOrganization *IssuerOrganization     `json:"issuer_organization,omitempty"`
}

// FromDomainCredential converts a domain Credential entity to a response DTO.
// Preloaded holder/issuer/revoker users are read directly from the domain
// entity (populated by the repository's GORM Preload).
func FromDomainCredential(c domain.Credential) Credential {
	out := Credential{
		ID:                              c.ID,
		HolderUserID:                    c.HolderUserID,
		SubmitterUserID:                 c.SubmitterUserID,
		IssuerUserID:                    c.IssuerUserID,
		SubmittedIssuerOrganizationName: c.SubmittedIssuerOrganizationName,
		IssuerOrganizationID:            c.IssuerOrganizationID,
		SubmittedTypeName:               c.SubmittedTypeName,
		TypeID:                          c.TypeID,
		SubmittedCompetencies:           c.SubmittedCompetencies,
		UnresolvedMetadata:              normalizeUnresolvedMetadata(c.UnresolvedMetadata()),
		Number:                          c.Number,
		RevokerUserID:                   c.RevokerUserID,
		Name:                            c.Name,
		Meta:                            c.Meta,
		TokenID:                         c.TokenID,
		FileHash:                        c.FileHash,
		FileURI:                         c.FileURI,
		ExtractState:                    c.ExtractState(),
		ExtractEnqueuedAt:               c.ExtractEnqueuedAt,
		ExtractFailedAt:                 c.ExtractFailedAt,
		ExtractError:                    c.ExtractError,
		ExtractedAt:                     c.ExtractedAt,
		IssuedAt:                        c.IssuedAt,
		RevokedAt:                       c.RevokedAt,
		ExpiresAt:                       c.ExpiresAt,
		Status:                          c.Status(),
		ApprovedAt:                      c.ApprovedAt,
		RejecterUserID:                  c.RejecterUserID,
		RejectedAt:                      c.RejectedAt,
		RejectionReason:                 c.RejectionReason,
		CreatedAt:                       c.CreatedAt,
		UpdatedAt:                       c.UpdatedAt,
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

// normalizeUnresolvedMetadata guarantees the JSON field is `[]` rather than
// `null` when nothing blocks approval — the UI reads `.length` to gate Approve.
func normalizeUnresolvedMetadata(u []string) []string {
	if u == nil {
		return []string{}
	}
	return u
}

// CredentialVerify is the response payload for POST /api/credentials/verify.
// VerdictCode is the 6-digit domain code for the verify outcome; Description
// is its localized message, resolved by the handler via the request's i18n
// localizer (Accept-Language). Credential is the matched credential, if any.
type CredentialVerify struct {
	VerdictCode       int         `json:"verdict_code"`
	SimilarityScore   *float64    `json:"similarity_score,omitempty"`
	SimilarityPercent *string     `json:"similarity_percent,omitempty"`
	Description       string      `json:"description"`
	Credential        *Credential `json:"credential,omitempty"`
}

package response

import (
	"time"

	"CredChain_Golang/domain"
)

// IssuerOrganization is the response DTO for credential_issuer_organizations rows.
type IssuerOrganization struct {
	ID              string                 `json:"id"`
	SubmitterUserID *string                `json:"submitter_user_id"`
	ApproverUserID  *string                `json:"approver_user_id"`
	RejecterUserID  *string                `json:"rejecter_user_id"`
	Name            string                 `json:"name"`
	Active          bool                   `json:"active"`
	RejectionReason *string                `json:"rejection_reason"`
	ActivatedAt     *time.Time             `json:"activated_at"`
	RejectedAt      *time.Time             `json:"rejected_at"`
	Status          domain.ReferenceStatus `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       *time.Time             `json:"updated_at"`

	Submitter *User `json:"submitter,omitempty"`
	Approver  *User `json:"approver,omitempty"`
	Rejecter  *User `json:"rejecter,omitempty"`
}

// FromDomainIssuerOrganization converts a domain entity to its response DTO.
func FromDomainIssuerOrganization(o domain.CredentialIssuerOrganization) IssuerOrganization {
	res := IssuerOrganization{
		ID:              o.Id,
		SubmitterUserID: o.SubmitterUserId,
		ApproverUserID:  o.ApproverUserId,
		RejecterUserID:  o.RejecterUserId,
		Name:            o.Name,
		Active:          o.Active,
		RejectionReason: o.RejectionReason,
		ActivatedAt:     o.ActivatedAt,
		RejectedAt:      o.RejectedAt,
		Status:          o.Status(),
		CreatedAt:       o.CreatedAt,
		UpdatedAt:       o.UpdatedAt,
	}
	if o.Submitter != nil {
		u := FromDomainUser(*o.Submitter)
		res.Submitter = &u
	}
	if o.Approver != nil {
		u := FromDomainUser(*o.Approver)
		res.Approver = &u
	}
	if o.Rejecter != nil {
		u := FromDomainUser(*o.Rejecter)
		res.Rejecter = &u
	}
	return res
}

package response

import (
	"time"

	"CredChain_Golang/domain"
)

// CredentialType is the response DTO for credential_types rows.
type CredentialType struct {
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

// FromDomainCredentialType converts a domain entity to its response DTO.
func FromDomainCredentialType(t domain.CredentialType) CredentialType {
	res := CredentialType{
		ID:              t.Id,
		SubmitterUserID: t.SubmitterUserId,
		ApproverUserID:  t.ApproverUserId,
		RejecterUserID:  t.RejecterUserId,
		Name:            t.Name,
		Active:          t.Active,
		RejectionReason: t.RejectionReason,
		ActivatedAt:     t.ActivatedAt,
		RejectedAt:      t.RejectedAt,
		Status:          t.Status(),
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
	if t.Submitter != nil {
		u := FromDomainUser(*t.Submitter)
		res.Submitter = &u
	}
	if t.Approver != nil {
		u := FromDomainUser(*t.Approver)
		res.Approver = &u
	}
	if t.Rejecter != nil {
		u := FromDomainUser(*t.Rejecter)
		res.Rejecter = &u
	}
	return res
}

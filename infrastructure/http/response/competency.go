package response

import (
	"time"

	"CredChain_Golang/domain"
)

// Competency is the response DTO for competencies rows.
type Competency struct {
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

// FromDomainCompetency converts a domain entity to its response DTO.
func FromDomainCompetency(c domain.Competency) Competency {
	res := Competency{
		ID:              c.Id,
		SubmitterUserID: c.SubmitterUserId,
		ApproverUserID:  c.ApproverUserId,
		RejecterUserID:  c.RejecterUserId,
		Name:            c.Name,
		Active:          c.Active,
		RejectionReason: c.RejectionReason,
		ActivatedAt:     c.ActivatedAt,
		RejectedAt:      c.RejectedAt,
		Status:          c.Status(),
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if c.Submitter != nil {
		u := FromDomainUser(*c.Submitter)
		res.Submitter = &u
	}
	if c.Approver != nil {
		u := FromDomainUser(*c.Approver)
		res.Approver = &u
	}
	if c.Rejecter != nil {
		u := FromDomainUser(*c.Rejecter)
		res.Rejecter = &u
	}
	return res
}

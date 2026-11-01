package model

import (
	"time"

	"CredChain_Golang/domain"
)

// Competency maps to competencies. Note: the column is `active` (NOT
// `is_active`). The GORM tag intentionally omits `default:true` so GORM inserts
// an explicit value for Active=false instead of letting the DB default silently
// override it.
type Competency struct {
	Id              string     `gorm:"primaryKey;type:char(26);column:id"`
	SubmitterUserId *string    `gorm:"type:char(26);column:submitter_user_id;index"`
	ApproverUserId  *string    `gorm:"type:char(26);column:approver_user_id"`
	RejecterUserId  *string    `gorm:"type:char(26);column:rejecter_user_id"`
	Name            string     `gorm:"type:varchar(256);column:name;not null"`
	Active          bool       `gorm:"type:boolean;column:active;not null"`
	RejectionReason *string    `gorm:"type:text;column:rejection_reason"`
	ActivatedAt     *time.Time `gorm:"column:activated_at;index"`
	RejectedAt      *time.Time `gorm:"column:rejected_at"`
	CreatedAt       time.Time  `gorm:"autoCreateTime;column:created_at"`
	UpdatedAt       *time.Time `gorm:"autoUpdateTime;column:updated_at"`

	Submitter *User `gorm:"foreignKey:Id;references:SubmitterUserId"`
	Approver  *User `gorm:"foreignKey:Id;references:ApproverUserId"`
	Rejecter  *User `gorm:"foreignKey:Id;references:RejecterUserId"`
}

func (Competency) TableName() string { return "competencies" }

func (m Competency) ToDomain() domain.Competency {
	c := domain.Competency{
		Id:              m.Id,
		SubmitterUserId: m.SubmitterUserId,
		ApproverUserId:  m.ApproverUserId,
		RejecterUserId:  m.RejecterUserId,
		Name:            m.Name,
		Active:          m.Active,
		RejectionReason: m.RejectionReason,
		ActivatedAt:     m.ActivatedAt,
		RejectedAt:      m.RejectedAt,
		CreatedAt:       m.CreatedAt,
		UpdatedAt:       m.UpdatedAt,
	}
	if m.Submitter != nil && m.Submitter.Id != "" {
		u := m.Submitter.ToDomain()
		c.Submitter = &u
	}
	if m.Approver != nil && m.Approver.Id != "" {
		u := m.Approver.ToDomain()
		c.Approver = &u
	}
	if m.Rejecter != nil && m.Rejecter.Id != "" {
		u := m.Rejecter.ToDomain()
		c.Rejecter = &u
	}
	return c
}

func FromDomainCompetency(c domain.Competency) Competency {
	return Competency{
		Id:              c.Id,
		SubmitterUserId: c.SubmitterUserId,
		ApproverUserId:  c.ApproverUserId,
		RejecterUserId:  c.RejecterUserId,
		Name:            c.Name,
		Active:          c.Active,
		RejectionReason: c.RejectionReason,
		ActivatedAt:     c.ActivatedAt,
		RejectedAt:      c.RejectedAt,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

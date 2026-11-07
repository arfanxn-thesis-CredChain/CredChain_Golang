package model

import (
	"time"

	"CredChain_Golang/domain"
)

type Notification struct {
	ID             string                          `gorm:"primaryKey;type:varchar(26);column:id;index:notifications_user_feed_idx,priority:3,sort:desc"`
	NotifiedUserID string                          `gorm:"type:char(26);not null;uniqueIndex:notifications_user_subject_type_unique,priority:1;index:notifications_user_feed_idx,priority:1;index:notifications_user_unread_idx,where:read_at IS NULL"`
	SubjectType    *domain.NotificationSubjectType `gorm:"type:text;uniqueIndex:notifications_user_subject_type_unique,priority:2;check:notifications_subject_check,(subject_type IS NULL AND subject_id IS NULL) OR (subject_type IS NOT NULL AND subject_id IS NOT NULL AND ((subject_type = 'credential' AND type IN ('credential_registered','credential_approved','credential_rejected','credential_revoked')) OR (subject_type = 'credential_type' AND type = 'credential_type_submitted') OR (subject_type = 'credential_issuer_organization' AND type = 'credential_issuer_organization_submitted') OR (subject_type = 'competency' AND type = 'competency_submitted')))"`
	SubjectID      *string                         `gorm:"type:char(26);uniqueIndex:notifications_user_subject_type_unique,priority:3"`
	Type           domain.NotificationType         `gorm:"type:text;not null;uniqueIndex:notifications_user_subject_type_unique,priority:4;check:notifications_type_check,type IN ('credential_registered','credential_approved','credential_rejected','credential_revoked','credential_type_submitted','credential_issuer_organization_submitted','competency_submitted')"`
	Data           map[string]any                  `gorm:"type:jsonb;serializer:json"`
	CreatedAt      time.Time                       `gorm:"autoCreateTime;not null;index:notifications_user_feed_idx,priority:2,sort:desc"`
	UpdatedAt      *time.Time                      `gorm:"autoUpdateTime:false"`
	ReadAt         *time.Time
	NotifiedUser   *User `gorm:"foreignKey:NotifiedUserID;references:Id;constraint:OnDelete:RESTRICT"`
}

func (Notification) TableName() string { return "notifications" }

func (m Notification) ToDomain() domain.Notification {
	return domain.Notification{ID: m.ID, NotifiedUserID: m.NotifiedUserID, SubjectType: m.SubjectType, SubjectID: m.SubjectID, Type: m.Type, Data: m.Data, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt, ReadAt: m.ReadAt}
}

func FromDomainNotification(n domain.Notification) Notification {
	return Notification{ID: n.ID, NotifiedUserID: n.NotifiedUserID, SubjectType: n.SubjectType, SubjectID: n.SubjectID, Type: n.Type, Data: n.Data, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt, ReadAt: n.ReadAt}
}

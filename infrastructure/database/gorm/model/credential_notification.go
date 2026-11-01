package model

import (
	"time"

	"CredChain_Golang/domain"
)

type CredentialNotification struct {
	ID             string     `gorm:"primaryKey;type:varchar(26);column:id"`
	NotifiedUserID string     `gorm:"type:char(26);column:notified_user_id;not null;uniqueIndex:credential_notifications_user_credential_type_unique,priority:1;index:credential_notifications_user_feed_idx,priority:1;index:credential_notifications_user_unread_idx,priority:1,where:read_at IS NULL"`
	CredentialID   string     `gorm:"type:char(26);column:credential_id;not null;uniqueIndex:credential_notifications_user_credential_type_unique,priority:2"`
	Type           string     `gorm:"type:text;column:type;not null;uniqueIndex:credential_notifications_user_credential_type_unique,priority:3;check:credential_notifications_type_check,type IN ('registered','approved','rejected','revoked')"`
	CreatedAt      time.Time  `gorm:"autoCreateTime;column:created_at;not null;index:credential_notifications_user_feed_idx,priority:2,sort:desc"`
	ReadAt         *time.Time `gorm:"column:read_at"`

	NotifiedUser *User       `gorm:"foreignKey:NotifiedUserID;references:Id;constraint:OnDelete:RESTRICT"`
	Credential   *Credential `gorm:"foreignKey:CredentialID;references:Id;constraint:OnDelete:RESTRICT"`
}

func (CredentialNotification) TableName() string {
	return "credential_notifications"
}

func (m CredentialNotification) ToDomain() domain.CredentialNotification {
	return domain.CredentialNotification{
		ID:             m.ID,
		NotifiedUserID: m.NotifiedUserID,
		CredentialID:   m.CredentialID,
		Type:           domain.CredentialNotificationType(m.Type),
		CreatedAt:      m.CreatedAt,
		ReadAt:         m.ReadAt,
	}
}

func FromDomainCredentialNotification(n domain.CredentialNotification) CredentialNotification {
	return CredentialNotification{
		ID:             n.ID,
		NotifiedUserID: n.NotifiedUserID,
		CredentialID:   n.CredentialID,
		Type:           string(n.Type),
		CreatedAt:      n.CreatedAt,
		ReadAt:         n.ReadAt,
	}
}

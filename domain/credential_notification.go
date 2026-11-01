package domain

import (
	"context"
	"time"
)

type CredentialNotificationType string

const (
	CredentialNotificationTypeRegistered CredentialNotificationType = "registered"
	CredentialNotificationTypeApproved   CredentialNotificationType = "approved"
	CredentialNotificationTypeRejected   CredentialNotificationType = "rejected"
	CredentialNotificationTypeRevoked    CredentialNotificationType = "revoked"
)

type CredentialNotificationStatus string

const (
	CredentialNotificationStatusAll    CredentialNotificationStatus = "all"
	CredentialNotificationStatusUnread CredentialNotificationStatus = "unread"
)

// CredentialNotification has exactly six persisted columns plus an optional
// related credential summary loaded strictly for the requested page.
type CredentialNotification struct {
	ID             string
	NotifiedUserID string
	CredentialID   string
	Type           CredentialNotificationType
	CreatedAt      time.Time
	ReadAt         *time.Time

	// Credential is the optional current related credential relation, loaded
	// in a batch and scoped strictly to holder_user_id == NotifiedUserID.
	// When inaccessible or missing, this is nil.
	Credential *Credential `json:"-"`
}

type CredentialNotificationListOptions struct {
	Status CredentialNotificationStatus // "all" or "unread"; empty defaults to "all"
	Limit  int                          // 1..100; zero defaults to 20
	Cursor string                       // opaque created_at + ULID position
}

type CredentialNotificationPage struct {
	Items      []CredentialNotification
	Total      int64
	NextCursor *string
}

type CredentialNotificationRepository interface {
	Store(ctx context.Context, notifications ...CredentialNotification) error
	List(ctx context.Context, notifiedUserID string, options CredentialNotificationListOptions) (CredentialNotificationPage, error)
	UnreadCount(ctx context.Context, notifiedUserID string) (int64, error)
	MarkRead(ctx context.Context, notifiedUserID, id string) (*CredentialNotification, error)
	MarkAllRead(ctx context.Context, notifiedUserID string) (int64, error)
}

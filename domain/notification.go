package domain

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

type NotificationType string

type NotificationSubjectType string

const (
	NotificationSubjectCredential                   NotificationSubjectType = "credential"
	NotificationSubjectCredentialType               NotificationSubjectType = "credential_type"
	NotificationSubjectCredentialIssuerOrganization NotificationSubjectType = "credential_issuer_organization"
	NotificationSubjectCompetency                   NotificationSubjectType = "competency"

	NotificationTypeCredentialRegistered                  NotificationType = "credential_registered"
	NotificationTypeCredentialApproved                    NotificationType = "credential_approved"
	NotificationTypeCredentialRejected                    NotificationType = "credential_rejected"
	NotificationTypeCredentialRevoked                     NotificationType = "credential_revoked"
	NotificationTypeCredentialTypeSubmitted               NotificationType = "credential_type_submitted"
	NotificationTypeCredentialIssuerOrganizationSubmitted NotificationType = "credential_issuer_organization_submitted"
	NotificationTypeCompetencySubmitted                   NotificationType = "competency_submitted"
)

func (t NotificationType) SubjectType() NotificationSubjectType {
	switch t {
	case NotificationTypeCredentialRegistered, NotificationTypeCredentialApproved, NotificationTypeCredentialRejected, NotificationTypeCredentialRevoked:
		return NotificationSubjectCredential
	case NotificationTypeCredentialTypeSubmitted:
		return NotificationSubjectCredentialType
	case NotificationTypeCredentialIssuerOrganizationSubmitted:
		return NotificationSubjectCredentialIssuerOrganization
	case NotificationTypeCompetencySubmitted:
		return NotificationSubjectCompetency
	default:
		return ""
	}
}

func (t NotificationType) ReviewerOnly() bool {
	return t == NotificationTypeCredentialTypeSubmitted || t == NotificationTypeCredentialIssuerOrganizationSubmitted || t == NotificationTypeCompetencySubmitted
}

type NotificationStatus string

const (
	NotificationStatusAll    NotificationStatus = "all"
	NotificationStatusUnread NotificationStatus = "unread"
)

type NotificationSubject struct {
	ID     string
	Name   string
	Number *string
	Status *string
}

// Notification stores recipient-owned history. Subject is an authorized current summary, not a snapshot.
type Notification struct {
	ID             string
	NotifiedUserID string
	SubjectType    *NotificationSubjectType
	SubjectID      *string
	Type           NotificationType
	Data           map[string]any
	CreatedAt      time.Time
	UpdatedAt      *time.Time
	ReadAt         *time.Time
	Subject        *NotificationSubject `json:"-"`
}

func (n Notification) Validate() error {
	if n.Type.SubjectType() == "" || (n.SubjectType == nil) != (n.SubjectID == nil) {
		return NewError(CodeSystemValidation)
	}
	if n.SubjectType != nil {
		if *n.SubjectType != n.Type.SubjectType() {
			return NewError(CodeSystemValidation)
		}
		if _, err := ulid.ParseStrict(*n.SubjectID); err != nil {
			return NewError(CodeSystemValidation, WithError(err))
		}
	}
	return nil
}

type NotificationListOptions struct {
	Status NotificationStatus
	Limit  int
	Cursor string
}

type NotificationPage struct {
	Items      []Notification
	Total      int64
	NextCursor *string
}

// NotificationScope is created by the service, never bound from HTTP input.
type NotificationScope struct {
	UserID    string
	CanReview bool
}

type NotificationRepository interface {
	Store(ctx context.Context, notifications ...Notification) error
	List(ctx context.Context, scope NotificationScope, options NotificationListOptions) (NotificationPage, error)
	UnreadCount(ctx context.Context, scope NotificationScope) (int64, error)
	MarkRead(ctx context.Context, scope NotificationScope, id string) (*Notification, error)
	MarkAllRead(ctx context.Context, scope NotificationScope) (int64, error)
}

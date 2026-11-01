package credential

import (
	"context"
	"errors"

	"CredChain_Golang/domain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type CredentialNotificationService interface {
	List(ctx context.Context, notifiedUserID string, options domain.CredentialNotificationListOptions) (domain.CredentialNotificationPage, error)
	UnreadCount(ctx context.Context, notifiedUserID string) (int64, error)
	MarkRead(ctx context.Context, notifiedUserID, id string) (*domain.CredentialNotification, error)
	MarkAllRead(ctx context.Context, notifiedUserID string) (int64, error)
}

type credentialNotificationService struct {
	repo domain.CredentialNotificationRepository
}

type CredentialNotificationServiceParams struct {
	fx.In
	Repo domain.CredentialNotificationRepository
}

func NewCredentialNotificationService(p CredentialNotificationServiceParams) CredentialNotificationService {
	return &credentialNotificationService{
		repo: p.Repo,
	}
}

func (s *credentialNotificationService) List(ctx context.Context, notifiedUserID string, options domain.CredentialNotificationListOptions) (domain.CredentialNotificationPage, error) {
	return s.repo.List(ctx, notifiedUserID, options)
}

func (s *credentialNotificationService) UnreadCount(ctx context.Context, notifiedUserID string) (int64, error) {
	return s.repo.UnreadCount(ctx, notifiedUserID)
}

func (s *credentialNotificationService) MarkRead(ctx context.Context, notifiedUserID, id string) (*domain.CredentialNotification, error) {
	notif, err := s.repo.MarkRead(ctx, notifiedUserID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewError(domain.CodeCredentialNotificationNotFound, domain.WithMetadata("id", id))
		}
		return nil, err
	}
	return notif, nil
}

func (s *credentialNotificationService) MarkAllRead(ctx context.Context, notifiedUserID string) (int64, error) {
	return s.repo.MarkAllRead(ctx, notifiedUserID)
}

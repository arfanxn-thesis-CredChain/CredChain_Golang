package notification

import (
	"context"
	"errors"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type NotificationService interface {
	List(context.Context, string, domain.NotificationListOptions) (domain.NotificationPage, error)
	UnreadCount(context.Context, string) (int64, error)
	MarkRead(context.Context, string, string) (*domain.Notification, error)
	MarkAllRead(context.Context, string) (int64, error)
}

type notificationService struct {
	repo      domain.NotificationRepository
	authority chain.AuthorityService
}

type NotificationServiceParams struct {
	fx.In
	Repo      domain.NotificationRepository
	Authority chain.AuthorityService
}

func NewNotificationService(p NotificationServiceParams) NotificationService {
	return &notificationService{repo: p.Repo, authority: p.Authority}
}

func (s *notificationService) List(ctx context.Context, userID string, options domain.NotificationListOptions) (domain.NotificationPage, error) {
	return s.repo.List(ctx, notificationScope(ctx, userID, s.authority), options)
}
func (s *notificationService) UnreadCount(ctx context.Context, userID string) (int64, error) {
	return s.repo.UnreadCount(ctx, notificationScope(ctx, userID, s.authority))
}
func (s *notificationService) MarkRead(ctx context.Context, userID, id string) (*domain.Notification, error) {
	n, err := s.repo.MarkRead(ctx, notificationScope(ctx, userID, s.authority), id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.NewError(domain.CodeNotificationNotFound, domain.WithMetadata("id", id))
	}
	return n, err
}
func (s *notificationService) MarkAllRead(ctx context.Context, userID string) (int64, error) {
	return s.repo.MarkAllRead(ctx, notificationScope(ctx, userID, s.authority))
}

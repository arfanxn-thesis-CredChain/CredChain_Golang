package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockNotificationRepository struct{ mock.Mock }

func (m *MockNotificationRepository) Store(ctx context.Context, rows ...domain.Notification) error {
	if len(m.ExpectedCalls) == 0 {
		return nil
	}
	return m.Called(ctx, rows).Error(0)
}
func (m *MockNotificationRepository) List(ctx context.Context, scope domain.NotificationScope, options domain.NotificationListOptions) (domain.NotificationPage, error) {
	args := m.Called(ctx, scope, options)
	return args.Get(0).(domain.NotificationPage), args.Error(1)
}
func (m *MockNotificationRepository) UnreadCount(ctx context.Context, scope domain.NotificationScope) (int64, error) {
	args := m.Called(ctx, scope)
	return args.Get(0).(int64), args.Error(1)
}
func (m *MockNotificationRepository) MarkRead(ctx context.Context, scope domain.NotificationScope, id string) (*domain.Notification, error) {
	args := m.Called(ctx, scope, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}
func (m *MockNotificationRepository) MarkAllRead(ctx context.Context, scope domain.NotificationScope) (int64, error) {
	args := m.Called(ctx, scope)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.NotificationRepository = (*MockNotificationRepository)(nil)

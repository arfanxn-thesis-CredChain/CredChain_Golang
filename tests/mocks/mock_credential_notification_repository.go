package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockCredentialNotificationRepository struct {
	mock.Mock
}

func (m *MockCredentialNotificationRepository) Store(ctx context.Context, notifications ...domain.CredentialNotification) error {
	if len(m.ExpectedCalls) == 0 {
		return nil
	}
	args := m.Called(ctx, notifications)
	return args.Error(0)
}

func (m *MockCredentialNotificationRepository) List(ctx context.Context, notifiedUserID string, options domain.CredentialNotificationListOptions) (domain.CredentialNotificationPage, error) {
	args := m.Called(ctx, notifiedUserID, options)
	return args.Get(0).(domain.CredentialNotificationPage), args.Error(1)
}

func (m *MockCredentialNotificationRepository) UnreadCount(ctx context.Context, notifiedUserID string) (int64, error) {
	args := m.Called(ctx, notifiedUserID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCredentialNotificationRepository) MarkRead(ctx context.Context, notifiedUserID, id string) (*domain.CredentialNotification, error) {
	args := m.Called(ctx, notifiedUserID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CredentialNotification), args.Error(1)
}

func (m *MockCredentialNotificationRepository) MarkAllRead(ctx context.Context, notifiedUserID string) (int64, error) {
	args := m.Called(ctx, notifiedUserID)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.CredentialNotificationRepository = (*MockCredentialNotificationRepository)(nil)

package credential

import (
	"context"
	"errors"
	"testing"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func TestCredentialNotificationService_List(t *testing.T) {
	repo := new(mocks.MockCredentialNotificationRepository)
	svc := NewCredentialNotificationService(CredentialNotificationServiceParams{Repo: repo})

	expectedPage := domain.CredentialNotificationPage{
		Items: []domain.CredentialNotification{
			{
				ID:             "01HNOTIF111111111111111111",
				NotifiedUserID: "01HUSER1111111111111111111",
				CredentialID:   "01HCRED1111111111111111111",
				Type:           domain.CredentialNotificationTypeApproved,
				CreatedAt:      time.Now(),
			},
		},
		Total: 1,
	}
	opts := domain.CredentialNotificationListOptions{
		Status: domain.CredentialNotificationStatusAll,
		Limit:  20,
	}

	repo.On("List", mock.Anything, "01HUSER1111111111111111111", opts).Return(expectedPage, nil)

	page, err := svc.List(context.Background(), "01HUSER1111111111111111111", opts)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), page.Total)
	assert.Len(t, page.Items, 1)
	repo.AssertExpectations(t)
}

func TestCredentialNotificationService_UnreadCount(t *testing.T) {
	repo := new(mocks.MockCredentialNotificationRepository)
	svc := NewCredentialNotificationService(CredentialNotificationServiceParams{Repo: repo})

	repo.On("UnreadCount", mock.Anything, "01HUSER1111111111111111111").Return(int64(5), nil)

	count, err := svc.UnreadCount(context.Background(), "01HUSER1111111111111111111")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	repo.AssertExpectations(t)
}

func TestCredentialNotificationService_MarkRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := new(mocks.MockCredentialNotificationRepository)
		svc := NewCredentialNotificationService(CredentialNotificationServiceParams{Repo: repo})

		now := time.Now()
		expected := &domain.CredentialNotification{
			ID:             "01HNOTIF111111111111111111",
			NotifiedUserID: "01HUSER1111111111111111111",
			CredentialID:   "01HCRED1111111111111111111",
			Type:           domain.CredentialNotificationTypeApproved,
			CreatedAt:      now.Add(-time.Hour),
			ReadAt:         &now,
		}

		repo.On("MarkRead", mock.Anything, "01HUSER1111111111111111111", "01HNOTIF111111111111111111").Return(expected, nil)

		res, err := svc.MarkRead(context.Background(), "01HUSER1111111111111111111", "01HNOTIF111111111111111111")
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
		repo.AssertExpectations(t)
	})

	t.Run("not found mapped to domain error", func(t *testing.T) {
		repo := new(mocks.MockCredentialNotificationRepository)
		svc := NewCredentialNotificationService(CredentialNotificationServiceParams{Repo: repo})

		repo.On("MarkRead", mock.Anything, "01HUSER1111111111111111111", "non-existent").Return(nil, gorm.ErrRecordNotFound)

		res, err := svc.MarkRead(context.Background(), "01HUSER1111111111111111111", "non-existent")
		assert.Nil(t, res)
		assert.Error(t, err)

		var appErr *domain.Error
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, domain.CodeCredentialNotificationNotFound, appErr.Code)
		repo.AssertExpectations(t)
	})
}

func TestCredentialNotificationService_MarkAllRead(t *testing.T) {
	repo := new(mocks.MockCredentialNotificationRepository)
	svc := NewCredentialNotificationService(CredentialNotificationServiceParams{Repo: repo})

	repo.On("MarkAllRead", mock.Anything, "01HUSER1111111111111111111").Return(int64(3), nil)

	count, err := svc.MarkAllRead(context.Background(), "01HUSER1111111111111111111")
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
	repo.AssertExpectations(t)
}

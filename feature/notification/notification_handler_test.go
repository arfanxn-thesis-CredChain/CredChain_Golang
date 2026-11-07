package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/http/response"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/gintest"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func notificationTestString(s string) *string { return &s }

type mockNotificationService struct {
	mock.Mock
}

func (m *mockNotificationService) List(ctx context.Context, notifiedUserID string, options domain.NotificationListOptions) (domain.NotificationPage, error) {
	args := m.Called(ctx, notifiedUserID, options)
	return args.Get(0).(domain.NotificationPage), args.Error(1)
}

func (m *mockNotificationService) UnreadCount(ctx context.Context, notifiedUserID string) (int64, error) {
	args := m.Called(ctx, notifiedUserID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockNotificationService) MarkRead(ctx context.Context, notifiedUserID, id string) (*domain.Notification, error) {
	args := m.Called(ctx, notifiedUserID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Notification), args.Error(1)
}

func (m *mockNotificationService) MarkAllRead(ctx context.Context, notifiedUserID string) (int64, error) {
	args := m.Called(ctx, notifiedUserID)
	return args.Get(0).(int64), args.Error(1)
}

func TestNotificationHandler_List(t *testing.T) {
	t.Run("success with default options", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodGet),
			gintest.WithPath("/api/notifications"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)

		svc := new(mockNotificationService)
		expectedOpts := domain.NotificationListOptions{
			Status: domain.NotificationStatusAll,
			Limit:  20,
		}
		credName := "Cert 1"
		credNum := "NUM-1"
		svc.On("List", mock.Anything, authUser.Id, expectedOpts).Return(domain.NotificationPage{
			Items: []domain.Notification{
				{
					ID:             "01HNOTIF111111111111111111",
					NotifiedUserID: authUser.Id,
					SubjectID:      notificationTestString("01HCRED1111111111111111111"),
					Type:           domain.NotificationTypeCredentialApproved,
					CreatedAt:      time.Now(),
					Subject: &domain.NotificationSubject{
						ID:     "01HCRED1111111111111111111",
						Name:   credName,
						Number: &credNum,
					},
				},
			},
			Total: 1,
		}, nil)

		h := NewNotificationHandler(NotificationHandlerParams{Svc: svc})
		h.List(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp response.Response[response.NotificationPageResponse]
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, domain.CodeNotificationFetchSuccess, resp.Code)
		assert.Equal(t, int64(1), resp.Data.Total)
		assert.Len(t, resp.Data.Items, 1)
		assert.NotNil(t, resp.Data.Items[0].Subject)
		assert.Equal(t, "Cert 1", resp.Data.Items[0].Subject.Name)
		svc.AssertExpectations(t)
	})

	t.Run("invalid status fails validation", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodGet),
			gintest.WithPath("/api/notifications"),
			gintest.WithQueryString("status=invalid"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)

		svc := new(mockNotificationService)
		h := NewNotificationHandler(NotificationHandlerParams{Svc: svc})
		h.List(c)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestNotificationHandler_UnreadCount(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/api/notifications/unread-count"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := new(mockNotificationService)
	svc.On("UnreadCount", mock.Anything, authUser.Id).Return(int64(4), nil)

	h := NewNotificationHandler(NotificationHandlerParams{Svc: svc})
	h.UnreadCount(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp response.Response[response.NotificationUnreadCountResponse]
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeNotificationUnreadCountSuccess, resp.Code)
	assert.Equal(t, int64(4), resp.Data.UnreadCount)
	svc.AssertExpectations(t)
}

func TestNotificationHandler_MarkRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodPut),
			gintest.WithPath("/api/notifications/01HNOTIF111111111111111111/read"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)
		c.Params = gin.Params{{Key: "id", Value: "01HNOTIF111111111111111111"}}

		svc := new(mockNotificationService)
		now := time.Now()
		svc.On("MarkRead", mock.Anything, authUser.Id, "01HNOTIF111111111111111111").Return(&domain.Notification{
			ID:             "01HNOTIF111111111111111111",
			NotifiedUserID: authUser.Id,
			SubjectID:      notificationTestString("01HCRED1111111111111111111"),
			Type:           domain.NotificationTypeCredentialApproved,
			CreatedAt:      now.Add(-time.Hour),
			ReadAt:         &now,
		}, nil)

		h := NewNotificationHandler(NotificationHandlerParams{Svc: svc})
		h.MarkRead(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp response.Response[response.NotificationResponse]
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, domain.CodeNotificationMarkReadSuccess, resp.Code)
		assert.Equal(t, "01HNOTIF111111111111111111", resp.Data.ID)
		assert.NotNil(t, resp.Data.ReadAt)
		svc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodPut),
			gintest.WithPath("/api/notifications/missing/read"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)
		c.Params = gin.Params{{Key: "id", Value: "missing"}}

		svc := new(mockNotificationService)
		svc.On("MarkRead", mock.Anything, authUser.Id, "missing").Return(nil, domain.NewError(domain.CodeNotificationNotFound))

		h := NewNotificationHandler(NotificationHandlerParams{Svc: svc})
		h.MarkRead(c)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		svc.AssertExpectations(t)
	})
}

func TestNotificationHandler_MarkAllRead(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPut),
		gintest.WithPath("/api/notifications/read-all"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := new(mockNotificationService)
	svc.On("MarkAllRead", mock.Anything, authUser.Id).Return(int64(2), nil)

	h := NewNotificationHandler(NotificationHandlerParams{Svc: svc})
	h.MarkAllRead(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp response.Response[response.NotificationReadCountResponse]
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeNotificationMarkAllReadSuccess, resp.Code)
	assert.Equal(t, int64(2), resp.Data.UpdatedCount)
	svc.AssertExpectations(t)
}

package credential

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

type mockCredentialNotificationService struct {
	mock.Mock
}

func (m *mockCredentialNotificationService) List(ctx context.Context, notifiedUserID string, options domain.CredentialNotificationListOptions) (domain.CredentialNotificationPage, error) {
	args := m.Called(ctx, notifiedUserID, options)
	return args.Get(0).(domain.CredentialNotificationPage), args.Error(1)
}

func (m *mockCredentialNotificationService) UnreadCount(ctx context.Context, notifiedUserID string) (int64, error) {
	args := m.Called(ctx, notifiedUserID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCredentialNotificationService) MarkRead(ctx context.Context, notifiedUserID, id string) (*domain.CredentialNotification, error) {
	args := m.Called(ctx, notifiedUserID, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CredentialNotification), args.Error(1)
}

func (m *mockCredentialNotificationService) MarkAllRead(ctx context.Context, notifiedUserID string) (int64, error) {
	args := m.Called(ctx, notifiedUserID)
	return args.Get(0).(int64), args.Error(1)
}

func TestCredentialNotificationHandler_List(t *testing.T) {
	t.Run("success with default options", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodGet),
			gintest.WithPath("/api/credential-notifications"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)

		svc := new(mockCredentialNotificationService)
		expectedOpts := domain.CredentialNotificationListOptions{
			Status: domain.CredentialNotificationStatusAll,
			Limit:  20,
		}
		credName := "Cert 1"
		credNum := "NUM-1"
		svc.On("List", mock.Anything, authUser.Id, expectedOpts).Return(domain.CredentialNotificationPage{
			Items: []domain.CredentialNotification{
				{
					ID:             "01HNOTIF111111111111111111",
					NotifiedUserID: authUser.Id,
					CredentialID:   "01HCRED1111111111111111111",
					Type:           domain.CredentialNotificationTypeApproved,
					CreatedAt:      time.Now(),
					Credential: &domain.Credential{
						ID:     "01HCRED1111111111111111111",
						Name:   credName,
						Number: &credNum,
					},
				},
			},
			Total: 1,
		}, nil)

		h := NewCredentialNotificationHandler(CredentialNotificationHandlerParams{Svc: svc})
		h.List(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp response.Response[response.CredentialNotificationPageResponse]
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, domain.CodeCredentialNotificationFetchSuccess, resp.Code)
		assert.Equal(t, int64(1), resp.Data.Total)
		assert.Len(t, resp.Data.Items, 1)
		assert.NotNil(t, resp.Data.Items[0].Credential)
		assert.Equal(t, "Cert 1", resp.Data.Items[0].Credential.Name)
		svc.AssertExpectations(t)
	})

	t.Run("invalid status fails validation", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodGet),
			gintest.WithPath("/api/credential-notifications"),
			gintest.WithQueryString("status=invalid"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)

		svc := new(mockCredentialNotificationService)
		h := NewCredentialNotificationHandler(CredentialNotificationHandlerParams{Svc: svc})
		h.List(c)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		svc.AssertNotCalled(t, "List", mock.Anything, mock.Anything, mock.Anything)
	})
}

func TestCredentialNotificationHandler_UnreadCount(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/api/credential-notifications/unread-count"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := new(mockCredentialNotificationService)
	svc.On("UnreadCount", mock.Anything, authUser.Id).Return(int64(4), nil)

	h := NewCredentialNotificationHandler(CredentialNotificationHandlerParams{Svc: svc})
	h.UnreadCount(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp response.Response[response.CredentialNotificationUnreadCountResponse]
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialNotificationUnreadCountSuccess, resp.Code)
	assert.Equal(t, int64(4), resp.Data.UnreadCount)
	svc.AssertExpectations(t)
}

func TestCredentialNotificationHandler_MarkRead(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodPut),
			gintest.WithPath("/api/credential-notifications/01HNOTIF111111111111111111/read"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)
		c.Params = gin.Params{{Key: "id", Value: "01HNOTIF111111111111111111"}}

		svc := new(mockCredentialNotificationService)
		now := time.Now()
		svc.On("MarkRead", mock.Anything, authUser.Id, "01HNOTIF111111111111111111").Return(&domain.CredentialNotification{
			ID:             "01HNOTIF111111111111111111",
			NotifiedUserID: authUser.Id,
			CredentialID:   "01HCRED1111111111111111111",
			Type:           domain.CredentialNotificationTypeApproved,
			CreatedAt:      now.Add(-time.Hour),
			ReadAt:         &now,
		}, nil)

		h := NewCredentialNotificationHandler(CredentialNotificationHandlerParams{Svc: svc})
		h.MarkRead(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp response.Response[response.CredentialNotificationResponse]
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Equal(t, domain.CodeCredentialNotificationMarkReadSuccess, resp.Code)
		assert.Equal(t, "01HNOTIF111111111111111111", resp.Data.ID)
		assert.NotNil(t, resp.Data.ReadAt)
		svc.AssertExpectations(t)
	})

	t.Run("not found", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodPut),
			gintest.WithPath("/api/credential-notifications/missing/read"),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)
		c.Params = gin.Params{{Key: "id", Value: "missing"}}

		svc := new(mockCredentialNotificationService)
		svc.On("MarkRead", mock.Anything, authUser.Id, "missing").Return(nil, domain.NewError(domain.CodeCredentialNotificationNotFound))

		h := NewCredentialNotificationHandler(CredentialNotificationHandlerParams{Svc: svc})
		h.MarkRead(c)

		assert.Equal(t, http.StatusNotFound, rr.Code)
		svc.AssertExpectations(t)
	})
}

func TestCredentialNotificationHandler_MarkAllRead(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPut),
		gintest.WithPath("/api/credential-notifications/read-all"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := new(mockCredentialNotificationService)
	svc.On("MarkAllRead", mock.Anything, authUser.Id).Return(int64(2), nil)

	h := NewCredentialNotificationHandler(CredentialNotificationHandlerParams{Svc: svc})
	h.MarkAllRead(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp response.Response[response.CredentialNotificationReadCountResponse]
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialNotificationMarkAllReadSuccess, resp.Code)
	assert.Equal(t, int64(2), resp.Data.UpdatedCount)
	svc.AssertExpectations(t)
}

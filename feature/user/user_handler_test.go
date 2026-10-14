package user

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/gintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockUserService implements UserService for handler tests.
// Only TransferSuperAdmin is exercised here; other methods panic if called.
type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) Paginate(ctx context.Context, q *domainQuery.Query) ([]domain.User, int, error) {
	panic("not implemented")
}
func (m *mockUserService) Find(ctx context.Context, id string) (*domain.User, error) {
	panic("not implemented")
}
func (m *mockUserService) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	panic("not implemented")
}
func (m *mockUserService) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	panic("not implemented")
}
func (m *mockUserService) UpdateEmail(ctx context.Context, id string, email string, idToken string) (string, error) {
	panic("not implemented")
}
func (m *mockUserService) UpdateLanguage(ctx context.Context, id string, language domain.Language) (domain.Language, error) {
	args := m.Called(ctx, id, language)
	return args.Get(0).(domain.Language), args.Error(1)
}
func (m *mockUserService) UpdateRole(ctx context.Context, updates ...domain.UserRoleUpdate) ([]domain.User, int64, error) {
	panic("not implemented")
}
func (m *mockUserService) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	panic("not implemented")
}
func (m *mockUserService) Delete(ctx context.Context, ids ...string) (int64, error) {
	panic("not implemented")
}
func (m *mockUserService) TransferSuperAdmin(ctx context.Context, targetId string) error {
	return m.Called(ctx, targetId).Error(0)
}
func (m *mockUserService) Restore(ctx context.Context, ids []string) ([]domain.User, int64, error) {
	panic("not implemented")
}

func TestUserHandler_UpdateSelfLanguage(t *testing.T) {
	for _, tc := range []struct {
		name     string
		language domain.Language
		status   int
	}{
		{"valid", domain.LanguageEN, http.StatusOK},
		{"invalid", "fr", http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			authUser := fixtures.NewDomainUser(fixtures.WithID("self-user"))
			c, rr := gintest.NewContext(t,
				gintest.WithMethod(http.MethodPut),
				gintest.WithPath("/api/users/self/language"),
				gintest.WithBody(UserUpdateSelfLanguageRequest{Language: tc.language}),
				gintest.WithUser(&authUser),
				gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
			)
			svc := &mockUserService{}
			if tc.status == http.StatusOK {
				svc.On("UpdateLanguage", mock.Anything, "self-user", domain.LanguageEN).Return(domain.LanguageEN, nil)
			}
			(&userHandler{userSvc: svc}).UpdateSelfLanguage(c)
			assert.Equal(t, tc.status, rr.Code)
			if tc.status == http.StatusOK {
				assert.Contains(t, rr.Body.String(), `"language":"en"`)
			}
			svc.AssertExpectations(t)
		})
	}
}

func TestUserHandler_TransferSuperAdmin_ValidationError(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/users/self/transfer-super-admin"),
		gintest.WithBody(UserTransferSuperAdminRequest{Id: ""}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockUserService{}
	h := &userHandler{userSvc: svc}
	h.TransferSuperAdmin(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "TransferSuperAdmin", mock.Anything, mock.Anything)
}

func TestUserHandler_TransferSuperAdmin_ServiceError(t *testing.T) {
	const targetId = "123e4567-e89b-12d3-a456-426614174000"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/users/self/transfer-super-admin"),
		gintest.WithBody(UserTransferSuperAdminRequest{Id: targetId}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockUserService{}
	svc.On("TransferSuperAdmin", mock.Anything, targetId).
		Return(domain.NewError(domain.CodeUserTransferSuperAdminBlockchainSyncFailed))

	h := &userHandler{userSvc: svc}
	h.TransferSuperAdmin(c)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeUserTransferSuperAdminBlockchainSyncFailed, resp.Code)
	svc.AssertExpectations(t)
}

func TestUserHandler_TransferSuperAdmin_Success(t *testing.T) {
	const targetId = "123e4567-e89b-12d3-a456-426614174000"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/users/self/transfer-super-admin"),
		gintest.WithBody(UserTransferSuperAdminRequest{Id: targetId}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockUserService{}
	svc.On("TransferSuperAdmin", mock.Anything, targetId).Return(nil)

	h := &userHandler{userSvc: svc}
	h.TransferSuperAdmin(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeUserTransferSuperAdminSuccess, resp.Code)
	svc.AssertExpectations(t)
}

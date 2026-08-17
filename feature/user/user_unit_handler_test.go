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

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockUserUnitService implements UserUnitService for handler tests.
type mockUserUnitService struct{ mock.Mock }

func (m *mockUserUnitService) Paginate(ctx context.Context, q *domainQuery.Query) ([]domain.UserUnit, error) {
	args := m.Called(ctx, q)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserUnitService) Find(ctx context.Context, id string) (*domain.UserUnit, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserUnitService) Store(ctx context.Context, name string, parentId *string) (*domain.UserUnit, error) {
	args := m.Called(ctx, name, parentId)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserUnitService) Update(ctx context.Context, id string, name *string, parentId *string) (*domain.UserUnit, error) {
	args := m.Called(ctx, id, name, parentId)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockUserUnitService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

func TestUserUnitHandler_Store_Validation(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/user-units"),
		gintest.WithBody(UserUnitStoreRequest{Name: ""}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockUserUnitService{}
	h := &userUnitHandler{userUnitSvc: svc}
	h.Store(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "Store", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserUnitHandler_Paginate_AppliesDefaultLimit(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/api/user-units"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockUserUnitService{}
	svc.On("Paginate", mock.Anything, mock.MatchedBy(func(q *domainQuery.Query) bool {
		return q != nil && q.Limit == lookupDefaultPageSize
	})).Return([]domain.UserUnit{}, nil)

	h := &userUnitHandler{userUnitSvc: svc}
	h.Paginate(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	svc.AssertExpectations(t)
}

func TestUserUnitHandler_Destroy_Success(t *testing.T) {
	const unitId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodDelete),
		gintest.WithPath("/api/user-units/"+unitId),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: unitId}}

	svc := &mockUserUnitService{}
	svc.On("Destroy", mock.Anything, mock.Anything).Return(int64(1), nil)

	h := &userUnitHandler{userUnitSvc: svc}
	h.Destroy(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeUserUnitDestroySuccess, resp.Code)
	svc.AssertExpectations(t)
}

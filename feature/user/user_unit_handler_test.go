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

func (m *mockUserUnitService) Update(ctx context.Context, id string, name *string, parentId *string, setParent bool) (*domain.UserUnit, error) {
	args := m.Called(ctx, id, name, parentId, setParent)
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

// TestUserUnitHandler_Update_ParentPresenceDetection is the crux of drag-to-root:
// a body containing "parent_id": null must set the parent (setParent=true) so the
// service can clear it, while a body that omits parent_id must leave it untouched
// (setParent=false). A *string alone cannot distinguish JSON null from absent.
func TestUserUnitHandler_Update_ParentPresenceDetection(t *testing.T) {
	const unitId = "01J0000000000000000000000B2"
	unit := domain.UserUnit{Id: unitId, Name: "Unit"}

	t.Run("parent_id null present -> setParent true, nil parent", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodPut),
			gintest.WithPath("/api/user-units/"+unitId),
			gintest.WithBody(map[string]any{"parent_id": nil}),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)
		c.Params = gin.Params{{Key: "id", Value: unitId}}

		svc := &mockUserUnitService{}
		svc.On("Update", mock.Anything, unitId, (*string)(nil), (*string)(nil), true).Return(&unit, nil)
		h := &userUnitHandler{userUnitSvc: svc}
		h.Update(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		svc.AssertExpectations(t)
	})

	t.Run("parent_id absent -> setParent false", func(t *testing.T) {
		authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
		c, rr := gintest.NewContext(t,
			gintest.WithMethod(http.MethodPut),
			gintest.WithPath("/api/user-units/"+unitId),
			gintest.WithBody(map[string]any{"name": "Renamed"}),
			gintest.WithUser(&authUser),
			gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
		)
		c.Params = gin.Params{{Key: "id", Value: unitId}}

		svc := &mockUserUnitService{}
		svc.On("Update", mock.Anything, unitId,
			mock.MatchedBy(func(n *string) bool { return n != nil && *n == "Renamed" }),
			(*string)(nil), false).Return(&unit, nil)
		h := &userUnitHandler{userUnitSvc: svc}
		h.Update(c)

		assert.Equal(t, http.StatusOK, rr.Code)
		svc.AssertExpectations(t)
	})
}

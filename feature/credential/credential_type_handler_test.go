package credential

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

// mockCredentialTypeService implements CredentialTypeService for handler tests.
type mockCredentialTypeService struct{ mock.Mock }

func (m *mockCredentialTypeService) Paginate(ctx context.Context, q *domainQuery.Query) ([]domain.CredentialType, int, error) {
	args := m.Called(ctx, q)
	if v := args.Get(0); v != nil {
		return v.([]domain.CredentialType), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

func (m *mockCredentialTypeService) Find(ctx context.Context, id string) (*domain.CredentialType, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialType), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialTypeService) Store(ctx context.Context, name string, active *bool) (*domain.CredentialType, error) {
	args := m.Called(ctx, name, active)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialType), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialTypeService) Update(ctx context.Context, id string, name *string, active *bool) (*domain.CredentialType, error) {
	args := m.Called(ctx, id, name, active)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialType), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialTypeService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCredentialTypeService) Submit(ctx context.Context, name string) (*domain.CredentialType, error) {
	args := m.Called(ctx, name)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialType), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialTypeService) Approve(ctx context.Context, id string) (*domain.CredentialType, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialType), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialTypeService) Reject(ctx context.Context, id string, reason string) (*domain.CredentialType, error) {
	args := m.Called(ctx, id, reason)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialType), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestCredentialTypeHandler_Store_Validation(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/credential-types"),
		gintest.WithBody(CredentialTypeStoreRequest{Name: ""}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockCredentialTypeService{}
	h := &credentialTypeHandler{credentialTypeSvc: svc}
	h.Store(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "Store", mock.Anything, mock.Anything, mock.Anything)
}

func TestCredentialTypeHandler_Paginate_AppliesDefaultLimit(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/api/credential-types"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockCredentialTypeService{}
	svc.On("Paginate", mock.Anything, mock.MatchedBy(func(q *domainQuery.Query) bool {
		return q != nil && q.Limit == lookupDefaultPageSize
	})).Return([]domain.CredentialType{}, 0, nil)

	h := &credentialTypeHandler{credentialTypeSvc: svc}
	h.Paginate(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	svc.AssertExpectations(t)
}

func TestCredentialTypeHandler_Destroy_Success(t *testing.T) {
	const typeId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodDelete),
		gintest.WithPath("/api/credential-types/"+typeId),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: typeId}}

	svc := &mockCredentialTypeService{}
	svc.On("Destroy", mock.Anything, mock.Anything).Return(int64(1), nil)

	h := &credentialTypeHandler{credentialTypeSvc: svc}
	h.Destroy(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCredentialTypeDestroySuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCredentialTypeHandler_Submit_Success(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/credential-types/submit"),
		gintest.WithBody(CredentialTypeSubmitRequest{Name: "Bachelor of Science"}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	expected := domain.CredentialType{
		Id:              "01J0000000000000000000000A1",
		Name:            "Bachelor of Science",
		Active:          false,
		SubmitterUserId: &authUser.Id,
	}

	svc := &mockCredentialTypeService{}
	svc.On("Submit", mock.Anything, "Bachelor of Science").Return(&expected, nil)

	h := &credentialTypeHandler{credentialTypeSvc: svc}
	h.Submit(c)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCredentialTypeSubmitSuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCredentialTypeHandler_Approve_Success(t *testing.T) {
	const typeId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/credential-types/"+typeId+"/approve"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: typeId}}

	expected := domain.CredentialType{
		Id:             typeId,
		Name:           "Bachelor of Science",
		Active:         true,
		ApproverUserId: &authUser.Id,
	}

	svc := &mockCredentialTypeService{}
	svc.On("Approve", mock.Anything, typeId).Return(&expected, nil)

	h := &credentialTypeHandler{credentialTypeSvc: svc}
	h.Approve(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCredentialTypeApproveSuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCredentialTypeHandler_Reject_Success(t *testing.T) {
	const typeId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/credential-types/"+typeId+"/reject"),
		gintest.WithBody(CredentialTypeRejectRequest{Reason: "Duplicate entry"}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: typeId}}

	reason := "Duplicate entry"
	expected := domain.CredentialType{
		Id:              typeId,
		Name:            "Bachelor of Science",
		Active:          false,
		RejecterUserId:  &authUser.Id,
		RejectionReason: &reason,
	}

	svc := &mockCredentialTypeService{}
	svc.On("Reject", mock.Anything, typeId, "Duplicate entry").Return(&expected, nil)

	h := &credentialTypeHandler{credentialTypeSvc: svc}
	h.Reject(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCredentialTypeRejectSuccess, resp.Code)
	svc.AssertExpectations(t)
}

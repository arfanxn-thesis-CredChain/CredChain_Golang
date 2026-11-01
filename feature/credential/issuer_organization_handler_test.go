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

// mockCredentialIssuerOrganizationService implements
// CredentialIssuerOrganizationService for handler tests.
type mockCredentialIssuerOrganizationService struct{ mock.Mock }

func (m *mockCredentialIssuerOrganizationService) Paginate(ctx context.Context, q *domainQuery.Query) ([]domain.CredentialIssuerOrganization, int, error) {
	args := m.Called(ctx, q)
	if v := args.Get(0); v != nil {
		return v.([]domain.CredentialIssuerOrganization), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

func (m *mockCredentialIssuerOrganizationService) Find(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialIssuerOrganization), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialIssuerOrganizationService) Store(ctx context.Context, name string, active *bool) (*domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, name, active)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialIssuerOrganization), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialIssuerOrganizationService) Update(ctx context.Context, id string, name *string, active *bool) (*domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, id, name, active)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialIssuerOrganization), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialIssuerOrganizationService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCredentialIssuerOrganizationService) Submit(ctx context.Context, name string) (*domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, name)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialIssuerOrganization), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialIssuerOrganizationService) Approve(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialIssuerOrganization), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCredentialIssuerOrganizationService) Reject(ctx context.Context, id string, reason string) (*domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, id, reason)
	if v := args.Get(0); v != nil {
		return v.(*domain.CredentialIssuerOrganization), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestCredentialIssuerOrganizationHandler_Store_Validation(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/issuer-organizations"),
		gintest.WithBody(IssuerOrganizationStoreRequest{Name: ""}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockCredentialIssuerOrganizationService{}
	h := &credentialIssuerOrganizationHandler{issuerOrganizationSvc: svc}
	h.Store(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
}

func TestCredentialIssuerOrganizationHandler_Paginate_AppliesDefaultLimit(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/api/issuer-organizations"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockCredentialIssuerOrganizationService{}
	svc.On("Paginate", mock.Anything, mock.MatchedBy(func(q *domainQuery.Query) bool {
		return q != nil && q.Limit == lookupDefaultPageSize
	})).Return([]domain.CredentialIssuerOrganization{}, 0, nil)

	h := &credentialIssuerOrganizationHandler{issuerOrganizationSvc: svc}
	h.Paginate(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	svc.AssertExpectations(t)
}

func TestCredentialIssuerOrganizationHandler_Destroy_Success(t *testing.T) {
	const orgId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodDelete),
		gintest.WithPath("/api/issuer-organizations/"+orgId),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: orgId}}

	svc := &mockCredentialIssuerOrganizationService{}
	svc.On("Destroy", mock.Anything, mock.Anything).Return(int64(1), nil)

	h := &credentialIssuerOrganizationHandler{issuerOrganizationSvc: svc}
	h.Destroy(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeIssuerOrganizationDestroySuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCredentialIssuerOrganizationHandler_Submit_Success(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/issuer-organizations/submit"),
		gintest.WithBody(IssuerOrganizationSubmitRequest{Name: "Acme Academy"}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	expected := domain.CredentialIssuerOrganization{
		Id:              "01J0000000000000000000000A1",
		Name:            "Acme Academy",
		Active:          false,
		SubmitterUserId: &authUser.Id,
	}

	svc := &mockCredentialIssuerOrganizationService{}
	svc.On("Submit", mock.Anything, "Acme Academy").Return(&expected, nil)

	h := &credentialIssuerOrganizationHandler{issuerOrganizationSvc: svc}
	h.Submit(c)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeIssuerOrganizationSubmitSuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCredentialIssuerOrganizationHandler_Approve_Success(t *testing.T) {
	const orgId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/issuer-organizations/"+orgId+"/approve"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: orgId}}

	expected := domain.CredentialIssuerOrganization{
		Id:             orgId,
		Name:           "Acme Academy",
		Active:         true,
		ApproverUserId: &authUser.Id,
	}

	svc := &mockCredentialIssuerOrganizationService{}
	svc.On("Approve", mock.Anything, orgId).Return(&expected, nil)

	h := &credentialIssuerOrganizationHandler{issuerOrganizationSvc: svc}
	h.Approve(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeIssuerOrganizationApproveSuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCredentialIssuerOrganizationHandler_Reject_Success(t *testing.T) {
	const orgId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/issuer-organizations/"+orgId+"/reject"),
		gintest.WithBody(IssuerOrganizationRejectRequest{Reason: "Unverified entity"}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: orgId}}

	reason := "Unverified entity"
	expected := domain.CredentialIssuerOrganization{
		Id:              orgId,
		Name:            "Acme Academy",
		Active:          false,
		RejecterUserId:  &authUser.Id,
		RejectionReason: &reason,
	}

	svc := &mockCredentialIssuerOrganizationService{}
	svc.On("Reject", mock.Anything, orgId, "Unverified entity").Return(&expected, nil)

	h := &credentialIssuerOrganizationHandler{issuerOrganizationSvc: svc}
	h.Reject(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeIssuerOrganizationRejectSuccess, resp.Code)
	svc.AssertExpectations(t)
}

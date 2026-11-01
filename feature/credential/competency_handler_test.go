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

// mockCompetencyService implements CompetencyService for handler tests.
type mockCompetencyService struct{ mock.Mock }

func (m *mockCompetencyService) Paginate(ctx context.Context, q *domainQuery.Query) ([]domain.Competency, int, error) {
	args := m.Called(ctx, q)
	if v := args.Get(0); v != nil {
		return v.([]domain.Competency), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

func (m *mockCompetencyService) Find(ctx context.Context, id string) (*domain.Competency, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Competency), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCompetencyService) Store(ctx context.Context, name string, active *bool) (*domain.Competency, error) {
	args := m.Called(ctx, name, active)
	if v := args.Get(0); v != nil {
		return v.(*domain.Competency), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCompetencyService) Update(ctx context.Context, id string, name *string, active *bool) (*domain.Competency, error) {
	args := m.Called(ctx, id, name, active)
	if v := args.Get(0); v != nil {
		return v.(*domain.Competency), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCompetencyService) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCompetencyService) Submit(ctx context.Context, name string) (*domain.Competency, error) {
	args := m.Called(ctx, name)
	if v := args.Get(0); v != nil {
		return v.(*domain.Competency), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCompetencyService) Approve(ctx context.Context, id string) (*domain.Competency, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.Competency), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockCompetencyService) Reject(ctx context.Context, id string, reason string) (*domain.Competency, error) {
	args := m.Called(ctx, id, reason)
	if v := args.Get(0); v != nil {
		return v.(*domain.Competency), args.Error(1)
	}
	return nil, args.Error(1)
}

func TestCompetencyHandler_Store_Validation(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/competencies"),
		gintest.WithBody(CompetencyStoreRequest{Name: ""}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockCompetencyService{}
	h := &competencyHandler{competencySvc: svc}
	h.Store(c)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	svc.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
}

func TestCompetencyHandler_Paginate_AppliesDefaultLimit(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodGet),
		gintest.WithPath("/api/competencies"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	svc := &mockCompetencyService{}
	svc.On("Paginate", mock.Anything, mock.MatchedBy(func(q *domainQuery.Query) bool {
		return q != nil && q.Limit == competencyDefaultPageSize
	})).Return([]domain.Competency{}, 0, nil)

	h := &competencyHandler{competencySvc: svc}
	h.Paginate(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	svc.AssertExpectations(t)
}

func TestCompetencyHandler_Destroy_Success(t *testing.T) {
	const competencyId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodDelete),
		gintest.WithPath("/api/competencies/"+competencyId),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: competencyId}}

	svc := &mockCompetencyService{}
	svc.On("Destroy", mock.Anything, mock.Anything).Return(int64(1), nil)

	h := &competencyHandler{competencySvc: svc}
	h.Destroy(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCompetencyDestroySuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCompetencyHandler_Submit_Success(t *testing.T) {
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/competencies/submit"),
		gintest.WithBody(CompetencySubmitRequest{Name: "Go Programming"}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)

	expected := domain.Competency{
		Id:              "01J0000000000000000000000A1",
		Name:            "Go Programming",
		Active:          false,
		SubmitterUserId: &authUser.Id,
	}

	svc := &mockCompetencyService{}
	svc.On("Submit", mock.Anything, "Go Programming").Return(&expected, nil)

	h := &competencyHandler{competencySvc: svc}
	h.Submit(c)

	assert.Equal(t, http.StatusCreated, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCompetencySubmitSuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCompetencyHandler_Approve_Success(t *testing.T) {
	const competencyId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/competencies/"+competencyId+"/approve"),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: competencyId}}

	expected := domain.Competency{
		Id:             competencyId,
		Name:           "Go Programming",
		Active:         true,
		ApproverUserId: &authUser.Id,
	}

	svc := &mockCompetencyService{}
	svc.On("Approve", mock.Anything, competencyId).Return(&expected, nil)

	h := &competencyHandler{competencySvc: svc}
	h.Approve(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCompetencyApproveSuccess, resp.Code)
	svc.AssertExpectations(t)
}

func TestCompetencyHandler_Reject_Success(t *testing.T) {
	const competencyId = "01J0000000000000000000000A1"
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	c, rr := gintest.NewContext(t,
		gintest.WithMethod(http.MethodPost),
		gintest.WithPath("/api/competencies/"+competencyId+"/reject"),
		gintest.WithBody(CompetencyRejectRequest{Reason: "Vague skill description"}),
		gintest.WithUser(&authUser),
		gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)),
	)
	c.Params = gin.Params{{Key: "id", Value: competencyId}}

	reason := "Vague skill description"
	expected := domain.Competency{
		Id:              competencyId,
		Name:            "Go Programming",
		Active:          false,
		RejecterUserId:  &authUser.Id,
		RejectionReason: &reason,
	}

	svc := &mockCompetencyService{}
	svc.On("Reject", mock.Anything, competencyId, "Vague skill description").Return(&expected, nil)

	h := &competencyHandler{competencySvc: svc}
	h.Reject(c)

	assert.Equal(t, http.StatusOK, rr.Code)
	var resp struct {
		Code int `json:"code"`
	}
	assert.NoError(t, json.NewDecoder(rr.Body).Decode(&resp))
	assert.Equal(t, domain.CodeCompetencyRejectSuccess, resp.Code)
	svc.AssertExpectations(t)
}

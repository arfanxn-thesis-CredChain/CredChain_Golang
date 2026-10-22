package overview

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/http/response"
	"CredChain_Golang/tests/gintest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSvc struct{ mock.Mock }

func (m *mockSvc) CredentialRecap(ctx context.Context) (*response.CredentialRecap, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*response.CredentialRecap), args.Error(1)
}

func (m *mockSvc) Get(ctx context.Context, q *domainQuery.Query) (*response.Overview, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*response.Overview), args.Error(1)
}

func TestHandlerGet_IssuerSuccess(t *testing.T) {
	svc := new(mockSvc)
	handler := NewOverviewHandler(OverviewHandlerParams{Svc: svc})

	user := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	ginCtx, rr := gintest.NewContext(t, gintest.WithUser(user), gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)))

	now := time.Now()
	svc.On("Get", mock.Anything, mock.Anything).Return(&response.Overview{
		CredentialCounts: &response.OverviewCredentialCounts{Total: 10, Active: 8, Revoked: 1, Pending: 1, Failed: 1},
		UserCounts:       &response.OverviewUserCounts{Total: 5, Holder: 3, Issuer: 1, Admin: 1, Active: 4, Trashed: 1},
		Recents: &response.OverviewRecents{
			ActiveCredentials: []response.Credential{{ID: "c1", Name: "Degree", IssuedAt: now}},
		},
		ChainDetails: &response.OverviewChainDetails{AuthorityContract: "0xAA", RegistryContract: "0xBB", LastBlock: 100},
	}, nil)

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, float64(domain.CodeOverviewSuccess), body["code"])

	data := body["data"].(map[string]interface{})
	assert.NotNil(t, data["credential_counts"])
	assert.NotNil(t, data["user_counts"])
	assert.NotNil(t, data["chain_details"])
}

func TestHandlerCredentialRecap_Envelope(t *testing.T) {
	svc := new(mockSvc)
	handler := NewOverviewHandler(OverviewHandlerParams{Svc: svc})
	c, rr := gintest.NewContext(t, gintest.WithUser(&domain.User{Role: domain.RoleIssuer}), gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)))
	svc.On("CredentialRecap", mock.Anything).Return(&response.CredentialRecap{AsOf: time.Now(), Rows: []domain.CredentialRecapRow{}, Units: []response.CredentialRecapUnit{}}, nil)
	handler.CredentialRecap(c)
	require.Equal(t, http.StatusOK, rr.Code)
	var body struct {
		Code int `json:"code"`
		Data struct {
			Rows  []domain.CredentialRecapRow    `json:"rows"`
			Units []response.CredentialRecapUnit `json:"units"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
	assert.Equal(t, domain.CodeOverviewSuccess, body.Code)
	require.NotNil(t, body.Data.Rows)
	require.NotNil(t, body.Data.Units)
}

func TestHandlerGet_HolderSuccess(t *testing.T) {
	svc := new(mockSvc)
	handler := NewOverviewHandler(OverviewHandlerParams{Svc: svc})

	user := &domain.User{Id: "holder1", Role: domain.RoleHolder, Email: "holder@test.com", WalletAddress: "0x2"}
	ginCtx, rr := gintest.NewContext(t, gintest.WithUser(user), gintest.WithI18nBundle(gintest.LoadTestI18nBundle(t)))

	svc.On("Get", mock.Anything, mock.Anything).Return(&response.Overview{
		CredentialCounts: &response.OverviewCredentialCounts{Total: 5, Active: 4, Revoked: 1, Pending: 0, Failed: 0},
		Recents:          &response.OverviewRecents{ActiveCredentials: []response.Credential{}},
	}, nil)

	handler.Get(ginCtx)

	assert.Equal(t, http.StatusOK, rr.Code)
	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))

	data := body["data"].(map[string]interface{})
	assert.NotNil(t, data["credential_counts"])
	assert.Nil(t, data["user_counts"])
	assert.Nil(t, data["chain_details"])
}

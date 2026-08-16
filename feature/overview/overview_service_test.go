package overview

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/tests/gintest"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockOverviewRepo struct{ mock.Mock }

func (m *mockOverviewRepo) CredentialCounts(ctx context.Context, q *domainQuery.Query, holderUserID *string) (*domain.OverviewCredentialCounts, error) {
	args := m.Called(ctx, q, holderUserID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OverviewCredentialCounts), args.Error(1)
}

func (m *mockOverviewRepo) UserCounts(ctx context.Context, q *domainQuery.Query) (*domain.OverviewUserCounts, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.OverviewUserCounts), args.Error(1)
}

type mockCredRepo struct{ mock.Mock }

func (m *mockCredRepo) Get(ctx context.Context, q *domainQuery.Query) ([]domain.Credential, int, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.Credential), args.Int(1), args.Error(2)
}

func (m *mockCredRepo) Find(ctx context.Context, id string, q *domainQuery.Query) (*domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepo) FindByIds(ctx context.Context, ids []string, q *domainQuery.Query) ([]domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepo) FindVerifiableById(ctx context.Context, id string, q *domainQuery.Query) (*domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepo) FindVerifiableByIds(ctx context.Context, ids []string, q *domainQuery.Query) ([]domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepo) Store(ctx context.Context, creds ...domain.Credential) ([]domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepo) Update(ctx context.Context, creds ...domain.Credential) ([]domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepo) FindByFileHashes(ctx context.Context, hashes []string, q *domainQuery.Query) ([]domain.Credential, error) {
	return nil, nil
}
func (m *mockCredRepo) FindByHolderId(ctx context.Context, holderID string, q *domainQuery.Query) ([]domain.Credential, error) {
	return nil, nil
}

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Get(ctx context.Context, q *domainQuery.Query) ([]domain.User, int, error) {
	args := m.Called(ctx, q)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]domain.User), args.Int(1), args.Error(2)
}

func (m *mockUserRepo) Find(ctx context.Context, id string) (*domain.User, error) { return nil, nil }
func (m *mockUserRepo) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) FindByRole(ctx context.Context, role domain.Role) ([]domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	return nil, nil
}
func (m *mockUserRepo) Delete(ctx context.Context, ids ...string) (int64, error)  { return 0, nil }
func (m *mockUserRepo) Restore(ctx context.Context, ids ...string) (int64, error) { return 0, nil }
func (m *mockUserRepo) UpdateRole(ctx context.Context, users ...domain.User) ([]domain.User, int64, error) {
	return nil, 0, nil
}

func setupTestService(t *testing.T, user *domain.User) (*overviewService, *mockOverviewRepo, *mockCredRepo, *mockUserRepo, *gin.Context) {
	t.Helper()
	repo := new(mockOverviewRepo)
	credRepo := new(mockCredRepo)
	userRepo := new(mockUserRepo)
	cfg := &config.Config{
		AuthorityContract: ptrStr("0xAuthority"),
		RegistryContract:  ptrStr("0xRegistry"),
	}
	ginCtx, _ := gintest.NewContext(t, gintest.WithUser(user))
	ctx := ginCtx.Request.Context()
	ginCtx.Request = httptest.NewRequest("GET", "/overview", nil).WithContext(ctx)
	svc := &overviewService{
		overviewRepo: repo,
		credRepo:     credRepo,
		userRepo:     userRepo,
		cfg:          cfg,
		chainClient:  nil,
	}
	return svc, repo, credRepo, userRepo, ginCtx
}

func TestGet_Issuer(t *testing.T) {
	issuer := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	svc, repo, credRepo, userRepo, ginCtx := setupTestService(t, issuer)
	q := &domainQuery.Query{}

	repo.On("CredentialCounts", mock.Anything, q, (*string)(nil)).Return(&domain.OverviewCredentialCounts{Total: 10, Active: 8, Revoked: 1, Pending: 1, Failed: 1}, nil)
	repo.On("UserCounts", mock.Anything, q).Return(&domain.OverviewUserCounts{Total: 5, Holder: 3, Issuer: 1, Admin: 1, Active: 4, Trashed: 1}, nil)
	credRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.Credential{}, 0, nil)
	userRepo.On("Get", mock.Anything, mock.Anything).Return(
		[]domain.User{{Id: "u1", Role: domain.RoleHolder, Email: "holder@test.com"}}, 1, nil)

	result, err := svc.Get(ginCtx.Request.Context(), q)
	require.NoError(t, err)

	assert.NotNil(t, result.CredentialCounts)
	assert.Equal(t, 10, result.CredentialCounts.Total)
	assert.NotNil(t, result.UserCounts)
	assert.Equal(t, 5, result.UserCounts.Total)
	assert.NotNil(t, result.Recents)
	assert.NotNil(t, result.ChainDetails)
	assert.Equal(t, "0xAuthority", result.ChainDetails.AuthorityContract)
	assert.Equal(t, uint64(0), result.ChainDetails.LastBlock)
	assert.Equal(t, "", result.ChainDetails.RelayerAddress)
	assert.Equal(t, "0.00", result.ChainDetails.RelayerBalance)
	assert.NotEmpty(t, result.Recents.StoredUsers)

	repo.AssertExpectations(t)
}

func TestGet_Holder(t *testing.T) {
	holder := &domain.User{Id: "holder1", Role: domain.RoleHolder, Email: "holder@test.com", WalletAddress: "0x2"}
	svc, repo, credRepo, _, ginCtx := setupTestService(t, holder)
	q := &domainQuery.Query{}

	repo.On("CredentialCounts", mock.Anything, q, &holder.Id).Return(&domain.OverviewCredentialCounts{Total: 5, Active: 4, Revoked: 1, Pending: 0, Failed: 0}, nil)
	credRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.Credential{}, 0, nil)

	result, err := svc.Get(ginCtx.Request.Context(), q)
	require.NoError(t, err)

	assert.NotNil(t, result.CredentialCounts)
	assert.Equal(t, 5, result.CredentialCounts.Total)
	assert.Nil(t, result.UserCounts)
	assert.NotNil(t, result.Recents)
	assert.Nil(t, result.ChainDetails)
	assert.Empty(t, result.Recents.StoredUsers)

	repo.AssertExpectations(t)
}

func TestGet_RepoError(t *testing.T) {
	issuer := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	svc, repo, _, _, ginCtx := setupTestService(t, issuer)
	q := &domainQuery.Query{}

	repo.On("CredentialCounts", mock.Anything, q, (*string)(nil)).Return(nil, errors.New("db down"))

	_, err := svc.Get(ginCtx.Request.Context(), q)
	require.Error(t, err)
}

func TestGet_LimitParam(t *testing.T) {
	issuer := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	svc, repo, credRepo, userRepo, ginCtx := setupTestService(t, issuer)

	q := &domainQuery.Query{Limit: 3}

	repo.On("CredentialCounts", mock.Anything, q, (*string)(nil)).Return(&domain.OverviewCredentialCounts{Total: 10, Active: 8, Revoked: 1, Pending: 1, Failed: 1}, nil)
	repo.On("UserCounts", mock.Anything, q).Return(&domain.OverviewUserCounts{Total: 5, Holder: 3, Issuer: 1, Admin: 1, Active: 4, Trashed: 1}, nil)
	credRepo.On("Get", mock.Anything, mock.MatchedBy(func(q2 *domainQuery.Query) bool {
		return q2.Limit == 3
	})).Return([]domain.Credential{{ID: "c1"}}, 1, nil)
	userRepo.On("Get", mock.Anything, mock.MatchedBy(func(q2 *domainQuery.Query) bool {
		return q2.Limit == 3
	})).Return([]domain.User{{Id: "u1", Role: domain.RoleHolder, Email: "h@t.com"}}, 1, nil)

	result, err := svc.Get(ginCtx.Request.Context(), q)
	require.NoError(t, err)

	assert.Len(t, result.Recents.ActiveCredentials, 1)
	assert.Len(t, result.Recents.StoredUsers, 1)
	repo.AssertExpectations(t)
}

func TestGet_LimitDefault(t *testing.T) {
	issuer := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	svc, repo, credRepo, userRepo, ginCtx := setupTestService(t, issuer)

	q := &domainQuery.Query{} // Limit = 0, should default to 5

	repo.On("CredentialCounts", mock.Anything, q, (*string)(nil)).Return(&domain.OverviewCredentialCounts{Total: 10}, nil)
	repo.On("UserCounts", mock.Anything, q).Return(&domain.OverviewUserCounts{Total: 5}, nil)
	credRepo.On("Get", mock.Anything, mock.MatchedBy(func(q2 *domainQuery.Query) bool {
		return q2.Limit == 5
	})).Return([]domain.Credential{}, 0, nil)
	userRepo.On("Get", mock.Anything, mock.MatchedBy(func(q2 *domainQuery.Query) bool {
		return q2.Limit == 5
	})).Return([]domain.User{}, 0, nil)

	_, err := svc.Get(ginCtx.Request.Context(), q)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGet_DateFilter(t *testing.T) {
	issuer := &domain.User{Id: "issuer1", Role: domain.RoleIssuer, Email: "issuer@test.com", WalletAddress: "0x1"}
	svc, repo, credRepo, userRepo, ginCtx := setupTestService(t, issuer)

	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "date", Operator: domainQuery.OperatorBetween, Values: []string{"2026-01-01", "2026-06-30"}},
		},
	}

	repo.On("CredentialCounts", mock.Anything, q, (*string)(nil)).Return(&domain.OverviewCredentialCounts{Total: 10}, nil)
	repo.On("UserCounts", mock.Anything, q).Return(&domain.OverviewUserCounts{Total: 5}, nil)
	credRepo.On("Get", mock.Anything, mock.MatchedBy(func(q2 *domainQuery.Query) bool {
		return hasFilter(q2, "issued_at", domainQuery.OperatorBetween)
	})).Return([]domain.Credential{}, 0, nil).Once()
	credRepo.On("Get", mock.Anything, mock.MatchedBy(func(q2 *domainQuery.Query) bool {
		return hasFilter(q2, "revoked_at", domainQuery.OperatorBetween)
	})).Return([]domain.Credential{}, 0, nil).Once()
	userRepo.On("Get", mock.Anything, mock.MatchedBy(func(q2 *domainQuery.Query) bool {
		return hasFilter(q2, "created_at", domainQuery.OperatorBetween)
	})).Return([]domain.User{}, 0, nil)

	_, err := svc.Get(ginCtx.Request.Context(), q)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func hasFilter(q *domainQuery.Query, column string, operator domainQuery.Operator) bool {
	for _, f := range q.Filters {
		if f.Column == column && f.Operator == operator && len(f.Values) == 2 {
			return true
		}
	}
	return false
}

func ptrStr(s string) *string { return &s }

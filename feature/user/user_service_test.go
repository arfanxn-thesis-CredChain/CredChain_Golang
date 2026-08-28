package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/oauth"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/mocks"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/api/idtoken"
)

func mkSvcCfg() *config.Config {
	raw := make([]byte, 32)
	for i := range raw {
		raw[i] = 0xAA
	}
	s := string(raw)
	return &config.Config{JWTSecret: &s, WalletEncryptionKey: &s}
}

func ctxWithAuth(u *domain.User) context.Context {
	return context.WithValue(context.Background(), httpContext.UserKey, u)
}

func TestUserService_Store_PolicyFails(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	policy.On("Store", mock.Anything, mock.Anything).Return(errors.New("policy denied"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Store(ctxWithAuth(&authUser), fixtures.NewDomainUser())
	assert.Error(t, err)
}

func TestUserService_Store_BatchDuplicateEmails(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	policy.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	u1 := fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Store(ctxWithAuth(&authUser), u1, u2)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "users.0.email")
	assert.Contains(t, verrs, "users.1.email")
	for _, ve := range verrs {
		vErr, ok := ve.(validation.Error)
		assert.True(t, ok)
		assert.Equal(t, "validation_store_email_duplicate_batch", vErr.Code())
	}
}

func TestUserService_Store_DBDuplicateEmails(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	policy.On("Store", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return(
		[]domain.User{fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	u1 := fixtures.NewDomainUser(fixtures.WithEmail("new@x.com"))
	u2 := fixtures.NewDomainUser(fixtures.WithEmail("dup@x.com"))
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Store(ctxWithAuth(&authUser), u1, u2)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "users.1.email")
	for _, ve := range verrs {
		vErr, ok := ve.(validation.Error)
		assert.True(t, ok)
		assert.Equal(t, "validation_store_email_duplicate_db", vErr.Code())
	}
}

func TestUserService_Paginate(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	repo.On("Get", mock.Anything, mock.Anything).Return([]domain.User{}, 0, nil)
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UnitRepo: &mocks.MockUserUnitRepository{}, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})

	_, _, err := svc.Paginate(context.Background(), &domainQuery.Query{})
	assert.NoError(t, err)
}

func TestUserService_Paginate_UnitFilterExpandsDescendants(t *testing.T) {
	unitRepo := new(mocks.MockUserUnitRepository)
	unitRepo.On("FindWithDescendants", mock.Anything, "unit-faculty").Return([]domain.UserUnit{
		{Id: "unit-faculty", Name: "Faculty"},
		{Id: "unit-informatics", ParentId: lo.ToPtr("unit-faculty"), Name: "Informatics"},
	}, nil)

	var captured *domainQuery.Query
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.User{}, 0, nil).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domainQuery.Query)
		})

	svc := NewUserService(UserServiceParams{
		UserRepo: userRepo, UnitRepo: unitRepo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})

	q := &domainQuery.Query{Filters: []domainQuery.Filter{
		domainQuery.NewFilter("unit_id", domainQuery.OperatorEqual, "unit-faculty"),
	}}
	_, _, err := svc.Paginate(context.Background(), q)
	require.NoError(t, err)

	require.NotNil(t, captured)
	require.Len(t, captured.Filters, 1)
	assert.Equal(t, domainQuery.OperatorIn, captured.Filters[0].Operator)
	assert.ElementsMatch(t, []string{"unit-faculty", "unit-informatics"}, captured.Filters[0].Values)
}

func TestUserService_Find(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	u := fixtures.NewDomainUser()
	repo.On("Find", mock.Anything, "u1").Return(&u, nil)
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})

	_, err := svc.Find(context.Background(), "u1")
	assert.NoError(t, err)
}

func TestUserService_Find_PropagatesError(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	repo.On("Find", mock.Anything, "missing").Return(nil, errors.New("not found"))
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: nil, Config: mkSvcCfg(),
		Logger: zap.NewNop(), Policy: nil,
	})

	_, err := svc.Find(context.Background(), "missing")
	assert.Error(t, err)
}

func TestUserService_UpdateEmail(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "new@x.com"},
	}, nil)
	repo := &mocks.MockUserRepository{}
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	u := fixtures.NewDomainUser(fixtures.WithEmail("new@x.com"))
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{u}, nil)

	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)

	svc := newEmailSvc(oauthClient, repo, uow)
	got, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.NoError(t, err)
	assert.Equal(t, "new@x.com", got)
}

func TestUserService_Delete_SelfDeleteForbidden(t *testing.T) {
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: NewUserPolicy(),
	})

	authUser := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Delete(ctxWithAuth(&authUser), "self")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserDeleteSelfTargetForbidden, de.Code)
}

func TestUserService_UpdateRole_BlockchainSyncFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("UpdateRolePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdateRolePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("UpdateRole", mock.Anything, mock.Anything).Return([]domain.User{target}, 1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain error"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	_, _, err := svc.UpdateRole(ctxWithAuth(&authUser), domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleBlockchainSyncFailed, de.Code)
}

func TestUserService_UpdateRole_Success_CallsAuthorityService(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("UpdateRolePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdateRolePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("UpdateRole", mock.Anything, mock.Anything).Return([]domain.User{target}, 1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	users, _, err := svc.UpdateRole(ctxWithAuth(&authUser), domain.UserRoleUpdate{UserID: "u1", Role: domain.RoleIssuer})
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Delete_BlockchainRevokeFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("DeletePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("DeletePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Delete", mock.Anything, mock.Anything).Return(1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain error"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	_, err := svc.Delete(ctxWithAuth(&authUser), "u1")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserDeleteBlockchainSyncFailed, de.Code)
}

func TestUserService_Delete_Success_CallsAuthorityServiceWithRoleNone(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("DeletePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("DeletePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Delete", mock.Anything, mock.Anything).Return(1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.MatchedBy(func(users []domain.User) bool {
		return len(users) == 1 && users[0].Role == domain.RoleNone
	})).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	count, err := svc.Delete(ctxWithAuth(&authUser), "u1")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, count)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Delete_AlreadyTrashed_SkipsChainSync(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	trashed := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	trashed.DeletedAt = &deletedAt

	policy.On("DeletePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("DeletePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{trashed}, nil)
	repo.On("Delete", mock.Anything, mock.Anything).Return(0, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	count, err := svc.Delete(ctxWithAuth(&authUser), "u1")
	assert.NoError(t, err)
	assert.EqualValues(t, 0, count, "rowsAffected reflects only newly-deleted rows; already-trashed contributes nothing")
	auth.AssertNotCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Delete_MixedLiveAndTrashed_OnlyLiveSyncsToChain(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	trashed := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder), fixtures.WithWalletAddress("0xAAA"))
	trashed.DeletedAt = &deletedAt
	live := fixtures.NewDomainUser(fixtures.WithID("u2"), fixtures.WithRole(domain.RoleHolder), fixtures.WithWalletAddress("0xBBB"))

	policy.On("DeletePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("DeletePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{trashed, live}, nil)
	repo.On("Delete", mock.Anything, mock.Anything).Return(1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.MatchedBy(func(users []domain.User) bool {
		return len(users) == 1 && users[0].WalletAddress == "0xBBB" && users[0].Role == domain.RoleNone
	})).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	count, err := svc.Delete(ctxWithAuth(&authUser), "u1", "u2")
	assert.NoError(t, err)
	assert.EqualValues(t, 1, count)
	auth.AssertNumberOfCalls(t, "UpdateUserRole", 1)
}

func newEmailSvc(oauthClient oauth.GoogleOAuthClient, repo domain.UserRepository, uow domain.UnitOfWork) UserService {
	if repo == nil {
		repo = &mocks.MockUserRepository{}
	}
	if uow == nil {
		uow = &mocks.MockUnitOfWork{}
	}
	clientID := ""
	walletKey := string(make([]byte, 32))
	jwt := "s"
	return NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: &config.Config{
			JWTSecret:           &jwt,
			WalletEncryptionKey: &walletKey,
			GoogleClientID:      &clientID,
		},
		AuthorityService: &mocks.MockAuthorityService{},
		Logger:           zap.NewNop(),
		Policy:           &mocks.MockUserPolicy{},
		OAuthClient:      oauthClient,
	})
}

func TestUserService_UpdateEmail_InvalidIdToken_Rejects(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "bad-token", mock.Anything).Return(nil, errors.New("invalid"))

	svc := newEmailSvc(oauthClient, nil, nil)
	_, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "bad-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserEmailInvalidIdToken, de.Code)
}

func TestUserService_UpdateEmail_MismatchedEmail_Rejects(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "other@x.com"},
	}, nil)

	svc := newEmailSvc(oauthClient, nil, nil)
	_, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserEmailMismatchedIdToken, de.Code)
}

func TestUserService_UpdateEmail_ConflictWithExistingUser_Rejects(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "new@x.com"},
	}, nil)
	repo := &mocks.MockUserRepository{}
	existing := fixtures.NewDomainUser(fixtures.WithID("other"), fixtures.WithEmail("new@x.com"))
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{existing}, nil)

	svc := newEmailSvc(oauthClient, repo, nil)
	_, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserEmailConflict, de.Code)
}

func TestUserService_UpdateEmail_Success_RevokesRefreshTokens(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", mock.Anything).Return(&idtoken.Payload{
		Claims: map[string]any{"email": "new@x.com"},
	}, nil)
	repo := &mocks.MockUserRepository{}
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	updated := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithEmail("new@x.com"))
	repo.On("Update", mock.Anything, mock.MatchedBy(func(users []domain.User) bool {
		return len(users) == 1 && users[0].Id == "u1" && users[0].Email == "new@x.com"
	})).Return([]domain.User{updated}, nil)

	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)

	svc := newEmailSvc(oauthClient, repo, uow)
	email, err := svc.UpdateEmail(context.Background(), "u1", "new@x.com", "ok-token")
	assert.NoError(t, err)
	assert.Equal(t, "new@x.com", email)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh)
}

func TestUserService_UpdateBatch_PolicyPreFetchFails(t *testing.T) {
	policy := &mocks.MockUserPolicy{}
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(errors.New("denied"))
	svc := NewUserService(UserServiceParams{
		UserRepo: &mocks.MockUserRepository{}, UoW: &mocks.MockUnitOfWork{}, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: policy,
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Update(ctxWithAuth(&auth), domain.User{Id: "u1"})
	assert.Error(t, err)
}

func TestUserService_UpdateBatch_NotFound(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	policy := &mocks.MockUserPolicy{}
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: policy,
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	_, err := svc.Update(ctxWithAuth(&auth), domain.User{Id: "missing"})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateNotFound, de.Code)
}

func TestUserService_UpdateBatch_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	policy := &mocks.MockUserPolicy{}
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	n := "Renamed"

	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	updated := target
	updated.Name = &n
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{updated}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: &mocks.MockAuthorityService{}, Logger: zap.NewNop(), Policy: policy,
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	auth := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	out, err := svc.Update(ctxWithAuth(&auth), domain.User{Id: "u1", Name: &n})
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	assert.Equal(t, "Renamed", *out[0].Name)
}

func mkUpdateSvc(repo *mocks.MockUserRepository, uow domain.UnitOfWork, auth *mocks.MockAuthorityService, policy *mocks.MockUserPolicy) UserService {
	return NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
}

func TestUserService_Update_OnlyProfile_NoChainSync_NoTokenRevoke(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	name := "NewName"
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	out, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Name: &name})
	assert.NoError(t, err)
	assert.Len(t, out, 1)
	auth.AssertNotCalled(t, "UpdateUserRole")
}

func TestUserService_Update_OnlyEmail_TokenRevoked_NoChainSync(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	email := "new@x.com"
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Email: email})
	assert.NoError(t, err)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh)
	auth.AssertNotCalled(t, "UpdateUserRole")
}

func TestUserService_Update_OnlyRole_ChainSync_NoTokenRevoke(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Role: domain.RoleIssuer})
	assert.NoError(t, err)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Update_ProfileAndEmail_TokenRevoked(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	name := "NewName"
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Name: &name, Email: "new@x.com"})
	assert.NoError(t, err)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh)
	auth.AssertNotCalled(t, "UpdateUserRole")
}

func TestUserService_Update_ProfileAndRole_ChainSync(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	name := "NewName"
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Name: &name, Role: domain.RoleIssuer})
	assert.NoError(t, err)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Update_EmailAndRole_BothEffects(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Email: "new@x.com", Role: domain.RoleIssuer})
	assert.NoError(t, err)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Update_AllThree_AllEffects(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	name := "NewName"
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Name: &name, Email: "new@x.com", Role: domain.RoleIssuer})
	assert.NoError(t, err)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Update_SameRole_SilentlySkippedNoChainSync(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Role: domain.RoleHolder})
	assert.NoError(t, err)
	auth.AssertNotCalled(t, "UpdateUserRole")
}

func TestUserService_Update_EmailConflict_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	conflicting := fixtures.NewDomainUser(fixtures.WithID("other"), fixtures.WithEmail("new@x.com"))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{conflicting}, nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Email: "new@x.com"})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserEmailConflict, de.Code)
	repo.AssertNotCalled(t, "Update")
}

func TestUserService_Update_BlockchainSyncFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain error"))
	svc := mkUpdateSvc(repo, uow, auth, policy)
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Role: domain.RoleIssuer})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdateBlockchainSyncFailed, de.Code)
}

func TestUserService_Update_TokenRevokeFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(0, errors.New("revoke error"))
	svc := mkUpdateSvc(repo, uow, auth, policy)
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Email: "new@x.com"})
	assert.Error(t, err)
}

func TestUserService_Update_PolicyPostFetchFails_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("UpdatePostFetch", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("denied"))
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	svc := mkUpdateSvc(repo, uow, auth, policy)
	name := "x"
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Name: &name})
	assert.Error(t, err)
	repo.AssertNotCalled(t, "Update")
}

func TestUserService_Update_AdminPromotingToAdmin_Forbidden(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: NewUserPolicy(),
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Role: domain.RoleAdmin})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSignerAdminRequiredForbidden, de.Code)
}

func TestUserService_Update_AdminUpdatingPeerAdmin_Forbidden(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	authUser := fixtures.NewDomainUser(fixtures.WithID("a1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleAdmin))
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: NewUserPolicy(),
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	name := "x"
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Name: &name})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUpdatePeerAdminForbidden, de.Code)
}

func TestUserService_Update_AssigningSuperAdmin_Forbidden(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: NewUserPolicy(),
		OAuthClient: &mocks.MockGoogleOAuthClient{},
	})
	_, err := svc.Update(ctxWithAuth(&authUser), domain.User{Id: "u1", Role: domain.RoleSuperAdmin})
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRoleSuperAdminBatchForbidden, de.Code)
}

func TestUserService_TransferSuperAdmin_PolicyFails(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	policy.On("TransferSuperAdminPreFetch", mock.Anything, "self-id").
		Return(domain.NewError(domain.CodeUserTransferSuperAdminSelfTargetForbidden))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithID("self-id"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := svc.TransferSuperAdmin(ctxWithAuth(&authUser), "self-id")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserTransferSuperAdminSelfTargetForbidden, de.Code)
}

func TestUserService_TransferSuperAdmin_TargetNotFound(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	policy.On("TransferSuperAdminPreFetch", mock.Anything, "missing-id").Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := svc.TransferSuperAdmin(ctxWithAuth(&authUser), "missing-id")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserTransferSuperAdminTargetNotFound, de.Code)
}

func TestUserService_TransferSuperAdmin_TargetTrashed(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	now := time.Now()
	trashed := fixtures.NewDomainUser(fixtures.WithID("trashed-id"), fixtures.WithRole(domain.RoleAdmin))
	trashed.DeletedAt = &now

	policy.On("TransferSuperAdminPreFetch", mock.Anything, "trashed-id").Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{trashed}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := svc.TransferSuperAdmin(ctxWithAuth(&authUser), "trashed-id")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserTransferSuperAdminTrashedForbidden, de.Code)
}

func TestUserService_TransferSuperAdmin_BlockchainSyncFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	target := fixtures.NewDomainUser(fixtures.WithID("target-id"), fixtures.WithRole(domain.RoleAdmin))

	policy.On("TransferSuperAdminPreFetch", mock.Anything, "target-id").Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	auth.On("TransferSuperAdmin", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain reverted"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleSuperAdmin))
	err := svc.TransferSuperAdmin(ctxWithAuth(&authUser), "target-id")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserTransferSuperAdminBlockchainSyncFailed, de.Code)
	repo.AssertNotCalled(t, "UpdateRole", mock.Anything, mock.Anything)
	tokenRepo.AssertNotCalled(t, "RevokeByUserIdAndType", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_TransferSuperAdmin_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("super-admin-id"), fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("target-id"), fixtures.WithRole(domain.RoleAdmin))

	policy.On("TransferSuperAdminPreFetch", mock.Anything, "target-id").Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	auth.On("TransferSuperAdmin", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateRole", mock.Anything, mock.MatchedBy(func(users []domain.User) bool {
		if len(users) != 2 {
			return false
		}
		idRole := map[string]domain.Role{}
		for _, u := range users {
			idRole[u.Id] = u.Role
		}
		return idRole["super-admin-id"] == domain.RoleAdmin &&
			idRole["target-id"] == domain.RoleSuperAdmin
	})).Return([]domain.User{authUser, target}, 2, nil)
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "super-admin-id", domain.UserTokenTypeRefresh).Return(1, nil)
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "target-id", domain.UserTokenTypeRefresh).Return(1, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	err := svc.TransferSuperAdmin(ctxWithAuth(&authUser), "target-id")
	assert.NoError(t, err)
	auth.AssertCalled(t, "TransferSuperAdmin", mock.Anything, mock.Anything, mock.Anything)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "super-admin-id", domain.UserTokenTypeRefresh)
	tokenRepo.AssertCalled(t, "RevokeByUserIdAndType", mock.Anything, "target-id", domain.UserTokenTypeRefresh)
}

func TestUserService_TransferSuperAdmin_UpdateRoleFailure_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	tokenRepo := &mocks.MockUserTokenRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("super-admin-id"), fixtures.WithRole(domain.RoleSuperAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("target-id"), fixtures.WithRole(domain.RoleAdmin))

	policy.On("TransferSuperAdminPreFetch", mock.Anything, "target-id").Return(nil)
	uow.On("User").Return(repo)
	uow.On("UserToken").Return(tokenRepo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	auth.On("TransferSuperAdmin", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateRole", mock.Anything, mock.Anything).Return(nil, 0, errors.New("db update failed"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})

	err := svc.TransferSuperAdmin(ctxWithAuth(&authUser), "target-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db update failed")
	tokenRepo.AssertNotCalled(t, "RevokeByUserIdAndType", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Restore_Success(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	delTime := time.Now().Add(-1 * time.Hour)
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	target.DeletedAt = &delTime

	policy.On("RestorePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("RestorePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil).Once()
	repo.On("Restore", mock.Anything, mock.Anything).Return(int64(1), nil)
	restored := target
	restored.DeletedAt = nil
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{restored}, nil).Once()
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})
	users, count, err := svc.Restore(ctxWithAuth(&authUser), []string{"u1"})
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.EqualValues(t, 1, count)
	assert.Nil(t, users[0].DeletedAt)
	auth.AssertCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Restore_AdminRestoresAdmin(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	delTime := time.Now().Add(-1 * time.Hour)
	target := fixtures.NewDomainUser(fixtures.WithID("admin2"), fixtures.WithRole(domain.RoleAdmin))
	target.DeletedAt = &delTime

	policy.On("RestorePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("RestorePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil).Once()
	repo.On("Restore", mock.Anything, mock.Anything).Return(int64(1), nil)
	restored := target
	restored.DeletedAt = nil
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{restored}, nil).Once()
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})
	users, count, err := svc.Restore(ctxWithAuth(&authUser), []string{"admin2"})
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.EqualValues(t, 1, count)
	assert.Nil(t, users[0].DeletedAt)
	assert.Equal(t, domain.RoleAdmin, users[0].Role)
}

func TestUserService_Restore_SuperAdminTargetForbidden(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	delTime := time.Now().Add(-1 * time.Hour)
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleSuperAdmin))
	target.DeletedAt = &delTime

	policy.On("RestorePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("RestorePostFetch", mock.Anything, mock.Anything).Return(
		domain.NewError(domain.CodeUserRestoreSuperAdminTargetForbidden,
			domain.WithMetadata("user_id", target.Id)))
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})
	_, _, err := svc.Restore(ctxWithAuth(&authUser), []string{"u1"})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRestoreSuperAdminTargetForbidden, de.Code)
	auth.AssertNotCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Restore_LiveTargetForbidden(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))

	policy.On("RestorePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("RestorePostFetch", mock.Anything, mock.Anything).Return(
		domain.NewError(domain.CodeUserRestoreNotTrashedForbidden,
			domain.WithMetadata("user_id", target.Id)))
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})
	_, _, err := svc.Restore(ctxWithAuth(&authUser), []string{"u1"})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRestoreNotTrashedForbidden, de.Code)
	auth.AssertNotCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Restore_SelfTargetForbidden(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("self"), fixtures.WithRole(domain.RoleAdmin))

	policy.On("RestorePreFetch", mock.Anything, mock.Anything).Return(
		domain.NewError(domain.CodeUserRestoreSelfTargetForbidden,
			domain.WithMetadata("user_id", authUser.Id)))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})
	_, _, err := svc.Restore(ctxWithAuth(&authUser), []string{"self"})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRestoreSelfTargetForbidden, de.Code)
	auth.AssertNotCalled(t, "UpdateUserRole", mock.Anything, mock.Anything, mock.Anything)
}

func TestUserService_Store_InactiveUnitRejected(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	unitRepo := &mocks.MockUserUnitRepository{}
	policy.On("Store", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	unitRepo.On("FindByIds", mock.Anything, mock.MatchedBy(func(ids []string) bool {
		return len(ids) == 1 && ids[0] == "unit-off"
	})).Return([]domain.UserUnit{
		{Id: "unit-off", Name: "Off", Active: false},
	}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
		UnitRepo: unitRepo,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	u := fixtures.NewDomainUser(fixtures.WithUnitID("unit-off"))
	_, err := svc.Store(ctxWithAuth(&authUser), u)
	verrs, ok := err.(validation.Errors)
	require.True(t, ok)
	assert.Contains(t, verrs, "users.0.unit_id")
	for _, ve := range verrs {
		vErr, ok := ve.(validation.Error)
		assert.True(t, ok)
		assert.Equal(t, "validation_unit_inactive", vErr.Code())
	}
}

func TestUserService_Update_InactiveUnitRejected(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := &mocks.MockUnitOfWork{}
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}
	unitRepo := &mocks.MockUserUnitRepository{}
	policy.On("UpdatePreFetch", mock.Anything, mock.Anything).Return(nil)
	unitRepo.On("FindByIds", mock.Anything, mock.MatchedBy(func(ids []string) bool {
		return len(ids) == 1 && ids[0] == "unit-off"
	})).Return([]domain.UserUnit{
		{Id: "unit-off", Name: "Off", Active: false},
	}, nil)

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
		UnitRepo: unitRepo,
	})

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	u := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithUnitID("unit-off"))
	_, err := svc.Update(ctxWithAuth(&authUser), u)
	verrs, ok := err.(validation.Errors)
	require.True(t, ok)
	assert.Contains(t, verrs, "users.0.unit_id")
	for _, ve := range verrs {
		vErr, ok := ve.(validation.Error)
		assert.True(t, ok)
		assert.Equal(t, "validation_unit_inactive", vErr.Code())
	}
}

func TestUserService_Restore_BlockchainSyncFailed_RollsBack(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	auth := &mocks.MockAuthorityService{}
	policy := &mocks.MockUserPolicy{}

	authUser := fixtures.NewDomainUser(fixtures.WithID("admin1"), fixtures.WithRole(domain.RoleAdmin))
	delTime := time.Now().Add(-1 * time.Hour)
	target := fixtures.NewDomainUser(fixtures.WithID("u1"), fixtures.WithRole(domain.RoleHolder))
	target.DeletedAt = &delTime

	policy.On("RestorePreFetch", mock.Anything, mock.Anything).Return(nil)
	policy.On("RestorePostFetch", mock.Anything, mock.Anything).Return(nil)
	uow.On("User").Return(repo)
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{target}, nil)
	repo.On("Restore", mock.Anything, mock.Anything).Return(int64(1), nil)
	restored := target
	restored.DeletedAt = nil
	repo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{restored}, nil).Once()
	auth.On("UpdateUserRole", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("chain error"))

	svc := NewUserService(UserServiceParams{
		UserRepo: repo, UoW: uow, Config: mkSvcCfg(),
		AuthorityService: auth, Logger: zap.NewNop(), Policy: policy,
	})
	_, _, err := svc.Restore(ctxWithAuth(&authUser), []string{"u1"})
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserRestoreBlockchainSyncFailed, de.Code)
}

package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/api/idtoken"
	"gorm.io/gorm"
)

func mkAuthSvcCfg() *config.Config {
	s := "test-secret"
	access := 15
	refresh := 168
	return &config.Config{
		JWTSecret:              &s,
		JWTAccessExpiryMinutes: &access,
		JWTRefreshExpiryHours:  &refresh,
	}
}

func TestAuthService_GoogleLogin_InvalidToken(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "bad-token", "").Return(nil, errors.New("invalid"))

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      &mocks.MockUserRepository{},
		UserTokenRepo: &mocks.MockUserTokenRepository{},
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   oauthClient,
		Config:        mkAuthSvcCfg(),
	})

	_, _, _, err := svc.GoogleLogin(context.Background(), "bad-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthGoogleLoginInvalidToken, de.Code)
}

func TestAuthService_GoogleLogin_EmptyEmail(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", "").Return(&idtoken.Payload{
		Claims: map[string]any{},
	}, nil)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      &mocks.MockUserRepository{},
		UserTokenRepo: &mocks.MockUserTokenRepository{},
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   oauthClient,
		Config:        mkAuthSvcCfg(),
	})

	_, _, _, err := svc.GoogleLogin(context.Background(), "ok-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthGoogleLoginInvalidToken, de.Code)
}

func TestAuthService_GoogleLogin_UserNotFound(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", "").Return(&idtoken.Payload{
		Claims: map[string]any{"email": "missing@x.com"},
	}, nil)

	repo := &mocks.MockUserRepository{}
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return([]domain.User{}, nil)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      repo,
		UserTokenRepo: &mocks.MockUserTokenRepository{},
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   oauthClient,
		Config:        mkAuthSvcCfg(),
	})

	_, _, _, err := svc.GoogleLogin(context.Background(), "ok-token")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthGoogleLoginUserNotFound, de.Code)
}

func TestAuthService_GoogleLogin_Success(t *testing.T) {
	oauthClient := &mocks.MockGoogleOAuthClient{}
	oauthClient.On("Validate", mock.Anything, "ok-token", "").Return(&idtoken.Payload{
		Claims: map[string]any{"email": "alice@x.com"},
	}, nil)

	repo := &mocks.MockUserRepository{}
	repo.On("FindByEmails", mock.Anything, mock.Anything).Return(
		[]domain.User{fixtures.NewDomainUser(fixtures.WithEmail("alice@x.com"))}, nil)

	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, mock.Anything, domain.UserTokenTypeRefresh).Return(0, nil)
	tokenRepo.On("Store", mock.Anything, mock.Anything).Return(
		[]domain.UserToken{fixtures.NewDomainUserToken()}, nil)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      repo,
		UserTokenRepo: tokenRepo,
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   oauthClient,
		Config:        mkAuthSvcCfg(),
	})

	user, token, accessToken, err := svc.GoogleLogin(context.Background(), "ok-token")
	assert.NoError(t, err)
	assert.NotEmpty(t, user.Email)
	assert.NotEmpty(t, token.Token)
	assert.NotEmpty(t, accessToken)
}

func TestAuthService_Refresh_TokenNotFound(t *testing.T) {
	repo := &mocks.MockUserTokenRepository{}
	repo.On("FindByToken", mock.Anything, "nope").Return(nil, gorm.ErrRecordNotFound)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      &mocks.MockUserRepository{},
		UserTokenRepo: repo,
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   &mocks.MockGoogleOAuthClient{},
		Config:        mkAuthSvcCfg(),
	})

	_, _, _, err := svc.Refresh(context.Background(), "nope")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthRefreshInvalidToken, de.Code)
}

func TestAuthService_Refresh_TokenRevoked(t *testing.T) {
	repo := &mocks.MockUserTokenRepository{}
	now := time.Now()
	repo.On("FindByToken", mock.Anything, "revoked").Return(&domain.UserToken{
		RevokedAt: &now,
	}, nil)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      &mocks.MockUserRepository{},
		UserTokenRepo: repo,
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   &mocks.MockGoogleOAuthClient{},
		Config:        mkAuthSvcCfg(),
	})

	_, _, _, err := svc.Refresh(context.Background(), "revoked")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthRefreshTokenRevoked, de.Code)
}

func TestAuthService_Refresh_TokenExpired(t *testing.T) {
	repo := &mocks.MockUserTokenRepository{}
	exp := time.Now().Add(-time.Hour)
	repo.On("FindByToken", mock.Anything, "expired").Return(&domain.UserToken{
		ExpiresAt: &exp,
	}, nil)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      &mocks.MockUserRepository{},
		UserTokenRepo: repo,
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   &mocks.MockGoogleOAuthClient{},
		Config:        mkAuthSvcCfg(),
	})

	_, _, _, err := svc.Refresh(context.Background(), "expired")
	assert.Error(t, err)
	var de *domain.Error
	assert.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeAuthRefreshTokenExpired, de.Code)
}

func TestAuthService_Refresh_Success(t *testing.T) {
	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("FindByToken", mock.Anything, "ok").Return(&domain.UserToken{
		UserId: "u1", ExpiresAt: nil,
	}, nil)

	userRepo := &mocks.MockUserRepository{}
	u := fixtures.NewDomainUser(fixtures.WithID("u1"))
	userRepo.On("Find", mock.Anything, "u1").Return(&u, nil)

	uow := &mocks.MockUnitOfWork{}
	mocks.RunUnitOfWorkFn(uow, uow)
	uow.On("UserToken").Return(tokenRepo)

	tokenRepo.On("Update", mock.Anything, mock.Anything).Return(&domain.UserToken{}, nil)
	tokenRepo.On("Store", mock.Anything, mock.Anything).Return(
		[]domain.UserToken{fixtures.NewDomainUserToken()}, nil)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      userRepo,
		UserTokenRepo: tokenRepo,
		UoW:           uow,
		OAuthClient:   &mocks.MockGoogleOAuthClient{},
		Config:        mkAuthSvcCfg(),
	})

	user, token, accessToken, err := svc.Refresh(context.Background(), "ok")
	assert.NoError(t, err)
	assert.NotEmpty(t, user.Id)
	assert.NotEmpty(t, token.Token)
	assert.NotEmpty(t, accessToken)
}

func TestAuthService_Refresh_DeletedUserRejected(t *testing.T) {
	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("FindByToken", mock.Anything, "ok").Return(&domain.UserToken{UserId: "u1"}, nil)
	userRepo := &mocks.MockUserRepository{}
	u := fixtures.NewDomainUser(fixtures.WithID("u1"))
	now := time.Now()
	u.DeletedAt = &now
	userRepo.On("Find", mock.Anything, "u1").Return(&u, nil)
	uow := &mocks.MockUnitOfWork{}
	// Existing implementation wrongly enters rotation for a deleted user.
	uow.On("Execute", mock.Anything, mock.Anything).Return(errors.New("rotation attempted"))
	svc := NewAuthService(AuthServiceParams{UserRepo: userRepo, UserTokenRepo: tokenRepo, UoW: uow, Config: mkAuthSvcCfg()})
	_, _, _, err := svc.Refresh(context.Background(), "ok")
	var de *domain.Error
	if assert.ErrorAs(t, err, &de) {
		assert.Equal(t, domain.CodeAuthRefreshTokenRevoked, de.Code)
	}
}

func TestAuthService_Refresh_DatabaseFailureIsNotInvalidSession(t *testing.T) {
	repo := &mocks.MockUserTokenRepository{}
	repo.On("FindByToken", mock.Anything, "ok").Return(nil, errors.New("database unavailable"))
	svc := NewAuthService(AuthServiceParams{UserTokenRepo: repo, Config: mkAuthSvcCfg()})
	_, _, _, err := svc.Refresh(context.Background(), "ok")
	var de *domain.Error
	if assert.ErrorAs(t, err, &de) {
		assert.Equal(t, domain.CodeSystemInternal, de.Code)
	}
}

func TestAuthService_LogoutByToken_OnlyActiveCredentialsRevoke(t *testing.T) {
	for _, state := range []string{"active", "revoked", "expired", "missing", "database"} {
		t.Run(state, func(t *testing.T) {
			repo := &mocks.MockUserTokenRepository{}
			token := fixtures.NewDomainUserToken(fixtures.TokenWithUserID("u1"))
			now := time.Now().Add(-time.Minute)
			var lookupErr error
			if state == "revoked" {
				token.RevokedAt = &now
			}
			if state == "expired" {
				token.ExpiresAt = &now
			}
			if state == "missing" {
				lookupErr = gorm.ErrRecordNotFound
			}
			if state == "database" {
				lookupErr = errors.New("database down")
			}
			repo.On("FindByToken", mock.Anything, "token").Return(&token, lookupErr)
			if state == "active" {
				repo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)
			}
			svc := NewAuthService(AuthServiceParams{UserTokenRepo: repo, Config: mkAuthSvcCfg()})
			err := svc.LogoutByToken(context.Background(), "token")
			if state == "database" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestAuthService_Logout_Success(t *testing.T) {
	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(1, nil)

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      &mocks.MockUserRepository{},
		UserTokenRepo: tokenRepo,
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   &mocks.MockGoogleOAuthClient{},
		Config:        mkAuthSvcCfg(),
	})

	err := svc.Logout(context.Background(), "u1")
	assert.NoError(t, err)
}

func TestAuthService_Logout_Error(t *testing.T) {
	tokenRepo := &mocks.MockUserTokenRepository{}
	tokenRepo.On("RevokeByUserIdAndType", mock.Anything, "u1", domain.UserTokenTypeRefresh).Return(0, errors.New("db err"))

	svc := NewAuthService(AuthServiceParams{
		UserRepo:      &mocks.MockUserRepository{},
		UserTokenRepo: tokenRepo,
		UoW:           &mocks.MockUnitOfWork{},
		OAuthClient:   &mocks.MockGoogleOAuthClient{},
		Config:        mkAuthSvcCfg(),
	})

	err := svc.Logout(context.Background(), "u1")
	assert.Error(t, err)
}

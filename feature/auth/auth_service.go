package auth

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/crypto"
	"CredChain_Golang/infrastructure/oauth"
	"CredChain_Golang/infrastructure/security"

	"github.com/oklog/ulid/v2"
	"go.uber.org/fx"
)

type AuthService interface {
	GoogleLogin(ctx context.Context, idToken string) (domain.User, domain.UserToken, string, error)
	Refresh(ctx context.Context, refreshToken string) (domain.User, domain.UserToken, string, error)
	Logout(ctx context.Context, userId string) error
	LogoutByToken(ctx context.Context, refreshToken string) error
}

type authService struct {
	userRepo      domain.UserRepository
	userTokenRepo domain.UserTokenRepository
	uow           domain.UnitOfWork
	oauthClient   oauth.GoogleOAuthClient
	config        *config.Config
}

type AuthServiceParams struct {
	fx.In
	UserRepo      domain.UserRepository
	UserTokenRepo domain.UserTokenRepository
	UoW           domain.UnitOfWork
	OAuthClient   oauth.GoogleOAuthClient
	Config        *config.Config
}

func NewAuthService(p AuthServiceParams) AuthService {
	return &authService{
		userRepo:      p.UserRepo,
		userTokenRepo: p.UserTokenRepo,
		uow:           p.UoW,
		oauthClient:   p.OAuthClient,
		config:        p.Config,
	}
}

func (s *authService) GoogleLogin(ctx context.Context, idToken string) (domain.User, domain.UserToken, string, error) {
	validateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	payload, err := s.oauthClient.Validate(validateCtx, idToken, "")
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginInvalidToken, domain.WithError(err))
	}

	email := oauth.ExtractEmailFromGoogleIdToken(payload)
	if email == "" {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginInvalidToken)
	}

	users, err := s.userRepo.FindByEmails(ctx, email)
	if err != nil || len(users) == 0 {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginUserNotFound)
	}
	user := users[0]

	if user.DeletedAt != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginAccountDeleted)
	}

	_, err = s.userTokenRepo.RevokeByUserIdAndType(ctx, user.Id, domain.UserTokenTypeRefresh)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}

	accessToken, err := security.GenerateJWT(user.Id, []byte(*s.config.JWTSecret), time.Duration(*s.config.JWTAccessExpiryMinutes)*time.Minute)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthGoogleLoginJWTFailed, domain.WithError(err))
	}

	refreshTokenStr := crypto.MustGenerateRandomHex32()
	expiresAt := time.Now().Add(time.Duration(*s.config.JWTRefreshExpiryHours) * time.Hour)

	refreshToken := domain.UserToken{
		Id:        ulid.MustNew(ulid.Now(), nil).String(),
		UserId:    user.Id,
		Type:      domain.UserTokenTypeRefresh,
		Token:     refreshTokenStr,
		ExpiresAt: &expiresAt,
	}
	storedTokens, err := s.userTokenRepo.Store(ctx, refreshToken)
	if err != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}

	return user, storedTokens[0], accessToken, nil
}

func (s *authService) Refresh(ctx context.Context, refreshToken string) (domain.User, domain.UserToken, string, error) {
	token, err := s.userTokenRepo.FindByToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshInvalidToken)
		}
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}

	if token.RevokedAt != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshTokenRevoked)
	}

	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshTokenExpired)
	}

	user, err := s.userRepo.Find(ctx, token.UserId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshUserNotFound)
		}
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}
	if user.DeletedAt != nil {
		return domain.User{}, domain.UserToken{}, "", domain.NewError(domain.CodeAuthRefreshTokenRevoked)
	}

	var newRefreshToken domain.UserToken
	var newAccessToken string

	err = s.uow.Execute(ctx, func(uow domain.UnitOfWork) error {
		now := time.Now()
		token.LastUsedAt = &now
		token.RevokedAt = &now
		_, err = uow.UserToken().Update(ctx, *token)
		if err != nil {
			var de *domain.Error
			if errors.As(err, &de) {
				return err
			}
			return domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
		}

		newAccessToken, err = security.GenerateJWT(user.Id, []byte(*s.config.JWTSecret), time.Duration(*s.config.JWTAccessExpiryMinutes)*time.Minute)
		if err != nil {
			return domain.NewError(domain.CodeAuthRefreshJWTFailed, domain.WithError(err))
		}

		newRefreshTokenStr := crypto.MustGenerateRandomHex32()
		expiresAt := time.Now().Add(time.Duration(*s.config.JWTRefreshExpiryHours) * time.Hour)

		newRefreshToken = domain.UserToken{
			Id:        ulid.MustNew(ulid.Now(), nil).String(),
			UserId:    user.Id,
			Type:      domain.UserTokenTypeRefresh,
			Token:     newRefreshTokenStr,
			ExpiresAt: &expiresAt,
		}
		storedTokens, err := uow.UserToken().Store(ctx, newRefreshToken)
		if err != nil {
			return domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
		}
		newRefreshToken = storedTokens[0]

		return nil
	})

	if err != nil {
		return domain.User{}, domain.UserToken{}, "", err
	}

	return *user, newRefreshToken, newAccessToken, nil
}

func (s *authService) LogoutByToken(ctx context.Context, refreshToken string) error {
	token, err := s.userTokenRepo.FindByToken(ctx, refreshToken)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
	}
	if token.Type != domain.UserTokenTypeRefresh || token.RevokedAt != nil ||
		(token.ExpiresAt != nil && !token.ExpiresAt.After(time.Now())) {
		return nil
	}
	return s.Logout(ctx, token.UserId)
}

func (s *authService) Logout(ctx context.Context, userId string) error {
	_, err := s.userTokenRepo.RevokeByUserIdAndType(ctx, userId, domain.UserTokenTypeRefresh)
	return err
}

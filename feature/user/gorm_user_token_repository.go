package user

import (
	"context"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"

	"gorm.io/gorm"
)

// gormUserTokenRepository implements domain.UserTokenRepository using GORM
type gormUserTokenRepository struct {
	db *gorm.DB
}

// NewGormUserTokenRepository creates a new GORM-based user token repository
func NewGormUserTokenRepository(db *gorm.DB) domain.UserTokenRepository {
	return &gormUserTokenRepository{db: db}
}

// Store creates new user tokens
func (r *gormUserTokenRepository) Store(ctx context.Context, tokens ...domain.UserToken) ([]domain.UserToken, error) {
	gormTokens := make([]model.UserToken, len(tokens))
	for i, token := range tokens {
		gormTokens[i] = model.FromDomainUserToken(token)
	}

	if err := r.db.WithContext(ctx).Create(&gormTokens).Error; err != nil {
		return nil, err
	}

	stored := make([]domain.UserToken, len(gormTokens))
	for i, gormToken := range gormTokens {
		stored[i] = gormToken.ToDomain()
	}
	return stored, nil
}

// FindByToken retrieves a token by its token string
func (r *gormUserTokenRepository) FindByToken(ctx context.Context, token string) (*domain.UserToken, error) {
	var gormToken model.UserToken
	if err := r.db.WithContext(ctx).Where("token = ?", token).First(&gormToken).Error; err != nil {
		return nil, err
	}
	domainToken := gormToken.ToDomain()
	return &domainToken, nil
}

// Find retrieves a token by ID
func (r *gormUserTokenRepository) Find(ctx context.Context, id string) (*domain.UserToken, error) {
	var gormToken model.UserToken
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&gormToken).Error; err != nil {
		return nil, err
	}
	domainToken := gormToken.ToDomain()
	return &domainToken, nil
}

// FindByUserId retrieves all tokens for a user
func (r *gormUserTokenRepository) FindByUserId(ctx context.Context, userId string) ([]domain.UserToken, error) {
	var gormTokens []model.UserToken
	if err := r.db.WithContext(ctx).Where("user_id = ?", userId).Find(&gormTokens).Error; err != nil {
		return nil, err
	}

	tokens := make([]domain.UserToken, len(gormTokens))
	for i, gormToken := range gormTokens {
		tokens[i] = gormToken.ToDomain()
	}
	return tokens, nil
}

// Revoke marks tokens as revoked, returns number of revoked tokens
func (r *gormUserTokenRepository) Revoke(ctx context.Context, tokenIDs ...string) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&model.UserToken{}).
		Where("id IN ?", tokenIDs).
		Update("revoked_at", now)
	return result.RowsAffected, result.Error
}

// RevokeByUserIdAndType revokes all tokens of a specific type for a user
func (r *gormUserTokenRepository) RevokeByUserIdAndType(ctx context.Context, userId string, tokenType domain.UserTokenType) (int64, error) {
	return r.RevokeByUserIdsAndType(ctx, []string{userId}, tokenType)
}

func (r *gormUserTokenRepository) RevokeByUserIdsAndType(ctx context.Context, userIds []string, tokenType domain.UserTokenType) (int64, error) {
	if len(userIds) == 0 {
		return 0, nil
	}
	result := r.db.WithContext(ctx).Model(&model.UserToken{}).
		Where("user_id IN ? AND type = ?", userIds, tokenType).
		Update("revoked_at", time.Now())
	return result.RowsAffected, result.Error
}

// Update updates a single user token
func (r *gormUserTokenRepository) Update(ctx context.Context, token domain.UserToken) (*domain.UserToken, error) {
	modelToken := model.FromDomainUserToken(token)
	// Consume only an active refresh token. A concurrent rotation must not mint a second successor.
	result := r.db.WithContext(ctx).Model(&model.UserToken{}).
		Where("id = ? AND type = ? AND revoked_at IS NULL AND (expires_at IS NULL OR expires_at > ?)", token.Id, domain.UserTokenTypeRefresh, time.Now()).
		Updates(modelToken)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, domain.NewError(domain.CodeAuthRefreshTokenRevoked)
	}
	updated, err := r.Find(ctx, token.Id)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

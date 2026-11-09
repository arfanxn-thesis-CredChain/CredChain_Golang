package mocks

import (
	"context"

	"CredChain_Golang/domain"

	"github.com/stretchr/testify/mock"
)

type MockUserTokenRepository struct {
	mock.Mock
}

func (m *MockUserTokenRepository) Store(ctx context.Context, tokens ...domain.UserToken) ([]domain.UserToken, error) {
	args := m.Called(ctx, tokens)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserToken), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserTokenRepository) Find(ctx context.Context, id string) (*domain.UserToken, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserToken), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserTokenRepository) FindByToken(ctx context.Context, token string) (*domain.UserToken, error) {
	args := m.Called(ctx, token)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserToken), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserTokenRepository) FindByUserId(ctx context.Context, userId string) ([]domain.UserToken, error) {
	args := m.Called(ctx, userId)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserToken), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserTokenRepository) Revoke(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserTokenRepository) RevokeByUserIdAndType(ctx context.Context, userId string, tokenType domain.UserTokenType) (int64, error) {
	args := m.Called(ctx, userId, tokenType)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserTokenRepository) RevokeByUserIdsAndType(ctx context.Context, ids []string, tokenType domain.UserTokenType) (int64, error) {
	args := m.Called(ctx, ids, tokenType)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserTokenRepository) Update(ctx context.Context, token domain.UserToken) (*domain.UserToken, error) {
	args := m.Called(ctx, token)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserToken), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ domain.UserTokenRepository = (*MockUserTokenRepository)(nil)

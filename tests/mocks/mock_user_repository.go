package mocks

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Get(ctx context.Context, q *domainQuery.Query) ([]domain.User, int, error) {
	args := m.Called(ctx, q)
	if v := args.Get(0); v != nil {
		return v.([]domain.User), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

func (m *MockUserRepository) Find(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) FindByEmails(ctx context.Context, emails ...string) ([]domain.User, error) {
	args := m.Called(ctx, emails)
	if v := args.Get(0); v != nil {
		return v.([]domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) FindByRole(ctx context.Context, role domain.Role) ([]domain.User, error) {
	args := m.Called(ctx, role)
	if v := args.Get(0); v != nil {
		return v.([]domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.User, error) {
	args := m.Called(ctx, ids)
	if v := args.Get(0); v != nil {
		return v.([]domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	args := m.Called(ctx, users)
	if v := args.Get(0); v != nil {
		return v.([]domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) Store(ctx context.Context, users ...domain.User) ([]domain.User, error) {
	args := m.Called(ctx, users)
	if v := args.Get(0); v != nil {
		return v.([]domain.User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserRepository) Delete(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserRepository) Restore(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserRepository) UpdateRole(ctx context.Context, users ...domain.User) ([]domain.User, int64, error) {
	args := m.Called(ctx, users)
	if v := args.Get(0); v != nil {
		return v.([]domain.User), int64(args.Int(1)), args.Error(2)
	}
	return nil, int64(args.Int(1)), args.Error(2)
}

func (m *MockUserRepository) CountByUnitIds(ctx context.Context, unitIds ...string) (int64, error) {
	args := m.Called(ctx, unitIds)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.UserRepository = (*MockUserRepository)(nil)

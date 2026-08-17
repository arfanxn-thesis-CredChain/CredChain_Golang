package mocks

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"github.com/stretchr/testify/mock"
)

type MockUserUnitRepository struct {
	mock.Mock
}

func (m *MockUserUnitRepository) Store(ctx context.Context, units ...domain.UserUnit) ([]domain.UserUnit, error) {
	args := m.Called(ctx, units)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUnitRepository) Find(ctx context.Context, id string) (*domain.UserUnit, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUnitRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.UserUnit, error) {
	args := m.Called(ctx, ids)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUnitRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.UserUnit, error) {
	args := m.Called(ctx, query)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUnitRepository) Update(ctx context.Context, units ...domain.UserUnit) ([]domain.UserUnit, error) {
	args := m.Called(ctx, units)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUnitRepository) FindWithDescendants(ctx context.Context, id string) ([]domain.UserUnit, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.([]domain.UserUnit), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockUserUnitRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return int64(args.Int(0)), args.Error(1)
}

func (m *MockUserUnitRepository) CountByParentIds(ctx context.Context, parentIds ...string) (int64, error) {
	args := m.Called(ctx, parentIds)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.UserUnitRepository = (*MockUserUnitRepository)(nil)

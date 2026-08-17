package mocks

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"

	"github.com/stretchr/testify/mock"
)

type MockCredentialRepository struct {
	mock.Mock
}

func (m *MockCredentialRepository) Get(ctx context.Context, q *domainQuery.Query) ([]domain.Credential, int, error) {
	args := m.Called(ctx, q)
	if v := args.Get(0); v != nil {
		return v.([]domain.Credential), args.Int(1), args.Error(2)
	}
	return nil, args.Int(1), args.Error(2)
}

func (m *MockCredentialRepository) Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	args := m.Called(ctx, id, query)
	if v := args.Get(0); v != nil {
		return v.(*domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) FindByHolderId(ctx context.Context, holderID string, query *domainQuery.Query) ([]domain.Credential, error) {
	args := m.Called(ctx, holderID, query)
	if v := args.Get(0); v != nil {
		return v.([]domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) FindByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]domain.Credential, error) {
	args := m.Called(ctx, ids, query)
	if v := args.Get(0); v != nil {
		return v.([]domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) FindVerifiableById(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	args := m.Called(ctx, id, query)
	if v := args.Get(0); v != nil {
		return v.(*domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) FindVerifiableByIds(ctx context.Context, ids []string, query *domainQuery.Query) ([]domain.Credential, error) {
	args := m.Called(ctx, ids, query)
	if v := args.Get(0); v != nil {
		return v.([]domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) FindByFileHashes(ctx context.Context, hashes []string, query *domainQuery.Query) ([]domain.Credential, error) {
	args := m.Called(ctx, hashes, query)
	if v := args.Get(0); v != nil {
		return v.([]domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) Store(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	args := m.Called(ctx, credentials)
	if v := args.Get(0); v != nil {
		return v.([]domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) Update(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	args := m.Called(ctx, credentials)
	if v := args.Get(0); v != nil {
		return v.([]domain.Credential), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockCredentialRepository) CountByTypeIds(ctx context.Context, typeIds ...string) (int64, error) {
	args := m.Called(ctx, typeIds)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCredentialRepository) CountByIssuerOrganizationIds(ctx context.Context, organizationIds ...string) (int64, error) {
	args := m.Called(ctx, organizationIds)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCredentialRepository) CountActiveByFileHashes(ctx context.Context, hashes ...string) (int64, error) {
	args := m.Called(ctx, hashes)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.CredentialRepository = (*MockCredentialRepository)(nil)

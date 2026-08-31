package credential

import (
	"context"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"

	"github.com/stretchr/testify/mock"
)

type mockCredentialService struct{ mock.Mock }

func (m *mockCredentialService) Paginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]domain.Credential), args.Int(1), args.Error(2)
}

func (m *mockCredentialService) Find(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	args := m.Called(ctx, id, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Credential), args.Error(1)
}

func (m *mockCredentialService) SelfPaginate(ctx context.Context, query *domainQuery.Query) ([]domain.Credential, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]domain.Credential), args.Int(1), args.Error(2)
}

func (m *mockCredentialService) SelfFind(ctx context.Context, id string, query *domainQuery.Query) (*domain.Credential, error) {
	args := m.Called(ctx, id, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Credential), args.Error(1)
}

func (m *mockCredentialService) Issue(ctx context.Context, items []CredentialIssuance) ([]domain.Credential, error) {
	args := m.Called(ctx, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *mockCredentialService) Submit(ctx context.Context, items []CredentialSubmission) ([]domain.Credential, error) {
	args := m.Called(ctx, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *mockCredentialService) Revoke(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *mockCredentialService) Verify(ctx context.Context, file pyai.ExtractFile) (int, *domain.Credential, *float64, *string, error) {
	args := m.Called(ctx, file)
	var cred *domain.Credential
	if args.Get(1) != nil {
		cred = args.Get(1).(*domain.Credential)
	}
	var similarity *float64
	if args.Get(2) != nil {
		similarity = args.Get(2).(*float64)
	}
	var field *string
	if args.Get(3) != nil {
		field = args.Get(3).(*string)
	}
	return args.Int(0), cred, similarity, field, args.Error(4)
}

func (m *mockCredentialService) ReExtract(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *mockCredentialService) Approve(ctx context.Context, ids ...string) ([]domain.Credential, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *mockCredentialService) Reject(ctx context.Context, rejections []CredentialRejection) ([]domain.Credential, error) {
	args := m.Called(ctx, rejections)
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *mockCredentialService) Update(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	args := m.Called(ctx, credentials)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Credential), args.Error(1)
}

func (m *mockCredentialService) DownloadFile(ctx context.Context, id string) ([]byte, string, string, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]byte), args.String(1), args.String(2), args.Error(3)
}

func (m *mockCredentialService) LinkCompetencies(ctx context.Context, credentialID string, competencyIDs []string) error {
	args := m.Called(ctx, credentialID, competencyIDs)
	return args.Error(0)
}

func (m *mockCredentialService) ResolveMetadata(ctx context.Context, in CredentialMetadataResolution) (*domain.Credential, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Credential), args.Error(1)
}

func (m *mockCredentialService) SuggestMetadataMatches(ctx context.Context, credentialID string) (*CredentialMetadataSuggestions, error) {
	args := m.Called(ctx, credentialID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CredentialMetadataSuggestions), args.Error(1)
}

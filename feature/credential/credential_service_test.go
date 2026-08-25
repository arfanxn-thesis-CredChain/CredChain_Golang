package credential

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/feature/user"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	"CredChain_Golang/infrastructure/chain/contracts"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/jobs"
	"CredChain_Golang/infrastructure/storage"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/mocks"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type testCredentialMocks struct {
	credRepo *mocks.MockCredentialRepository
	verRepo  *mocks.MockCredentialVerificationRepository
	extRepo  *mocks.MockCredentialExtractionRepository
	aiClient *mocks.MockPythonAIClient
	regSvc   *mocks.MockRegistryService
}

func newTestCredentialService(m *testCredentialMocks) *credentialService {
	return &credentialService{
		repo:             m.credRepo,
		verificationRepo: m.verRepo,
		extractionRepo:   m.extRepo,
		aiClient:         m.aiClient,
		registryService:  m.regSvc,
		policy:           &credentialPolicy{},
		logger:           zap.NewNop(),
	}
}

func testConfig() *config.Config {
	return &config.Config{
		FileEncryptionKey:         lo.ToPtr("12345678901234567890123456789012"),
		CredentialFileStoragePath: lo.ToPtr("credentials"),
		StoragePath:               lo.ToPtr("uploads"),
	}
}

func ctxWithAuth(u *domain.User) context.Context {
	return context.WithValue(context.Background(), httpContext.UserKey, u)
}

// localMockEnqueuer is a test-only mock for jobs.Enqueuer defined inline here
// to avoid an import cycle (jobs imports testutil/mocks for its own tests).
type localMockEnqueuer struct{ mock.Mock }

func (m *localMockEnqueuer) EnqueueExtract(ctx context.Context, args jobs.CredentialExtractArgs) error {
	return m.Called(ctx, args).Error(0)
}

// ── Inline repo mocks for the issue endpoint (B2) ──────────────────────────

type mockCredentialTypeRepository struct{ mock.Mock }

func (m *mockCredentialTypeRepository) Store(ctx context.Context, types ...domain.CredentialType) ([]domain.CredentialType, error) {
	args := m.Called(ctx, types)
	return args.Get(0).([]domain.CredentialType), args.Error(1)
}

func (m *mockCredentialTypeRepository) Find(ctx context.Context, id string) (*domain.CredentialType, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CredentialType), args.Error(1)
}

func (m *mockCredentialTypeRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.CredentialType, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.CredentialType), args.Error(1)
}

// Get returns a zero total: credential_service only ever uses this repo to look
// a row up, never to paginate, so expectations stay two-valued Return(rows, err).
func (m *mockCredentialTypeRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialType, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]domain.CredentialType), 0, args.Error(1)
}

func (m *mockCredentialTypeRepository) Update(ctx context.Context, types ...domain.CredentialType) ([]domain.CredentialType, error) {
	args := m.Called(ctx, types)
	return args.Get(0).([]domain.CredentialType), args.Error(1)
}

func (m *mockCredentialTypeRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.CredentialTypeRepository = (*mockCredentialTypeRepository)(nil)

type mockCredentialIssuerOrganizationRepository struct{ mock.Mock }

func (m *mockCredentialIssuerOrganizationRepository) Store(ctx context.Context, orgs ...domain.CredentialIssuerOrganization) ([]domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, orgs)
	return args.Get(0).([]domain.CredentialIssuerOrganization), args.Error(1)
}

func (m *mockCredentialIssuerOrganizationRepository) Find(ctx context.Context, id string) (*domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CredentialIssuerOrganization), args.Error(1)
}

func (m *mockCredentialIssuerOrganizationRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.CredentialIssuerOrganization), args.Error(1)
}

// Get returns a zero total — see mockCredentialTypeRepository.Get.
func (m *mockCredentialIssuerOrganizationRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.CredentialIssuerOrganization, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]domain.CredentialIssuerOrganization), 0, args.Error(1)
}

func (m *mockCredentialIssuerOrganizationRepository) Update(ctx context.Context, orgs ...domain.CredentialIssuerOrganization) ([]domain.CredentialIssuerOrganization, error) {
	args := m.Called(ctx, orgs)
	return args.Get(0).([]domain.CredentialIssuerOrganization), args.Error(1)
}

func (m *mockCredentialIssuerOrganizationRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.CredentialIssuerOrganizationRepository = (*mockCredentialIssuerOrganizationRepository)(nil)

type mockCompetencyRepository struct{ mock.Mock }

func (m *mockCompetencyRepository) Store(ctx context.Context, competencies ...domain.Competency) ([]domain.Competency, error) {
	args := m.Called(ctx, competencies)
	return args.Get(0).([]domain.Competency), args.Error(1)
}

func (m *mockCompetencyRepository) Find(ctx context.Context, id string) (*domain.Competency, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Competency), args.Error(1)
}

func (m *mockCompetencyRepository) FindByIds(ctx context.Context, ids ...string) ([]domain.Competency, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]domain.Competency), args.Error(1)
}

// Get returns a zero total — see mockCredentialTypeRepository.Get.
func (m *mockCompetencyRepository) Get(ctx context.Context, query *domainQuery.Query) ([]domain.Competency, int, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]domain.Competency), 0, args.Error(1)
}

func (m *mockCompetencyRepository) Update(ctx context.Context, competencies ...domain.Competency) ([]domain.Competency, error) {
	args := m.Called(ctx, competencies)
	return args.Get(0).([]domain.Competency), args.Error(1)
}

func (m *mockCompetencyRepository) Destroy(ctx context.Context, ids ...string) (int64, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.CompetencyRepository = (*mockCompetencyRepository)(nil)

type mockCompetencyCredentialRepository struct{ mock.Mock }

func (m *mockCompetencyCredentialRepository) Store(ctx context.Context, links ...domain.CompetencyCredential) ([]domain.CompetencyCredential, error) {
	args := m.Called(ctx, links)
	return args.Get(0).([]domain.CompetencyCredential), args.Error(1)
}

func (m *mockCompetencyCredentialRepository) Destroy(ctx context.Context, links ...domain.CompetencyCredential) (int64, error) {
	args := m.Called(ctx, links)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCompetencyCredentialRepository) DestroyByCredentialId(ctx context.Context, credentialId string) (int64, error) {
	args := m.Called(ctx, credentialId)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCompetencyCredentialRepository) FindByCredentialId(ctx context.Context, credentialId string) ([]domain.CompetencyCredential, error) {
	args := m.Called(ctx, credentialId)
	return args.Get(0).([]domain.CompetencyCredential), args.Error(1)
}

func (m *mockCompetencyCredentialRepository) FindByCompetencyId(ctx context.Context, competencyId string) ([]domain.CompetencyCredential, error) {
	args := m.Called(ctx, competencyId)
	return args.Get(0).([]domain.CompetencyCredential), args.Error(1)
}

func (m *mockCompetencyCredentialRepository) CountByCompetencyIds(ctx context.Context, competencyIds ...string) (int64, error) {
	args := m.Called(ctx, competencyIds)
	return args.Get(0).(int64), args.Error(1)
}

var _ domain.CompetencyCredentialRepository = (*mockCompetencyCredentialRepository)(nil)

// captureStoreCredentialRepository embeds MockCredentialRepository and records
// every cred passed to Store, returning them unchanged (their pre-generated
// IDs are preserved so competency-link assertions can key off them).
type captureStoreCredentialRepository struct {
	*mocks.MockCredentialRepository
	stored []domain.Credential
}

func (r *captureStoreCredentialRepository) Store(ctx context.Context, credentials ...domain.Credential) ([]domain.Credential, error) {
	r.stored = append(r.stored, credentials...)
	return credentials, nil
}

// newIssueRepos returns three mocks preconfigured for a valid issuance:
// an active type, an existing org, and no competencies in the batch.
func newIssueRepos() (*mockCredentialTypeRepository, *mockCredentialIssuerOrganizationRepository, *mockCompetencyRepository) {
	typeRepo := &mockCredentialTypeRepository{}
	typeRepo.On("Find", mock.Anything, mock.Anything).
		Return(&domain.CredentialType{Id: "type-1", Active: true}, nil)
	orgRepo := &mockCredentialIssuerOrganizationRepository{}
	orgRepo.On("Find", mock.Anything, mock.Anything).
		Return(&domain.CredentialIssuerOrganization{Id: "org-1"}, nil)
	compRepo := &mockCompetencyRepository{}
	compRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.Competency{}, nil)
	return typeRepo, orgRepo, compRepo
}

func TestVerify_CacheHit(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	credID := "01J0000000000000000000000A"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyAuthentic,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, credID, mock.Anything).Return(&domain.Credential{
		ID:     credID,
		Holder: &domain.User{},
		Issuer: &domain.User{},
	}, nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyAuthentic, code)
	assert.NotNil(t, cred)
	assert.Equal(t, credID, cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
	m.extRepo.AssertNotCalled(t, "FindRankedByIds", mock.Anything, mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
}

func TestVerify_CacheHit_RevokedCredential(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	now := time.Now()
	credID := "cred-1"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyAuthentic,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, credID, mock.Anything).Return(&domain.Credential{
		ID:        credID,
		RevokedAt: &now,
		Holder:    &domain.User{},
		Issuer:    &domain.User{},
	}, nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
	assert.NotNil(t, cred)
	assert.Equal(t, credID, cred.ID)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
	m.extRepo.AssertNotCalled(t, "FindRankedByIds", mock.Anything, mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
}

func TestVerify_CacheHit_RevokedOverridesPartyDisabled(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	now := time.Now()
	delTime := time.Now().Add(-1 * time.Hour)
	holder := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	holder.DeletedAt = &delTime
	credID := "cred-1"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyAuthentic,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, credID, mock.Anything).Return(&domain.Credential{
		ID:        credID,
		RevokedAt: &now,
		Holder:    &holder,
		Issuer:    &domain.User{},
	}, nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
	assert.NotNil(t, cred)
	assert.Equal(t, credID, cred.ID)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
	m.extRepo.AssertNotCalled(t, "FindRankedByIds", mock.Anything, mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
}

func TestVerify_CacheHit_NonAuthenticPreserved(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	now := time.Now()
	credID := "cred-1"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyTampered,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, credID, mock.Anything).Return(&domain.Credential{
		ID:        credID,
		RevokedAt: &now,
		Holder:    &domain.User{},
		Issuer:    &domain.User{},
	}, nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
	m.extRepo.AssertNotCalled(t, "FindRankedByIds", mock.Anything, mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
}

func TestVerify_CacheHit_CredentialNotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	credID := "cred-1"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyAuthentic,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, credID, mock.Anything).Return(nil, nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyIntegrityWarning, code)
	assert.Nil(t, cred)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
	m.extRepo.AssertNotCalled(t, "FindRankedByIds", mock.Anything, mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
}

func TestVerify_ExactAuthentic(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f", RevokedAt: nil, Holder: &domain.User{}, Issuer: &domain.User{}, TokenID: lo.ToPtr("12345")},
	}, nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Holder:    common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			Hash:      "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f",
			Issuer:    common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
			Revoker:   common.Address{},
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(0),
			Uri:       "testUri",
		}}, nil,
	)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyAuthentic, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
}

func TestVerify_ExactRevoked(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	now := time.Now()
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f", RevokedAt: &now, TokenID: lo.ToPtr("12345")},
	}, nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Holder:    common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			Hash:      "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f",
			Issuer:    common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
			Revoker:   common.Address{},
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(2000),
			Uri:       "testUri",
		}}, nil,
	)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_ExactIntegrityWarning(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f", RevokedAt: nil, TokenID: lo.ToPtr("12345")},
	}, nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:   big.NewInt(99999),
			Hash: "0xnonmatchinghash00000000000000000000000000000000000000000000000000",
		}}, nil,
	)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyIntegrityWarning, code)
	assert.NotNil(t, cred)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_ExactExpired(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	past := time.Now().Add(-24 * time.Hour)
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f", RevokedAt: nil, ExpiresAt: &past, Holder: &domain.User{}, Issuer: &domain.User{}, TokenID: lo.ToPtr("12345")},
	}, nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Holder:    common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			Hash:      "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f",
			Issuer:    common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
			Revoker:   common.Address{},
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(0),
			Uri:       "testUri",
		}}, nil,
	)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyExpired, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
}

func TestVerify_RevokedBeatsExpired(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	now := time.Now()
	past := now.Add(-24 * time.Hour)
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-1", FileHash: "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f", RevokedAt: &now, ExpiresAt: &past, TokenID: lo.ToPtr("12345")},
	}, nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Holder:    common.HexToAddress("0x1234567890abcdef1234567890abcdef12345678"),
			Hash:      "0x1b2fd4f3ca18fadafcd57a833257bbd533935aa2849e92e34c79387577fc725f",
			Issuer:    common.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12"),
			Revoker:   common.Address{},
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(2000),
			Uri:       "testUri",
		}}, nil,
	)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_CacheHit_ExpiredReEvaluated(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	past := time.Now().Add(-24 * time.Hour)
	credID := "cred-1"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyAuthentic,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, credID, mock.Anything).Return(&domain.Credential{
		ID:        credID,
		ExpiresAt: &past,
		Holder:    &domain.User{},
		Issuer:    &domain.User{},
	}, nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyExpired, code)
	assert.NotNil(t, cred)
	assert.Equal(t, credID, cred.ID)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.aiClient.AssertNotCalled(t, "Verify", mock.Anything, mock.Anything, mock.Anything)
	m.extRepo.AssertNotCalled(t, "FindRankedByIds", mock.Anything, mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
}

func TestVerify_NoExpiryUnchanged(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	credID := "cred-1"
	cached := &domain.CredentialVerification{
		VerdictCode:         domain.CodeCredentialVerifyAuthentic,
		MatchedCredentialID: &credID,
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(cached, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, credID, mock.Anything).Return(&domain.Credential{
		ID:     credID,
		Holder: &domain.User{},
		Issuer: &domain.User{},
	}, nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyAuthentic, code)
	assert.NotNil(t, cred)
	assert.Equal(t, credID, cred.ID)
	m.aiClient.AssertNotCalled(t, "ExtractIDs", mock.Anything, mock.Anything)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
}

func TestVerify_FuzzyNoIdentifiers(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyNoIdentifiers, code)
	assert.Nil(t, cred)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_FuzzyNoMatch(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyNoMatch, code)
	assert.Nil(t, cred)
	assert.Nil(t, score)
	assert.Nil(t, percent)
}

func TestVerify_FuzzyTampered(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.CredentialExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%",
	}, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.NotNil(t, score)
	assert.Equal(t, 0.3, *score)
	assert.NotNil(t, percent)
	assert.Equal(t, "30%", *percent)
}

func TestVerify_FuzzySuspicious(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.CredentialExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "suspicious", SimilarityScore: 0.5, SimilarityPercent: "50%",
	}, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifySuspicious, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.Equal(t, 0.5, *score)
	assert.Equal(t, "50%", *percent)
}

func TestVerify_FuzzyLowSimilarity(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.CredentialExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "low_similarity", SimilarityScore: 0.4, SimilarityPercent: "40%",
	}, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyLowSimilarity, code)
	assert.NotNil(t, cred)
	assert.Equal(t, 0.4, *score)
	assert.Equal(t, "40%", *percent)
}

func TestVerify_FuzzyNotSimilar(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.CredentialExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "not_similar", SimilarityScore: 0.2, SimilarityPercent: "20%",
	}, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyNotSimilar, code)
	assert.NotNil(t, cred)
	assert.Equal(t, 0.2, *score)
	assert.Equal(t, "20%", *percent)
}

func TestVerify_TieBreakNonRevokedPreferred(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "ID1"},
		{Type: "student_id", Value: "ID2"},
	}, nil)

	now := time.Now()
	earlier := now.Add(-24 * time.Hour)

	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-revoked", IDs: []domain.CredentialExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{1.0, 0.0}},
		{CredentialID: "cred-live", IDs: []domain.CredentialExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{2.0, 0.0}},
	}, nil)

	m.credRepo.On("FindVerifiableByIds", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-revoked", RevokedAt: &now, IssuedAt: earlier},
		{ID: "cred-live", RevokedAt: nil, IssuedAt: earlier},
	}, nil)

	var actualEmbedding []float64
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).
		Return(&pyai.VerifyResult{Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%"}, nil).
		Run(func(args mock.Arguments) {
			actualEmbedding = args.Get(2).([]float64)
		})

	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-live", mock.Anything).Return(&domain.Credential{ID: "cred-live"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-live", cred.ID)
	assert.Equal(t, []float64{2.0, 0.0}, actualEmbedding)
}

func TestVerify_TieBreakNewestIssuedAt(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "ID1"},
		{Type: "student_id", Value: "ID2"},
	}, nil)

	now := time.Now()
	earlier := now.Add(-48 * time.Hour)
	newer := now.Add(-24 * time.Hour)

	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-old", IDs: []domain.CredentialExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{1.0, 0.0}},
		{CredentialID: "cred-new", IDs: []domain.CredentialExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{3.0, 0.0}},
	}, nil)

	m.credRepo.On("FindVerifiableByIds", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{
		{ID: "cred-old", RevokedAt: nil, IssuedAt: earlier},
		{ID: "cred-new", RevokedAt: nil, IssuedAt: newer},
	}, nil)

	var actualEmbedding []float64
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).
		Return(&pyai.VerifyResult{Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%"}, nil).
		Run(func(args mock.Arguments) {
			actualEmbedding = args.Get(2).([]float64)
		})

	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-new", mock.Anything).Return(&domain.Credential{ID: "cred-new"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test-file")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-new", cred.ID)
	assert.Equal(t, []float64{3.0, 0.0}, actualEmbedding)
}

func TestIssue_ValidationErrors(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{}, nil)
	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredentialHashStatus{}, nil,
	)
	typeRepo, orgRepo, compRepo := newIssueRepos()
	svc := &credentialService{
		cfg:             testConfig(),
		registryService: regSvc,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
		typeRepo:        typeRepo,
		orgRepo:         orgRepo,
		competencyRepo:  compRepo,
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "a", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "x.pdf", FileBytes: []byte("x")},
		{HolderUserID: "holder-2", Name: "b", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "x.pdf", FileBytes: []byte("y")},
	}
	results, err := svc.Issue(ctx, items)
	assert.Nil(t, results)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok, "expected validation.Errors, got %T", err)
	assert.Contains(t, verrs, "credentials.0.holder_user_id")
	assert.Contains(t, verrs, "credentials.1.holder_user_id")
}

func TestIssue_ChainRollback(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	enq := &localMockEnqueuer{}
	userRepo := &mocks.MockUserRepository{}
	stor := &storage.Storage{Config: &config.Config{StoragePath: lo.ToPtr(t.TempDir())}}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return(
		[]domain.User{{Id: "holder-valid"}}, nil)
	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredentialHashStatus{}, nil,
	)
	regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)
	uow := mocks.NewPropagatingUnitOfWork()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("Store", mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-1", FileURI: lo.ToPtr("up/test.pdf")}}, nil)
	uow.On("Credential").Return(innerCredRepo)
	typeRepo, orgRepo, compRepo := newIssueRepos()
	svc := &credentialService{
		uow:             uow,
		cfg:             testConfig(),
		registryService: regSvc,
		storage:         stor,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
		enqueuer:        enq,
		typeRepo:        typeRepo,
		orgRepo:         orgRepo,
		competencyRepo:  compRepo,
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-valid", Name: "doc", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "x.pdf", FileBytes: []byte("test")},
	}
	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	_, ok := err.(validation.Errors)
	assert.False(t, ok, "expected *domain.Error for chain rollback, not validation.Errors")
}

func TestIssue_DuplicateFileHash(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return(
		[]domain.User{{Id: "holder-1"}, {Id: "holder-2"}}, nil)
	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredentialHashStatus{
			{Status: 1},
			{Status: 0},
		}, nil,
	)
	typeRepo, orgRepo, compRepo := newIssueRepos()
	svc := &credentialService{
		cfg:             testConfig(),
		registryService: regSvc,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
		typeRepo:        typeRepo,
		orgRepo:         orgRepo,
		competencyRepo:  compRepo,
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "a", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "x.pdf", FileBytes: []byte("dup")},
		{HolderUserID: "holder-2", Name: "b", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "x.pdf", FileBytes: []byte("unique")},
	}
	results, err := svc.Issue(ctx, items)
	assert.Nil(t, results)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok, "expected validation.Errors for duplicate hash, got %T", err)
	assert.Contains(t, verrs, "credentials.0.file")
	assert.NotContains(t, verrs, "credentials.1.file")
}

func TestReExtract_HappyPath(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	enq := &localMockEnqueuer{}
	uow := &mocks.MockUnitOfWork{}

	fileURI := "uploads/test.pdf"
	targets := []domain.Credential{
		{ID: "cred-1", ExtractStatus: domain.ExtractStatusFailed, FileURI: &fileURI},
		{ID: "cred-2", ExtractStatus: domain.ExtractStatusFailed, FileURI: &fileURI},
	}
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, mock.Anything, mock.Anything).Return(targets, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything).Return(targets, nil)
	uow.On("Credential").Return(innerCredRepo)
	mocks.RunUnitOfWorkFn(uow, uow)
	enq.On("EnqueueExtract", mock.Anything, mock.Anything).Return(nil)

	svc := &credentialService{
		cfg:      testConfig(),
		uow:      uow,
		policy:   &credentialPolicy{},
		logger:   zap.NewNop(),
		enqueuer: enq,
	}
	updated, err := svc.ReExtract(ctx, "cred-1", "cred-2")
	assert.NoError(t, err)
	assert.Len(t, updated, 2)
	enq.AssertNumberOfCalls(t, "EnqueueExtract", 2)
}

func TestReExtract_NotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	enq := &localMockEnqueuer{}
	uow := mocks.NewPropagatingUnitOfWork()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"cred-1", "cred-2"}, (*domainQuery.Query)(nil)).Return(
		[]domain.Credential{{ID: "cred-1"}}, nil)
	uow.On("Credential").Return(innerCredRepo)

	svc := &credentialService{
		cfg:      testConfig(),
		uow:      uow,
		policy:   &credentialPolicy{},
		logger:   zap.NewNop(),
		enqueuer: enq,
	}
	_, err := svc.ReExtract(ctx, "cred-1", "cred-2")
	assert.Error(t, err)
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialReExtractNotFound, domErr.Code)
	}
	enq.AssertNotCalled(t, "EnqueueExtract", mock.Anything, mock.Anything)
}

func TestReExtract_NotFailed(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	enq := &localMockEnqueuer{}
	uow := mocks.NewPropagatingUnitOfWork()
	fileURI := "uploads/test.pdf"
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"cred-1"}, (*domainQuery.Query)(nil)).Return(
		[]domain.Credential{{ID: "cred-1", ExtractStatus: domain.ExtractStatusSucceeded, FileURI: &fileURI}}, nil)
	uow.On("Credential").Return(innerCredRepo)

	svc := &credentialService{
		cfg:      testConfig(),
		uow:      uow,
		policy:   &credentialPolicy{},
		logger:   zap.NewNop(),
		enqueuer: enq,
	}
	_, err := svc.ReExtract(ctx, "cred-1")
	assert.Error(t, err)
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialReExtractNotEligible, domErr.Code)
	}
	enq.AssertNotCalled(t, "EnqueueExtract", mock.Anything, mock.Anything)
}

func TestFind_NotFound(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "missing", mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	_, err := svc.Find(context.Background(), "missing", nil)
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialFetchNotFound, domErr.Code)
	}
}

func TestFind_HappyPath(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{ID: "cred-1"}, nil)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	got, err := svc.Find(context.Background(), "cred-1", nil)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "cred-1", got.ID)
}

func TestFind_RepoError(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-x", mock.Anything).Return(nil, assert.AnError)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	_, err := svc.Find(context.Background(), "cred-x", nil)
	assert.Error(t, err)
}

func TestRevoke_HappyPath(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	tokID := "1"
	targets := []domain.Credential{{ID: "c1", TokenID: &tokID}}
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return(targets, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(targets, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	svc := &credentialService{
		uow:             uow,
		registryService: regSvc,
		policy:          &credentialPolicy{},
		logger:          zap.NewNop(),
	}
	revoked, err := svc.Revoke(ctx, "c1")
	assert.NoError(t, err)
	assert.Len(t, revoked, 1)
}

func TestRevoke_NotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"missing"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	svc := &credentialService{
		uow:    uow,
		policy: &credentialPolicy{},
		logger: zap.NewNop(),
	}
	_, err := svc.Revoke(ctx, "missing")
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialRevokeNotFound, domErr.Code)
	}
}

func TestRevoke_AlreadyRevoked(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	now := time.Now()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return(
		[]domain.Credential{{ID: "c1", RevokedAt: &now}}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	svc := &credentialService{
		uow:    uow,
		policy: &credentialPolicy{},
		logger: zap.NewNop(),
	}
	_, err := svc.Revoke(ctx, "c1")
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialRevokeAlreadyRevoked, domErr.Code)
	}
}

func TestRevoke_ChainRollback(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	tokID := "1"
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return(
		[]domain.Credential{{ID: "c1", TokenID: &tokID}}, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "c1"}}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(assert.AnError)

	svc := &credentialService{
		uow:             uow,
		registryService: regSvc,
		policy:          &credentialPolicy{},
		logger:          zap.NewNop(),
	}
	_, err := svc.Revoke(ctx, "c1")
	assert.Error(t, err)
}

func TestRevoke_DeletesVerificationCache(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	tokID := "1"
	targets := []domain.Credential{
		{ID: "c1", TokenID: &tokID, FileHash: "0xabc"},
		{ID: "c2", TokenID: &tokID, FileHash: "0xdef"},
	}
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1", "c2"}, (*domainQuery.Query)(nil)).Return(targets, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(targets, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	verRepo := &mocks.MockCredentialVerificationRepository{}
	verRepo.On("DeleteByUploadedFileHashes", mock.Anything, mock.Anything).Return(nil)

	svc := &credentialService{
		uow:              uow,
		registryService:  regSvc,
		policy:           &credentialPolicy{},
		verificationRepo: verRepo,
		logger:           zap.NewNop(),
	}
	revoked, err := svc.Revoke(ctx, "c1", "c2")
	assert.NoError(t, err)
	assert.Len(t, revoked, 2)
	verRepo.AssertCalled(t, "DeleteByUploadedFileHashes", mock.Anything, []string{"0xabc", "0xdef"})
}

func TestRevoke_VerificationCacheDeleteFailureIsNonFatal(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	tokID := "1"
	targets := []domain.Credential{
		{ID: "c1", TokenID: &tokID, FileHash: "0xabc"},
	}
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return(targets, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(targets, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("RevokeCredentials", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	verRepo := &mocks.MockCredentialVerificationRepository{}
	verRepo.On("DeleteByUploadedFileHashes", mock.Anything, mock.Anything).Return(assert.AnError)

	svc := &credentialService{
		uow:              uow,
		registryService:  regSvc,
		policy:           &credentialPolicy{},
		verificationRepo: verRepo,
		logger:           zap.NewNop(),
	}
	revoked, err := svc.Revoke(ctx, "c1")
	assert.NoError(t, err)
	assert.Len(t, revoked, 1)
	verRepo.AssertCalled(t, "DeleteByUploadedFileHashes", mock.Anything, []string{"0xabc"})
}

func TestReExtractCompensate_Success(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "c1"}}, nil)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}
	err := svc.reExtractCompensate(context.Background(), domain.Credential{ID: "c1", ExtractError: lo.ToPtr("orig err")})
	assert.NoError(t, err)
}

func TestCredentialIssuedAtToChain(t *testing.T) {
	assert.Equal(t, uint64(0), credentialIssuedAtToChain(time.Time{}), "zero issued_at writes 0 on chain")

	ts := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, uint64(ts.Unix()), credentialIssuedAtToChain(ts))
}

func TestCredentialExpiresAtToChain(t *testing.T) {
	assert.Equal(t, uint64(0), credentialExpiresAtToChain(nil), "NULL DB expiry writes 0 on chain")

	ts := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, uint64(ts.Unix()), credentialExpiresAtToChain(&ts))
}

func TestSyncBlockchainRevoke_EmptyInput(t *testing.T) {
	svc := &credentialService{logger: zap.NewNop()}
	err := svc.syncBlockchainRevoke(context.Background(), domain.Wallet{}, []string{})
	assert.NoError(t, err)
}

func TestSyncBlockchainRevoke_InvalidTokenID(t *testing.T) {
	svc := &credentialService{logger: zap.NewNop()}
	err := svc.syncBlockchainRevoke(context.Background(), domain.Wallet{}, []string{"not-a-number"})
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialRevokeBlockchainSyncFailed, domErr.Code)
	}
}

func TestSelfPaginate_InjectsHolderFilter(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}}
	var captured *domainQuery.Query
	m.credRepo.On("Get", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			captured = args.Get(1).(*domainQuery.Query)
		}).
		Return([]domain.Credential{{ID: "c1", HolderUserID: "holder-1"}}, 1, nil)

	svc := newTestCredentialService(m)
	creds, total, err := svc.SelfPaginate(ctx, &domainQuery.Query{})

	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, creds, 1)
	if assert.NotNil(t, captured) {
		found := lo.ContainsBy(captured.Filters, func(f domainQuery.Filter) bool {
			return f.Column == "holder_user_id" &&
				f.Operator == domainQuery.OperatorEqual &&
				f.GetValue() == "holder-1"
		})
		assert.True(t, found, "SelfPaginate must inject holder_user_id filter scoped to the auth user")
	}
}

func TestSelfPaginate_NilQuery(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithID("holder-2"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}}
	m.credRepo.On("Get", mock.Anything, mock.Anything).
		Return([]domain.Credential{}, 0, nil)

	svc := newTestCredentialService(m)
	_, _, err := svc.SelfPaginate(ctx, nil)
	assert.NoError(t, err)
}

func TestSelfFind_OwnedReturnsCredential(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}}
	m.credRepo.On("Find", mock.Anything, "c1", mock.Anything).
		Return(&domain.Credential{ID: "c1", HolderUserID: "holder-1"}, nil)

	svc := newTestCredentialService(m)
	cred, err := svc.SelfFind(ctx, "c1", &domainQuery.Query{})
	assert.NoError(t, err)
	if assert.NotNil(t, cred) {
		assert.Equal(t, "c1", cred.ID)
	}
}

func TestSelfFind_NotOwnedReturnsNotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}}
	m.credRepo.On("Find", mock.Anything, "c2", mock.Anything).
		Return(&domain.Credential{ID: "c2", HolderUserID: "other-holder"}, nil)

	svc := newTestCredentialService(m)
	cred, err := svc.SelfFind(ctx, "c2", &domainQuery.Query{})
	assert.Nil(t, cred)
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialFetchNotFound, domErr.Code,
			"ownership mismatch must be reported as 404, not leaked")
	}
}

func TestSelfFind_MissingReturnsNotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}}
	m.credRepo.On("Find", mock.Anything, "missing", mock.Anything).
		Return((*domain.Credential)(nil), gorm.ErrRecordNotFound)

	svc := newTestCredentialService(m)
	cred, err := svc.SelfFind(ctx, "missing", &domainQuery.Query{})
	assert.Nil(t, cred)
	var domErr *domain.Error
	if assert.ErrorAs(t, err, &domErr) {
		assert.Equal(t, domain.CodeCredentialFetchNotFound, domErr.Code)
	}
}

func TestVerify_HolderDisabled_OverridesAuthentic(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	delTime := time.Now().Add(-1 * time.Hour)
	holder := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	holder.DeletedAt = &delTime
	issuer := fixtures.NewDomainUser(fixtures.WithID("issuer-1"), fixtures.WithRole(domain.RoleIssuer))

	cred := domain.Credential{
		ID:           "c1",
		HolderUserID: "holder-1",
		IssuerUserID: "issuer-1",
		Holder:       &holder,
		Issuer:       &issuer,
	}
	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return((*domain.CredentialVerification)(nil), nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Hash:      "0x9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658",
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(0),
		}}, nil,
	)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{func() domain.Credential {
		c := cred
		c.TokenID = lo.ToPtr("12345")
		return c
	}()}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyHolderDisabled, code)
}

func TestVerify_IssuerDisabled_OverridesAuthentic(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	delTime := time.Now().Add(-1 * time.Hour)
	holder := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	issuer := fixtures.NewDomainUser(fixtures.WithID("issuer-1"), fixtures.WithRole(domain.RoleIssuer))
	issuer.DeletedAt = &delTime

	cred := domain.Credential{
		ID:           "c1",
		HolderUserID: "holder-1",
		IssuerUserID: "issuer-1",
		Holder:       &holder,
		Issuer:       &issuer,
	}
	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return((*domain.CredentialVerification)(nil), nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Hash:      "0x9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658",
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(0),
		}}, nil,
	)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{func() domain.Credential {
		c := cred
		c.TokenID = lo.ToPtr("12345")
		return c
	}()}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyIssuerDisabled, code)
}

func TestVerify_PartyDisabled_BothDeleted(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	delTime := time.Now().Add(-1 * time.Hour)
	holder := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	holder.DeletedAt = &delTime
	issuer := fixtures.NewDomainUser(fixtures.WithID("issuer-1"), fixtures.WithRole(domain.RoleIssuer))
	issuer.DeletedAt = &delTime

	cred := domain.Credential{
		ID:           "c1",
		HolderUserID: "holder-1",
		IssuerUserID: "issuer-1",
		Holder:       &holder,
		Issuer:       &issuer,
	}
	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return((*domain.CredentialVerification)(nil), nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Hash:      "0x9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658",
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(0),
		}}, nil,
	)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{func() domain.Credential {
		c := cred
		c.TokenID = lo.ToPtr("12345")
		return c
	}()}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyPartyDisabled, code)
}

func TestVerify_DoesNotOverrideRevoked_WhenHolderDeleted(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	delTime := time.Now().Add(-1 * time.Hour)
	holder := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	holder.DeletedAt = &delTime
	issuer := fixtures.NewDomainUser(fixtures.WithID("issuer-1"), fixtures.WithRole(domain.RoleIssuer))

	now := time.Now()
	cred := domain.Credential{
		ID:           "c1",
		HolderUserID: "holder-1",
		IssuerUserID: "issuer-1",
		Holder:       &holder,
		Issuer:       &issuer,
		RevokedAt:    &now,
	}
	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return((*domain.CredentialVerification)(nil), nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Hash:      "0x9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658",
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(2000),
		}}, nil,
	)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{func() domain.Credential {
		c := cred
		c.TokenID = lo.ToPtr("12345")
		return c
	}()}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyRevoked, code)
}

func TestVerify_PartyDisabled_MissingHolderTreatedAsDisabled(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	issuer := fixtures.NewDomainUser(fixtures.WithID("issuer-1"), fixtures.WithRole(domain.RoleIssuer))
	cred := domain.Credential{
		ID:           "c1",
		HolderUserID: "holder-1",
		IssuerUserID: "issuer-1",
		Holder:       nil,
		Issuer:       &issuer,
	}
	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return((*domain.CredentialVerification)(nil), nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return(
		[]contracts.CredentialRegistryCredential{{
			Id:        big.NewInt(12345),
			Hash:      "0x9c22ff5f21f0b81b113e63f7db6da94fedef11b2119b4088b89664fb9a3cb658",
			IssuedAt:  big.NewInt(1000),
			RevokedAt: big.NewInt(0),
		}}, nil,
	)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{func() domain.Credential {
		c := cred
		c.TokenID = lo.ToPtr("12345")
		return c
	}()}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, _, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyHolderDisabled, code)
}

func TestVerify_DoesNotOverrideTampered_WhenHolderDeleted(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	delTime := time.Now().Add(-1 * time.Hour)
	holder := fixtures.NewDomainUser(fixtures.WithID("holder-1"), fixtures.WithRole(domain.RoleHolder))
	holder.DeletedAt = &delTime
	issuer := fixtures.NewDomainUser(fixtures.WithID("issuer-1"), fixtures.WithRole(domain.RoleIssuer))

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return((*domain.CredentialVerification)(nil), nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "12345"},
	}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-1", IDs: []domain.CredentialExtractedID{{Value: "12345"}}, Embedding: []float64{0.1, 0.2}},
	}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{
		Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%",
	}, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-1", mock.Anything).Return(&domain.Credential{
		ID:           "cred-1",
		HolderUserID: "holder-1",
		IssuerUserID: "issuer-1",
		Holder:       &holder,
		Issuer:       &issuer,
	}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, score, percent, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})
	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-1", cred.ID)
	assert.NotNil(t, score)
	assert.Equal(t, 0.3, *score)
	assert.NotNil(t, percent)
	assert.Equal(t, "30%", *percent)
}

func TestVerify_Fuzzy_TieBreak_FindByIdsErrorFallsBack(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	m := &testCredentialMocks{
		credRepo: &mocks.MockCredentialRepository{},
		verRepo:  &mocks.MockCredentialVerificationRepository{},
		extRepo:  &mocks.MockCredentialExtractionRepository{},
		aiClient: &mocks.MockPythonAIClient{},
		regSvc:   &mocks.MockRegistryService{},
	}

	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{
		{Type: "student_id", Value: "ID1"},
		{Type: "student_id", Value: "ID2"},
	}, nil)

	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{
		{CredentialID: "cred-a", IDs: []domain.CredentialExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{1.0, 0.0}},
		{CredentialID: "cred-b", IDs: []domain.CredentialExtractedID{{Value: "ID1"}, {Value: "ID2"}}, Embedding: []float64{2.0, 0.0}},
	}, nil)

	// FindVerifiableByIds errors → verifyPickBestMatch should fall back to tied[0] without crashing
	m.credRepo.On("FindVerifiableByIds", mock.Anything, mock.Anything, mock.Anything).Return(nil, assert.AnError)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, []float64{1.0, 0.0}).Return(
		&pyai.VerifyResult{Verdict: "tampered", SimilarityScore: 0.3, SimilarityPercent: "30%"}, nil,
	)
	m.credRepo.On("FindVerifiableById", mock.Anything, "cred-a", mock.Anything).Return(&domain.Credential{ID: "cred-a"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)

	svc := newTestCredentialService(m)
	code, cred, _, _, err := svc.Verify(ctx, pyai.ExtractFile{Data: []byte("test")})

	assert.NoError(t, err)
	assert.Equal(t, domain.CodeCredentialVerifyTampered, code)
	assert.NotNil(t, cred)
	assert.Equal(t, "cred-a", cred.ID)
	m.credRepo.AssertCalled(t, "FindVerifiableByIds", mock.Anything, mock.Anything, mock.Anything)
	m.aiClient.AssertCalled(t, "Verify", mock.Anything, mock.Anything, []float64{1.0, 0.0})
}

func TestVerifyCacheVerdict_StoreFails(t *testing.T) {
	verRepo := &mocks.MockCredentialVerificationRepository{}
	verRepo.On("Store", mock.Anything, mock.Anything).Return(assert.AnError)
	svc := &credentialService{verificationRepo: verRepo, logger: zap.NewNop()}
	svc.verifyCacheVerdict(context.Background(), "0xhash", domain.CodeCredentialVerifyNoMatch, nil, nil, nil)
	verRepo.AssertCalled(t, "Store", mock.Anything, mock.Anything)
}

func TestIssue_GlobalDuplicateHash_Batch(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holderA := fixtures.NewDomainUser(
		fixtures.WithID("hA"),
		fixtures.WithWalletAddress("0x"+"a1"),
	)
	holderB := fixtures.NewDomainUser(
		fixtures.WithID("hB"),
		fixtures.WithWalletAddress("0x"+"b2"),
	)
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{
			{Status: 0}, {Status: 0},
		}, nil)

	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).
		Return([]domain.User{holderA, holderB}, nil)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	typeRepo, orgRepo, compRepo := newIssueRepos()
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	data := []byte("a")
	items := []CredentialIssuance{
		{HolderUserID: "hA", Name: "C1", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: data},
		{HolderUserID: "hB", Name: "C2", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: data},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.1.file")
}

func TestIssue_GlobalDuplicateHash_OnChain(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(
		fixtures.WithID("h"),
		fixtures.WithWalletAddress("0x"+"c1"),
	)
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{
			{Status: 1},
		}, nil)

	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).
		Return([]domain.User{holder}, nil)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	typeRepo, orgRepo, compRepo := newIssueRepos()
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.file")
}

func TestIssue_RevokedHash_Allowed(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(
		fixtures.WithID("h"),
		fixtures.WithWalletAddress("0x"+"d1"),
	)
	ctx := ctxWithAuth(&issuer)
	enq := &localMockEnqueuer{}
	enq.On("EnqueueExtract", mock.Anything, mock.Anything).Return(nil)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{
			{Status: 2},
		}, nil)
	regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return([]*big.Int{big.NewInt(1)}, nil)

	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).
		Return([]domain.User{holder}, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	innerCredRepo := &mocks.MockCredentialRepository{}
	innerCredRepo.On("Store", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-1", FileURI: lo.ToPtr("up/test.pdf")}}, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-1", TokenID: lo.ToPtr("1")}}, nil)
	uow.On("Credential").Return(innerCredRepo)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	typeRepo, orgRepo, compRepo := newIssueRepos()
	svc := newTestCredentialService(m)
	svc.uow = uow
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.storage = &storage.Storage{Config: &config.Config{StoragePath: lo.ToPtr(t.TempDir())}}
	svc.enqueuer = enq
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("y")},
	}

	_, err := svc.Issue(ctx, items)
	assert.NoError(t, err)
}

func TestIssue_HolderNotFound(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)

	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).
		Return([]domain.User{}, nil)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	typeRepo, orgRepo, compRepo := newIssueRepos()
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	items := []CredentialIssuance{
		{HolderUserID: "ghost", Name: "C", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("z")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.holder_user_id")
}

func TestIssue_TypeNotFound(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(fixtures.WithID("h"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{holder}, nil)

	typeRepo := &mockCredentialTypeRepository{}
	typeRepo.On("Find", mock.Anything, "type-missing").Return((*domain.CredentialType)(nil), gorm.ErrRecordNotFound)
	_, orgRepo, compRepo := newIssueRepos()

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", TypeID: "type-missing", IssuerOrganizationID: "org-1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.type_id")
}

func TestIssue_TypeInactive(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(fixtures.WithID("h"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{holder}, nil)

	typeRepo := &mockCredentialTypeRepository{}
	typeRepo.On("Find", mock.Anything, "type-1").Return(&domain.CredentialType{Id: "type-1", Active: false}, nil)
	_, orgRepo, compRepo := newIssueRepos()

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", TypeID: "type-1", IssuerOrganizationID: "org-1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.type_id")
}

func TestIssue_OrgNotFound(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(fixtures.WithID("h"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{holder}, nil)

	typeRepo, _, compRepo := newIssueRepos()
	orgRepo := &mockCredentialIssuerOrganizationRepository{}
	orgRepo.On("Find", mock.Anything, "org-missing").Return((*domain.CredentialIssuerOrganization)(nil), gorm.ErrRecordNotFound)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", TypeID: "type-1", IssuerOrganizationID: "org-missing", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.issuer_organization_id")
}

func TestIssue_NumberDuplicate(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(fixtures.WithID("h"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{holder}, nil)

	typeRepo, orgRepo, compRepo := newIssueRepos()
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Get", mock.Anything, mock.Anything).
		Return([]domain.Credential{{ID: "existing", Number: lo.ToPtr("N-001")}}, 1, nil)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: credRepo}
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	colliding := "N-001"
	free := "N-002"
	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C1", TypeID: "type-1", IssuerOrganizationID: "org-1", Number: &colliding, Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
		{HolderUserID: "h", Name: "C2", TypeID: "type-1", IssuerOrganizationID: "org-1", Number: &free, Filename: "b.pdf", MIMEType: "application/pdf", FileBytes: []byte("y")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.number")
	assert.NotContains(t, verrs, "credentials.1.number")
	credRepo.AssertNumberOfCalls(t, "Get", 1)
}

func TestIssue_NumberDuplicate_ScopedPerOrg(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(fixtures.WithID("h"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{holder}, nil)

	typeRepo, orgRepo, compRepo := newIssueRepos()
	orgFilter := func(q *domainQuery.Query) string {
		for _, f := range q.Filters {
			if f.Column == "issuer_organization_id" {
				return f.GetValue()
			}
		}
		return ""
	}
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Get", mock.Anything, mock.MatchedBy(func(q *domainQuery.Query) bool {
		return orgFilter(q) == "org-a"
	})).Return([]domain.Credential{{ID: "existing-a", Number: lo.ToPtr("N-001")}}, 1, nil)
	credRepo.On("Get", mock.Anything, mock.MatchedBy(func(q *domainQuery.Query) bool {
		return orgFilter(q) == "org-b"
	})).Return([]domain.Credential{}, 0, nil)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: credRepo}
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	number := "N-001"
	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C1", TypeID: "type-1", IssuerOrganizationID: "org-a", Number: &number, Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
		{HolderUserID: "h", Name: "C2", TypeID: "type-1", IssuerOrganizationID: "org-b", Number: &number, Filename: "b.pdf", MIMEType: "application/pdf", FileBytes: []byte("y")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.number", "org-a item with existing N-001 must be flagged")
	assert.NotContains(t, verrs, "credentials.1.number", "org-b item with same number must NOT be flagged")
	credRepo.AssertNumberOfCalls(t, "Get", 2)
}

func TestIssue_CompetencyNotFound(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(fixtures.WithID("h"), fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&issuer)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{holder}, nil)

	typeRepo, orgRepo, _ := newIssueRepos()
	compRepo := &mockCompetencyRepository{}
	compRepo.On("FindByIds", mock.Anything, mock.Anything).
		Return([]domain.Competency{{Id: "comp-b"}}, nil)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: &mocks.MockCredentialRepository{}}
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", TypeID: "type-1", IssuerOrganizationID: "org-1", CompetencyIDs: []string{"comp-a", "comp-b"}, Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.competency_ids")
}

func TestIssue_SetsSubmitterApproverAndCompetencyLinks(t *testing.T) {
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	holder := fixtures.NewDomainUser(
		fixtures.WithID("h"),
		fixtures.WithRole(domain.RoleHolder),
		fixtures.WithWalletAddress("0x"+"a1b2"),
	)
	ctx := ctxWithAuth(&issuer)

	enq := &localMockEnqueuer{}
	enq.On("EnqueueExtract", mock.Anything, mock.Anything).Return(nil)

	regSvc := &mocks.MockRegistryService{}
	regSvc.On("GetCredentialHashStatuses", mock.Anything, mock.Anything).
		Return([]contracts.CredentialRegistryCredentialHashStatus{{Status: 0}}, nil)
	regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return([]*big.Int{big.NewInt(1)}, nil)

	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, mock.Anything).Return([]domain.User{holder}, nil)

	typeRepo, orgRepo, _ := newIssueRepos()
	compRepo := &mockCompetencyRepository{}
	compRepo.On("FindByIds", mock.Anything, mock.Anything).
		Return([]domain.Competency{{Id: "comp-a"}, {Id: "comp-b"}}, nil)

	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.Credential{}, 0, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	captureRepo := &captureStoreCredentialRepository{MockCredentialRepository: &mocks.MockCredentialRepository{}}
	captureRepo.On("Update", mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "stored-1", TokenID: lo.ToPtr("1")}}, nil)
	uow.On("Credential").Return(captureRepo)

	var storedLinks []domain.CompetencyCredential
	compCredRepo := &mockCompetencyCredentialRepository{}
	compCredRepo.On("Store", mock.Anything, mock.Anything).
		Return([]domain.CompetencyCredential{}, nil).
		Run(func(args mock.Arguments) {
			storedLinks = append(storedLinks, args.Get(1).([]domain.CompetencyCredential)...)
		})
	uow.On("CompetencyCredential").Return(compCredRepo)

	m := &testCredentialMocks{regSvc: regSvc, credRepo: credRepo}
	svc := newTestCredentialService(m)
	svc.uow = uow
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.storage = &storage.Storage{Config: &config.Config{StoragePath: lo.ToPtr(t.TempDir())}}
	svc.enqueuer = enq
	svc.typeRepo = typeRepo
	svc.orgRepo = orgRepo
	svc.competencyRepo = compRepo

	number := "N-001"
	items := []CredentialIssuance{
		{
			HolderUserID:         "h",
			Name:                 "C",
			TypeID:               "type-1",
			IssuerOrganizationID: "org-1",
			Number:               &number,
			CompetencyIDs:        []string{"comp-a", "comp-b"},
			Filename:             "a.pdf",
			MIMEType:             "application/pdf",
			FileBytes:            []byte("payload"),
		},
	}

	created, err := svc.Issue(ctx, items)
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.NotNil(t, created[0].TokenID, "issue must return the on-chain token id")
	assert.Equal(t, "1", *created[0].TokenID)

	if assert.NotEmpty(t, captureRepo.stored) {
		c := captureRepo.stored[0]
		assert.Equal(t, "h", c.HolderUserID)
		assert.Equal(t, issuer.Id, c.SubmitterUserID)
		assert.Equal(t, issuer.Id, c.IssuerUserID)
		assert.Equal(t, c.SubmitterUserID, c.IssuerUserID)
		assert.Equal(t, "type-1", c.TypeID)
		assert.Equal(t, "org-1", c.IssuerOrganizationID)
		assert.Equal(t, "N-001", *c.Number)
		assert.NotNil(t, c.ApproverUserID)
		assert.Equal(t, issuer.Id, *c.ApproverUserID)
		assert.NotNil(t, c.ApprovedAt)
	}
	if assert.NotEmpty(t, captureRepo.stored) {
		credID := captureRepo.stored[0].ID
		assert.Equal(t, []domain.CompetencyCredential{
			{CompetencyId: "comp-a", CredentialId: credID},
			{CompetencyId: "comp-b", CredentialId: credID},
		}, storedLinks)
	}
	compCredRepo.AssertCalled(t, "Store", mock.Anything, mock.Anything)
}

// ── Submit (SQLite-backed tests) ──────────────────────────────────────────

// newCredentialServiceWithSQLite builds a *credentialService over real GORM
// repositories and a real GormUnitOfWork backed by in-memory SQLite, seeded
// with an active credential type (id "t1") and an issuer organization (id "o1").
// The partial unique index and CountActiveByFileHashes therefore behave like
// production.
func newCredentialServiceWithSQLite(t *testing.T) (*credentialService, *gormCredentialRepository) {
	t.Helper()
	d := db.OpenInMemorySQLite(t)
	credRepo := NewGormCredentialRepository(d).(*gormCredentialRepository)
	typeRepo := NewGormCredentialTypeRepository(d)
	orgRepo := NewGormCredentialIssuerOrganizationRepository(d)
	compRepo := NewGormCompetencyRepository(d)
	uow := gormInfra.NewGormUnitOfWork(d,
		user.NewGormUserRepository,
		NewGormCredentialRepository,
		user.NewGormUserTokenRepository,
		NewGormCompetencyCredentialRepository,
	)

	ctx := context.Background()
	if _, err := typeRepo.Store(ctx, domain.CredentialType{Id: "t1", Name: "Degree", Active: true}); err != nil {
		t.Fatalf("seed type: %v", err)
	}
	if _, err := orgRepo.Store(ctx, domain.CredentialIssuerOrganization{Id: "o1", Name: "UI"}); err != nil {
		t.Fatalf("seed org: %v", err)
	}

	cfg := &config.Config{
		FileEncryptionKey:         lo.ToPtr("12345678901234567890123456789012"),
		CredentialFileStoragePath: lo.ToPtr("credentials"),
		StoragePath:               lo.ToPtr(t.TempDir()),
	}
	svc := &credentialService{
		repo:           credRepo,
		uow:            uow,
		cfg:            cfg,
		storage:        &storage.Storage{Config: cfg},
		policy:         &credentialPolicy{},
		typeRepo:       typeRepo,
		orgRepo:        orgRepo,
		competencyRepo: compRepo,
		logger:         zap.NewNop(),
	}
	return svc, credRepo
}

func TestSubmit_DuplicateFileRejectedAtSubmission(t *testing.T) {
	svc, _ := newCredentialServiceWithSQLite(t)

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&authUser)

	issuedAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	first := CredentialSubmission{
		Name: "Degree", TypeID: "t1", IssuerOrganizationID: "o1",
		IssuedAt: &issuedAt, FileBytes: []byte("same-bytes"),
		Filename: "a.pdf", MIMEType: "application/pdf",
	}
	_, err := svc.Submit(ctx, []CredentialSubmission{first})
	require.NoError(t, err)

	_, err = svc.Submit(ctx, []CredentialSubmission{first})
	var verrs validation.Errors
	require.ErrorAs(t, err, &verrs)
	assert.Contains(t, verrs, "credentials.0.file", "duplicate file must be rejected at submission, not approval")
}

func TestSubmit_SetsUnextractedExtractStatus(t *testing.T) {
	svc, repo := newCredentialServiceWithSQLite(t)

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&authUser)

	issuedAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	submitted, err := svc.Submit(ctx, []CredentialSubmission{
		{
			Name: "Degree", TypeID: "t1", IssuerOrganizationID: "o1",
			IssuedAt: &issuedAt, FileBytes: []byte("b"),
			Filename: "a.pdf", MIMEType: "application/pdf",
		},
	})
	require.NoError(t, err)
	require.Len(t, submitted, 1)

	stored, err := repo.Find(ctx, submitted[0].ID, nil)
	require.NoError(t, err)
	assert.Equal(t, domain.ExtractStatusUnextracted, stored.ExtractStatus)
	assert.Nil(t, stored.ApprovedAt)
	assert.Equal(t, authUser.Id, stored.SubmitterUserID)
	assert.Equal(t, stored.HolderUserID, stored.SubmitterUserID)
	assert.Equal(t, issuedAt.UTC(), stored.IssuedAt.UTC())
	assert.Nil(t, stored.TokenID)
}

func TestSubmit_HappyPathStoresPendingCredential(t *testing.T) {
	svc, repo := newCredentialServiceWithSQLite(t)

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&authUser)

	number := "N-001"
	issuedAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	submitted, err := svc.Submit(ctx, []CredentialSubmission{
		{
			Name: "Degree", TypeID: "t1", IssuerOrganizationID: "o1",
			Number: &number, IssuedAt: &issuedAt, FileBytes: []byte("c"),
			Filename: "a.pdf", MIMEType: "application/pdf",
		},
	})
	require.NoError(t, err)
	require.Len(t, submitted, 1)

	stored, err := repo.Find(ctx, submitted[0].ID, nil)
	require.NoError(t, err)
	assert.Equal(t, "N-001", *stored.Number)
	assert.Equal(t, domain.CredentialLifecycleStatusPending, stored.LifecycleStatus())
}

func TestSubmit_TypeNotFound(t *testing.T) {
	svc, _ := newCredentialServiceWithSQLite(t)

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&authUser)

	issuedAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	_, err := svc.Submit(ctx, []CredentialSubmission{
		{
			Name: "Degree", TypeID: "missing-type", IssuerOrganizationID: "o1",
			IssuedAt: &issuedAt, FileBytes: []byte("d"),
			Filename: "a.pdf", MIMEType: "application/pdf",
		},
	})
	var verrs validation.Errors
	require.ErrorAs(t, err, &verrs)
	assert.Contains(t, verrs, "credentials.0.type_id")
}

func TestSubmit_NumberDuplicate(t *testing.T) {
	svc, _ := newCredentialServiceWithSQLite(t)

	authUser := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	ctx := ctxWithAuth(&authUser)

	number := "N-001"
	issuedAt := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	first := CredentialSubmission{
		Name: "Degree", TypeID: "t1", IssuerOrganizationID: "o1",
		Number: &number, IssuedAt: &issuedAt, FileBytes: []byte("x"),
		Filename: "a.pdf", MIMEType: "application/pdf",
	}
	_, err := svc.Submit(ctx, []CredentialSubmission{first})
	require.NoError(t, err)

	second := CredentialSubmission{
		Name: "Diploma", TypeID: "t1", IssuerOrganizationID: "o1",
		Number: &number, IssuedAt: &issuedAt, FileBytes: []byte("y"),
		Filename: "b.pdf", MIMEType: "application/pdf",
	}
	_, err = svc.Submit(ctx, []CredentialSubmission{second})
	var verrs validation.Errors
	require.ErrorAs(t, err, &verrs)
	assert.Contains(t, verrs, "credentials.0.number", "duplicate number rejected on second submission")
}

// ── Review: Approve / Reject (B4) ─────────────────────────────────────────

// newCredentialServiceForReview builds a *credentialService wired with a
// propagating UoW, a user repo that answers holder lookups for the review
// flows, a no-op enqueuer, and the given registry service mock.
func newCredentialServiceForReview(t *testing.T, uow domain.UnitOfWork, credRepo *mocks.MockCredentialRepository, regSvc *mocks.MockRegistryService) *credentialService {
	t.Helper()
	userRepo := &mocks.MockUserRepository{}
	userRepo.On("FindByIds", mock.Anything, []string{"h1"}).Return([]domain.User{
		{Id: "h1", WalletAddress: "0x1111111111111111111111111111111111111111"},
	}, nil)
	enq := &localMockEnqueuer{}
	enq.On("EnqueueExtract", mock.Anything, mock.Anything).Return(nil)
	return &credentialService{
		uow:             uow,
		userRepo:        userRepo,
		registryService: regSvc,
		cfg:             testConfig(),
		enqueuer:        enq,
		policy:          &credentialPolicy{},
		logger:          zap.NewNop(),
	}
}

func reviewPendingCredential() domain.Credential {
	return domain.Credential{
		ID: "c1", HolderUserID: "h1", SubmitterUserID: "h1", IssuerUserID: "h1",
		TypeID: "t1", IssuerOrganizationID: "o1", Name: "Degree", FileHash: "0x1",
		FileURI: lo.ToPtr("f.pdf"), IssuedAt: time.Now(),
		ExtractStatus: domain.ExtractStatusUnextracted,
	}
}

func TestApprove_ChainFailure_RollsBackAndStaysPending(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	pending := []domain.Credential{reviewPendingCredential()}

	innerCredRepo := new(mocks.MockCredentialRepository)
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return(pending, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything).Return(pending, nil)

	regSvc := new(mocks.MockRegistryService)
	regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("rpc down"))

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	svc := newCredentialServiceForReview(t, uow, innerCredRepo, regSvc)

	_, err := svc.Approve(ctx, "c1")
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialReviewBlockchainSyncFailed, de.Code)

	// The approval Update ran inside the UoW but the mint failed — with the
	// real GormUnitOfWork the approval is rolled back and the row stays pending.
	innerCredRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestApprove_HappyPath_MintsAndApproves(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)
	pending := []domain.Credential{reviewPendingCredential()}

	var approvalUpdates []domain.Credential
	innerCredRepo := new(mocks.MockCredentialRepository)
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return(pending, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything).
		Return(pending, nil).
		Run(func(args mock.Arguments) {
			for _, c := range args.Get(1).([]domain.Credential) {
				if c.ApproverUserID != nil {
					approvalUpdates = append(approvalUpdates, c)
				}
			}
		})

	regSvc := new(mocks.MockRegistryService)
	regSvc.On("IssueCredentials", mock.Anything, mock.Anything, mock.Anything).
		Return([]*big.Int{big.NewInt(7)}, nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)

	svc := newCredentialServiceForReview(t, uow, innerCredRepo, regSvc)

	approved, err := svc.Approve(ctx, "c1")
	require.NoError(t, err)
	require.Len(t, approved, 1)

	regSvc.AssertCalled(t, "IssueCredentials", mock.Anything, mock.Anything, mock.Anything)

	require.Len(t, approvalUpdates, 1)
	u := approvalUpdates[0]
	assert.Equal(t, user.Id, *u.ApproverUserID)
	assert.Equal(t, user.Id, u.IssuerUserID)
	assert.NotNil(t, u.ApprovedAt)
	assert.Equal(t, domain.ExtractStatusPending, u.ExtractStatus)
}

func TestApprove_NotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	innerCredRepo := new(mocks.MockCredentialRepository)
	innerCredRepo.On("FindByIds", mock.Anything, []string{"ghost"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)
	regSvc := new(mocks.MockRegistryService)
	svc := newCredentialServiceForReview(t, uow, innerCredRepo, regSvc)

	_, err := svc.Approve(ctx, "ghost")
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialReviewNotFound, de.Code)
	regSvc.AssertNotCalled(t, "IssueCredentials", mock.Anything, mock.Anything, mock.Anything)
}

func TestApprove_NotPending(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		cred domain.Credential
		want int
	}{
		{"already approved", domain.Credential{ID: "c1", ApprovedAt: &now}, domain.CodeCredentialReviewAlreadyApproved},
		{"already rejected", domain.Credential{ID: "c1", RejectedAt: &now}, domain.CodeCredentialReviewAlreadyRejected},
		{"already revoked", domain.Credential{ID: "c1", RevokedAt: &now}, domain.CodeCredentialReviewAlreadyRevoked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
			ctx := ctxWithAuth(&user)

			innerCredRepo := new(mocks.MockCredentialRepository)
			innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{tt.cred}, nil)
			uow := mocks.NewPropagatingUnitOfWork()
			uow.On("Credential").Return(innerCredRepo)
			regSvc := new(mocks.MockRegistryService)
			svc := newCredentialServiceForReview(t, uow, innerCredRepo, regSvc)

			_, err := svc.Approve(ctx, "c1")
			var de *domain.Error
			require.ErrorAs(t, err, &de)
			assert.Equal(t, tt.want, de.Code)
			innerCredRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
			regSvc.AssertNotCalled(t, "IssueCredentials", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestReject_HappyPath(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	pending := []domain.Credential{
		{ID: "c1", HolderUserID: "h1", ExtractStatus: domain.ExtractStatusUnextracted},
		{ID: "c2", HolderUserID: "h1", ExtractStatus: domain.ExtractStatusUnextracted},
	}

	var updateArgs []domain.Credential
	innerCredRepo := new(mocks.MockCredentialRepository)
	innerCredRepo.On("FindByIds", mock.Anything, []string{"c1", "c2"}, (*domainQuery.Query)(nil)).Return(pending, nil)
	innerCredRepo.On("Update", mock.Anything, mock.Anything).
		Return(pending, nil).
		Run(func(args mock.Arguments) {
			updateArgs = args.Get(1).([]domain.Credential)
		})

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)
	svc := newCredentialServiceForReview(t, uow, innerCredRepo, new(mocks.MockRegistryService))

	rejected, err := svc.Reject(ctx, []CredentialRejection{
		{ID: "c1", Reason: "unreadable scan"},
		{ID: "c2", Reason: "expired document"},
	})
	require.NoError(t, err)
	require.Len(t, rejected, 2)

	require.Len(t, updateArgs, 2)
	assert.Equal(t, "unreadable scan", *updateArgs[0].RejectionReason)
	assert.Equal(t, "expired document", *updateArgs[1].RejectionReason)
	assert.Equal(t, user.Id, *updateArgs[0].RejecterUserID)
	assert.NotNil(t, updateArgs[0].RejectedAt)
}

func TestReject_NotFound(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
	ctx := ctxWithAuth(&user)

	innerCredRepo := new(mocks.MockCredentialRepository)
	innerCredRepo.On("FindByIds", mock.Anything, []string{"ghost"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{}, nil)
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(innerCredRepo)
	svc := newCredentialServiceForReview(t, uow, innerCredRepo, new(mocks.MockRegistryService))

	_, err := svc.Reject(ctx, []CredentialRejection{{ID: "ghost", Reason: "x"}})
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialReviewNotFound, de.Code)
}

func TestReject_NotPending(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		cred domain.Credential
		want int
	}{
		{"already approved", domain.Credential{ID: "c1", ApprovedAt: &now}, domain.CodeCredentialReviewAlreadyApproved},
		{"already rejected", domain.Credential{ID: "c1", RejectedAt: &now}, domain.CodeCredentialReviewAlreadyRejected},
		{"already revoked", domain.Credential{ID: "c1", RevokedAt: &now}, domain.CodeCredentialReviewAlreadyRevoked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
			ctx := ctxWithAuth(&user)

			innerCredRepo := new(mocks.MockCredentialRepository)
			innerCredRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{tt.cred}, nil)
			uow := mocks.NewPropagatingUnitOfWork()
			uow.On("Credential").Return(innerCredRepo)
			svc := newCredentialServiceForReview(t, uow, innerCredRepo, new(mocks.MockRegistryService))

			_, err := svc.Reject(ctx, []CredentialRejection{{ID: "c1", Reason: "x"}})
			var de *domain.Error
			require.ErrorAs(t, err, &de)
			assert.Equal(t, tt.want, de.Code)
			innerCredRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
		})
	}
}

// ── Update (B5) ───────────────────────────────────────────────────────────

func TestCredentialUpdate_PendingRow_EditsAllFields(t *testing.T) {
	ctx := context.Background()
	target := domain.Credential{
		ID:                   "c1",
		IssuerOrganizationID: "org-1",
		TypeID:               "type-1",
		Number:               lo.ToPtr("N-001"),
		Name:                 "old",
	}
	updated := domain.Credential{
		ID:                   "c1",
		Name:                 "new",
		Number:               lo.ToPtr("N-002"),
		TypeID:               "type-2",
		IssuerOrganizationID: "org-2",
		IssuedAt:             time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		ExpiresAt:            lo.ToPtr(time.Date(2027, 8, 1, 0, 0, 0, 0, time.UTC)),
		Meta:                 map[string]any{"k": "v"},
	}
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{target}, nil)
	credRepo.On("Get", mock.Anything, mock.Anything).Return([]domain.Credential{}, 0, nil)
	credRepo.On("Update", mock.Anything, mock.Anything).Return([]domain.Credential{updated}, nil)
	typeRepo := &mockCredentialTypeRepository{}
	typeRepo.On("Find", mock.Anything, "type-2").Return(&domain.CredentialType{Id: "type-2", Active: true}, nil)
	orgRepo := &mockCredentialIssuerOrganizationRepository{}
	orgRepo.On("Find", mock.Anything, "org-2").Return(&domain.CredentialIssuerOrganization{Id: "org-2"}, nil)
	svc := &credentialService{repo: credRepo, typeRepo: typeRepo, orgRepo: orgRepo, logger: zap.NewNop()}

	got, err := svc.Update(ctx, updated)
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "new", got[0].Name)
	credRepo.AssertNumberOfCalls(t, "Get", 1)
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestCredentialUpdate_ApprovedRow_Rejected(t *testing.T) {
	testCredentialUpdateNotPending(t, domain.Credential{ID: "c1", ApprovedAt: lo.ToPtr(time.Now())})
}

func TestCredentialUpdate_RejectedRow_Rejected(t *testing.T) {
	testCredentialUpdateNotPending(t, domain.Credential{ID: "c1", RejectedAt: lo.ToPtr(time.Now())})
}

func TestCredentialUpdate_RevokedRow_Rejected(t *testing.T) {
	testCredentialUpdateNotPending(t, domain.Credential{ID: "c1", RevokedAt: lo.ToPtr(time.Now())})
}

func testCredentialUpdateNotPending(t *testing.T, target domain.Credential) {
	t.Helper()
	ctx := context.Background()
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{target}, nil)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}

	_, err := svc.Update(ctx, domain.Credential{ID: "c1", Name: "new"})
	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.CodeCredentialUpdateNotPending, domErr.Code)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestCredentialUpdate_TypeInactiveRejected(t *testing.T) {
	ctx := context.Background()
	target := domain.Credential{
		ID: "c1", IssuerOrganizationID: "org-1", TypeID: "type-1", Number: lo.ToPtr("N-001"),
	}
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{target}, nil)
	typeRepo := &mockCredentialTypeRepository{}
	typeRepo.On("Find", mock.Anything, "type-2").Return(&domain.CredentialType{Id: "type-2", Active: false}, nil)
	svc := &credentialService{repo: credRepo, typeRepo: typeRepo, logger: zap.NewNop()}

	_, err := svc.Update(ctx, domain.Credential{ID: "c1", TypeID: "type-2"})
	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.CodeCredentialIssueTypeInactive, domErr.Code)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestCredentialUpdate_OrgNotFoundRejected(t *testing.T) {
	ctx := context.Background()
	target := domain.Credential{
		ID: "c1", IssuerOrganizationID: "org-1", TypeID: "type-1", Number: lo.ToPtr("N-001"),
	}
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{target}, nil)
	orgRepo := &mockCredentialIssuerOrganizationRepository{}
	orgRepo.On("Find", mock.Anything, "org-missing").Return((*domain.CredentialIssuerOrganization)(nil), gorm.ErrRecordNotFound)
	svc := &credentialService{repo: credRepo, orgRepo: orgRepo, logger: zap.NewNop()}

	_, err := svc.Update(ctx, domain.Credential{ID: "c1", IssuerOrganizationID: "org-missing"})
	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.CodeCredentialIssueOrganizationNotFound, domErr.Code)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestCredentialUpdate_NumberDuplicateRejected(t *testing.T) {
	ctx := context.Background()
	target := domain.Credential{
		ID: "c1", IssuerOrganizationID: "org-1", TypeID: "type-1", Number: lo.ToPtr("N-001"),
	}
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("FindByIds", mock.Anything, []string{"c1"}, (*domainQuery.Query)(nil)).Return([]domain.Credential{target}, nil)
	credRepo.On("Get", mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "other", Number: lo.ToPtr("N-002")}}, 1, nil)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}

	_, err := svc.Update(ctx, domain.Credential{ID: "c1", Number: lo.ToPtr("N-002")})
	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.CodeCredentialIssueNumberDuplicate, domErr.Code)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestCredentialUpdate_NotFound(t *testing.T) {
	ctx := context.Background()
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("FindByIds", mock.Anything, []string{"c1", "c2"}, (*domainQuery.Query)(nil)).Return(
		[]domain.Credential{{ID: "c1"}}, nil)
	svc := &credentialService{repo: credRepo, logger: zap.NewNop()}

	_, err := svc.Update(ctx, domain.Credential{ID: "c1"}, domain.Credential{ID: "c2"})
	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.CodeCredentialUpdateNotFound, domErr.Code)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

// ── LinkCompetencies (competency replace-set) ─────────────────────────────

func TestLinkCompetencies_HappyPathReplacesSet(t *testing.T) {
	ctx := context.Background()

	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-1", (*domainQuery.Query)(nil)).
		Return(&domain.Credential{ID: "cred-1"}, nil)

	compRepo := &mockCompetencyRepository{}
	compRepo.On("FindByIds", mock.Anything, []string{"comp-a", "comp-b"}).
		Return([]domain.Competency{{Id: "comp-a"}, {Id: "comp-b"}}, nil)

	var storedLinks []domain.CompetencyCredential
	compCredRepo := &mockCompetencyCredentialRepository{}
	compCredRepo.On("DestroyByCredentialId", mock.Anything, "cred-1").Return(int64(2), nil)
	compCredRepo.On("Store", mock.Anything, mock.Anything).
		Return([]domain.CompetencyCredential{}, nil).
		Run(func(args mock.Arguments) {
			storedLinks = append(storedLinks, args.Get(1).([]domain.CompetencyCredential)...)
		})

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(credRepo)
	uow.On("CompetencyCredential").Return(compCredRepo)

	svc := &credentialService{uow: uow, competencyRepo: compRepo}
	err := svc.LinkCompetencies(ctx, "cred-1", []string{"comp-a", "comp-b"})

	require.NoError(t, err)
	assert.Equal(t, []domain.CompetencyCredential{
		{CompetencyId: "comp-a", CredentialId: "cred-1"},
		{CompetencyId: "comp-b", CredentialId: "cred-1"},
	}, storedLinks)
	compCredRepo.AssertCalled(t, "DestroyByCredentialId", mock.Anything, "cred-1")
}

func TestLinkCompetencies_CredentialNotFound(t *testing.T) {
	ctx := context.Background()

	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-missing", (*domainQuery.Query)(nil)).
		Return(nil, gorm.ErrRecordNotFound)

	compCredRepo := &mockCompetencyCredentialRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(credRepo)
	uow.On("CompetencyCredential").Return(compCredRepo)

	svc := &credentialService{uow: uow, competencyRepo: &mockCompetencyRepository{}}
	err := svc.LinkCompetencies(ctx, "cred-missing", []string{"comp-a"})

	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.CodeCredentialCompetencyLinkCredentialNotFound, domErr.Code)
	assert.Equal(t, "cred-missing", domErr.Metadata["credential_id"])
	compCredRepo.AssertNotCalled(t, "DestroyByCredentialId", mock.Anything, mock.Anything)
}

func TestLinkCompetencies_CompetencyNotFound(t *testing.T) {
	ctx := context.Background()

	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-1", (*domainQuery.Query)(nil)).
		Return(&domain.Credential{ID: "cred-1"}, nil)

	compRepo := &mockCompetencyRepository{}
	compRepo.On("FindByIds", mock.Anything, []string{"comp-a", "comp-x"}).
		Return([]domain.Competency{{Id: "comp-a"}}, nil)

	compCredRepo := &mockCompetencyCredentialRepository{}
	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(credRepo)
	uow.On("CompetencyCredential").Return(compCredRepo)

	svc := &credentialService{uow: uow, competencyRepo: compRepo}
	err := svc.LinkCompetencies(ctx, "cred-1", []string{"comp-a", "comp-x"})

	var domErr *domain.Error
	require.ErrorAs(t, err, &domErr)
	assert.Equal(t, domain.CodeCredentialCompetencyLinkCompetencyNotFound, domErr.Code)
	assert.Equal(t, []string{"comp-x"}, domErr.Metadata["competency_ids"])
	compCredRepo.AssertNotCalled(t, "DestroyByCredentialId", mock.Anything, mock.Anything)
}

func TestLinkCompetencies_EmptySetClearsLinks(t *testing.T) {
	ctx := context.Background()

	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Find", mock.Anything, "cred-1", (*domainQuery.Query)(nil)).
		Return(&domain.Credential{ID: "cred-1"}, nil)

	compCredRepo := &mockCompetencyCredentialRepository{}
	compCredRepo.On("DestroyByCredentialId", mock.Anything, "cred-1").Return(int64(2), nil)

	uow := mocks.NewPropagatingUnitOfWork()
	uow.On("Credential").Return(credRepo)
	uow.On("CompetencyCredential").Return(compCredRepo)

	svc := &credentialService{uow: uow, competencyRepo: &mockCompetencyRepository{}}
	err := svc.LinkCompetencies(ctx, "cred-1", []string{})

	require.NoError(t, err)
	compCredRepo.AssertCalled(t, "DestroyByCredentialId", mock.Anything, "cred-1")
	compCredRepo.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
}

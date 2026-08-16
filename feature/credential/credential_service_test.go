package credential

import (
	"context"
	"math/big"
	"testing"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	"CredChain_Golang/infrastructure/chain/contracts"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/jobs"
	"CredChain_Golang/infrastructure/storage"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/mocks"

	"github.com/ethereum/go-ethereum/common"
	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	svc := &credentialService{
		cfg:             testConfig(),
		registryService: regSvc,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "a", Filename: "x.pdf", FileBytes: []byte("x")},
		{HolderUserID: "holder-2", Name: "b", Filename: "x.pdf", FileBytes: []byte("y")},
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
	svc := &credentialService{
		uow:             uow,
		cfg:             testConfig(),
		registryService: regSvc,
		storage:         stor,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
		enqueuer:        enq,
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-valid", Name: "doc", Filename: "x.pdf", FileBytes: []byte("test")},
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
	svc := &credentialService{
		cfg:             testConfig(),
		registryService: regSvc,
		policy:          &credentialPolicy{},
		userRepo:        userRepo,
		logger:          zap.NewNop(),
	}
	items := []CredentialIssuance{
		{HolderUserID: "holder-1", Name: "a", Filename: "x.pdf", FileBytes: []byte("dup")},
		{HolderUserID: "holder-2", Name: "b", Filename: "x.pdf", FileBytes: []byte("unique")},
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
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()

	data := []byte("a")
	items := []CredentialIssuance{
		{HolderUserID: "hA", Name: "C1", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: data},
		{HolderUserID: "hB", Name: "C2", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: data},
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
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("x")},
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
	svc := newTestCredentialService(m)
	svc.uow = uow
	svc.userRepo = userRepo
	svc.cfg = testConfig()
	svc.storage = &storage.Storage{Config: &config.Config{StoragePath: lo.ToPtr(t.TempDir())}}
	svc.enqueuer = enq

	items := []CredentialIssuance{
		{HolderUserID: "h", Name: "C", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("y")},
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
	svc := newTestCredentialService(m)
	svc.userRepo = userRepo
	svc.cfg = testConfig()

	items := []CredentialIssuance{
		{HolderUserID: "ghost", Name: "C", Filename: "a.pdf", MIMEType: "application/pdf", FileBytes: []byte("z")},
	}

	_, err := svc.Issue(ctx, items)
	assert.Error(t, err)
	verrs, ok := err.(validation.Errors)
	assert.True(t, ok)
	assert.Contains(t, verrs, "credentials.0.holder_user_id")
}

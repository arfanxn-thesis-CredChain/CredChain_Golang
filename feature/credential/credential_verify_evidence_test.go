package credential

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"CredChain_Golang/domain"
	pyai "CredChain_Golang/infrastructure/ai/pyai"
	"CredChain_Golang/infrastructure/chain/contracts"
	"CredChain_Golang/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestVerify_ExactEvidenceCannotComeFromCache(t *testing.T) {
	for _, code := range []int{400401, 400402, 400410, 400411, 400412, 400413, 400404} {
		t.Run(domainCodeName(code), func(t *testing.T) {
			m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}, verRepo: &mocks.MockCredentialVerificationRepository{}, extRepo: &mocks.MockCredentialExtractionRepository{}, aiClient: &mocks.MockPythonAIClient{}, regSvc: &mocks.MockRegistryService{}}
			id := "record"
			score := 1.0
			m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{}, nil)
			m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(&domain.CredentialVerification{VerdictCode: code, MatchedCredentialID: &id, SimilarityScore: &score}, nil)
			m.credRepo.On("FindVerifiableById", mock.Anything, id, mock.Anything).Return(&domain.Credential{ID: id, Holder: &domain.User{}, Issuer: &domain.User{}}, nil)
			result, err := newTestCredentialService(m).Verify(context.Background(), pyai.ExtractFile{Data: []byte("test-file")})
			require.NoError(t, err)
			assert.False(t, result.ExactMatchConfirmed)
			m.unitRepo.AssertNumberOfCalls(t, "Find", 0)
			m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
		})
	}
}

func TestVerify_UncachedAIScoreOneDoesNotConfirmExactEvidence(t *testing.T) {
	m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}, verRepo: &mocks.MockCredentialVerificationRepository{}, extRepo: &mocks.MockCredentialExtractionRepository{}, aiClient: &mocks.MockPythonAIClient{}, regSvc: &mocks.MockRegistryService{}}
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return(nil, nil)
	m.verRepo.On("FindByUploadedFileHash", mock.Anything, mock.Anything).Return(nil, nil)
	m.aiClient.On("ExtractIDs", mock.Anything, mock.Anything).Return([]pyai.ExtractedID{{Type: "student_id", Value: "12345"}}, nil)
	m.extRepo.On("FindRankedByIds", mock.Anything, mock.Anything, 10).Return([]domain.CredentialExtraction{{CredentialID: "record", IDs: []domain.CredentialExtractedID{{Value: "12345"}}, Embedding: []float64{0.1}}}, nil)
	m.aiClient.On("Verify", mock.Anything, mock.Anything, mock.Anything).Return(&pyai.VerifyResult{Verdict: "tampered", SimilarityScore: 1, SimilarityPercent: "100%"}, nil)
	m.credRepo.On("FindVerifiableById", mock.Anything, "record", mock.Anything).Return(&domain.Credential{ID: "record"}, nil)
	m.verRepo.On("Store", mock.Anything, mock.Anything).Return(nil)
	result, err := newTestCredentialService(m).Verify(context.Background(), pyai.ExtractFile{Data: []byte("test-file")})
	require.NoError(t, err)
	require.NotNil(t, result.SimilarityScore)
	assert.Equal(t, 1.0, *result.SimilarityScore)
	assert.False(t, result.ExactMatchConfirmed)
	m.unitRepo.AssertNumberOfCalls(t, "Find", 0)
	m.regSvc.AssertNotCalled(t, "GetCredentialsByIds", mock.Anything, mock.Anything)
	m.aiClient.AssertExpectations(t)
}

func TestVerify_ResolvesOnlySelectedExactHolderUnit(t *testing.T) {
	m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}, regSvc: &mocks.MockRegistryService{}}
	units := &mocks.MockUserUnitRepository{}
	unitID := "unit"
	cred := exactVerifyCredential("record", "12345", verifyFileHash)
	cred.Holder.UnitID = &unitID
	older := exactVerifyCredential("older", "12346", verifyFileHash)
	olderUnitID := "other-unit"
	older.Holder.UnitID = &olderUnitID
	older.IssuedAt = time.Unix(900, 0).UTC()
	olderChain := exactVerifyChain("older", 12346, verifyFileHash)
	olderChain.IssuedAt.SetInt64(900)
	m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{older, cred}, nil)
	m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return([]contracts.CredentialRegistryCredential{olderChain, exactVerifyChain("record", 12345, verifyFileHash)}, nil)
	units.On("Find", mock.Anything, unitID).Return(&domain.UserUnit{Name: "Engineering", Active: false}, nil).Once()
	svc := NewCredentialService(CredentialServiceParams{Repo: m.credRepo, RegistryService: m.regSvc, UnitRepo: units})
	result, err := svc.Verify(context.Background(), pyai.ExtractFile{Data: []byte("test-file")})
	require.NoError(t, err)
	require.NotNil(t, result.HolderUnitName)
	assert.Equal(t, "Engineering", *result.HolderUnitName)
	units.AssertExpectations(t)
}

func TestVerify_ExactHolderUnitMissingAndErrors(t *testing.T) {
	for _, state := range []string{"no-unit", "no-holder", "not-found", "nil-result", "error", "chain-error", "mismatch"} {
		t.Run(state, func(t *testing.T) {
			units := &mocks.MockUserUnitRepository{}
			m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}, regSvc: &mocks.MockRegistryService{}, unitRepo: units}
			cred := exactVerifyCredential("record", "12345", verifyFileHash)
			unitID := "unit"
			if state != "no-unit" {
				cred.Holder.UnitID = &unitID
			}
			chain := exactVerifyChain("record", 12345, verifyFileHash)
			var chainErr error
			lookupErr := errors.New("unit storage unavailable")
			switch state {
			case "no-holder":
				cred.Holder = nil
			case "not-found":
				units.On("Find", mock.Anything, unitID).Return(nil, gorm.ErrRecordNotFound).Once()
			case "nil-result":
				units.On("Find", mock.Anything, unitID).Return(nil, nil).Once()
			case "error":
				units.On("Find", mock.Anything, unitID).Return(nil, lookupErr).Once()
			case "chain-error":
				chainErr = errors.New("RPC unavailable")
			case "mismatch":
				chain.Hash = "different"
			}
			m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{cred}, nil)
			m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return([]contracts.CredentialRegistryCredential{chain}, chainErr)
			result, err := newTestCredentialService(m).Verify(context.Background(), pyai.ExtractFile{Data: []byte("test-file")})
			if state == "error" {
				require.ErrorIs(t, err, lookupErr)
			} else if state == "chain-error" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, state != "error" && state != "chain-error" && state != "mismatch" && state != "no-holder", result.ExactMatchConfirmed)
			if state == "no-holder" {
				assert.Equal(t, domain.CodeCredentialVerifyIntegrityWarning, result.VerdictCode)
			}
			assert.Nil(t, result.HolderUnitName)
			units.AssertExpectations(t)
			if state == "no-unit" || state == "no-holder" || state == "chain-error" || state == "mismatch" {
				units.AssertNumberOfCalls(t, "Find", 0)
			}
		})
	}
}

func domainCodeName(code int) string { return strconv.Itoa(code) }

func TestVerify_LiveExactEvidenceSurvivesLifecycleWarnings(t *testing.T) {
	for _, state := range []string{"valid", "revoked", "expired", "holder-disabled", "issuer-disabled", "both-disabled", "mismatch"} {
		t.Run(state, func(t *testing.T) {
			m := &testCredentialMocks{credRepo: &mocks.MockCredentialRepository{}, verRepo: &mocks.MockCredentialVerificationRepository{}, extRepo: &mocks.MockCredentialExtractionRepository{}, aiClient: &mocks.MockPythonAIClient{}, regSvc: &mocks.MockRegistryService{}}
			cred := exactVerifyCredential("record", "12345", verifyFileHash)
			chain := exactVerifyChain("record", 12345, verifyFileHash)
			now := time.Now()
			switch state {
			case "revoked":
				cred.RevokedAt = &now
				chain.RevokedAt.SetInt64(2000)
			case "expired":
				past := time.Unix(1500, 0)
				cred.ExpiresAt = &past
				chain.ExpiresAt.SetInt64(1500)
			case "holder-disabled":
				cred.Holder.DeletedAt = &now
			case "issuer-disabled":
				cred.Issuer.DeletedAt = &now
			case "both-disabled":
				cred.Holder.DeletedAt = &now
				cred.Issuer.DeletedAt = &now
			case "mismatch":
				chain.Hash = "different"
			}
			m.credRepo.On("FindByFileHashes", mock.Anything, mock.Anything, mock.Anything).Return([]domain.Credential{cred}, nil)
			m.regSvc.On("GetCredentialsByIds", mock.Anything, mock.Anything).Return([]contracts.CredentialRegistryCredential{chain}, nil)
			result, err := newTestCredentialService(m).Verify(context.Background(), pyai.ExtractFile{Data: []byte("test-file")})
			require.NoError(t, err)
			assert.Equal(t, state != "mismatch", result.ExactMatchConfirmed)
		})
	}
}

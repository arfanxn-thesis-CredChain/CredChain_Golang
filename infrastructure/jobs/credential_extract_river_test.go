package jobs

import (
	"context"
	"encoding/json"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/ai/pyai"
	infraCrypto "CredChain_Golang/infrastructure/crypto"
	"CredChain_Golang/infrastructure/storage"
	"CredChain_Golang/tests/mocks"

	"github.com/riverqueue/river/rivertype"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func setupTestStorage(t *testing.T, key []byte, plaintext []byte) (*storage.Storage, string) {
	t.Helper()
	s := &storage.Storage{Config: &config.Config{StoragePath: lo.ToPtr(t.TempDir())}}
	relPath := "test.pdf"
	encHex, err := infraCrypto.Encrypt(plaintext, key)
	require.NoError(t, err)
	_, err = s.SaveBytes([]byte(encHex), relPath)
	require.NoError(t, err)
	return s, relPath
}

func TestWorkExtract_Success(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	s, relPath := setupTestStorage(t, key, []byte("test-data"))

	credRepo := &mocks.MockCredentialRepository{}
	extRepo := &mocks.MockCredentialExtractionRepository{}
	aiClient := &mocks.MockPythonAIClient{}

	credRepo.On("Find", mock.Anything, "cred-id", mock.Anything).
		Return(&domain.Credential{FileHash: "0xabc"}, nil)
	aiClient.On("Extract", mock.Anything, mock.Anything).
		Return([]pyai.PythonExtractResult{
			{Text: "extracted-text", IDs: []pyai.ExtractedID{{Type: "id", Value: "123"}}, Embedding: []float64{0.1, 0.2}},
		}, nil)
	extRepo.On("Store", mock.Anything, mock.Anything).Return(nil)
	credRepo.On("Update", mock.Anything, mock.Anything).
		Return([]domain.Credential{{ID: "cred-id"}}, nil)

	w := &CredentialExtractWorker{
		credRepo:       credRepo,
		extractionRepo: extRepo,
		aiClient:       aiClient,
		storage:        s,
		config:         &config.Config{FileEncryptionKey: lo.ToPtr(string(key))},
		logger:         zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{CredentialID: "cred-id", FileURI: relPath})
	assert.NoError(t, err)
	extRepo.AssertCalled(t, "Store", mock.Anything, mock.Anything)
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestWorkExtract_FileNotFound(t *testing.T) {
	s := &storage.Storage{Config: &config.Config{StoragePath: lo.ToPtr(t.TempDir())}}
	w := &CredentialExtractWorker{
		aiClient: &mocks.MockPythonAIClient{},
		storage:  s,
		logger:   zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{
		CredentialID: "cred-id", FileURI: "nonexistent.pdf",
	})
	assert.Error(t, err)
}

func TestWorkExtract_EmptyEmbedding(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	s, relPath := setupTestStorage(t, key, []byte("test-data"))

	credRepo := &mocks.MockCredentialRepository{}
	extRepo := &mocks.MockCredentialExtractionRepository{}
	aiClient := &mocks.MockPythonAIClient{}

	aiClient.On("Extract", mock.Anything, mock.Anything).
		Return([]pyai.PythonExtractResult{{Text: "", IDs: nil, Embedding: nil}}, nil)

	w := &CredentialExtractWorker{
		credRepo:       credRepo,
		extractionRepo: extRepo,
		aiClient:       aiClient,
		storage:        s,
		config:         &config.Config{FileEncryptionKey: lo.ToPtr(string(key))},
		logger:         zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{CredentialID: "cred-id", FileURI: relPath})
	assert.Error(t, err)
	extRepo.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestWorkExtract_StoreFails(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	s, relPath := setupTestStorage(t, key, []byte("test-data"))

	credRepo := &mocks.MockCredentialRepository{}
	extRepo := &mocks.MockCredentialExtractionRepository{}
	aiClient := &mocks.MockPythonAIClient{}

	credRepo.On("Find", mock.Anything, "cred-id", mock.Anything).
		Return(&domain.Credential{FileHash: "0xabc"}, nil)
	aiClient.On("Extract", mock.Anything, mock.Anything).
		Return([]pyai.PythonExtractResult{
			{Text: "t", IDs: []pyai.ExtractedID{{Type: "a", Value: "b"}}, Embedding: []float64{0.5}},
		}, nil)
	extRepo.On("Store", mock.Anything, mock.Anything).Return(assert.AnError)

	w := &CredentialExtractWorker{
		credRepo:       credRepo,
		extractionRepo: extRepo,
		aiClient:       aiClient,
		storage:        s,
		config:         &config.Config{FileEncryptionKey: lo.ToPtr(string(key))},
		logger:         zap.NewNop(),
	}
	err := w.workExtract(context.Background(), CredentialExtractArgs{CredentialID: "cred-id", FileURI: relPath})
	assert.Error(t, err)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandleError_TerminalFailure(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "cred-id"}}, nil)

	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id", FileURI: "/tmp/x"})
	require.NoError(t, err)

	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     5,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandleError(context.Background(), job, assert.AnError)
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

func TestHandleError_NonTerminal(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id"})
	require.NoError(t, err)
	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     1,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandleError(context.Background(), job, assert.AnError)
	credRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestHandlePanic_Terminal(t *testing.T) {
	credRepo := &mocks.MockCredentialRepository{}
	credRepo.On("Update", mock.Anything, mock.Anything, mock.Anything).Return(
		[]domain.Credential{{ID: "cred-id"}}, nil)
	args, err := json.Marshal(CredentialExtractArgs{CredentialID: "cred-id"})
	require.NoError(t, err)
	w := &CredentialExtractWorker{credRepo: credRepo, logger: zap.NewNop()}
	job := &rivertype.JobRow{
		Kind:        "credential_extract",
		Attempt:     5,
		MaxAttempts: 5,
		EncodedArgs: args,
	}
	w.HandlePanic(context.Background(), job, "ouch", "stacktrace")
	credRepo.AssertCalled(t, "Update", mock.Anything, mock.Anything, mock.Anything)
}

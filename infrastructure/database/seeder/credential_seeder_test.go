package seeder_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/infrastructure/storage"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockExtractionRepo struct {
	stored []domain.CredentialExtraction
}

func (m *mockExtractionRepo) Store(ctx context.Context, extraction domain.CredentialExtraction) error {
	m.stored = append(m.stored, extraction)
	return nil
}

func (m *mockExtractionRepo) FindByCredentialId(ctx context.Context, credentialID string) (*domain.CredentialExtraction, error) {
	for _, e := range m.stored {
		if e.CredentialID == credentialID {
			return &e, nil
		}
	}
	return nil, nil
}

func (m *mockExtractionRepo) FindRankedByIds(ctx context.Context, values []string, limit int) ([]domain.CredentialExtraction, error) {
	return nil, nil
}

func TestCredentialSeeder_DeterministicAndValid(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	userUnitRepo := user.NewGormUserUnitRepository(gormDB)
	typeRepo := credential.NewGormCredentialTypeRepository(gormDB)
	issuerOrgRepo := credential.NewGormCredentialIssuerOrganizationRepository(gormDB)
	competencyRepo := credential.NewGormCompetencyRepository(gormDB)
	compCredRepo := credential.NewGormCompetencyCredentialRepository(gormDB)
	credRepo := credential.NewGormCredentialRepository(gormDB)

	ctx := context.Background()

	// Seed dependencies first
	seedMnemonic := "test test test test test test test test test test test junk"
	encKey := string(fixtures.TestWalletEncryptionKey())
	userCfg := &config.Config{
		HardhatMnemonic:     lo.ToPtr(seedMnemonic),
		WalletEncryptionKey: lo.ToPtr(encKey),
	}
	require.NoError(t, seeder.NewUserUnitSeeder(userUnitRepo).Seed(ctx))
	require.NoError(t, seeder.NewUserSeeder(userRepo, userUnitRepo, userCfg).Seed(ctx))
	require.NoError(t, seeder.NewCredentialTypeSeeder(typeRepo).Seed(ctx))
	require.NoError(t, seeder.NewCredentialIssuerOrganizationSeeder(issuerOrgRepo).Seed(ctx))
	require.NoError(t, seeder.NewCompetencySeeder(competencyRepo).Seed(ctx))

	tmpDir, err := os.MkdirTemp("", "credchain-seeder-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	credStoragePath := "credentials"
	credCfg := &config.Config{
		StoragePath:               &tmpDir,
		CredentialFileStoragePath: &credStoragePath,
		FileEncryptionKey:         lo.ToPtr("01234567890123456789012345678901"),
	}
	stor, err := storage.NewStorage(storage.StorageParams{Config: credCfg})
	require.NoError(t, err)

	mockExtract := &mockExtractionRepo{}
	s := seeder.NewCredentialSeeder(
		credRepo,
		userRepo,
		typeRepo,
		issuerOrgRepo,
		competencyRepo,
		compCredRepo,
		mockExtract,
		stor,
		credCfg,
	)

	err = s.Seed(ctx)
	require.NoError(t, err)

	// Verify credentials persisted
	creds, total, err := credRepo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 41, total)
	assert.Len(t, creds, 41)

	// Verify all CHECK constraints and invariants on each credential
	hashMap := make(map[string]bool)
	for _, c := range creds {
		// chk_credentials_approved_xor_rejected
		assert.False(t, c.ApprovedAt != nil && c.RejectedAt != nil, "cannot be both approved and rejected")

		// chk_credentials_approved_metadata_resolved
		if c.ApprovedAt != nil {
			assert.NotNil(t, c.TypeID, "approved cred must have TypeID")
			assert.NotNil(t, c.IssuerOrganizationID, "approved cred must have IssuerOrganizationID")
		}

		// chk_credentials_type_present
		assert.True(t, c.TypeID != nil || c.SubmittedTypeName != nil, "must have type_id or submitted_type_name")

		// chk_credentials_issuer_organization_present
		assert.True(t, c.IssuerOrganizationID != nil || c.SubmittedIssuerOrganizationName != nil, "must have org_id or submitted_org_name")

		// unique file hash
		assert.NotEmpty(t, c.FileHash)
		assert.False(t, hashMap[c.FileHash], "duplicate file hash detected")
		hashMap[c.FileHash] = true

		// file exists on disk
		require.NotNil(t, c.FileURI)
		filePath := filepath.Join(tmpDir, credStoragePath, *c.FileURI)
		fileBytes, err := os.ReadFile(filePath)
		assert.NoError(t, err, "stored file should exist on disk")
		assert.NotEmpty(t, fileBytes)
	}

	// Succeeded extractions (exactly 12 approved_extracted rows)
	assert.Len(t, mockExtract.stored, 12)
	for _, ext := range mockExtract.stored {
		assert.Len(t, ext.Embedding, 768)
		assert.NotEmpty(t, ext.Text)
		assert.NotEmpty(t, ext.IDs)
	}
}

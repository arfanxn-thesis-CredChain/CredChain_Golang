package seeder_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// seedCredentials runs the full dependency chain and returns everything a
// caller needs to assert against the seeded credentials.
type seededCredentials struct {
	creds       []domain.Credential
	usersByID   map[string]domain.User
	extractions []domain.CredentialExtraction
	compCred    domain.CompetencyCredentialRepository
	storageDir  string
	credDir     string
}

func seedCredentials(t *testing.T) seededCredentials {
	t.Helper()

	gormDB := db.OpenInMemorySQLite(t)
	userRepo := user.NewGormUserRepository(gormDB)
	userUnitRepo := user.NewGormUserUnitRepository(gormDB)
	typeRepo := credential.NewGormCredentialTypeRepository(gormDB)
	issuerOrgRepo := credential.NewGormCredentialIssuerOrganizationRepository(gormDB)
	competencyRepo := credential.NewGormCompetencyRepository(gormDB)
	compCredRepo := credential.NewGormCompetencyCredentialRepository(gormDB)
	credRepo := credential.NewGormCredentialRepository(gormDB)

	ctx := context.Background()

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
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

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
	require.NoError(t, s.Seed(ctx))

	creds, _, err := credRepo.Get(ctx, nil)
	require.NoError(t, err)
	require.NotEmpty(t, creds)

	users, _, err := userRepo.Get(ctx, nil)
	require.NoError(t, err)

	return seededCredentials{
		creds:       creds,
		usersByID:   lo.SliceToMap(users, func(u domain.User) (string, domain.User) { return u.Id, u }),
		extractions: mockExtract.stored,
		compCred:    compCredRepo,
		storageDir:  tmpDir,
		credDir:     credStoragePath,
	}
}

// isDirectIssuance distinguishes the two workflows from persisted state alone:
// direct issuance stamps the officer as submitter, self-submission stamps the
// holder.
func isDirectIssuance(c domain.Credential) bool {
	return c.SubmitterUserID != c.HolderUserID
}

// TestCredentialSeeder_SchemaInvariants covers what the Postgres CHECK
// constraints and unique indexes enforce in production. Tests run on SQLite via
// AutoMigrate, which carries none of them, so they are asserted here.
func TestCredentialSeeder_SchemaInvariants(t *testing.T) {
	s := seedCredentials(t)

	activeHashes := make(map[string]bool)
	orgNumbers := make(map[string]bool)
	for _, c := range s.creds {
		// chk_credentials_approved_xor_rejected
		assert.False(t, c.ApprovedAt != nil && c.RejectedAt != nil,
			"%s: cannot be both approved and rejected", c.Name)

		// chk_credentials_approved_metadata_resolved
		if c.ApprovedAt != nil {
			assert.NotNil(t, c.TypeID, "%s: approved row needs a resolved type", c.Name)
			assert.NotNil(t, c.IssuerOrganizationID, "%s: approved row needs a resolved organization", c.Name)
		}

		// chk_credentials_type_present / chk_credentials_issuer_organization_present
		assert.True(t, c.TypeID != nil || c.SubmittedTypeName != nil,
			"%s: needs type_id or submitted_type_name", c.Name)
		assert.True(t, c.IssuerOrganizationID != nil || c.SubmittedIssuerOrganizationName != nil,
			"%s: needs issuer_organization_id or submitted_issuer_organization_name", c.Name)

		// idx_credentials_file_hash_active — unique on file_hash, but only
		// among rows that are neither revoked nor rejected.
		assert.NotEmpty(t, c.FileHash)
		if c.RevokedAt == nil && c.RejectedAt == nil {
			assert.False(t, activeHashes[c.FileHash], "%s: duplicate active file hash", c.Name)
			activeHashes[c.FileHash] = true
		}

		// uq_credentials_issuer_org_number
		if c.IssuerOrganizationID != nil && c.Number != nil {
			key := *c.IssuerOrganizationID + "|" + *c.Number
			assert.False(t, orgNumbers[key], "%s: duplicate number within organization", c.Name)
			orgNumbers[key] = true
		}

		// token_id is written back by seed-chain after minting; the seeder must
		// never invent one.
		assert.Nil(t, c.TokenID, "%s: token_id belongs to seed-chain", c.Name)

		require.NotNil(t, c.FileURI)
		fileBytes, err := os.ReadFile(filepath.Join(s.storageDir, s.credDir, *c.FileURI))
		assert.NoError(t, err, "%s: stored file should exist on disk", c.Name)
		assert.NotEmpty(t, fileBytes)
	}
}

// TestCredentialSeeder_WorkflowInvariants asserts every row is a shape one of
// the two real code paths could have produced.
func TestCredentialSeeder_WorkflowInvariants(t *testing.T) {
	s := seedCredentials(t)

	for _, c := range s.creds {
		if isDirectIssuance(c) {
			require.NotNil(t, c.IssuerUserID, "%s: direct issuance has an issuer", c.Name)
			assert.Equal(t, c.SubmitterUserID, *c.IssuerUserID,
				"%s: direct issuance issuer must be the submitting officer", c.Name)
			assert.NotNil(t, c.ApprovedAt, "%s: direct issuance is approved at creation", c.Name)
			assert.Nil(t, c.RejectedAt, "%s: a directly issued row is never rejected", c.Name)
			continue
		}

		// submitPrepareCredentials stamps the holder as submitter, leaves issuer nil.
		assert.Equal(t, c.HolderUserID, c.SubmitterUserID,
			"%s: a submission is filed by its holder", c.Name)

		switch c.Status() {
		case domain.CredentialStatusPending:
			assert.Nil(t, c.IssuerUserID, "%s: pending submission has nil issuer_user_id", c.Name)
			assert.Nil(t, c.RejecterUserID, "%s: pending row has no rejecter", c.Name)
		case domain.CredentialStatusApproved, domain.CredentialStatusRevoked:
			// Approve assigns issuer_user_id to the reviewing officer, who is
			// the wallet that signs the mint.
			require.NotNil(t, c.IssuerUserID)
			assert.NotEqual(t, c.HolderUserID, *c.IssuerUserID,
				"%s: a holder cannot end up as the issuer of an approved row", c.Name)
		case domain.CredentialStatusRejected:
			require.NotNil(t, c.RejecterUserID)
			assert.NotNil(t, c.RejectionReason, "%s: Reject always records a reason", c.Name)
			// Reject leaves issuer_user_id nil.
			assert.Nil(t, c.IssuerUserID, "%s: rejected submission leaves issuer_user_id nil", c.Name)
		}
	}
}

// TestCredentialSeeder_ReviewInvariants asserts the ordering rules the review
// endpoints enforce.
func TestCredentialSeeder_ReviewInvariants(t *testing.T) {
	s := seedCredentials(t)

	for _, c := range s.creds {
		switch c.Status() {
		case domain.CredentialStatusApproved:
			assert.Empty(t, c.UnresolvedMetadata(),
				"%s: Approve refuses a row with unresolved metadata", c.Name)
		case domain.CredentialStatusRevoked:
			// Revoke requires the row to be approved first.
			assert.NotNil(t, c.ApprovedAt, "%s: only an approved row can be revoked", c.Name)
			require.NotNil(t, c.RevokerUserID)
			assert.True(t, c.RevokedAt.After(*c.ApprovedAt),
				"%s: revocation comes after approval", c.Name)
		}

		if c.ApprovedAt != nil {
			assert.False(t, c.ApprovedAt.Before(c.CreatedAt),
				"%s: approved before it existed", c.Name)
		}
		require.NotNil(t, c.UpdatedAt)
		assert.False(t, c.UpdatedAt.Before(c.CreatedAt), "%s: updated before created", c.Name)
	}
}

// TestCredentialSeeder_OfficersAreIssuerPlus is the invariant seed-chain
// depends on: it signs each mint and revoke batch with the wallet named by
// issuer_user_id / revoker_user_id, and CredentialRegistry reverts
// RoleBelowIssuerError for anything below Issuer.
func TestCredentialSeeder_OfficersAreIssuerPlus(t *testing.T) {
	s := seedCredentials(t)

	requireIssuerPlus := func(t *testing.T, id, role, name string) {
		t.Helper()
		u, ok := s.usersByID[id]
		require.True(t, ok, "%s: %s %s is not a seeded user", name, role, id)
		assert.Nil(t, u.DeletedAt, "%s: %s is soft-deleted, its wallet is off-chain", name, role)
		assert.GreaterOrEqual(t, u.Role.Rank(), domain.RoleIssuer.Rank(),
			"%s: %s has role %s, below issuer", name, role, u.Role)
	}

	var approved int
	for _, c := range s.creds {
		if c.Status() == domain.CredentialStatusPending || c.Status() == domain.CredentialStatusRejected {
			continue
		}
		approved++
		require.NotNil(t, c.IssuerUserID)
		requireIssuerPlus(t, *c.IssuerUserID, "issuer", c.Name)
		if c.RevokerUserID != nil {
			requireIssuerPlus(t, *c.RevokerUserID, "revoker", c.Name)
		}
	}
	assert.Positive(t, approved, "no approved credentials to mint on chain")
}

// TestCredentialSeeder_ExtractInvariants ties the derived ExtractState back to
// the Mongo documents the fuzzy verify path reads.
func TestCredentialSeeder_ExtractInvariants(t *testing.T) {
	s := seedCredentials(t)

	docs := lo.SliceToMap(s.extractions,
		func(e domain.CredentialExtraction) (string, domain.CredentialExtraction) { return e.CredentialID, e })
	assert.Len(t, docs, len(s.extractions), "one extraction document per credential")

	states := make(map[domain.ExtractState]int)
	for _, c := range s.creds {
		state := c.ExtractState()
		states[state]++

		if c.Status() == domain.CredentialStatusPending || c.Status() == domain.CredentialStatusRejected {
			// Extraction is enqueued by Approve, so an unapproved row has none.
			assert.Equal(t, domain.ExtractStateUnextracted, state,
				"%s: extraction starts at approval", c.Name)
		}

		doc, hasDoc := docs[c.ID]
		if state != domain.ExtractStateSucceeded {
			assert.False(t, hasDoc, "%s: only a succeeded extraction has a document", c.Name)
			continue
		}
		require.True(t, hasDoc, "%s: extracted_at with no document is unverifiable", c.Name)
		assert.Equal(t, c.FileHash, doc.FileHash, "%s: document must describe the same file", c.Name)
		assert.Len(t, doc.Embedding, 768)
		assert.NotEmpty(t, doc.Text)
		assert.NotEmpty(t, doc.IDs)

		if state == domain.ExtractStateFailed {
			assert.NotNil(t, c.ExtractError, "%s: a failed extraction records why", c.Name)
		}
	}

	for _, state := range []domain.ExtractState{
		domain.ExtractStateUnextracted,
		domain.ExtractStatePending,
		domain.ExtractStateSucceeded,
		domain.ExtractStateFailed,
	} {
		assert.Positive(t, states[state], "no credential in extract state %s", state)
	}
}

// TestCredentialSeeder_CompetencyLinks asserts the pairing ResolveMetadata
// guarantees: a staged competency gets its resolved_id and its join row
// together.
func TestCredentialSeeder_CompetencyLinks(t *testing.T) {
	s := seedCredentials(t)
	ctx := context.Background()

	var resolvedSeen, stagedSeen int
	for _, c := range s.creds {
		links, err := s.compCred.FindByCredentialId(ctx, c.ID)
		require.NoError(t, err)
		linked := lo.SliceToMap(links, func(l domain.CompetencyCredential) (string, struct{}) {
			return l.CompetencyId, struct{}{}
		})

		for _, sc := range c.SubmittedCompetencies {
			if sc.ResolvedID == nil {
				stagedSeen++
				continue
			}
			resolvedSeen++
			assert.Contains(t, linked, *sc.ResolvedID,
				"%s: resolved competency %s has no join row", c.Name, sc.Name)
		}

		if c.Status() == domain.CredentialStatusApproved || c.Status() == domain.CredentialStatusRevoked {
			assert.NotEmpty(t, links, "%s: an approved credential attests competencies", c.Name)
		}
	}
	assert.Positive(t, resolvedSeen, "no resolved submitted competencies seeded")
	assert.Positive(t, stagedSeen, "no unresolved submitted competencies seeded")
}

// TestCredentialSeeder_HashReuseAfterTerminalState covers the partial unique
// index: a document rejected once may be resubmitted, so exactly one file_hash
// is expected to appear twice.
func TestCredentialSeeder_HashReuseAfterTerminalState(t *testing.T) {
	s := seedCredentials(t)

	byHash := lo.GroupBy(s.creds, func(c domain.Credential) string { return c.FileHash })
	reused := lo.PickBy(byHash, func(_ string, group []domain.Credential) bool { return len(group) > 1 })
	require.Len(t, reused, 1, "expected exactly one reused file hash")

	for _, group := range reused {
		require.Len(t, group, 2)
		statuses := lo.Map(group, func(c domain.Credential, _ int) domain.CredentialStatus { return c.Status() })
		assert.Contains(t, statuses, domain.CredentialStatusRejected,
			"reuse is only legal once the earlier row is rejected or revoked")
		assert.Contains(t, statuses, domain.CredentialStatusPending,
			"the resubmission should still be awaiting review")
	}
}

// TestCredentialSeeder_ScenarioCoverage guards the point of the rewrite: both
// workflows, every status, and every role that can act are represented.
func TestCredentialSeeder_ScenarioCoverage(t *testing.T) {
	s := seedCredentials(t)

	var directIssued, selfSubmitted, expired int
	statuses := make(map[domain.CredentialStatus]int)
	actingRoles := make(map[domain.Role]int)
	holderRoles := make(map[domain.Role]int)

	for _, c := range s.creds {
		statuses[c.Status()]++
		if c.ExpiresAt != nil {
			// Expiry only exercises the verify path when it is already past.
			assert.True(t, c.ExpiresAt.Before(time.Now()),
				"%s: a future expiry never reaches the expired verdict", c.Name)
			expired++
		}
		if isDirectIssuance(c) {
			directIssued++
		} else {
			selfSubmitted++
		}
		for _, id := range []*string{c.IssuerUserID, c.RejecterUserID, c.RevokerUserID} {
			if id != nil {
				actingRoles[s.usersByID[*id].Role]++
			}
		}
		holderRoles[s.usersByID[c.HolderUserID].Role]++
	}

	assert.Positive(t, directIssued, "direct issuance workflow unrepresented")
	assert.Positive(t, selfSubmitted, "self-submission workflow unrepresented")
	assert.Positive(t, expired, "no credential carries an expiry date")

	// Reject never resolves what the submitter staged, so a rejected row whose
	// free-text names were never linked is the shape the review UI must handle.
	var rejectedUnresolved int
	for _, c := range s.creds {
		if c.Status() == domain.CredentialStatusRejected && len(c.UnresolvedMetadata()) > 0 {
			rejectedUnresolved++
		}
	}
	assert.Positive(t, rejectedUnresolved, "no rejected credential keeps its staged names unresolved")

	for _, status := range []domain.CredentialStatus{
		domain.CredentialStatusPending,
		domain.CredentialStatusApproved,
		domain.CredentialStatusRejected,
		domain.CredentialStatusRevoked,
	} {
		assert.Positive(t, statuses[status], "no credential with status %s", status)
	}

	// Issuer, Admin and SuperAdmin all rank at or above Issuer, so all three
	// can issue and review.
	for _, role := range []domain.Role{domain.RoleIssuer, domain.RoleAdmin, domain.RoleSuperAdmin} {
		assert.Positive(t, actingRoles[role], "role %s never acts on a credential", role)
	}
	assert.Positive(t, holderRoles[domain.RoleHolder], "no credential is held by a holder")

	// A soft-deleted holder is what makes the holder_disabled verify verdict
	// reachable. Its issuer stays active, so the row still mints on chain.
	var disabledHolder int
	for _, c := range s.creds {
		if c.Status() != domain.CredentialStatusApproved {
			continue
		}
		if s.usersByID[c.HolderUserID].DeletedAt != nil {
			disabledHolder++
		}
	}
	assert.Positive(t, disabledHolder, "no approved credential held by a disabled user")
}

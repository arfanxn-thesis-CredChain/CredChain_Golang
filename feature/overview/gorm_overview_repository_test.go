package overview

import (
	"context"
	"testing"
	"time"

	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/tests/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func seedRepoTestData(t *testing.T, gormDB *gorm.DB) {
	t.Helper()
	now := time.Now()

	users := []model.User{
		{Id: "01J00000000000000000000001", Name: ptr("John"), Email: "john@test.com", Role: "holder", WalletAddress: "0x1", CreatedAt: now},
		{Id: "01J00000000000000000000002", Name: ptr("IssuerA"), Email: "issuer@test.com", Role: "issuer", WalletAddress: "0x2", CreatedAt: now.Add(-24 * time.Hour)},
		{Id: "01J00000000000000000000003", Name: ptr("Admin"), Email: "admin@test.com", Role: "admin", WalletAddress: "0x3", CreatedAt: now.Add(-48 * time.Hour)},
		{Id: "01J00000000000000000000004", Name: ptr("Trashed"), Email: "trashed@test.com", Role: "holder", WalletAddress: "0x4", CreatedAt: now.Add(-72 * time.Hour), DeletedAt: gorm.DeletedAt{Time: now, Valid: true}},
	}
	require.NoError(t, gormDB.Create(&users).Error)

	creds := []model.Credential{
		{Id: "01J000000000000000000000C1", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Active-Degree", FileHash: "0xaa", ExtractEnqueuedAt: ptr(now), ExtractedAt: ptr(now), IssuedAt: now},
		{Id: "01J000000000000000000000C2", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Revoked-Diploma", FileHash: "0xbb", ExtractEnqueuedAt: ptr(now), ExtractedAt: ptr(now), IssuedAt: now.Add(-1 * time.Hour), RevokedAt: ptr(now), RevokerUserId: ptr("01J00000000000000000000002")},
		{Id: "01J000000000000000000000C3", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Pending-Extract", FileHash: "0xcc", ExtractEnqueuedAt: ptr(now), IssuedAt: now.Add(-2 * time.Hour)},
		{Id: "01J000000000000000000000C4", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Failed-Extract", FileHash: "0xdd", ExtractEnqueuedAt: ptr(now), ExtractFailedAt: ptr(now), IssuedAt: now.Add(-3 * time.Hour)},
		{Id: "01J000000000000000000000C5", HolderUserId: "01J00000000000000000000003", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Admin-Active", FileHash: "0xee", ExtractEnqueuedAt: ptr(now), ExtractedAt: ptr(now), IssuedAt: now.Add(-4 * time.Hour)},
	}
	require.NoError(t, gormDB.Create(&creds).Error)
}

func TestCredentialCounts_Issuer(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	q := &domainQuery.Query{}
	counts, err := repo.CredentialCounts(ctx, q, nil)
	require.NoError(t, err)

	assert.Equal(t, 5, counts.Total)
	assert.Equal(t, 3, counts.Active)
	assert.Equal(t, 1, counts.Revoked)
	assert.Equal(t, 1, counts.Pending)
	assert.Equal(t, 1, counts.Failed)
}

func TestCredentialCounts_Holder(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	q := &domainQuery.Query{}
	holderID := "01J00000000000000000000001"
	counts, err := repo.CredentialCounts(ctx, q, &holderID)
	require.NoError(t, err)

	assert.Equal(t, 4, counts.Total)
	assert.Equal(t, 2, counts.Active)
	assert.Equal(t, 1, counts.Revoked)
	assert.Equal(t, 1, counts.Pending)
	assert.Equal(t, 1, counts.Failed)
}

func TestCredentialCounts_DateFilter(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	now := time.Now()
	from := now.Add(-30 * time.Minute)
	to := now.Add(24 * time.Hour)

	q := &domainQuery.Query{
		Filters: []domainQuery.Filter{
			{Column: "date", Operator: domainQuery.OperatorBetween, Values: []string{from.Format("2006-01-02"), to.Format("2006-01-02")}},
		},
	}
	counts, err := repo.CredentialCounts(ctx, q, nil)
	require.NoError(t, err)
	assert.Greater(t, counts.Total, 0)
}

func TestUserCounts(t *testing.T) {
	gormDB := db.OpenInMemorySQLite(t)
	seedRepoTestData(t, gormDB)
	repo := NewGormOverviewRepository(gormDB)
	ctx := context.Background()

	q := &domainQuery.Query{}
	counts, err := repo.UserCounts(ctx, q)
	require.NoError(t, err)

	assert.Equal(t, 4, counts.Total)
	assert.Equal(t, 2, counts.Holder)
	assert.Equal(t, 1, counts.Issuer)
	assert.Equal(t, 1, counts.Admin)
	assert.Equal(t, 0, counts.SuperAdmin)
	assert.Equal(t, 3, counts.Active)
	assert.Equal(t, 1, counts.Trashed)
}

func TestExtractDateRange_NilQuery(t *testing.T) {
	from, to := extractDateRange(nil)
	assert.True(t, from.Before(to))
	assert.Equal(t, 1, from.Year())
	assert.Equal(t, 9999, to.Year())
}

func ptr[T any](v T) *T { return &v }

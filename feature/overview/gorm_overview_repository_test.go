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
		{Id: "01J000000000000000000000C1", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Active-Degree", FileHash: "0xaa", ExtractEnqueuedAt: ptr(now), ExtractedAt: ptr(now), IssuedAt: now, ActivatedAt: ptr(now)},
		{Id: "01J000000000000000000000C2", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Revoked-Diploma", FileHash: "0xbb", ExtractEnqueuedAt: ptr(now), ExtractedAt: ptr(now), IssuedAt: now.Add(-1 * time.Hour), RevokedAt: ptr(now), RevokerUserId: ptr("01J00000000000000000000002")},
		{Id: "01J000000000000000000000C3", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Pending-Extract", FileHash: "0xcc", ExtractEnqueuedAt: ptr(now), IssuedAt: now.Add(-2 * time.Hour)},
		{Id: "01J000000000000000000000C4", HolderUserId: "01J00000000000000000000001", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Failed-Extract", FileHash: "0xdd", ExtractEnqueuedAt: ptr(now), ExtractFailedAt: ptr(now), IssuedAt: now.Add(-3 * time.Hour)},
		{Id: "01J000000000000000000000C5", HolderUserId: "01J00000000000000000000003", IssuerUserId: ptr("01J00000000000000000000002"), Name: "Admin-Active", FileHash: "0xee", ExtractEnqueuedAt: ptr(now), ExtractedAt: ptr(now), IssuedAt: now.Add(-4 * time.Hour), ActivatedAt: ptr(now)},
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
	assert.Equal(t, 2, counts.Active)
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
	assert.Equal(t, 1, counts.Active)
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

func TestCredentialRecap_GroupedLifecycle(t *testing.T) {
	g := db.OpenInMemorySQLite(t)
	asOf := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	past, future := asOf.Add(-time.Hour), asOf.Add(time.Hour)
	units := []model.UserUnit{{Id: "unit-a", Name: "Same", Active: true}, {Id: "unit-b", Name: "Same", Active: false}}
	require.NoError(t, g.Create(&units).Error)
	users := []model.User{
		{Id: "h1", Email: "h1@test", Role: "holder", UnitID: ptr("unit-a"), JoinedYear: ptr(2025)},
		{Id: "h2", Email: "h2@test", Role: "holder", UnitID: ptr("unit-b"), JoinedYear: ptr(2025)},
		{Id: "h3", Email: "h3@test", Role: "holder"},
		{Id: "admin", Email: "admin@test", Role: "admin"},
		{Id: "deleted", Email: "deleted@test", Role: "holder", DeletedAt: gorm.DeletedAt{Time: past, Valid: true}},
	}
	require.NoError(t, g.Create(&users).Error)
	creds := []model.Credential{
		{Id: "a1", HolderUserId: "h1", Name: "active OCR failure", FileHash: "a1", IssuedAt: past, ActivatedAt: &past, ExtractFailedAt: &past},
		{Id: "a2", HolderUserId: "h1", Name: "active no OCR", FileHash: "a2", IssuedAt: past, ActivatedAt: &past},
		{Id: "expired", HolderUserId: "h1", Name: "equal expiry", FileHash: "expired", IssuedAt: past, ActivatedAt: &past, ExpiresAt: &asOf},
		{Id: "pending", HolderUserId: "h2", Name: "pending", FileHash: "pending", IssuedAt: past},
		{Id: "rejected", HolderUserId: "h2", Name: "rejected expired", FileHash: "rejected", IssuedAt: past, RejectedAt: &past, ExpiresAt: &past},
		{Id: "revoked", HolderUserId: "h2", Name: "revoked rejected expired", FileHash: "revoked", IssuedAt: past, RevokedAt: &past, RejectedAt: &past, ExpiresAt: &past},
		{Id: "excluded1", HolderUserId: "admin", Name: "admin", FileHash: "excluded1", IssuedAt: past, ActivatedAt: &past},
		{Id: "excluded2", HolderUserId: "deleted", Name: "deleted", FileHash: "excluded2", IssuedAt: past, ActivatedAt: &past},
		{Id: "future", HolderUserId: "admin", Name: "future", FileHash: "future", IssuedAt: future},
	}
	require.NoError(t, g.Create(&creds).Error)
	rows, err := NewGormOverviewRepository(g).CredentialRecap(context.Background(), asOf)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	byUnit := map[string]struct{ holders, with, without, activeHolders, withoutActive, total, active, pending, rejected, expired, revoked int }{}
	var totals struct{ holders, with, without, activeHolders, withoutActive, total, active, pending, rejected, expired, revoked int }
	for _, r := range rows {
		totals.holders += r.TotalHolders
		totals.with += r.HoldersWithCredentials
		totals.without += r.HoldersWithoutCredentials
		totals.activeHolders += r.HoldersWithActiveCredentials
		totals.withoutActive += r.HoldersWithoutActiveCredentials
		totals.total += r.TotalCredentials
		totals.active += r.ActiveCredentials
		totals.pending += r.PendingCredentials
		totals.rejected += r.RejectedCredentials
		totals.expired += r.ExpiredCredentials
		totals.revoked += r.RevokedCredentials
		if r.UnitID == nil {
			assert.Nil(t, r.JoinedYear)
		} else {
			assert.Equal(t, 2025, *r.JoinedYear)
		}
		key := "null"
		if r.UnitID != nil {
			key = *r.UnitID
		}
		byUnit[key] = struct{ holders, with, without, activeHolders, withoutActive, total, active, pending, rejected, expired, revoked int }{r.TotalHolders, r.HoldersWithCredentials, r.HoldersWithoutCredentials, r.HoldersWithActiveCredentials, r.HoldersWithoutActiveCredentials, r.TotalCredentials, r.ActiveCredentials, r.PendingCredentials, r.RejectedCredentials, r.ExpiredCredentials, r.RevokedCredentials}
		assert.Equal(t, r.TotalHolders, r.HoldersWithCredentials+r.HoldersWithoutCredentials)
		assert.Equal(t, r.TotalHolders, r.HoldersWithActiveCredentials+r.HoldersWithoutActiveCredentials)
		assert.Equal(t, r.TotalCredentials, r.ActiveCredentials+r.PendingCredentials+r.RejectedCredentials+r.ExpiredCredentials+r.RevokedCredentials)
	}
	assert.Equal(t, 3, totals.holders)
	assert.Equal(t, 2, totals.with)
	assert.Equal(t, 1, totals.without)
	assert.Equal(t, 1, totals.activeHolders)
	assert.Equal(t, 2, totals.withoutActive)
	assert.Equal(t, 6, totals.total)
	assert.Equal(t, 2, totals.active)
	assert.Equal(t, 1, totals.pending)
	assert.Equal(t, 1, totals.rejected)
	assert.Equal(t, 1, totals.expired)
	assert.Equal(t, 1, totals.revoked)
	assert.Equal(t, 1, byUnit["unit-a"].holders)
	assert.Equal(t, 3, byUnit["unit-a"].total)
	assert.Equal(t, 2, byUnit["unit-a"].active)
	assert.Equal(t, 1, byUnit["unit-a"].expired)
	assert.Equal(t, 1, byUnit["unit-a"].activeHolders)
	assert.Equal(t, 3, byUnit["unit-b"].total)
	assert.Equal(t, 1, byUnit["unit-b"].pending)
	assert.Equal(t, 1, byUnit["unit-b"].rejected)
	assert.Equal(t, 1, byUnit["unit-b"].revoked)
	assert.Equal(t, 1, byUnit["null"].without)
	assert.Equal(t, 1, byUnit["null"].withoutActive)
}

func TestCredentialRecap_Empty(t *testing.T) {
	rows, err := NewGormOverviewRepository(db.OpenInMemorySQLite(t)).CredentialRecap(context.Background(), time.Now())
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestCredentialCounts_LifecycleNotExtraction(t *testing.T) {
	g := db.OpenInMemorySQLite(t)
	now := time.Now().UTC()
	expired := now.Add(-time.Hour)
	creds := []model.Credential{
		{Id: "active-no-ocr", HolderUserId: "h1", Name: "active", FileHash: "1", IssuedAt: now, ActivatedAt: &expired},
		{Id: "active-failed-ocr", HolderUserId: "h1", Name: "active", FileHash: "2", IssuedAt: now, ActivatedAt: &expired, ExtractFailedAt: &expired},
		{Id: "expired", HolderUserId: "h1", Name: "expired", FileHash: "3", IssuedAt: now, ActivatedAt: &expired, ExpiresAt: &expired},
		{Id: "rejected", HolderUserId: "h1", Name: "rejected", FileHash: "4", IssuedAt: now, ActivatedAt: &expired, RejectedAt: &expired},
		{Id: "revoked", HolderUserId: "h1", Name: "revoked", FileHash: "5", IssuedAt: now, RevokedAt: &expired},
	}
	require.NoError(t, g.Create(&creds).Error)
	repo := NewGormOverviewRepository(g)
	for _, holderID := range []*string{nil, ptr("h1")} {
		counts, err := repo.CredentialCounts(context.Background(), &domainQuery.Query{}, holderID)
		require.NoError(t, err)
		assert.Equal(t, 2, counts.Active)
		assert.Equal(t, 1, counts.Revoked)
		assert.Equal(t, 1, counts.Failed)
	}
}

func ptr[T any](v T) *T { return &v }

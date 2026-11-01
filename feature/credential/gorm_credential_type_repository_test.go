package credential

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/tests/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openCredentialTypeRepo(t *testing.T) *gormCredentialTypeRepository {
	t.Helper()
	return &gormCredentialTypeRepository{db: db.OpenInMemorySQLite(t)}
}

func TestGormCredentialTypeRepository_CRUD(t *testing.T) {
	repo := openCredentialTypeRepo(t)
	ctx := context.Background()

	stored, err := repo.Store(ctx,
		domain.CredentialType{Id: "t1", Name: "Certificate of Completion", Active: true},
	)
	require.NoError(t, err)
	assert.Len(t, stored, 1)

	found, err := repo.Find(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, "Certificate of Completion", found.Name)
	assert.True(t, found.Active)

	all, _, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	updated, err := repo.Update(ctx, domain.CredentialType{Id: "t1", Name: "Certificate", Active: false})
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "Certificate", updated[0].Name)
	assert.False(t, updated[0].Active)

	deleted, err := repo.Destroy(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = repo.Find(ctx, "t1")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func seedCredentialTypes(t *testing.T, repo *gormCredentialTypeRepository) {
	t.Helper()
	_, err := repo.Store(context.Background(),
		domain.CredentialType{Id: "t1", Name: "Certificate of Completion", Active: true},
		domain.CredentialType{Id: "t2", Name: "Certificate of Attendance", Active: true},
		domain.CredentialType{Id: "t3", Name: "Diploma", Active: false},
	)
	require.NoError(t, err)
}

func TestGormCredentialTypeRepository_Get_Search(t *testing.T) {
	repo := openCredentialTypeRepo(t)
	seedCredentialTypes(t, repo)
	ctx := context.Background()

	got, total, err := repo.Get(ctx, &domainQuery.Query{Search: "certificate"})
	require.NoError(t, err)
	assert.Equal(t, 2, total, "search is case-insensitive")
	assert.Len(t, got, 2)

	got, total, err = repo.Get(ctx, &domainQuery.Query{Search: "nonexistent"})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, got)
}

func TestGormCredentialTypeRepository_Get_FilterByIds(t *testing.T) {
	repo := openCredentialTypeRepo(t)
	seedCredentialTypes(t, repo)

	got, total, err := repo.Get(context.Background(), &domainQuery.Query{
		Filters: []domainQuery.Filter{{
			Column: "id", Operator: domainQuery.OperatorIn, Values: []string{"t1", "t3"},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, got, 2)
	assert.ElementsMatch(t, []string{"t1", "t3"}, []string{got[0].Id, got[1].Id})
}

// total counts every matching row; items respect the page limit.
func TestGormCredentialTypeRepository_Get_TotalIgnoresPagination(t *testing.T) {
	repo := openCredentialTypeRepo(t)
	seedCredentialTypes(t, repo)

	got, total, err := repo.Get(context.Background(), &domainQuery.Query{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, got, 2)
}

func TestGormCredentialTypeRepositoryFindByNames(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCredentialTypeRepository(d)
	ctx := context.Background()

	if _, err := repo.Store(ctx,
		domain.CredentialType{Name: "Diploma", Active: true},
		domain.CredentialType{Name: "Certificate of Completion", Active: true},
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Case-insensitive, whitespace-trimmed, batch — one query for many names.
	got, err := repo.FindByNames(ctx, "  diploma ", "CERTIFICATE OF COMPLETION", "Nonexistent")
	if err != nil {
		t.Fatalf("FindByNames: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2: %+v", len(got), got)
	}
}

func TestGormCredentialTypeRepositoryFindByNamesEmpty(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCredentialTypeRepository(d)

	got, err := repo.FindByNames(context.Background())
	if err != nil {
		t.Fatalf("FindByNames: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d rows, want 0", len(got))
	}
}

func TestGormCredentialTypeSuggestByName(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCredentialTypeRepository(d)
	ctx := context.Background()

	if _, err := repo.Store(ctx,
		domain.CredentialType{Name: "Diploma", Active: true},
		domain.CredentialType{Name: "Certificate of Completion", Active: true},
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := repo.SuggestByName(ctx, "Diplo", 5)
	if err != nil {
		t.Fatalf("SuggestByName: %v", err)
	}
	if len(got) == 0 || got[0].Name != "Diploma" {
		t.Fatalf("want Diploma first, got %+v", got)
	}
}

func TestGormCredentialTypeRepository_Get_HolderFilter(t *testing.T) {
	repo := openCredentialTypeRepo(t)
	now := time.Now()
	holderID := "holder-001"
	otherID := "other-002"

	require.NoError(t, repo.db.Create(&model.CredentialType{
		Id: "t1", Name: "Approved Active", Active: true, ActivatedAt: &now,
	}).Error)
	require.NoError(t, repo.db.Create(&model.CredentialType{
		Id: "t2", Name: "Approved Inactive", Active: false, ActivatedAt: &now,
	}).Error)
	require.NoError(t, repo.db.Create(&model.CredentialType{
		Id: "t3", Name: "Own Pending", Active: false, SubmitterUserId: &holderID,
	}).Error)
	require.NoError(t, repo.db.Create(&model.CredentialType{
		Id: "t4", Name: "Other Pending", Active: false, SubmitterUserId: &otherID,
	}).Error)
	require.NoError(t, repo.db.Create(&model.CredentialType{
		Id: "t5", Name: "Own Rejected", Active: false, SubmitterUserId: &holderID, RejectedAt: &now,
	}).Error)
	require.NoError(t, repo.db.Create(&model.CredentialType{
		Id: "t6", Name: "Other Rejected", Active: false, SubmitterUserId: &otherID, RejectedAt: &now,
	}).Error)
	require.NoError(t, repo.db.Create(&model.CredentialType{
		Id: "t7", Name: "Own Approved Inactive", Active: false, SubmitterUserId: &holderID, ActivatedAt: &now,
	}).Error)

	// Direct repository Get with pure context.Background() and holder_user_id filter
	got, total, err := repo.Get(context.Background(), &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "holder_user_id", Operator: domainQuery.OperatorEqual, Values: []string{holderID}}},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, got, 3)
	ids := []string{got[0].Id, got[1].Id, got[2].Id}
	assert.ElementsMatch(t, []string{"t1", "t3", "t5"}, ids)
}

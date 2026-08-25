package credential

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
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

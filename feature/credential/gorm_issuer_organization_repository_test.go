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

func openIssuerOrgRepo(t *testing.T) *gormIssuerOrganizationRepository {
	t.Helper()
	return &gormIssuerOrganizationRepository{db: db.OpenInMemorySQLite(t)}
}

func TestGormIssuerOrganizationRepository_CRUD(t *testing.T) {
	repo := openIssuerOrgRepo(t)
	ctx := context.Background()

	stored, err := repo.Store(ctx,
		domain.CredentialIssuerOrganization{Id: "o1", Name: "Faculty of Computer Science"},
	)
	require.NoError(t, err)
	assert.Len(t, stored, 1)

	found, err := repo.Find(ctx, "o1")
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Computer Science", found.Name)

	all, _, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	updated, err := repo.Update(ctx, domain.CredentialIssuerOrganization{Id: "o1", Name: "Faculty of Computer Science and Engineering"})
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "Faculty of Computer Science and Engineering", updated[0].Name)

	deleted, err := repo.Destroy(ctx, "o1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = repo.Find(ctx, "o1")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func seedIssuerOrgs(t *testing.T, repo *gormIssuerOrganizationRepository) {
	t.Helper()
	_, err := repo.Store(context.Background(),
		domain.CredentialIssuerOrganization{Id: "o1", Name: "Faculty of Computer Science"},
		domain.CredentialIssuerOrganization{Id: "o2", Name: "Faculty of Law"},
		domain.CredentialIssuerOrganization{Id: "o3", Name: "Student Council"},
	)
	require.NoError(t, err)
}

func TestGormIssuerOrganizationRepository_Get_Search(t *testing.T) {
	repo := openIssuerOrgRepo(t)
	seedIssuerOrgs(t, repo)
	ctx := context.Background()

	got, total, err := repo.Get(ctx, &domainQuery.Query{Search: "faculty"})
	require.NoError(t, err)
	assert.Equal(t, 2, total, "search is case-insensitive")
	assert.Len(t, got, 2)

	got, total, err = repo.Get(ctx, &domainQuery.Query{Search: "nonexistent"})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, got)
}

func TestGormIssuerOrganizationRepository_Get_FilterByIds(t *testing.T) {
	repo := openIssuerOrgRepo(t)
	seedIssuerOrgs(t, repo)

	got, total, err := repo.Get(context.Background(), &domainQuery.Query{
		Filters: []domainQuery.Filter{{
			Column: "id", Operator: domainQuery.OperatorIn, Values: []string{"o1", "o3"},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, got, 2)
	assert.ElementsMatch(t, []string{"o1", "o3"}, []string{got[0].Id, got[1].Id})
}

// total counts every matching row; items respect the page limit.
func TestGormIssuerOrganizationRepository_Get_TotalIgnoresPagination(t *testing.T) {
	repo := openIssuerOrgRepo(t)
	seedIssuerOrgs(t, repo)

	got, total, err := repo.Get(context.Background(), &domainQuery.Query{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, got, 2)
}

func TestGormIssuerOrganizationRepositoryFindByNames(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCredentialIssuerOrganizationRepository(d)
	ctx := context.Background()

	if _, err := repo.Store(ctx,
		domain.CredentialIssuerOrganization{Name: "Universitas Harkat Negeri", Active: true},
		domain.CredentialIssuerOrganization{Name: "BNSP", Active: true},
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Case-insensitive, whitespace-trimmed, batch — one query for many names.
	got, err := repo.FindByNames(ctx, "  universitas harkat negeri ", "bnsp", "Nonexistent")
	if err != nil {
		t.Fatalf("FindByNames: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2: %+v", len(got), got)
	}
}

func TestGormIssuerOrganizationRepositoryFindByNamesEmpty(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCredentialIssuerOrganizationRepository(d)

	got, err := repo.FindByNames(context.Background())
	if err != nil {
		t.Fatalf("FindByNames: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d rows, want 0", len(got))
	}
}

func TestGormIssuerOrganizationSuggestByName(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCredentialIssuerOrganizationRepository(d)
	ctx := context.Background()

	if _, err := repo.Store(ctx,
		domain.CredentialIssuerOrganization{Name: "Universitas Harkat Negeri", Active: true},
		domain.CredentialIssuerOrganization{Name: "BNSP", Active: true},
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := repo.SuggestByName(ctx, "Universitas Harkat", 5)
	if err != nil {
		t.Fatalf("SuggestByName: %v", err)
	}
	if len(got) == 0 || got[0].Name != "Universitas Harkat Negeri" {
		t.Fatalf("want Universitas Harkat Negeri first, got %+v", got)
	}
}

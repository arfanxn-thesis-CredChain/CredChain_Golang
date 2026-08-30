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

func openCompetencyRepo(t *testing.T) *gormCompetencyRepository {
	t.Helper()
	return &gormCompetencyRepository{db: db.OpenInMemorySQLite(t)}
}

func TestGormCompetencyRepository_CRUD(t *testing.T) {
	repo := openCompetencyRepo(t)
	ctx := context.Background()

	stored, err := repo.Store(ctx,
		domain.Competency{Id: "c1", Name: "Data Structures"},
	)
	require.NoError(t, err)
	assert.Len(t, stored, 1)

	found, err := repo.Find(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, "Data Structures", found.Name)

	all, _, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	updated, err := repo.Update(ctx, domain.Competency{Id: "c1", Name: "Data Structures and Algorithms"})
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "Data Structures and Algorithms", updated[0].Name)

	deleted, err := repo.Destroy(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = repo.Find(ctx, "c1")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func seedCompetencies(t *testing.T, repo *gormCompetencyRepository) {
	t.Helper()
	_, err := repo.Store(context.Background(),
		domain.Competency{Id: "c1", Name: "Data Structures"},
		domain.Competency{Id: "c2", Name: "Database Design"},
		domain.Competency{Id: "c3", Name: "Public Speaking"},
	)
	require.NoError(t, err)
}

func TestGormCompetencyRepository_Get_Search(t *testing.T) {
	repo := openCompetencyRepo(t)
	seedCompetencies(t, repo)
	ctx := context.Background()

	got, total, err := repo.Get(ctx, &domainQuery.Query{Search: "data"})
	require.NoError(t, err)
	assert.Equal(t, 2, total, "search is case-insensitive")
	assert.Len(t, got, 2)

	got, total, err = repo.Get(ctx, &domainQuery.Query{Search: "nonexistent"})
	require.NoError(t, err)
	assert.Zero(t, total)
	assert.Empty(t, got)
}

func TestGormCompetencyRepository_Get_FilterByIds(t *testing.T) {
	repo := openCompetencyRepo(t)
	seedCompetencies(t, repo)

	got, total, err := repo.Get(context.Background(), &domainQuery.Query{
		Filters: []domainQuery.Filter{{
			Column: "id", Operator: domainQuery.OperatorIn, Values: []string{"c1", "c3"},
		}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, got, 2)
	assert.ElementsMatch(t, []string{"c1", "c3"}, []string{got[0].Id, got[1].Id})
}

// total counts every matching row; items respect the page limit. The frontend's
// "has more" check depends on the two disagreeing.
func TestGormCompetencyRepository_Get_TotalIgnoresPagination(t *testing.T) {
	repo := openCompetencyRepo(t)
	seedCompetencies(t, repo)

	got, total, err := repo.Get(context.Background(), &domainQuery.Query{Page: 1, Limit: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, got, 2)
}

func TestGormCompetencyRepositoryFindByNames(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCompetencyRepository(d)
	ctx := context.Background()

	if _, err := repo.Store(ctx,
		domain.Competency{Name: "Discrete Mathematics", Active: true},
		domain.Competency{Name: "Linear Algebra", Active: true},
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Case-insensitive, whitespace-trimmed, batch — one query for many names.
	got, err := repo.FindByNames(ctx, "  discrete mathematics ", "LINEAR ALGEBRA", "Nonexistent")
	if err != nil {
		t.Fatalf("FindByNames: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows, want 2: %+v", len(got), got)
	}
}

func TestGormCompetencyRepositoryFindByNamesEmpty(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCompetencyRepository(d)

	got, err := repo.FindByNames(context.Background())
	if err != nil {
		t.Fatalf("FindByNames: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d rows, want 0", len(got))
	}
}

func TestGormCompetencySuggestByName(t *testing.T) {
	d := db.OpenInMemorySQLite(t)
	repo := NewGormCompetencyRepository(d)
	ctx := context.Background()

	if _, err := repo.Store(ctx,
		domain.Competency{Name: "Discrete Mathematics", Active: true},
		domain.Competency{Name: "Database Systems", Active: true},
	); err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := repo.SuggestByName(ctx, "Discrete Math", 5)
	if err != nil {
		t.Fatalf("SuggestByName: %v", err)
	}
	if len(got) == 0 || got[0].Name != "Discrete Mathematics" {
		t.Fatalf("want Discrete Mathematics first, got %+v", got)
	}
}

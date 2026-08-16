package credential

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
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

	all, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	updated, err := repo.Update(ctx, domain.Competency{Id: "c1", Name: "Data Structures and Algorithms"})
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "Data Structures and Algorithms", updated[0].Name)

	deleted, err := repo.Delete(ctx, "c1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = repo.Find(ctx, "c1")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

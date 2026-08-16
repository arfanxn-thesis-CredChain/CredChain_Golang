package credential

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/tests/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openCompetencyCredentialRepo(t *testing.T) *gormCompetencyCredentialRepository {
	t.Helper()
	return &gormCompetencyCredentialRepository{db: db.OpenInMemorySQLite(t)}
}

func TestGormCompetencyCredentialRepository_LinkFindDelete(t *testing.T) {
	repo := openCompetencyCredentialRepo(t)
	ctx := context.Background()

	stored, err := repo.Store(ctx,
		domain.CompetencyCredential{CompetencyId: "comp-1", CredentialId: "cred-1"},
		domain.CompetencyCredential{CompetencyId: "comp-2", CredentialId: "cred-1"},
	)
	require.NoError(t, err)
	assert.Len(t, stored, 2)

	byCred, err := repo.FindByCredentialId(ctx, "cred-1")
	require.NoError(t, err)
	assert.Len(t, byCred, 2)

	byComp, err := repo.FindByCompetencyId(ctx, "comp-1")
	require.NoError(t, err)
	assert.Len(t, byComp, 1)
	assert.Equal(t, "cred-1", byComp[0].CredentialId)

	deleted, err := repo.Delete(ctx, domain.CompetencyCredential{CompetencyId: "comp-1", CredentialId: "cred-1"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	byCred, err = repo.FindByCredentialId(ctx, "cred-1")
	require.NoError(t, err)
	assert.Len(t, byCred, 1)
	assert.Equal(t, "comp-2", byCred[0].CompetencyId)
}

func TestGormCompetencyCredentialRepository_CountByCompetencyIds(t *testing.T) {
	repo := openCompetencyCredentialRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.CompetencyCredential{CompetencyId: "comp-1", CredentialId: "cred-1"},
		domain.CompetencyCredential{CompetencyId: "comp-1", CredentialId: "cred-2"},
		domain.CompetencyCredential{CompetencyId: "comp-2", CredentialId: "cred-1"},
	)
	require.NoError(t, err)

	count, err := repo.CountByCompetencyIds(ctx, "comp-1")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	none, err := repo.CountByCompetencyIds(ctx, "missing")
	require.NoError(t, err)
	assert.Equal(t, int64(0), none)
}

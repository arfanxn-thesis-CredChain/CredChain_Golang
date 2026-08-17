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

	all, err := repo.Get(ctx, nil)
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

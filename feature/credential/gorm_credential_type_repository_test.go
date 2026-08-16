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

	all, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 1)

	updated, err := repo.Update(ctx, domain.CredentialType{Id: "t1", Name: "Certificate", Active: false})
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "Certificate", updated[0].Name)
	assert.False(t, updated[0].Active)

	deleted, err := repo.Delete(ctx, "t1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	_, err = repo.Find(ctx, "t1")
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

package user

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/tests/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openUnitRepo(t *testing.T) *gormUserUnitRepository {
	t.Helper()
	return &gormUserUnitRepository{db: db.OpenInMemorySQLite(t)}
}

func strPtrUnit(s string) *string { return &s }

func TestGormUserUnitRepository_CRUD(t *testing.T) {
	repo := openUnitRepo(t)
	ctx := context.Background()

	stored, err := repo.Store(ctx,
		domain.UserUnit{Id: "faculty", Name: "Faculty of Engineering"},
		domain.UserUnit{Id: "informatics", ParentId: strPtrUnit("faculty"), Name: "Informatics"},
	)
	require.NoError(t, err)
	assert.Len(t, stored, 2)

	found, err := repo.Find(ctx, "faculty")
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Engineering", found.Name)

	all, err := repo.Get(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	updated, err := repo.Update(ctx, domain.UserUnit{Id: "informatics", Name: "Informatics Engineering"})
	require.NoError(t, err)
	assert.Len(t, updated, 1)
	assert.Equal(t, "Informatics Engineering", updated[0].Name)
}

func TestGormUserUnitRepository_FindWithDescendants(t *testing.T) {
	repo := openUnitRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.UserUnit{Id: "root", Name: "University"},
		domain.UserUnit{Id: "faculty", ParentId: strPtrUnit("root"), Name: "Faculty"},
		domain.UserUnit{Id: "dept", ParentId: strPtrUnit("faculty"), Name: "Department"},
		domain.UserUnit{Id: "other", Name: "Other Faculty"},
	)
	require.NoError(t, err)

	got, err := repo.FindWithDescendants(ctx, "faculty")
	require.NoError(t, err)
	require.Len(t, got, 2)
	ids := map[string]bool{}
	for _, u := range got {
		ids[u.Id] = true
	}
	assert.True(t, ids["faculty"], "self included")
	assert.True(t, ids["dept"], "descendant included")
	assert.False(t, ids["root"], "ancestor NOT included")
	assert.False(t, ids["other"], "unrelated branch NOT included")
}

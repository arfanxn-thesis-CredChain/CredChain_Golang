package user

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
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

// seedUnitChain builds root > faculty > dept plus an unrelated sibling branch.
func seedUnitChain(t *testing.T, repo *gormUserUnitRepository) {
	t.Helper()
	_, err := repo.Store(context.Background(),
		domain.UserUnit{Id: "root", Name: "University"},
		domain.UserUnit{Id: "faculty", ParentId: strPtrUnit("root"), Name: "Faculty of Engineering"},
		domain.UserUnit{Id: "dept", ParentId: strPtrUnit("faculty"), Name: "Informatics"},
		domain.UserUnit{Id: "other", Name: "Faculty of Law"},
		domain.UserUnit{Id: "otherdept", ParentId: strPtrUnit("other"), Name: "Civil Law"},
	)
	require.NoError(t, err)
}

func unitIdSet(units []domain.UserUnit) map[string]bool {
	ids := make(map[string]bool, len(units))
	for _, u := range units {
		ids[u.Id] = true
	}
	return ids
}

func TestGormUserUnitRepository_Get_SearchReturnsMatchPlusAncestors(t *testing.T) {
	repo := openUnitRepo(t)
	seedUnitChain(t, repo)

	got, err := repo.Get(context.Background(), &domainQuery.Query{Search: "Informatics"})
	require.NoError(t, err)
	require.Len(t, got, 3, "leaf match returns its whole parent chain")
	ids := unitIdSet(got)
	assert.True(t, ids["dept"], "the match itself")
	assert.True(t, ids["faculty"], "parent")
	assert.True(t, ids["root"], "grandparent")
	assert.False(t, ids["other"], "unrelated branch excluded")
	assert.False(t, ids["otherdept"], "unrelated branch excluded")
}

func TestGormUserUnitRepository_Get_SearchOnRootReturnsWholeSubtree(t *testing.T) {
	repo := openUnitRepo(t)
	seedUnitChain(t, repo)

	got, err := repo.Get(context.Background(), &domainQuery.Query{Search: "University"})
	require.NoError(t, err)
	require.Len(t, got, 3, "matching a root brings its entire subtree")
	ids := unitIdSet(got)
	assert.True(t, ids["root"], "the match itself")
	assert.True(t, ids["faculty"], "child")
	assert.True(t, ids["dept"], "grandchild")
	assert.False(t, ids["other"], "unrelated root excluded")
}

// A mid-tree match needs both directions at once: the parent chain to stay
// reconstructable, the subtree because that is what the user asked to see.
func TestGormUserUnitRepository_Get_SearchOnMidTreeReturnsBothDirections(t *testing.T) {
	repo := openUnitRepo(t)
	seedUnitChain(t, repo)

	got, err := repo.Get(context.Background(), &domainQuery.Query{Search: "Faculty of Engineering"})
	require.NoError(t, err)
	require.Len(t, got, 3)
	ids := unitIdSet(got)
	assert.True(t, ids["root"], "ancestor")
	assert.True(t, ids["faculty"], "the match itself")
	assert.True(t, ids["dept"], "descendant")
	assert.False(t, ids["other"], "sibling branch excluded")
	assert.False(t, ids["otherdept"], "sibling branch excluded")
}

func TestGormUserUnitRepository_Get_SearchIsCaseInsensitive(t *testing.T) {
	repo := openUnitRepo(t)
	seedUnitChain(t, repo)

	got, err := repo.Get(context.Background(), &domainQuery.Query{Search: "iNfOrMaTiCs"})
	require.NoError(t, err)
	assert.True(t, unitIdSet(got)["dept"])
}

// Two sibling matches share the same ancestors; UNION must not duplicate them.
func TestGormUserUnitRepository_Get_SearchDedupesSharedAncestors(t *testing.T) {
	repo := openUnitRepo(t)
	seedUnitChain(t, repo)
	_, err := repo.Store(context.Background(),
		domain.UserUnit{Id: "dept2", ParentId: strPtrUnit("faculty"), Name: "Informatics Systems"},
	)
	require.NoError(t, err)

	got, err := repo.Get(context.Background(), &domainQuery.Query{Search: "Informatics"})
	require.NoError(t, err)
	assert.Len(t, got, 4, "root and faculty appear once each, not once per match")
}

func TestGormUserUnitRepository_Get_SearchMissReturnsEmpty(t *testing.T) {
	repo := openUnitRepo(t)
	seedUnitChain(t, repo)

	got, err := repo.Get(context.Background(), &domainQuery.Query{Search: "Nonexistent"})
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestGormUserUnitRepository_CountByParentIdsAndDestroy(t *testing.T) {
	repo := openUnitRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.UserUnit{Id: "root", Name: "Root"},
		domain.UserUnit{Id: "child1", ParentId: strPtrUnit("root"), Name: "Child 1"},
		domain.UserUnit{Id: "child2", ParentId: strPtrUnit("root"), Name: "Child 2"},
	)
	require.NoError(t, err)

	count, err := repo.CountByParentIds(ctx, "root")
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	deleted, err := repo.Destroy(ctx, "child1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), deleted)

	countAfter, err := repo.CountByParentIds(ctx, "root")
	require.NoError(t, err)
	assert.Equal(t, int64(1), countAfter)
}

func TestGormUserUnitRepository_UpdateParent(t *testing.T) {
	repo := openUnitRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx,
		domain.UserUnit{Id: "root", Name: "Root"},
		domain.UserUnit{Id: "child", ParentId: strPtrUnit("root"), Name: "Child"},
	)
	require.NoError(t, err)

	// nil parentId writes SQL NULL (promote to root).
	require.NoError(t, repo.UpdateParent(ctx, "child", nil))
	got, err := repo.Find(ctx, "child")
	require.NoError(t, err)
	assert.Nil(t, got.ParentId, "nil parentId must write NULL")

	// non-nil parentId writes the value (reparent).
	require.NoError(t, repo.UpdateParent(ctx, "child", strPtrUnit("root")))
	got, err = repo.Find(ctx, "child")
	require.NoError(t, err)
	require.NotNil(t, got.ParentId)
	assert.Equal(t, "root", *got.ParentId)
}

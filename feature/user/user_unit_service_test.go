package user

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newUserUnitServiceTest(t *testing.T) (UserUnitService, *gorm.DB) {
	t.Helper()
	d := db.OpenInMemorySQLite(t)
	unitRepo := NewGormUserUnitRepository(d)
	userRepo := NewGormUserRepository(d)
	return NewUserUnitService(UserUnitServiceParams{
		UnitRepo: unitRepo,
		UserRepo: userRepo,
	}), d
}

func TestUserUnitService_StoreAndPaginate(t *testing.T) {
	svc, _ := newUserUnitServiceTest(t)
	ctx := context.Background()

	root, err := svc.Store(ctx, "Faculty of Engineering", nil)
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Engineering", root.Name)
	assert.Nil(t, root.ParentId)

	child, err := svc.Store(ctx, "Informatics", lo.ToPtr(root.Id))
	require.NoError(t, err)
	assert.Equal(t, root.Id, *child.ParentId)

	list, err := svc.Paginate(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestUserUnitService_Store_MissingParentInvalid(t *testing.T) {
	svc, _ := newUserUnitServiceTest(t)
	ctx := context.Background()

	missing := "01J0000000000000000000000ZZ"
	_, err := svc.Store(ctx, "Orphan Unit", &missing)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUnitParentInvalid, de.Code)
}

func TestUserUnitService_Update_CycleRejected(t *testing.T) {
	svc, _ := newUserUnitServiceTest(t)
	ctx := context.Background()

	root, err := svc.Store(ctx, "Root", nil)
	require.NoError(t, err)
	child, err := svc.Store(ctx, "Child", lo.ToPtr(root.Id))
	require.NoError(t, err)
	grandchild, err := svc.Store(ctx, "Grandchild", lo.ToPtr(child.Id))
	require.NoError(t, err)

	t.Run("self-parent rejected", func(t *testing.T) {
		_, err := svc.Update(ctx, root.Id, nil, lo.ToPtr(root.Id), true)
		var de *domain.Error
		require.ErrorAs(t, err, &de)
		assert.Equal(t, domain.CodeUserUnitParentInvalid, de.Code)
	})

	t.Run("descendant as parent rejected", func(t *testing.T) {
		_, err := svc.Update(ctx, child.Id, nil, lo.ToPtr(grandchild.Id), true)
		var de *domain.Error
		require.ErrorAs(t, err, &de)
		assert.Equal(t, domain.CodeUserUnitParentInvalid, de.Code)
	})
}

func TestUserUnitService_Update_PreservesUntouchedFields(t *testing.T) {
	svc, _ := newUserUnitServiceTest(t)
	ctx := context.Background()

	root, err := svc.Store(ctx, "Root", nil)
	require.NoError(t, err)
	child, err := svc.Store(ctx, "Child", lo.ToPtr(root.Id))
	require.NoError(t, err)

	newName := "Renamed Child"
	updated, err := svc.Update(ctx, child.Id, &newName, nil, false)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.Equal(t, root.Id, *updated.ParentId, "name-only update must preserve the parent")

	other, err := svc.Store(ctx, "Other", nil)
	require.NoError(t, err)
	updated, err = svc.Update(ctx, child.Id, nil, lo.ToPtr(other.Id), true)
	require.NoError(t, err)
	assert.Equal(t, other.Id, *updated.ParentId)
	assert.Equal(t, newName, updated.Name, "parent-only update must preserve the name")
}

func TestUserUnitService_Destroy_ReferencedAndFree(t *testing.T) {
	svc, d := newUserUnitServiceTest(t)
	ctx := context.Background()

	withUser, err := svc.Store(ctx, "Unit With User", nil)
	require.NoError(t, err)
	withChild, err := svc.Store(ctx, "Unit With Child", nil)
	require.NoError(t, err)
	_, err = svc.Store(ctx, "Child Unit", lo.ToPtr(withChild.Id))
	require.NoError(t, err)
	free, err := svc.Store(ctx, "Free Unit", nil)
	require.NoError(t, err)

	user := fixtures.NewModelUser(
		fixtures.WithID("u1"),
		fixtures.WithEmail("u1@example.com"),
		fixtures.WithUnitID(withUser.Id),
	)
	require.NoError(t, d.Create(&user).Error)

	_, err = svc.Destroy(ctx, withUser.Id)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUnitDestroyInUse, de.Code)

	_, err = svc.Destroy(ctx, withChild.Id)
	de = nil
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeUserUnitDestroyInUse, de.Code)

	destroyed, err := svc.Destroy(ctx, free.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), destroyed)

	_, err = svc.Find(ctx, free.Id)
	require.Error(t, err)
}

func TestUserUnitService_Update_MoveToRoot(t *testing.T) {
	svc, _ := newUserUnitServiceTest(t)
	ctx := context.Background()

	root, err := svc.Store(ctx, "Root", nil)
	require.NoError(t, err)
	child, err := svc.Store(ctx, "Child", lo.ToPtr(root.Id))
	require.NoError(t, err)
	require.Equal(t, root.Id, *child.ParentId)

	// setParent=true with a nil parent clears parent_id (promote to root).
	updated, err := svc.Update(ctx, child.Id, nil, nil, true)
	require.NoError(t, err)
	assert.Nil(t, updated.ParentId, "move-to-root must clear parent_id")

	// setParent=false with a nil parent must NOT touch the (now root) parent.
	renamed := "Child Renamed"
	updated, err = svc.Update(ctx, child.Id, &renamed, nil, false)
	require.NoError(t, err)
	assert.Nil(t, updated.ParentId, "name-only update must not touch parent")
	assert.Equal(t, renamed, updated.Name)
}

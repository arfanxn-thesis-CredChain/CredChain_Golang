package credential

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/tests/db"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCredentialTypeServiceTest(t *testing.T) (CredentialTypeService, *gormCredentialTypeRepository, *gormCredentialRepository) {
	t.Helper()
	d := db.OpenInMemorySQLite(t)
	typeRepo := NewGormCredentialTypeRepository(d)
	credRepo := NewGormCredentialRepository(d)
	return NewCredentialTypeService(CredentialTypeServiceParams{
		TypeRepo:       typeRepo,
		CredentialRepo: credRepo,
	}), typeRepo.(*gormCredentialTypeRepository), credRepo.(*gormCredentialRepository)
}

func TestCredentialTypeService_StoreAndPaginate(t *testing.T) {
	svc, _, _ := newCredentialTypeServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Certificate of Completion", nil)
	require.NoError(t, err)
	assert.Equal(t, "Certificate of Completion", created.Name)
	assert.True(t, created.Active)

	list, total, err := svc.Paginate(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, 1, total)

	again, err := svc.Store(ctx, "certificate of completion", nil)
	require.NoError(t, err)
	assert.Equal(t, created.Id, again.Id, "upsert-by-name returns the existing row (case-insensitive)")
}

func TestCredentialTypeService_Store_UpsertTrimsAndMatchesCaseInsensitive(t *testing.T) {
	svc, _, _ := newCredentialTypeServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  Degree  ", nil)
	require.NoError(t, err)
	assert.Equal(t, "Degree", first.Name, "name is trimmed before persistence")

	second, err := svc.Store(ctx, "degree", nil)
	require.NoError(t, err)
	assert.Equal(t, first.Id, second.Id, "case-variant upsert converges on the same row")
}

func TestCredentialTypeService_Update_Deactivate(t *testing.T) {
	svc, _, _ := newCredentialTypeServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Degree", nil)
	require.NoError(t, err)

	updated, err := svc.Update(ctx, created.Id, nil, lo.ToPtr(false))
	require.NoError(t, err)
	assert.False(t, updated.Active)

	newName := "Degree Certificate"
	updated, err = svc.Update(ctx, created.Id, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
}

func TestCredentialTypeService_Update_RenamePreservesActive(t *testing.T) {
	svc, _, _ := newCredentialTypeServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Degree", nil)
	require.NoError(t, err)
	assert.True(t, created.Active)

	newName := "Degree Certificate"
	updated, err := svc.Update(ctx, created.Id, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.True(t, updated.Active, "name-only update must preserve the existing active state")
}

func TestCredentialTypeService_Destroy_ReferencedAndFree(t *testing.T) {
	svc, typeRepo, credRepo := newCredentialTypeServiceTest(t)
	ctx := context.Background()

	referenced, err := svc.Store(ctx, "Referenced Type", nil)
	require.NoError(t, err)
	free, err := svc.Store(ctx, "Free Type", nil)
	require.NoError(t, err)

	require.NoError(t, credRepo.db.Create(&model.Credential{
		Id: "c1", HolderUserId: "h1", SubmitterUserId: "s1", IssuerUserId: "i1",
		IssuerOrganizationId: "o1", TypeId: referenced.Id, Name: "C", FileHash: "0x1",
	}).Error)

	_, err = svc.Destroy(ctx, referenced.Id)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeDestroyInUse, de.Code)

	destroyed, err := svc.Destroy(ctx, free.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), destroyed)

	_, err = typeRepo.Find(ctx, free.Id)
	require.Error(t, err)
}

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

func newCredentialIssuerOrganizationServiceTest(t *testing.T) (CredentialIssuerOrganizationService, *gormIssuerOrganizationRepository, *gormCredentialRepository) {
	t.Helper()
	d := db.OpenInMemorySQLite(t)
	orgRepo := NewGormCredentialIssuerOrganizationRepository(d)
	credRepo := NewGormCredentialRepository(d)
	return NewCredentialIssuerOrganizationService(CredentialIssuerOrganizationServiceParams{
		OrgRepo:        orgRepo,
		CredentialRepo: credRepo,
	}), orgRepo.(*gormIssuerOrganizationRepository), credRepo.(*gormCredentialRepository)
}

func TestCredentialIssuerOrganizationService_StoreAndPaginate(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Faculty of Computer Science", nil)
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Computer Science", created.Name)
	assert.True(t, created.Active)

	list, total, err := svc.Paginate(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, 1, total)

	again, err := svc.Store(ctx, "faculty of computer science", nil)
	require.NoError(t, err)
	assert.Equal(t, created.Id, again.Id, "upsert-by-name returns the existing row (case-insensitive)")
}

func TestCredentialIssuerOrganizationService_Store_UpsertTrimsAndMatchesCaseInsensitive(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  University of Jakarta  ", nil)
	require.NoError(t, err)
	assert.Equal(t, "University of Jakarta", first.Name, "name is trimmed before persistence")

	second, err := svc.Store(ctx, "university of jakarta", nil)
	require.NoError(t, err)
	assert.Equal(t, first.Id, second.Id, "case-variant upsert converges on the same row")
}

func TestCredentialIssuerOrganizationService_Update_RenameTrims(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Faculty of Computer Science", nil)
	require.NoError(t, err)

	newName := "  Faculty of Computer Science and Engineering  "
	updated, err := svc.Update(ctx, created.Id, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Computer Science and Engineering", updated.Name, "rename is trimmed before persistence")

	updated, err = svc.Update(ctx, created.Id, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Computer Science and Engineering", updated.Name, "nil name update must preserve the existing name")
}

func TestCredentialIssuerOrganizationService_Update_Deactivate(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Faculty of Computer Science", nil)
	require.NoError(t, err)

	updated, err := svc.Update(ctx, created.Id, nil, lo.ToPtr(false))
	require.NoError(t, err)
	assert.False(t, updated.Active)

	updated, err = svc.Update(ctx, created.Id, nil, lo.ToPtr(true))
	require.NoError(t, err)
	assert.True(t, updated.Active)
}

func TestCredentialIssuerOrganizationService_Update_RenamePreservesActive(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Faculty of Computer Science", nil)
	require.NoError(t, err)

	deactivated, err := svc.Update(ctx, created.Id, nil, lo.ToPtr(false))
	require.NoError(t, err)
	require.False(t, deactivated.Active)

	newName := "Faculty of Computer Science and Engineering"
	updated, err := svc.Update(ctx, created.Id, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.False(t, updated.Active, "name-only update must preserve the existing active state")
}

func TestCredentialIssuerOrganizationService_Update_NameDuplicate(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "Faculty A", nil)
	require.NoError(t, err)
	second, err := svc.Store(ctx, "Faculty B", nil)
	require.NoError(t, err)

	newName := "faculty a"
	_, err = svc.Update(ctx, second.Id, &newName, nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationNameDuplicate, de.Code)

	_, err = svc.Update(ctx, first.Id, &newName, nil)
	require.NoError(t, err, "renaming to a case-variant of its own name is allowed")
}

func TestCredentialIssuerOrganizationService_Destroy_ReferencedAndFree(t *testing.T) {
	svc, orgRepo, credRepo := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	referenced, err := svc.Store(ctx, "Referenced Organization", nil)
	require.NoError(t, err)
	free, err := svc.Store(ctx, "Free Organization", nil)
	require.NoError(t, err)

	require.NoError(t, credRepo.db.Create(&model.Credential{
		Id: "c1", HolderUserId: "h1", SubmitterUserId: "s1", IssuerUserId: "i1",
		IssuerOrganizationId: referenced.Id, TypeId: "t1", Name: "C", FileHash: "0x1",
	}).Error)

	_, err = svc.Destroy(ctx, referenced.Id)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialIssuerOrganizationDestroyInUse, de.Code)

	destroyed, err := svc.Destroy(ctx, free.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), destroyed)

	_, err = orgRepo.Find(ctx, free.Id)
	require.Error(t, err)
}

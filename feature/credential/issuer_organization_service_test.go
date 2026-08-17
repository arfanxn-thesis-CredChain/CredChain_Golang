package credential

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/tests/db"

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

	created, err := svc.Store(ctx, "Faculty of Computer Science")
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Computer Science", created.Name)

	list, err := svc.Paginate(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	again, err := svc.Store(ctx, "faculty of computer science")
	require.NoError(t, err)
	assert.Equal(t, created.Id, again.Id, "upsert-by-name returns the existing row (case-insensitive)")
}

func TestCredentialIssuerOrganizationService_Store_UpsertTrimsAndMatchesCaseInsensitive(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  University of Jakarta  ")
	require.NoError(t, err)
	assert.Equal(t, "University of Jakarta", first.Name, "name is trimmed before persistence")

	second, err := svc.Store(ctx, "university of jakarta")
	require.NoError(t, err)
	assert.Equal(t, first.Id, second.Id, "case-variant upsert converges on the same row")
}

func TestCredentialIssuerOrganizationService_Update_RenameTrims(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Faculty of Computer Science")
	require.NoError(t, err)

	newName := "  Faculty of Computer Science and Engineering  "
	updated, err := svc.Update(ctx, created.Id, &newName)
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Computer Science and Engineering", updated.Name, "rename is trimmed before persistence")

	updated, err = svc.Update(ctx, created.Id, nil)
	require.NoError(t, err)
	assert.Equal(t, "Faculty of Computer Science and Engineering", updated.Name, "nil name update must preserve the existing name")
}

func TestCredentialIssuerOrganizationService_Update_NameDuplicate(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "Faculty A")
	require.NoError(t, err)
	second, err := svc.Store(ctx, "Faculty B")
	require.NoError(t, err)

	newName := "faculty a"
	_, err = svc.Update(ctx, second.Id, &newName)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationNameDuplicate, de.Code)

	_, err = svc.Update(ctx, first.Id, &newName)
	require.NoError(t, err, "renaming to a case-variant of its own name is allowed")
}

func TestCredentialIssuerOrganizationService_Destroy_ReferencedAndFree(t *testing.T) {
	svc, orgRepo, credRepo := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	referenced, err := svc.Store(ctx, "Referenced Organization")
	require.NoError(t, err)
	free, err := svc.Store(ctx, "Free Organization")
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

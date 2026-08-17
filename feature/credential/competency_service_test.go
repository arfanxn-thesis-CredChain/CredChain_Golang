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

func newCompetencyServiceTest(t *testing.T) (CompetencyService, *gormCompetencyRepository, *gormCompetencyCredentialRepository) {
	t.Helper()
	d := db.OpenInMemorySQLite(t)
	competencyRepo := NewGormCompetencyRepository(d)
	competencyCredentialRepo := NewGormCompetencyCredentialRepository(d)
	return NewCompetencyService(CompetencyServiceParams{
		CompetencyRepo:           competencyRepo,
		CompetencyCredentialRepo: competencyCredentialRepo,
	}), competencyRepo.(*gormCompetencyRepository), competencyCredentialRepo.(*gormCompetencyCredentialRepository)
}

func TestCompetencyService_StoreAndPaginate(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Blockchain Fundamentals")
	require.NoError(t, err)
	assert.Equal(t, "Blockchain Fundamentals", created.Name)

	list, err := svc.Paginate(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	again, err := svc.Store(ctx, "blockchain fundamentals")
	require.NoError(t, err)
	assert.Equal(t, created.Id, again.Id, "upsert-by-name returns the existing row (case-insensitive)")
}

func TestCompetencyService_Store_UpsertTrimsAndMatchesCaseInsensitive(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  Data Structures  ")
	require.NoError(t, err)
	assert.Equal(t, "Data Structures", first.Name, "name is trimmed before persistence")

	second, err := svc.Store(ctx, "data structures")
	require.NoError(t, err)
	assert.Equal(t, first.Id, second.Id, "case-variant upsert converges on the same row")
}

func TestCompetencyService_Update_RenameTrims(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Blockchain Fundamentals")
	require.NoError(t, err)

	newName := "  Blockchain Fundamentals and Applications  "
	updated, err := svc.Update(ctx, created.Id, &newName)
	require.NoError(t, err)
	assert.Equal(t, "Blockchain Fundamentals and Applications", updated.Name, "rename is trimmed before persistence")

	updated, err = svc.Update(ctx, created.Id, nil)
	require.NoError(t, err)
	assert.Equal(t, "Blockchain Fundamentals and Applications", updated.Name, "nil name update must preserve the existing name")
}

func TestCompetencyService_Update_NameDuplicate(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "Competency A")
	require.NoError(t, err)
	second, err := svc.Store(ctx, "Competency B")
	require.NoError(t, err)

	newName := "competency a"
	_, err = svc.Update(ctx, second.Id, &newName)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyNameDuplicate, de.Code)

	_, err = svc.Update(ctx, first.Id, &newName)
	require.NoError(t, err, "renaming to a case-variant of its own name is allowed")
}

func TestCompetencyService_Destroy_ReferencedAndFree(t *testing.T) {
	svc, competencyRepo, competencyCredentialRepo := newCompetencyServiceTest(t)
	ctx := context.Background()

	referenced, err := svc.Store(ctx, "Referenced Competency")
	require.NoError(t, err)
	free, err := svc.Store(ctx, "Free Competency")
	require.NoError(t, err)

	require.NoError(t, competencyCredentialRepo.db.Create(&model.CompetencyCredential{
		CompetencyId: referenced.Id, CredentialId: "cred-x",
	}).Error)

	_, err = svc.Destroy(ctx, referenced.Id)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyDestroyInUse, de.Code)

	destroyed, err := svc.Destroy(ctx, free.Id)
	require.NoError(t, err)
	assert.Equal(t, int64(1), destroyed)

	_, err = competencyRepo.Find(ctx, free.Id)
	require.Error(t, err)
}

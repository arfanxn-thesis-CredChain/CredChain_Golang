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

	created, err := svc.Store(ctx, "Blockchain Fundamentals", nil)
	require.NoError(t, err)
	assert.Equal(t, "Blockchain Fundamentals", created.Name)
	assert.True(t, created.Active)

	list, total, err := svc.Paginate(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, 1, total)

	again, err := svc.Store(ctx, "blockchain fundamentals", nil)
	require.NoError(t, err)
	assert.Equal(t, created.Id, again.Id, "upsert-by-name returns the existing row (case-insensitive)")
}

func TestCompetencyService_Store_UpsertTrimsAndMatchesCaseInsensitive(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  Data Structures  ", nil)
	require.NoError(t, err)
	assert.Equal(t, "Data Structures", first.Name, "name is trimmed before persistence")

	second, err := svc.Store(ctx, "data structures", nil)
	require.NoError(t, err)
	assert.Equal(t, first.Id, second.Id, "case-variant upsert converges on the same row")
}

func TestCompetencyService_Update_RenameTrims(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Blockchain Fundamentals", nil)
	require.NoError(t, err)

	newName := "  Blockchain Fundamentals and Applications  "
	updated, err := svc.Update(ctx, created.Id, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, "Blockchain Fundamentals and Applications", updated.Name, "rename is trimmed before persistence")

	updated, err = svc.Update(ctx, created.Id, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "Blockchain Fundamentals and Applications", updated.Name, "nil name update must preserve the existing name")
}

func TestCompetencyService_Update_Deactivate(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Blockchain Fundamentals", nil)
	require.NoError(t, err)

	updated, err := svc.Update(ctx, created.Id, nil, lo.ToPtr(false))
	require.NoError(t, err)
	assert.False(t, updated.Active)

	updated, err = svc.Update(ctx, created.Id, nil, lo.ToPtr(true))
	require.NoError(t, err)
	assert.True(t, updated.Active)
}

func TestCompetencyService_Update_RenamePreservesActive(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	created, err := svc.Store(ctx, "Blockchain Fundamentals", nil)
	require.NoError(t, err)

	deactivated, err := svc.Update(ctx, created.Id, nil, lo.ToPtr(false))
	require.NoError(t, err)
	require.False(t, deactivated.Active)

	newName := "Blockchain Fundamentals and Applications"
	updated, err := svc.Update(ctx, created.Id, &newName, nil)
	require.NoError(t, err)
	assert.Equal(t, newName, updated.Name)
	assert.False(t, updated.Active, "name-only update must preserve the existing active state")
}

func TestCompetencyService_Update_NameDuplicate(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "Competency A", nil)
	require.NoError(t, err)
	second, err := svc.Store(ctx, "Competency B", nil)
	require.NoError(t, err)

	newName := "competency a"
	_, err = svc.Update(ctx, second.Id, &newName, nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyNameDuplicate, de.Code)

	_, err = svc.Update(ctx, first.Id, &newName, nil)
	require.NoError(t, err, "renaming to a case-variant of its own name is allowed")
}

func TestCompetencyService_Destroy_ReferencedAndFree(t *testing.T) {
	svc, competencyRepo, competencyCredentialRepo := newCompetencyServiceTest(t)
	ctx := context.Background()

	referenced, err := svc.Store(ctx, "Referenced Competency", nil)
	require.NoError(t, err)
	free, err := svc.Store(ctx, "Free Competency", nil)
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

package credential

import (
	"context"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

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
		UoW:                      referenceTestUnitOfWork(d),
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

	_, err = svc.Store(ctx, "blockchain fundamentals", nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyNameDuplicate, de.Code)
}

func TestCompetencyService_Store_TrimsAndEnforcesCaseInsensitiveUnique(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  Data Structures  ", nil)
	require.NoError(t, err)
	assert.Equal(t, "Data Structures", first.Name, "name is trimmed before persistence")

	_, err = svc.Store(ctx, "data structures", nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyNameDuplicate, de.Code)
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

func TestCompetencyService_Submit_Approve_Reject_Lifecycle(t *testing.T) {
	svc, _, _ := newCompetencyServiceTest(t)
	holder := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))

	holderCtx := httpContext.WithUser(context.Background(), &holder)
	issuerCtx := httpContext.WithUser(context.Background(), &issuer)

	// 1. Submit by holder creates pending inactive row
	submitted, err := svc.Submit(holderCtx, "Distributed Systems")
	require.NoError(t, err)
	assert.Equal(t, "Distributed Systems", submitted.Name)
	assert.False(t, submitted.Active)
	assert.Equal(t, domain.ReferenceStatusPending, submitted.Status())
	require.NotNil(t, submitted.SubmitterUserId)
	assert.Equal(t, holder.Id, *submitted.SubmitterUserId)
	assert.Nil(t, submitted.ActivatedAt)
	assert.Nil(t, submitted.RejectedAt)

	// 2. Submitting duplicate pending returns CodeCompetencyAlreadyPending
	_, err = svc.Submit(holderCtx, "distributed systems")
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyAlreadyPending, de.Code)

	// 3. Direct Store cannot bypass pending proposal
	_, err = svc.Store(issuerCtx, "distributed systems", nil)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyAlreadyPending, de.Code)

	// 4. Issuer approves proposal
	approved, err := svc.Approve(issuerCtx, submitted.Id)
	require.NoError(t, err)
	assert.True(t, approved.Active)
	assert.Equal(t, domain.ReferenceStatusApproved, approved.Status())
	require.NotNil(t, approved.ApproverUserId)
	assert.Equal(t, issuer.Id, *approved.ApproverUserId)
	assert.NotNil(t, approved.ActivatedAt)

	// 5. Approving again returns CodeCompetencyAlreadyApproved
	_, err = svc.Approve(issuerCtx, submitted.Id)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyAlreadyApproved, de.Code)

	// 6. Submitting name that is already approved returns CodeCompetencyAlreadyApproved
	_, err = svc.Submit(holderCtx, "distributed systems")
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyAlreadyApproved, de.Code)

	// 7. Rejecting an approved row returns CodeCompetencyAlreadyApproved
	_, err = svc.Reject(issuerCtx, submitted.Id, "should fail")
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyAlreadyApproved, de.Code)

	// 8. Submit another proposal and reject it
	proposal2, err := svc.Submit(holderCtx, "Quantum Teleportation")
	require.NoError(t, err)

	rejected, err := svc.Reject(issuerCtx, proposal2.Id, "Not a valid verified skill")
	require.NoError(t, err)
	assert.False(t, rejected.Active)
	assert.Equal(t, domain.ReferenceStatusRejected, rejected.Status())
	require.NotNil(t, rejected.RejecterUserId)
	assert.Equal(t, issuer.Id, *rejected.RejecterUserId)
	require.NotNil(t, rejected.RejectionReason)
	assert.Equal(t, "Not a valid verified skill", *rejected.RejectionReason)
	assert.NotNil(t, rejected.RejectedAt)

	// 9. Approving rejected row fails
	_, err = svc.Approve(issuerCtx, proposal2.Id)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCompetencyAlreadyRejected, de.Code)

	// 10. Re-submitting rejected name creates a NEW pending row
	resubmitted, err := svc.Submit(holderCtx, "quantum teleportation")
	require.NoError(t, err)
	assert.NotEqual(t, proposal2.Id, resubmitted.Id)
	assert.Equal(t, domain.ReferenceStatusPending, resubmitted.Status())
}

func TestCompetencyService_Paginate_HolderVisibility(t *testing.T) {
	svc, competencyRepo, _ := newCompetencyServiceTest(t)
	now := time.Now()
	holderID := "holder-001"
	otherID := "other-002"

	require.NoError(t, competencyRepo.db.Create(&model.Competency{
		Id: "k1", Name: "Approved Active", Active: true, ActivatedAt: &now,
	}).Error)
	require.NoError(t, competencyRepo.db.Create(&model.Competency{
		Id: "k2", Name: "Approved Inactive", Active: false, ActivatedAt: &now,
	}).Error)
	require.NoError(t, competencyRepo.db.Create(&model.Competency{
		Id: "k3", Name: "Own Pending", Active: false, SubmitterUserId: &holderID,
	}).Error)
	require.NoError(t, competencyRepo.db.Create(&model.Competency{
		Id: "k4", Name: "Other Pending", Active: false, SubmitterUserId: &otherID,
	}).Error)
	require.NoError(t, competencyRepo.db.Create(&model.Competency{
		Id: "k5", Name: "Own Rejected", Active: false, SubmitterUserId: &holderID, RejectedAt: &now,
	}).Error)
	require.NoError(t, competencyRepo.db.Create(&model.Competency{
		Id: "k6", Name: "Other Rejected", Active: false, SubmitterUserId: &otherID, RejectedAt: &now,
	}).Error)
	require.NoError(t, competencyRepo.db.Create(&model.Competency{
		Id: "k7", Name: "Own Approved Inactive", Active: false, SubmitterUserId: &holderID, ActivatedAt: &now,
	}).Error)

	holderUser := domain.User{Id: holderID, Role: domain.RoleHolder}
	holderCtx := httpContext.WithUser(context.Background(), &holderUser)

	// 1. Default Paginate for Holder sees k1, k3, k5 only
	list, total, err := svc.Paginate(holderCtx, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, list, 3)
	ids := lo.Map(list, func(item domain.Competency, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"k1", "k3", "k5"}, ids)

	// 2. Holder filtering for active=false sees only own pending and rejected (k3, k5)
	list, total, err = svc.Paginate(holderCtx, &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "active", Operator: domainQuery.OperatorEqual, Values: []string{"false"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, list, 2)
	ids = lo.Map(list, func(item domain.Competency, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"k3", "k5"}, ids)

	// 3. Holder querying by ID for inactive references (k2, k7) gets 0 results
	list, total, err = svc.Paginate(holderCtx, &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "id", Operator: domainQuery.OperatorIn, Values: []string{"k2", "k7"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, list)

	// 4. Client spoofing attempt: client passes holder_user_id=other-002; service overrides it with authenticated user's ID
	list, total, err = svc.Paginate(holderCtx, &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "holder_user_id", Operator: domainQuery.OperatorEqual, Values: []string{otherID}}},
	})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	ids = lo.Map(list, func(item domain.Competency, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"k1", "k3", "k5"}, ids)

	// 5. Issuer sees ALL 7 items
	issuerUser := domain.User{Id: "issuer-001", Role: domain.RoleIssuer}
	issuerCtx := httpContext.WithUser(context.Background(), &issuerUser)
	list, total, err = svc.Paginate(issuerCtx, nil)
	require.NoError(t, err)
	assert.Equal(t, 7, total)
	assert.Len(t, list, 7)
}

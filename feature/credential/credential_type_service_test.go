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

func newCredentialTypeServiceTest(t *testing.T) (CredentialTypeService, *gormCredentialTypeRepository, *gormCredentialRepository) {
	t.Helper()
	d := db.OpenInMemorySQLite(t)
	typeRepo := NewGormCredentialTypeRepository(d)
	credRepo := NewGormCredentialRepository(d)
	return NewCredentialTypeService(CredentialTypeServiceParams{
		UoW:            referenceTestUnitOfWork(d),
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

	_, err = svc.Store(ctx, "certificate of completion", nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeNameDuplicate, de.Code)
}

func TestCredentialTypeService_Store_TrimsAndEnforcesCaseInsensitiveUnique(t *testing.T) {
	svc, _, _ := newCredentialTypeServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  Degree  ", nil)
	require.NoError(t, err)
	assert.Equal(t, "Degree", first.Name, "name is trimmed before persistence")

	_, err = svc.Store(ctx, "degree", nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeNameDuplicate, de.Code)
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
		Id: "c1", HolderUserId: "h1", SubmitterUserId: "s1", IssuerUserId: lo.ToPtr("i1"),
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

func TestCredentialTypeService_Submit_Approve_Reject_Lifecycle(t *testing.T) {
	svc, _, _ := newCredentialTypeServiceTest(t)
	holder := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))

	holderCtx := httpContext.WithUser(context.Background(), &holder)
	issuerCtx := httpContext.WithUser(context.Background(), &issuer)

	// 1. Submit by holder creates pending inactive row
	submitted, err := svc.Submit(holderCtx, "Master of Science")
	require.NoError(t, err)
	assert.Equal(t, "Master of Science", submitted.Name)
	assert.False(t, submitted.Active)
	assert.Equal(t, domain.ReferenceStatusPending, submitted.Status())
	require.NotNil(t, submitted.SubmitterUserId)
	assert.Equal(t, holder.Id, *submitted.SubmitterUserId)
	assert.Nil(t, submitted.ActivatedAt)
	assert.Nil(t, submitted.RejectedAt)

	// 2. Submitting duplicate pending returns CodeCredentialTypeAlreadyPending
	_, err = svc.Submit(holderCtx, "master of science")
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeAlreadyPending, de.Code)

	// 3. Direct Store cannot bypass pending proposal
	_, err = svc.Store(issuerCtx, "master of science", nil)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeAlreadyPending, de.Code)

	// 4. Issuer approves proposal
	approved, err := svc.Approve(issuerCtx, submitted.Id)
	require.NoError(t, err)
	assert.True(t, approved.Active)
	assert.Equal(t, domain.ReferenceStatusApproved, approved.Status())
	require.NotNil(t, approved.ApproverUserId)
	assert.Equal(t, issuer.Id, *approved.ApproverUserId)
	assert.NotNil(t, approved.ActivatedAt)

	// 5. Approving again returns CodeCredentialTypeAlreadyApproved
	_, err = svc.Approve(issuerCtx, submitted.Id)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeAlreadyApproved, de.Code)

	// 6. Submitting name that is already approved returns CodeCredentialTypeAlreadyApproved
	_, err = svc.Submit(holderCtx, "master of science")
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeAlreadyApproved, de.Code)

	// 7. Rejecting an approved row returns CodeCredentialTypeAlreadyApproved
	_, err = svc.Reject(issuerCtx, submitted.Id, "should fail")
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeAlreadyApproved, de.Code)

	// 8. Submit another proposal and reject it
	proposal2, err := svc.Submit(holderCtx, "PhD in Robotics")
	require.NoError(t, err)

	rejected, err := svc.Reject(issuerCtx, proposal2.Id, "Scope too narrow")
	require.NoError(t, err)
	assert.False(t, rejected.Active)
	assert.Equal(t, domain.ReferenceStatusRejected, rejected.Status())
	require.NotNil(t, rejected.RejecterUserId)
	assert.Equal(t, issuer.Id, *rejected.RejecterUserId)
	require.NotNil(t, rejected.RejectionReason)
	assert.Equal(t, "Scope too narrow", *rejected.RejectionReason)
	assert.NotNil(t, rejected.RejectedAt)

	// 9. Approving rejected row fails
	_, err = svc.Approve(issuerCtx, proposal2.Id)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeCredentialTypeAlreadyRejected, de.Code)

	// 10. Re-submitting rejected name creates a NEW pending row
	resubmitted, err := svc.Submit(holderCtx, "phd in robotics")
	require.NoError(t, err)
	assert.NotEqual(t, proposal2.Id, resubmitted.Id)
	assert.Equal(t, domain.ReferenceStatusPending, resubmitted.Status())
}

func TestCredentialTypeService_Paginate_HolderVisibility(t *testing.T) {
	svc, typeRepo, _ := newCredentialTypeServiceTest(t)
	now := time.Now()
	holderID := "holder-001"
	otherID := "other-002"

	require.NoError(t, typeRepo.db.Create(&model.CredentialType{
		Id: "t1", Name: "Approved Active", Active: true, ActivatedAt: &now,
	}).Error)
	require.NoError(t, typeRepo.db.Create(&model.CredentialType{
		Id: "t2", Name: "Approved Inactive", Active: false, ActivatedAt: &now,
	}).Error)
	require.NoError(t, typeRepo.db.Create(&model.CredentialType{
		Id: "t3", Name: "Own Pending", Active: false, SubmitterUserId: &holderID,
	}).Error)
	require.NoError(t, typeRepo.db.Create(&model.CredentialType{
		Id: "t4", Name: "Other Pending", Active: false, SubmitterUserId: &otherID,
	}).Error)
	require.NoError(t, typeRepo.db.Create(&model.CredentialType{
		Id: "t5", Name: "Own Rejected", Active: false, SubmitterUserId: &holderID, RejectedAt: &now,
	}).Error)
	require.NoError(t, typeRepo.db.Create(&model.CredentialType{
		Id: "t6", Name: "Other Rejected", Active: false, SubmitterUserId: &otherID, RejectedAt: &now,
	}).Error)
	require.NoError(t, typeRepo.db.Create(&model.CredentialType{
		Id: "t7", Name: "Own Approved Inactive", Active: false, SubmitterUserId: &holderID, ActivatedAt: &now,
	}).Error)

	holderUser := domain.User{Id: holderID, Role: domain.RoleHolder}
	holderCtx := httpContext.WithUser(context.Background(), &holderUser)

	// 1. Default Paginate for Holder sees t1, t3, t5 only
	list, total, err := svc.Paginate(holderCtx, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, list, 3)
	ids := lo.Map(list, func(item domain.CredentialType, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"t1", "t3", "t5"}, ids)

	// 2. Holder filtering for active=false sees only own pending and rejected (t3, t5)
	list, total, err = svc.Paginate(holderCtx, &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "active", Operator: domainQuery.OperatorEqual, Values: []string{"false"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, list, 2)
	ids = lo.Map(list, func(item domain.CredentialType, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"t3", "t5"}, ids)

	// 3. Holder querying by ID for inactive references (t2, t7) gets 0 results
	list, total, err = svc.Paginate(holderCtx, &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "id", Operator: domainQuery.OperatorIn, Values: []string{"t2", "t7"}}},
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
	ids = lo.Map(list, func(item domain.CredentialType, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"t1", "t3", "t5"}, ids)

	// 5. Issuer sees ALL 7 items
	issuerUser := domain.User{Id: "issuer-001", Role: domain.RoleIssuer}
	issuerCtx := httpContext.WithUser(context.Background(), &issuerUser)
	list, total, err = svc.Paginate(issuerCtx, nil)
	require.NoError(t, err)
	assert.Equal(t, 7, total)
	assert.Len(t, list, 7)
}

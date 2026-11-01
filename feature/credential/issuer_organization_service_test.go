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

	_, err = svc.Store(ctx, "faculty of computer science", nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationNameDuplicate, de.Code)
}

func TestCredentialIssuerOrganizationService_Store_TrimsAndEnforcesCaseInsensitiveUnique(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	ctx := context.Background()

	first, err := svc.Store(ctx, "  University of Jakarta  ", nil)
	require.NoError(t, err)
	assert.Equal(t, "University of Jakarta", first.Name, "name is trimmed before persistence")

	_, err = svc.Store(ctx, "university of jakarta", nil)
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationNameDuplicate, de.Code)
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
		Id: "c1", HolderUserId: "h1", SubmitterUserId: "s1", IssuerUserId: lo.ToPtr("i1"),
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

func TestCredentialIssuerOrganizationService_Submit_Approve_Reject_Lifecycle(t *testing.T) {
	svc, _, _ := newCredentialIssuerOrganizationServiceTest(t)
	holder := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	issuer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))

	holderCtx := httpContext.WithUser(context.Background(), &holder)
	issuerCtx := httpContext.WithUser(context.Background(), &issuer)

	// 1. Submit by holder creates pending inactive row
	submitted, err := svc.Submit(holderCtx, "MIT Open Learning")
	require.NoError(t, err)
	assert.Equal(t, "MIT Open Learning", submitted.Name)
	assert.False(t, submitted.Active)
	assert.Equal(t, domain.ReferenceStatusPending, submitted.Status())
	require.NotNil(t, submitted.SubmitterUserId)
	assert.Equal(t, holder.Id, *submitted.SubmitterUserId)
	assert.Nil(t, submitted.ActivatedAt)
	assert.Nil(t, submitted.RejectedAt)

	// 2. Submitting duplicate pending returns CodeIssuerOrganizationAlreadyPending
	_, err = svc.Submit(holderCtx, "mit open learning")
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationAlreadyPending, de.Code)

	// 3. Direct Store cannot bypass pending proposal
	_, err = svc.Store(issuerCtx, "mit open learning", nil)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationAlreadyPending, de.Code)

	// 4. Issuer approves proposal
	approved, err := svc.Approve(issuerCtx, submitted.Id)
	require.NoError(t, err)
	assert.True(t, approved.Active)
	assert.Equal(t, domain.ReferenceStatusApproved, approved.Status())
	require.NotNil(t, approved.ApproverUserId)
	assert.Equal(t, issuer.Id, *approved.ApproverUserId)
	assert.NotNil(t, approved.ActivatedAt)

	// 5. Approving again returns CodeIssuerOrganizationAlreadyApproved
	_, err = svc.Approve(issuerCtx, submitted.Id)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationAlreadyApproved, de.Code)

	// 6. Submitting name that is already approved returns CodeIssuerOrganizationAlreadyApproved
	_, err = svc.Submit(holderCtx, "mit open learning")
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationAlreadyApproved, de.Code)

	// 7. Rejecting an approved row returns CodeIssuerOrganizationAlreadyApproved
	_, err = svc.Reject(issuerCtx, submitted.Id, "should fail")
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationAlreadyApproved, de.Code)

	// 8. Submit another proposal and reject it
	proposal2, err := svc.Submit(holderCtx, "Fake University")
	require.NoError(t, err)

	rejected, err := svc.Reject(issuerCtx, proposal2.Id, "Unaccredited organization")
	require.NoError(t, err)
	assert.False(t, rejected.Active)
	assert.Equal(t, domain.ReferenceStatusRejected, rejected.Status())
	require.NotNil(t, rejected.RejecterUserId)
	assert.Equal(t, issuer.Id, *rejected.RejecterUserId)
	require.NotNil(t, rejected.RejectionReason)
	assert.Equal(t, "Unaccredited organization", *rejected.RejectionReason)
	assert.NotNil(t, rejected.RejectedAt)

	// 9. Approving rejected row fails
	_, err = svc.Approve(issuerCtx, proposal2.Id)
	require.ErrorAs(t, err, &de)
	assert.Equal(t, domain.CodeIssuerOrganizationAlreadyRejected, de.Code)

	// 10. Re-submitting rejected name creates a NEW pending row
	resubmitted, err := svc.Submit(holderCtx, "fake university")
	require.NoError(t, err)
	assert.NotEqual(t, proposal2.Id, resubmitted.Id)
	assert.Equal(t, domain.ReferenceStatusPending, resubmitted.Status())
}

func TestIssuerOrganizationService_Paginate_HolderVisibility(t *testing.T) {
	svc, orgRepo, _ := newCredentialIssuerOrganizationServiceTest(t)
	now := time.Now()
	holderID := "holder-001"
	otherID := "other-002"

	require.NoError(t, orgRepo.db.Create(&model.CredentialIssuerOrganization{
		Id: "o1", Name: "Approved Active", Active: true, ActivatedAt: &now,
	}).Error)
	require.NoError(t, orgRepo.db.Create(&model.CredentialIssuerOrganization{
		Id: "o2", Name: "Approved Inactive", Active: false, ActivatedAt: &now,
	}).Error)
	require.NoError(t, orgRepo.db.Create(&model.CredentialIssuerOrganization{
		Id: "o3", Name: "Own Pending", Active: false, SubmitterUserId: &holderID,
	}).Error)
	require.NoError(t, orgRepo.db.Create(&model.CredentialIssuerOrganization{
		Id: "o4", Name: "Other Pending", Active: false, SubmitterUserId: &otherID,
	}).Error)
	require.NoError(t, orgRepo.db.Create(&model.CredentialIssuerOrganization{
		Id: "o5", Name: "Own Rejected", Active: false, SubmitterUserId: &holderID, RejectedAt: &now,
	}).Error)
	require.NoError(t, orgRepo.db.Create(&model.CredentialIssuerOrganization{
		Id: "o6", Name: "Other Rejected", Active: false, SubmitterUserId: &otherID, RejectedAt: &now,
	}).Error)
	require.NoError(t, orgRepo.db.Create(&model.CredentialIssuerOrganization{
		Id: "o7", Name: "Own Approved Inactive", Active: false, SubmitterUserId: &holderID, ActivatedAt: &now,
	}).Error)

	holderUser := domain.User{Id: holderID, Role: domain.RoleHolder}
	holderCtx := httpContext.WithUser(context.Background(), &holderUser)

	// 1. Default Paginate for Holder sees o1, o3, o5 only
	list, total, err := svc.Paginate(holderCtx, nil)
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	require.Len(t, list, 3)
	ids := lo.Map(list, func(item domain.CredentialIssuerOrganization, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"o1", "o3", "o5"}, ids)

	// 2. Holder filtering for active=false sees only own pending and rejected (o3, o5)
	list, total, err = svc.Paginate(holderCtx, &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "active", Operator: domainQuery.OperatorEqual, Values: []string{"false"}}},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, list, 2)
	ids = lo.Map(list, func(item domain.CredentialIssuerOrganization, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"o3", "o5"}, ids)

	// 3. Holder querying by ID for inactive references (o2, o7) gets 0 results
	list, total, err = svc.Paginate(holderCtx, &domainQuery.Query{
		Filters: []domainQuery.Filter{{Column: "id", Operator: domainQuery.OperatorIn, Values: []string{"o2", "o7"}}},
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
	ids = lo.Map(list, func(item domain.CredentialIssuerOrganization, _ int) string { return item.Id })
	assert.ElementsMatch(t, []string{"o1", "o3", "o5"}, ids)

	// 5. Issuer sees ALL 7 items
	issuerUser := domain.User{Id: "issuer-001", Role: domain.RoleIssuer}
	issuerCtx := httpContext.WithUser(context.Background(), &issuerUser)
	list, total, err = svc.Paginate(issuerCtx, nil)
	require.NoError(t, err)
	assert.Equal(t, 7, total)
	assert.Len(t, list, 7)
}

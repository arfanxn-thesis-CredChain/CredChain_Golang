package credential

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/feature/notification"
	"CredChain_Golang/feature/user"
	gormInfra "CredChain_Golang/infrastructure/database/gorm"
	"CredChain_Golang/infrastructure/database/gorm/model"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func referenceTestUnitOfWork(d *gorm.DB) domain.UnitOfWork {
	return gormInfra.NewGormUnitOfWork(d, user.NewGormUserRepository, NewGormCredentialRepository,
		user.NewGormUserTokenRepository, NewGormCompetencyCredentialRepository,
		notification.NewGormNotificationRepository, NewGormCredentialTypeRepository,
		NewGormCredentialIssuerOrganizationRepository, NewGormCompetencyRepository)
}

func TestReferenceSubmissionNotifications(t *testing.T) {
	for _, resource := range []string{"credential_type", "credential_issuer_organization", "competency"} {
		for _, failNotifications := range []bool{false, true} {
			t.Run(resource+map[bool]string{false: "/fanout", true: "/rollback"}[failNotifications], func(t *testing.T) {
				d := db.OpenInMemorySQLite(t)
				require.NoError(t, d.AutoMigrate(&model.Notification{}))
				users := user.NewGormUserRepository(d)
				now := time.Now()
				submitter := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
				deleted := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
				deleted.DeletedAt = &now
				_, err := users.Store(context.Background(), submitter, deleted,
					fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder)),
					fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer)),
					fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin)),
					fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin)))
				require.NoError(t, err)
				uow := gormInfra.NewGormUnitOfWork(d, user.NewGormUserRepository, NewGormCredentialRepository,
					user.NewGormUserTokenRepository, NewGormCompetencyCredentialRepository,
					notification.NewGormNotificationRepository, NewGormCredentialTypeRepository,
					NewGormCredentialIssuerOrganizationRepository, NewGormCompetencyRepository)
				ctx := httpContext.WithUser(context.Background(), &submitter)
				var submit func(context.Context, string) error
				var table string
				switch resource {
				case "credential_type":
					svc := NewCredentialTypeService(CredentialTypeServiceParams{TypeRepo: NewGormCredentialTypeRepository(d), UoW: uow})
					submit = func(ctx context.Context, name string) error { _, err := svc.Submit(ctx, name); return err }
					table = "credential_types"
				case "credential_issuer_organization":
					svc := NewCredentialIssuerOrganizationService(CredentialIssuerOrganizationServiceParams{OrgRepo: NewGormCredentialIssuerOrganizationRepository(d), UoW: uow})
					submit = func(ctx context.Context, name string) error { _, err := svc.Submit(ctx, name); return err }
					table = "credential_issuer_organizations"
				case "competency":
					svc := NewCompetencyService(CompetencyServiceParams{CompetencyRepo: NewGormCompetencyRepository(d), UoW: uow})
					submit = func(ctx context.Context, name string) error { _, err := svc.Submit(ctx, name); return err }
					table = "competencies"
				}
				if failNotifications {
					require.NoError(t, d.Migrator().DropTable(&model.Notification{}))
				}
				err = submit(ctx, "New reference")
				var count int64
				require.NoError(t, d.Table(table).Where("name = ?", "New reference").Count(&count).Error)
				if failNotifications {
					require.Error(t, err)
					require.Zero(t, count)
					return
				}
				require.NoError(t, err)
				require.EqualValues(t, 1, count)
				var rows []model.Notification
				require.NoError(t, d.Find(&rows).Error)
				require.Len(t, rows, 3)
				for _, row := range rows {
					require.NotEqual(t, submitter.Id, row.NotifiedUserID)
					require.NotEqual(t, deleted.Id, row.NotifiedUserID)
					require.Equal(t, resource, string(*row.SubjectType))
					require.Nil(t, row.ReadAt)
				}
				require.Error(t, submit(ctx, "New reference"))
				require.NoError(t, d.Model(&model.Notification{}).Count(&count).Error)
				require.EqualValues(t, 3, count)
			})
		}
	}
}

func TestReferenceSubmissionRecipientBatchSize(t *testing.T) {
	for _, reviewerCount := range []int{0, 1, 25} {
		t.Run(fmt.Sprintf("reviewers=%d", reviewerCount), func(t *testing.T) {
			d := db.OpenInMemorySQLite(t)
			require.NoError(t, d.AutoMigrate(&model.Notification{}))
			submitter := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
			users := []domain.User{submitter}
			for i := 0; i < reviewerCount; i++ {
				users = append(users, fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer)))
			}
			_, err := user.NewGormUserRepository(d).Store(context.Background(), users...)
			require.NoError(t, err)
			var recipientQueries, notificationInserts int
			require.NoError(t, d.Callback().Query().After("gorm:query").Register("count_recipients", func(tx *gorm.DB) {
				if tx.Statement.Table == "users" && strings.Contains(tx.Statement.SQL.String(), "role") && strings.Contains(tx.Statement.SQL.String(), "deleted_at") {
					recipientQueries++
				}
			}))
			require.NoError(t, d.Callback().Create().After("gorm:create").Register("count_notifications", func(tx *gorm.DB) {
				if tx.Statement.Table == "notifications" {
					notificationInserts++
				}
			}))
			svc := NewCredentialTypeService(CredentialTypeServiceParams{TypeRepo: NewGormCredentialTypeRepository(d), UoW: referenceTestUnitOfWork(d)})
			ctx := httpContext.WithUser(context.Background(), &submitter)
			proposal, err := svc.Submit(ctx, "Batch reference")
			require.NoError(t, err)
			require.NotEmpty(t, proposal.Id)
			require.Equal(t, 2, recipientQueries, "one recipient lookup uses the existing count plus select, independent of batch size")
			wantInserts := 1
			if reviewerCount == 0 {
				wantInserts = 0
			}
			require.Equal(t, wantInserts, notificationInserts)
			var count int64
			require.NoError(t, d.Model(&model.Notification{}).Count(&count).Error)
			require.EqualValues(t, reviewerCount, count)
		})
	}
}

func TestReferenceReviewWorkloadIndependentOfInbox(t *testing.T) {
	for _, resource := range []string{"credential_type", "credential_issuer_organization", "competency"} {
		t.Run(resource, func(t *testing.T) {
			d := db.OpenInMemorySQLite(t)
			require.NoError(t, d.AutoMigrate(&model.Notification{}))
			submitter := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
			reviewer := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
			_, err := user.NewGormUserRepository(d).Store(context.Background(), submitter, reviewer)
			require.NoError(t, err)
			submitCtx := httpContext.WithUser(context.Background(), &submitter)
			reviewCtx := httpContext.WithUser(context.Background(), &reviewer)
			uow := referenceTestUnitOfWork(d)
			var submit func(string) (string, error)
			var approve, reject func(string) error
			var pending func(*domainQuery.Query) (int, error)
			var table string
			switch resource {
			case "credential_type":
				repo := NewGormCredentialTypeRepository(d)
				svc := NewCredentialTypeService(CredentialTypeServiceParams{TypeRepo: repo, UoW: uow})
				submit = func(name string) (string, error) {
					row, err := svc.Submit(submitCtx, name)
					if err != nil {
						return "", err
					}
					return row.Id, nil
				}
				approve = func(id string) error { _, err := svc.Approve(reviewCtx, id); return err }
				reject = func(id string) error { _, err := svc.Reject(reviewCtx, id, "Not suitable"); return err }
				pending = func(q *domainQuery.Query) (int, error) { _, total, err := repo.Get(reviewCtx, q); return total, err }
				table = "credential_types"
			case "credential_issuer_organization":
				repo := NewGormCredentialIssuerOrganizationRepository(d)
				svc := NewCredentialIssuerOrganizationService(CredentialIssuerOrganizationServiceParams{OrgRepo: repo, UoW: uow})
				submit = func(name string) (string, error) {
					row, err := svc.Submit(submitCtx, name)
					if err != nil {
						return "", err
					}
					return row.Id, nil
				}
				approve = func(id string) error { _, err := svc.Approve(reviewCtx, id); return err }
				reject = func(id string) error { _, err := svc.Reject(reviewCtx, id, "Not suitable"); return err }
				pending = func(q *domainQuery.Query) (int, error) { _, total, err := repo.Get(reviewCtx, q); return total, err }
				table = "credential_issuer_organizations"
			case "competency":
				repo := NewGormCompetencyRepository(d)
				svc := NewCompetencyService(CompetencyServiceParams{CompetencyRepo: repo, UoW: uow})
				submit = func(name string) (string, error) {
					row, err := svc.Submit(submitCtx, name)
					if err != nil {
						return "", err
					}
					return row.Id, nil
				}
				approve = func(id string) error { _, err := svc.Approve(reviewCtx, id); return err }
				reject = func(id string) error { _, err := svc.Reject(reviewCtx, id, "Not suitable"); return err }
				pending = func(q *domainQuery.Query) (int, error) { _, total, err := repo.Get(reviewCtx, q); return total, err }
				table = "competencies"
			}
			assertPending := func(want int) {
				t.Helper()
				total, err := pending(&domainQuery.Query{Page: 1, Limit: 1, Filters: []domainQuery.Filter{domainQuery.NewFilter("status", domainQuery.OperatorEqual, "pending")}})
				require.NoError(t, err)
				require.Equal(t, want, total)
			}
			approvedID, err := submit("Approve this")
			require.NoError(t, err)
			rejectedID, err := submit("Reject this")
			require.NoError(t, err)
			assertPending(2)
			inbox := notification.NewGormNotificationRepository(d)
			scope := domain.NotificationScope{UserID: reviewer.Id, CanReview: true}
			page, err := inbox.List(reviewCtx, scope, domain.NotificationListOptions{})
			require.NoError(t, err)
			require.Len(t, page.Items, 2)
			marked, err := inbox.MarkRead(reviewCtx, scope, page.Items[0].ID)
			require.NoError(t, err)
			require.NotNil(t, marked.ReadAt)
			assertPending(2)
			var before []model.Notification
			require.NoError(t, d.Order("id").Find(&before).Error)
			require.NoError(t, approve(approvedID))
			assertPending(1)
			require.NoError(t, reject(rejectedID))
			assertPending(0)
			require.NoError(t, d.Table(table).Where("id = ?", approvedID).Update("active", false).Error)
			assertPending(0)
			var after []model.Notification
			require.NoError(t, d.Order("id").Find(&after).Error)
			require.Equal(t, before, after, "review decisions must preserve every inbox row and read timestamp")
			page, err = inbox.List(reviewCtx, scope, domain.NotificationListOptions{})
			require.NoError(t, err)
			require.Len(t, page.Items, 2)
			for _, row := range page.Items {
				require.NotNil(t, row.Subject)
				if *row.SubjectID == approvedID {
					require.Equal(t, "approved", *row.Subject.Status)
				} else {
					require.Equal(t, rejectedID, *row.SubjectID)
					require.Equal(t, "rejected", *row.Subject.Status)
				}
			}
		})
	}
}

func TestReferenceStoreProducesZeroNotifications(t *testing.T) {
	for _, resource := range []string{"credential_type", "credential_issuer_organization", "competency"} {
		t.Run(resource, func(t *testing.T) {
			d := db.OpenInMemorySQLite(t)
			require.NoError(t, d.AutoMigrate(&model.Notification{}))
			users := user.NewGormUserRepository(d)
			creator := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer))
			_, err := users.Store(context.Background(), creator,
				fixtures.NewDomainUser(fixtures.WithRole(domain.RoleIssuer)),
				fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin)),
				fixtures.NewDomainUser(fixtures.WithRole(domain.RoleSuperAdmin)))
			require.NoError(t, err)

			uow := referenceTestUnitOfWork(d)
			ctx := httpContext.WithUser(context.Background(), &creator)

			switch resource {
			case "credential_type":
				svc := NewCredentialTypeService(CredentialTypeServiceParams{TypeRepo: NewGormCredentialTypeRepository(d), UoW: uow})
				res, err := svc.Store(ctx, "Direct Approved Type", nil)
				require.NoError(t, err)
				require.NotNil(t, res.ActivatedAt)
			case "credential_issuer_organization":
				svc := NewCredentialIssuerOrganizationService(CredentialIssuerOrganizationServiceParams{OrgRepo: NewGormCredentialIssuerOrganizationRepository(d), UoW: uow})
				res, err := svc.Store(ctx, "Direct Approved Org", nil)
				require.NoError(t, err)
				require.NotNil(t, res.ActivatedAt)
			case "competency":
				svc := NewCompetencyService(CompetencyServiceParams{CompetencyRepo: NewGormCompetencyRepository(d), UoW: uow})
				res, err := svc.Store(ctx, "Direct Approved Competency", nil)
				require.NoError(t, err)
				require.NotNil(t, res.ActivatedAt)
			}

			var notifCount int64
			require.NoError(t, d.Model(&model.Notification{}).Count(&notifCount).Error)
			require.Zero(t, notifCount, "direct approved store must produce zero reviewer notifications")
		})
	}
}

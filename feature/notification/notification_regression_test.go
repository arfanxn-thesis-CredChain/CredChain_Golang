package notification

import (
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"
	testdb "CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"

	"github.com/glebarez/sqlite"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func notificationDB(t *testing.T) (*gorm.DB, domain.NotificationRepository, domain.NotificationScope, domain.NotificationScope) {
	t.Helper()
	d := testdb.OpenInMemorySQLite(t)
	require.NoError(t, d.AutoMigrate(&model.Notification{}))
	a, b := fixtures.NewModelUser(), fixtures.NewModelUser()
	require.NoError(t, d.Create(&[]model.User{a, b}).Error)
	return d, NewGormNotificationRepository(d), domain.NotificationScope{UserID: a.Id}, domain.NotificationScope{UserID: b.Id}
}

func credentialRow(t *testing.T, d *gorm.DB, holder string, kind domain.NotificationType) domain.Notification {
	t.Helper()
	id := ulid.Make().String()
	number := "CERT-001"
	require.NoError(t, d.Create(&model.Credential{Id: id, HolderUserId: holder, SubmitterUserId: holder, IssuerOrganizationId: ulid.Make().String(), Name: "Bachelor Degree", Number: &number, FileHash: id, IssuedAt: time.Now()}).Error)
	typ := domain.NotificationSubjectCredential
	return domain.Notification{NotifiedUserID: holder, SubjectType: &typ, SubjectID: &id, Type: kind}
}

func TestNotificationReplayAndOwnership(t *testing.T) {
	d, repo, a, b := notificationDB(t)
	ctx := context.Background()
	n := credentialRow(t, d, a.UserID, domain.NotificationTypeCredentialRegistered)
	n.Data = map[string]any{"name": "original"}
	require.NoError(t, repo.Store(ctx, n, n))
	page, err := repo.List(ctx, a, domain.NotificationListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	stored := page.Items[0]
	require.Nil(t, stored.UpdatedAt)
	_, err = repo.MarkRead(ctx, b, stored.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	marked, err := repo.MarkRead(ctx, a, stored.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.ReadAt)
	require.Equal(t, marked.ReadAt, marked.UpdatedAt)
	n.Data["name"] = "changed"
	require.NoError(t, repo.Store(ctx, n))
	markedAgain, err := repo.MarkRead(ctx, a, stored.ID)
	require.NoError(t, err)
	require.Equal(t, marked.UpdatedAt, markedAgain.UpdatedAt)
	require.Equal(t, marked.ReadAt, markedAgain.ReadAt)
	require.Equal(t, stored.CreatedAt, markedAgain.CreatedAt)
	require.Equal(t, stored.ID, markedAgain.ID)
	require.Equal(t, "original", markedAgain.Data["name"])
	n.Type = domain.NotificationTypeCredentialApproved
	require.NoError(t, repo.Store(ctx, n))
	foreign := credentialRow(t, d, b.UserID, domain.NotificationTypeCredentialRegistered)
	require.NoError(t, repo.Store(ctx, foreign))
	updated, err := repo.MarkAllRead(ctx, a)
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)
	count, err := repo.UnreadCount(ctx, b)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	markedAgain, err = repo.MarkRead(ctx, a, stored.ID)
	require.NoError(t, err)
	require.Equal(t, marked.UpdatedAt, markedAgain.UpdatedAt)
}

func TestNotificationKeysetAndBounds(t *testing.T) {
	d, repo, a, b := notificationDB(t)
	ctx := context.Background()
	stamp := time.Date(2026, 10, 1, 12, 30, 0, 123456000, time.UTC)
	var rows []domain.Notification
	for range 101 {
		n := credentialRow(t, d, a.UserID, domain.NotificationTypeCredentialApproved)
		n.ID, n.CreatedAt = ulid.Make().String(), stamp
		rows = append(rows, n)
	}
	foreign := credentialRow(t, d, b.UserID, domain.NotificationTypeCredentialRejected)
	require.NoError(t, repo.Store(ctx, append(rows, foreign)...))
	for _, tc := range []struct{ limit, want int }{{0, 20}, {1, 1}, {100, 100}} {
		page, err := repo.List(ctx, a, domain.NotificationListOptions{Limit: tc.limit})
		require.NoError(t, err)
		require.Len(t, page.Items, tc.want)
		require.EqualValues(t, 101, page.Total)
	}
	page, err := repo.List(ctx, a, domain.NotificationListOptions{Limit: 2})
	require.NoError(t, err)
	require.Equal(t, rows[100].ID, page.Items[0].ID)
	require.Equal(t, rows[99].ID, page.Items[1].ID)
	newer := credentialRow(t, d, a.UserID, domain.NotificationTypeCredentialRevoked)
	newer.CreatedAt = stamp.Add(time.Second)
	require.NoError(t, repo.Store(ctx, newer))
	page, err = repo.List(ctx, a, domain.NotificationListOptions{Limit: 100, Cursor: *page.NextCursor})
	require.NoError(t, err)
	require.EqualValues(t, 102, page.Total)
	require.Len(t, page.Items, 99)
	require.Equal(t, rows[98].ID, page.Items[0].ID)
	require.Nil(t, page.NextCursor)
	encode := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	for _, options := range []domain.NotificationListOptions{
		{Limit: -1}, {Limit: 101}, {Status: "read"}, {Status: "UNREAD"}, {Cursor: "not-base64!"},
		{Cursor: encode("invalid|" + ulid.Make().String())}, {Cursor: encode(stamp.Format(time.RFC3339Nano) + "|invalid")},
		{Cursor: encode(stamp.Format(time.RFC3339Nano) + "|extra|parts")},
	} {
		_, err := repo.List(ctx, a, options)
		var de *domain.Error
		require.ErrorAs(t, err, &de)
		require.Equal(t, domain.CodeSystemValidation, de.Code)
	}
}

func TestNotificationMixedSubjectsAndLookupFailures(t *testing.T) {
	d, repo, a, b := notificationDB(t)
	ctx := context.Background()
	a.CanReview = true
	credential := credentialRow(t, d, a.UserID, domain.NotificationTypeCredentialRegistered)
	foreign := credentialRow(t, d, b.UserID, domain.NotificationTypeCredentialRegistered)
	foreign.NotifiedUserID = a.UserID
	missingID := ulid.Make().String()
	missing := credential
	missing.SubjectID = &missingID
	now := time.Now()
	id := *credential.SubjectID
	require.NoError(t, d.Create(&model.CredentialType{Id: id, Name: "Renamed type", ActivatedAt: &now}).Error)
	require.NoError(t, d.Create(&model.Competency{Id: id, Name: "Rejected skill", RejectedAt: &now}).Error)
	typeSubject, skillSubject := domain.NotificationSubjectCredentialType, domain.NotificationSubjectCompetency
	typeRow := domain.Notification{NotifiedUserID: a.UserID, SubjectType: &typeSubject, SubjectID: &id, Type: domain.NotificationTypeCredentialTypeSubmitted}
	skillRow := domain.Notification{NotifiedUserID: a.UserID, SubjectType: &skillSubject, SubjectID: &id, Type: domain.NotificationTypeCompetencySubmitted}
	require.NoError(t, repo.Store(ctx, credential, foreign, missing, typeRow, skillRow))
	page, err := repo.List(ctx, a, domain.NotificationListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Items, 5)
	for _, n := range page.Items {
		switch n.Type {
		case domain.NotificationTypeCredentialTypeSubmitted:
			require.Equal(t, "Renamed type", n.Subject.Name)
			require.Equal(t, "approved", *n.Subject.Status)
		case domain.NotificationTypeCompetencySubmitted:
			require.Equal(t, "Rejected skill", n.Subject.Name)
			require.Equal(t, "rejected", *n.Subject.Status)
		default:
			if *n.SubjectID == id {
				require.Equal(t, "Bachelor Degree", n.Subject.Name)
			} else {
				require.Nil(t, n.Subject)
			}
		}
	}
	a.CanReview = false
	page, err = repo.List(ctx, a, domain.NotificationListOptions{Limit: 1})
	require.NoError(t, err)
	require.EqualValues(t, 3, page.Total)
	var restricted model.Notification
	require.NoError(t, d.Where("type = ?", skillRow.Type).First(&restricted).Error)
	_, err = repo.MarkRead(ctx, a, restricted.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
	require.NoError(t, d.Migrator().DropTable(&model.Credential{}))
	_, err = repo.List(ctx, a, domain.NotificationListOptions{})
	require.Error(t, err, "lookup failures must not become unavailable targets")
}

func TestNotificationBatchAndPrimaryKeyConstraints(t *testing.T) {
	d, repo, a, _ := notificationDB(t)
	ctx := context.Background()
	rows := []domain.Notification{credentialRow(t, d, a.UserID, domain.NotificationTypeCredentialRegistered), credentialRow(t, d, a.UserID, domain.NotificationTypeCredentialApproved)}
	creates, updates, raw := 0, 0, 0
	require.NoError(t, d.Callback().Create().Before("gorm:create").Register("notifications_create", func(*gorm.DB) { creates++ }))
	require.NoError(t, d.Callback().Update().Before("gorm:update").Register("notifications_update", func(*gorm.DB) { updates++ }))
	require.NoError(t, d.Callback().Row().Before("gorm:row").Register("notifications_summary", func(*gorm.DB) { raw++ }))
	require.NoError(t, repo.Store(ctx, rows...))
	require.Equal(t, 1, creates)
	_, err := repo.List(ctx, a, domain.NotificationListOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, raw)
	count, err := repo.MarkAllRead(ctx, a)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	require.Equal(t, 1, updates)
	rows[0].ID = ulid.Make().String()
	rows[0].Type = domain.NotificationTypeCredentialRejected
	require.NoError(t, repo.Store(ctx, rows[0]))
	rows[1].ID = rows[0].ID
	rows[1].Type = domain.NotificationTypeCredentialRevoked
	require.Error(t, repo.Store(ctx, rows[1]))
	rows[1].ID = "invalid"
	require.Error(t, repo.Store(ctx, rows[1]))
}

func TestNotificationInitialSchema(t *testing.T) {
	d, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := d.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	require.NoError(t, d.Exec("PRAGMA foreign_keys = ON").Error)
	require.NoError(t, d.Exec("CREATE TABLE users (id CHAR(26) PRIMARY KEY)").Error)
	up, err := os.ReadFile("../../infrastructure/database/migrations/000001_initial_schema.up.sql")
	require.NoError(t, err)
	start := strings.Index(string(up), "CREATE TABLE notifications (")
	require.Positive(t, start)
	for statement := range strings.SplitSeq(string(up)[start:], ";") {
		if strings.TrimSpace(statement) != "" {
			require.NoError(t, d.Exec(statement).Error)
		}
	}
	var columns []struct{ Name string }
	require.NoError(t, d.Raw("PRAGMA table_info(notifications)").Scan(&columns).Error)
	var names []string
	for _, column := range columns {
		names = append(names, column.Name)
	}
	require.Equal(t, []string{"id", "notified_user_id", "subject_type", "subject_id", "type", "data", "created_at", "updated_at", "read_at"}, names)
	for _, index := range []string{"notifications_user_feed_idx", "notifications_user_unread_idx"} {
		var definition string
		require.NoError(t, d.Raw("SELECT sql FROM sqlite_master WHERE name = ?", index).Scan(&definition).Error)
		if strings.Contains(index, "unread") {
			require.Contains(t, definition, "WHERE read_at IS NULL")
		} else {
			require.Contains(t, definition, "notified_user_id, created_at DESC, id DESC")
		}
	}
	userID, subjectID := ulid.Make().String(), ulid.Make().String()
	typ := domain.NotificationSubjectCredential
	n := domain.Notification{NotifiedUserID: userID, SubjectType: &typ, SubjectID: &subjectID, Type: domain.NotificationTypeCredentialRegistered}
	repo := NewGormNotificationRepository(d)
	require.Error(t, repo.Store(context.Background(), n), "recipient FK remains")
	require.NoError(t, d.Exec("INSERT INTO users (id) VALUES (?)", userID).Error)
	require.NoError(t, repo.Store(context.Background(), n, n), "subject does not need an FK")
	for _, values := range [][]any{
		{nil, subjectID, "credential_registered"}, {"credential", nil, "credential_registered"},
		{"competency", subjectID, "credential_registered"}, {"unknown", subjectID, "credential_registered"},
		{nil, nil, "unknown"},
	} {
		require.Error(t, d.Exec("INSERT INTO notifications (id, notified_user_id, subject_type, subject_id, type) VALUES (?, ?, ?, ?, ?)", ulid.Make().String(), userID, values[0], values[1], values[2]).Error)
	}
	down, err := os.ReadFile("../../infrastructure/database/migrations/000001_initial_schema.down.sql")
	require.NoError(t, err)
	first := strings.TrimSpace(strings.Split(string(down), ";")[0])
	require.Equal(t, "DROP TABLE IF EXISTS notifications", first)
	require.NoError(t, d.Exec(first).Error)
}

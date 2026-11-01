package credential

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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func credentialNotificationDB(t *testing.T) (*gorm.DB, domain.CredentialNotificationRepository, string, string) {
	t.Helper()
	db := testdb.OpenInMemorySQLite(t)
	require.NoError(t, db.AutoMigrate(&model.CredentialNotification{}))
	a := fixtures.NewModelUser(fixtures.WithWalletAddress("0x1111111111111111111111111111111111111111"))
	b := fixtures.NewModelUser(fixtures.WithWalletAddress("0x2222222222222222222222222222222222222222"))
	require.NoError(t, db.Create(&[]model.User{a, b}).Error)
	return db, NewGormCredentialNotificationRepository(db), a.Id, b.Id
}

func credentialNotificationRow(t *testing.T, db *gorm.DB, holderID string, nType domain.CredentialNotificationType) domain.CredentialNotification {
	t.Helper()
	credentialID := ulid.Make().String()
	num := "CERT-001"
	c := model.Credential{
		Id:                   credentialID,
		HolderUserId:         holderID,
		SubmitterUserId:      holderID,
		IssuerOrganizationId: ulid.Make().String(),
		TypeId:               ulid.Make().String(),
		Name:                 "Bachelor Degree",
		Number:               &num,
		FileHash:             credentialID,
		IssuedAt:             time.Now(),
	}
	require.NoError(t, db.Create(&c).Error)
	return domain.CredentialNotification{
		NotifiedUserID: holderID,
		CredentialID:   credentialID,
		Type:           nType,
	}
}

func TestCredentialNotificationStoreReplayPreservesReadAndID(t *testing.T) {
	db, repo, a, b := credentialNotificationDB(t)
	ctx := context.Background()

	n := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRegistered)

	require.NoError(t, repo.Store(ctx))
	require.NoError(t, repo.Store(ctx, n, n))

	page, err := repo.List(ctx, a, domain.CredentialNotificationListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)

	stored := page.Items[0]
	_, err = ulid.ParseStrict(stored.ID)
	require.NoError(t, err)
	assert.False(t, stored.CreatedAt.IsZero())
	assert.Nil(t, stored.ReadAt)

	// Mark read
	marked, err := repo.MarkRead(ctx, a, stored.ID)
	require.NoError(t, err)
	require.NotNil(t, marked.ReadAt)
	readAt := *marked.ReadAt

	// Replay Store with same notified_user_id, credential_id, type
	require.NoError(t, repo.Store(ctx, n))

	page, err = repo.List(ctx, a, domain.CredentialNotificationListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, stored.ID, page.Items[0].ID)
	assert.Equal(t, stored.CreatedAt, page.Items[0].CreatedAt)
	require.NotNil(t, page.Items[0].ReadAt)
	assert.Equal(t, readAt, *page.Items[0].ReadAt)

	// Different type is a separate row
	nApproved := n
	nApproved.Type = domain.CredentialNotificationTypeApproved
	require.NoError(t, repo.Store(ctx, nApproved))

	// Different recipient is a separate row
	nRecipientB := credentialNotificationRow(t, db, b, domain.CredentialNotificationTypeRegistered)
	require.NoError(t, repo.Store(ctx, nRecipientB))

	countA, err := repo.UnreadCount(ctx, a)
	require.NoError(t, err)
	assert.EqualValues(t, 1, countA) // n was read, nApproved is unread

	countB, err := repo.UnreadCount(ctx, b)
	require.NoError(t, err)
	assert.EqualValues(t, 1, countB)
}

func TestCredentialNotificationListKeysetEqualTimestampsAndLivePosition(t *testing.T) {
	db, repo, a, b := credentialNotificationDB(t)
	ctx := context.Background()
	stamp := time.Date(2026, 10, 1, 12, 30, 0, 123456000, time.UTC)

	var input []domain.CredentialNotification
	for range 5 {
		n := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeApproved)
		n.ID, n.CreatedAt = ulid.Make().String(), stamp
		input = append(input, n)
	}

	foreign := credentialNotificationRow(t, db, b, domain.CredentialNotificationTypeRejected)
	require.NoError(t, repo.Store(ctx, append(input, foreign)...))

	page, err := repo.List(ctx, a, domain.CredentialNotificationListOptions{Limit: 2})
	require.NoError(t, err)
	assert.EqualValues(t, 5, page.Total)
	require.Len(t, page.Items, 2)
	assert.Equal(t, input[4].ID, page.Items[0].ID)
	assert.Equal(t, input[3].ID, page.Items[1].ID)
	require.NotNil(t, page.NextCursor)

	// Newer row added above cursor position does not disrupt continuation
	newer := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRevoked)
	newer.CreatedAt = stamp.Add(time.Second)
	require.NoError(t, repo.Store(ctx, newer))

	page, err = repo.List(ctx, a, domain.CredentialNotificationListOptions{Limit: 2, Cursor: *page.NextCursor})
	require.NoError(t, err)
	assert.EqualValues(t, 6, page.Total)
	require.Len(t, page.Items, 2)
	assert.Equal(t, input[2].ID, page.Items[0].ID)
	assert.Equal(t, input[1].ID, page.Items[1].ID)
	require.NotNil(t, page.NextCursor)

	page, err = repo.List(ctx, a, domain.CredentialNotificationListOptions{Limit: 2, Cursor: *page.NextCursor})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	assert.Equal(t, input[0].ID, page.Items[0].ID)
	assert.Nil(t, page.NextCursor)
}

func TestCredentialNotificationListBatchedAuthorizedCredentialSummary(t *testing.T) {
	db, repo, a, b := credentialNotificationDB(t)
	ctx := context.Background()

	// 1. Own valid credential
	n1 := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRegistered)

	// 2. Credential whose name is later updated
	n2 := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeApproved)

	// 3. Credential belonging to user B (simulates an unauthorized / foreign credential)
	foreignCredID := ulid.Make().String()
	require.NoError(t, db.Create(&model.Credential{
		Id:              foreignCredID,
		HolderUserId:    b, // holder is B!
		SubmitterUserId: b,
		Name:            "Foreign Secret",
		FileHash:        foreignCredID,
		IssuedAt:        time.Now(),
	}).Error)
	n3 := domain.CredentialNotification{
		NotifiedUserID: a,
		CredentialID:   foreignCredID,
		Type:           domain.CredentialNotificationTypeRegistered,
	}

	// 4. Missing / deleted credential
	missingCredID := ulid.Make().String()
	n4 := domain.CredentialNotification{
		NotifiedUserID: a,
		CredentialID:   missingCredID,
		Type:           domain.CredentialNotificationTypeRegistered,
	}

	require.NoError(t, repo.Store(ctx, n1, n2, n3, n4))

	// Update n2's credential name in the credentials table
	require.NoError(t, db.Model(&model.Credential{}).Where("id = ?", n2.CredentialID).
		Update("name", "Master Degree (Updated)").Error)

	page, err := repo.List(ctx, a, domain.CredentialNotificationListOptions{Limit: 10})
	require.NoError(t, err)
	assert.EqualValues(t, 4, page.Total)
	require.Len(t, page.Items, 4)

	itemMap := make(map[string]domain.CredentialNotification)
	for _, it := range page.Items {
		itemMap[it.CredentialID] = it
	}

	// n1 has valid summary, no sensitive leak
	it1 := itemMap[n1.CredentialID]
	require.NotNil(t, it1.Credential)
	assert.Equal(t, n1.CredentialID, it1.Credential.ID)
	assert.Equal(t, "Bachelor Degree", it1.Credential.Name)
	assert.Equal(t, "CERT-001", *it1.Credential.Number)
	assert.Empty(t, it1.Credential.FileHash) // sensitive field not populated

	// n2 reflects the updated name (not an old snapshot)
	it2 := itemMap[n2.CredentialID]
	require.NotNil(t, it2.Credential)
	assert.Equal(t, "Master Degree (Updated)", it2.Credential.Name)

	// n3 is retained with unavailable fallback (relation nil) because holder != a
	it3 := itemMap[n3.CredentialID]
	assert.Nil(t, it3.Credential)

	// n4 is retained with unavailable fallback (relation nil) because credential does not exist
	it4 := itemMap[n4.CredentialID]
	assert.Nil(t, it4.Credential)
}

func TestCredentialNotificationListBoundsAndValidation(t *testing.T) {
	db, repo, a, _ := credentialNotificationDB(t)
	ctx := context.Background()

	var input []domain.CredentialNotification
	for range 101 {
		input = append(input, credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRegistered))
	}
	require.NoError(t, repo.Store(ctx, input...))

	for _, tc := range []struct{ limit, want int }{{0, 20}, {1, 1}, {100, 100}} {
		page, err := repo.List(ctx, a, domain.CredentialNotificationListOptions{Limit: tc.limit})
		require.NoError(t, err)
		assert.Len(t, page.Items, tc.want)
		assert.EqualValues(t, 101, page.Total)
		assert.NotNil(t, page.NextCursor)
	}

	encode := func(s string) string { return base64.RawURLEncoding.EncodeToString([]byte(s)) }
	for _, options := range []domain.CredentialNotificationListOptions{
		{Limit: -1},
		{Limit: 101},
		{Status: "invalid"},
		{Status: "UNREAD"},
		{Status: "read"}, // only "all" and "unread" allowed
		{Cursor: "not-base64!"},
		{Cursor: encode("invalid|" + ulid.Make().String())},
		{Cursor: encode(time.Now().Format(time.RFC3339Nano) + "|invalid")},
		{Cursor: encode(time.Now().Format(time.RFC3339Nano) + "|extra|parts")},
	} {
		_, err := repo.List(ctx, a, options)
		var domainErr *domain.Error
		require.ErrorAs(t, err, &domainErr, "%+v", options)
		assert.Equal(t, domain.CodeSystemValidation, domainErr.Code)
	}
}

func TestCredentialNotificationMarkReadIsScopedAndPreservesFirstRead(t *testing.T) {
	db, repo, a, b := credentialNotificationDB(t)
	ctx := context.Background()

	n := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRejected)
	require.NoError(t, repo.Store(ctx, n))

	page, err := repo.List(ctx, a, domain.CredentialNotificationListOptions{Status: domain.CredentialNotificationStatusUnread})
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	id := page.Items[0].ID

	// Foreign and missing return gorm.ErrRecordNotFound
	_, err = repo.MarkRead(ctx, b, id)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = repo.MarkRead(ctx, a, ulid.Make().String())
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	// Mark read succeeds
	marked, err := repo.MarkRead(ctx, a, id)
	require.NoError(t, err)
	require.NotNil(t, marked.ReadAt)
	require.NotNil(t, marked.Credential)
	firstRead := *marked.ReadAt

	// Calling MarkRead again preserves the original ReadAt
	markedAgain, err := repo.MarkRead(ctx, a, id)
	require.NoError(t, err)
	assert.Equal(t, firstRead, *markedAgain.ReadAt)

	// Now unread count is 0
	unreadCount, err := repo.UnreadCount(ctx, a)
	require.NoError(t, err)
	assert.Zero(t, unreadCount)
}

func TestCredentialNotificationMarkAllReadOnlyCurrentRecipientUnreadRows(t *testing.T) {
	db, repo, a, b := credentialNotificationDB(t)
	ctx := context.Background()

	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	alreadyRead := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRegistered)
	alreadyRead.ReadAt = &past

	n2 := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeApproved)
	n3 := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRejected)
	foreign := credentialNotificationRow(t, db, b, domain.CredentialNotificationTypeRevoked)

	require.NoError(t, repo.Store(ctx, alreadyRead, n2, n3, foreign))

	count, err := repo.MarkAllRead(ctx, a)
	require.NoError(t, err)
	assert.EqualValues(t, 2, count)

	// User A now has 0 unread
	countA, err := repo.UnreadCount(ctx, a)
	require.NoError(t, err)
	assert.Zero(t, countA)

	// User B still has 1 unread
	countB, err := repo.UnreadCount(ctx, b)
	require.NoError(t, err)
	assert.EqualValues(t, 1, countB)

	// Already read row preserved original ReadAt
	page, err := repo.List(ctx, a, domain.CredentialNotificationListOptions{})
	require.NoError(t, err)
	for _, it := range page.Items {
		require.NotNil(t, it.ReadAt)
		if it.CredentialID == alreadyRead.CredentialID {
			assert.Equal(t, past, *it.ReadAt)
		}
	}
}

func TestCredentialNotificationMigrationConstraintsAndIndexes(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })

	require.NoError(t, db.Exec("PRAGMA foreign_keys = ON").Error)
	require.NoError(t, db.Exec("CREATE TABLE users (id CHAR(26) PRIMARY KEY)").Error)
	require.NoError(t, db.Exec("CREATE TABLE credentials (id CHAR(26) PRIMARY KEY, holder_user_id CHAR(26), name TEXT, number TEXT)").Error)

	up, err := os.ReadFile("../../infrastructure/database/migrations/000001_initial_schema.up.sql")
	require.NoError(t, err)
	start := strings.Index(string(up), "CREATE TABLE credential_notifications (")
	require.Greater(t, start, strings.Index(string(up), "CREATE TABLE credentials ("))
	require.Greater(t, start, strings.Index(string(up), "CREATE TABLE users ("))
	for statement := range strings.SplitSeq(string(up)[start:], ";") {
		if strings.TrimSpace(statement) != "" {
			require.NoError(t, db.Exec(statement).Error)
		}
	}

	for _, name := range []string{"credential_notifications_user_feed_idx", "credential_notifications_user_unread_idx"} {
		var sql string
		require.NoError(t, db.Raw("SELECT sql FROM sqlite_master WHERE name = ?", name).Scan(&sql).Error)
		assert.Contains(t, sql, "notified_user_id")
		if strings.Contains(name, "unread") {
			assert.Contains(t, sql, "WHERE read_at IS NULL")
		} else {
			assert.Contains(t, sql, "(notified_user_id, created_at DESC, id DESC)")
		}
	}
	var columns []struct{ Name string }
	require.NoError(t, db.Raw("PRAGMA table_info(credential_notifications)").Scan(&columns).Error)
	var names []string
	for _, column := range columns {
		names = append(names, column.Name)
	}
	assert.Equal(t, []string{"id", "notified_user_id", "credential_id", "type", "created_at", "read_at"}, names)

	repo := NewGormCredentialNotificationRepository(db)
	userA := ulid.Make().String()
	credA := ulid.Make().String()

	n := domain.CredentialNotification{
		NotifiedUserID: userA,
		CredentialID:   credA,
		Type:           domain.CredentialNotificationTypeRegistered,
	}

	// FK to users fails
	require.Error(t, repo.Store(context.Background(), n), "missing user FK")
	require.NoError(t, db.Exec("INSERT INTO users (id) VALUES (?)", userA).Error)

	// FK to credentials fails
	require.Error(t, repo.Store(context.Background(), n), "missing credential FK")
	require.NoError(t, db.Exec("INSERT INTO credentials (id, holder_user_id, name) VALUES (?, ?, 'Cert')", credA, userA).Error)

	// Now Store succeeds
	require.NoError(t, repo.Store(context.Background(), n, n))

	// Invalid type check fails
	nInvalid := n
	nInvalid.Type = "invalid_type"
	require.Error(t, repo.Store(context.Background(), nInvalid), "type CHECK constraint")

	// Down migration drops table
	down, err := os.ReadFile("../../infrastructure/database/migrations/000001_initial_schema.down.sql")
	require.NoError(t, err)
	statements := strings.Split(string(down), ";")
	assert.Equal(t, "DROP TABLE IF EXISTS credential_notifications", strings.TrimSpace(statements[0]))
	require.Less(t, strings.Index(string(down), "DROP TABLE IF EXISTS credential_notifications"), strings.Index(string(down), "DROP TABLE IF EXISTS credentials;"))
	require.Less(t, strings.Index(string(down), "DROP TABLE IF EXISTS credential_notifications"), strings.Index(string(down), "DROP TABLE IF EXISTS users;"))
	require.NoError(t, db.Exec(statements[0]).Error)
	assert.False(t, db.Migrator().HasTable("credential_notifications"))
}

func TestCredentialNotificationBatchOperationsUseOneStatement(t *testing.T) {
	db, repo, a, _ := credentialNotificationDB(t)
	ctx := context.Background()

	input := []domain.CredentialNotification{
		credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRegistered),
		credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeApproved),
		credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRejected),
	}

	creates, updates := 0, 0
	require.NoError(t, db.Callback().Create().Before("gorm:create").Register("cred_notif_count_create", func(*gorm.DB) { creates++ }))
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("cred_notif_count_update", func(*gorm.DB) { updates++ }))

	require.NoError(t, repo.Store(ctx, input...))
	assert.Equal(t, 1, creates, "batch Store must use exactly one insert statement")

	count, err := repo.MarkAllRead(ctx, a)
	require.NoError(t, err)
	assert.EqualValues(t, 3, count)
	assert.Equal(t, 1, updates, "MarkAllRead must use exactly one update statement")
}

func TestCredentialNotificationStoreDoesNotIgnoreOtherConstraints(t *testing.T) {
	db, repo, a, _ := credentialNotificationDB(t)
	ctx := context.Background()

	n := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRegistered)
	n.ID = ulid.Make().String()
	require.NoError(t, repo.Store(ctx, n))

	// PK collision with different type must fail (not swallowed by on conflict)
	other := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeApproved)
	other.ID = n.ID
	require.Error(t, repo.Store(ctx, other), "PK collision must not be swallowed")

	// Invalid ULID format fails validation
	invalid := credentialNotificationRow(t, db, a, domain.CredentialNotificationTypeRegistered)
	invalid.ID = "not-a-valid-ulid"
	require.Error(t, repo.Store(ctx, invalid))
}

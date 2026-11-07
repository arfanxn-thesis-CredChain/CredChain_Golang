package notification

import (
	"context"
	"testing"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/database/gorm/model"
	testdb "CredChain_Golang/tests/db"
	"CredChain_Golang/tests/fixtures"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestNotificationSubjectlessReplayReadAndVisibility(t *testing.T) {
	db := testdb.OpenInMemorySQLite(t)
	require.NoError(t, db.AutoMigrate(&model.Notification{}, &model.CredentialType{}, &model.CredentialIssuerOrganization{}, &model.Competency{}))
	u := fixtures.NewModelUser()
	require.NoError(t, db.Create(&u).Error)
	repo := NewGormNotificationRepository(db)
	ctx := context.Background()
	scope := domain.NotificationScope{UserID: u.Id}
	n := domain.Notification{NotifiedUserID: u.Id, Type: domain.NotificationTypeCredentialRegistered, Data: map[string]any{"note": "literal <text>"}}
	require.NoError(t, repo.Store(ctx, n, n))
	page, err := repo.List(ctx, scope, domain.NotificationListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Items, 2)
	require.Nil(t, page.Items[0].UpdatedAt)
	require.Equal(t, n.Data, page.Items[0].Data)
	marked, err := repo.MarkRead(ctx, scope, page.Items[0].ID)
	require.NoError(t, err)
	require.NotNil(t, marked.ReadAt)
	require.Equal(t, marked.ReadAt, marked.UpdatedAt)
	again, err := repo.MarkRead(ctx, scope, marked.ID)
	require.NoError(t, err)
	require.Equal(t, marked.UpdatedAt, again.UpdatedAt)
	typ, id := domain.NotificationSubjectCompetency, ulid.Make().String()
	n.Type, n.SubjectType, n.SubjectID = domain.NotificationTypeCompetencySubmitted, &typ, &id
	require.NoError(t, repo.Store(ctx, n, n))
	count, err := repo.UnreadCount(ctx, scope)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	scope.CanReview = true
	count, err = repo.UnreadCount(ctx, scope)
	require.NoError(t, err)
	require.EqualValues(t, 2, count)
	page, err = repo.List(ctx, scope, domain.NotificationListOptions{})
	require.NoError(t, err)
	require.Len(t, page.Items, 3)
	count, err = repo.MarkAllRead(ctx, domain.NotificationScope{UserID: u.Id})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	count, err = repo.UnreadCount(ctx, scope)
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
}

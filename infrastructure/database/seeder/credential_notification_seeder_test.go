package seeder_test

import (
	"context"
	"encoding/binary"
	"errors"
	"slices"
	"sort"
	"testing"
	"time"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	"CredChain_Golang/infrastructure/database/gorm/model"
	"CredChain_Golang/infrastructure/database/seeder"
	"CredChain_Golang/tests/mocks"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func notificationRows(t *testing.T, s seededCredentials) []domain.Notification {
	t.Helper()
	var rows []model.Notification
	require.NoError(t, s.db.Order("id").Find(&rows).Error)
	out := make([]domain.Notification, len(rows))
	for i, row := range rows {
		out[i] = row.ToDomain()
	}
	return out
}

func TestCredentialNotificationSeeder_HistoryAndReplay(t *testing.T) {
	s := seedCredentials(t)
	ctx := context.Background()
	seed := seeder.NewCredentialNotificationSeeder(s.credRepo, s.notificationRepo)
	require.Equal(t, "credential-notification", seed.Name())
	require.NoError(t, seed.Seed(ctx))
	rows := notificationRows(t, s)
	require.Len(t, s.creds, 41)
	require.Len(t, rows, 36)
	byCredential := map[string][]domain.Notification{}
	counts := map[domain.NotificationType]int{}
	read, unread := 0, 0
	for _, n := range rows {
		byCredential[*n.SubjectID] = append(byCredential[*n.SubjectID], n)
		counts[n.Type]++
		if n.ReadAt == nil {
			unread++
		} else {
			read++
			assert.Equal(t, n.CreatedAt.Add(time.Hour), *n.ReadAt)
		}
	}
	assert.Equal(t, map[domain.NotificationType]int{
		domain.NotificationTypeCredentialRegistered: 16,
		domain.NotificationTypeCredentialApproved:   10,
		domain.NotificationTypeCredentialRejected:   5,
		domain.NotificationTypeCredentialRevoked:    5,
	}, counts)
	assert.Positive(t, read)
	assert.Positive(t, unread)
	sorted := slices.Clone(s.creds)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	pending, deleted, expiredHistory, revokedHistory := 0, 0, 0, 0
	for slot, c := range sorted {
		expected := map[domain.NotificationType]time.Time{}
		if isDirectIssuance(c) {
			expected[domain.NotificationTypeCredentialRegistered] = c.CreatedAt
		} else if c.ActivatedAt != nil {
			expected[domain.NotificationTypeCredentialApproved] = *c.ActivatedAt
		}
		if c.RejectedAt != nil {
			expected[domain.NotificationTypeCredentialRejected] = *c.RejectedAt
		}
		if c.RevokedAt != nil {
			expected[domain.NotificationTypeCredentialRevoked] = *c.RevokedAt
			revokedHistory++
			assert.Len(t, byCredential[c.ID], 2)
		}
		if c.ExpiresAt != nil {
			expiredHistory++
			assert.NotEmpty(t, byCredential[c.ID])
		}
		if len(expected) == 0 {
			pending++
		}
		assert.Len(t, byCredential[c.ID], len(expected), c.Name)
		for _, n := range byCredential[c.ID] {
			assert.Equal(t, c.HolderUserID, n.NotifiedUserID)
			at, exists := expected[n.Type]
			assert.True(t, exists)
			assert.Equal(t, at, n.CreatedAt)
			ordinal := map[domain.NotificationType]int{
				domain.NotificationTypeCredentialRegistered: 0, domain.NotificationTypeCredentialApproved: 1,
				domain.NotificationTypeCredentialRejected: 2, domain.NotificationTypeCredentialRevoked: 3,
			}[n.Type]
			id, err := ulid.ParseStrict(n.ID)
			require.NoError(t, err)
			assert.Equal(t, uint32(10000+4*slot+ordinal), binary.BigEndian.Uint32(id[12:]))
			assert.Equal(t, ulid.Timestamp(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)), id.Time())
			assert.Equal(t, (slot+ordinal)%2 == 0, n.ReadAt != nil)
			if s.usersByID[n.NotifiedUserID].DeletedAt != nil {
				deleted++
			}
		}
	}
	assert.Equal(t, 10, pending)
	assert.Positive(t, deleted)
	assert.Positive(t, expiredHistory)
	assert.Positive(t, revokedHistory)

	for i, n := range rows {
		if n.ReadAt != nil {
			continue
		}
		marked, err := s.notificationRepo.MarkRead(ctx, domain.NotificationScope{UserID: n.NotifiedUserID}, n.ID)
		require.NoError(t, err)
		require.NotNil(t, marked.ReadAt)
		rows[i].ReadAt = marked.ReadAt
		rows[i].UpdatedAt = marked.UpdatedAt
		break
	}
	require.NoError(t, seed.Seed(ctx))
	assert.Equal(t, rows, notificationRows(t, s), "replay preserves IDs, event timestamps and user read state")
}

// Override only the batched read while retaining the existing repository contract.
type notificationCredentialRead struct {
	domain.CredentialRepository
	rows  []domain.Credential
	ids   []string
	err   error
	calls int
}

func (r *notificationCredentialRead) FindByIds(_ context.Context, ids []string, _ *domainQuery.Query) ([]domain.Credential, error) {
	r.calls++
	r.ids = slices.Clone(ids)
	return r.rows, r.err
}

func captureNotifications(t *testing.T, credentials []domain.Credential) ([]domain.Notification, []string) {
	t.Helper()
	r := &notificationCredentialRead{rows: credentials}
	n := &mocks.MockNotificationRepository{}
	var got []domain.Notification
	n.On("Store", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		got = slices.Clone(args.Get(1).([]domain.Notification))
	}).Return(nil).Once()
	require.NoError(t, seeder.NewCredentialNotificationSeeder(r, n).Seed(context.Background()))
	assert.Equal(t, 1, r.calls)
	n.AssertExpectations(t)
	return got, r.ids
}

func TestCredentialNotificationSeeder_SeedScopeAndStableSlots(t *testing.T) {
	s := seedCredentials(t)
	baseline, ids := captureNotifications(t, s.creds)
	reversed := slices.Clone(s.creds)
	slices.Reverse(reversed)
	again, _ := captureNotifications(t, reversed)
	assert.Equal(t, baseline, again)
	assert.Len(t, ids, len(s.creds))

	// A missing fixture must not shift later notification IDs or read states.
	removed := s.creds[0].ID
	remaining := slices.DeleteFunc(slices.Clone(s.creds), func(c domain.Credential) bool { return c.ID == removed })
	nonSeed := s.creds[0]
	nonSeed.ID = ulid.Make().String()
	remaining = append(remaining, nonSeed)
	partial, _ := captureNotifications(t, remaining)
	assert.Equal(t, slices.DeleteFunc(slices.Clone(baseline), func(n domain.Notification) bool { return *n.SubjectID == removed }), partial)

	// The real repository must not select a persisted, non-fixture credential.
	nonSeed.FileHash = "0x" + nonSeed.ID
	nonSeed.Number = nil
	require.NoError(t, s.db.Create(model.FromDomainCredential(nonSeed)).Error)
	require.NoError(t, seeder.NewCredentialNotificationSeeder(s.credRepo, s.notificationRepo).Seed(context.Background()))
	var count int64
	require.NoError(t, s.db.Model(&model.Notification{}).Where("subject_id = ?", nonSeed.ID).Count(&count).Error)
	assert.Zero(t, count)
}

func TestCredentialNotificationSeeder_EmptyPendingAndErrors(t *testing.T) {
	ctx := context.Background()
	t.Run("empty prerequisite", func(t *testing.T) {
		r := &notificationCredentialRead{}
		n := &mocks.MockNotificationRepository{}
		err := seeder.NewCredentialNotificationSeeder(r, n).Seed(ctx)
		require.ErrorContains(t, err, "run credential seeder first")
		n.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
	})
	t.Run("read error", func(t *testing.T) {
		cause := errors.New("read failure")
		r := &notificationCredentialRead{err: cause}
		n := &mocks.MockNotificationRepository{}
		err := seeder.NewCredentialNotificationSeeder(r, n).Seed(ctx)
		require.ErrorIs(t, err, cause)
		assert.ErrorContains(t, err, "fetch credentials")
		n.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
	})
	s := seedCredentials(t)
	t.Run("pending only", func(t *testing.T) {
		var pending []domain.Credential
		for _, c := range s.creds {
			if c.ActivatedAt == nil && c.RejectedAt == nil && c.RevokedAt == nil {
				pending = append(pending, c)
			}
		}
		require.NotEmpty(t, pending)
		r := &notificationCredentialRead{rows: pending}
		n := &mocks.MockNotificationRepository{}
		require.NoError(t, seeder.NewCredentialNotificationSeeder(r, n).Seed(ctx))
		n.AssertNotCalled(t, "Store", mock.Anything, mock.Anything)
	})
	t.Run("store error", func(t *testing.T) {
		cause := errors.New("store failure")
		r := &notificationCredentialRead{rows: s.creds}
		n := &mocks.MockNotificationRepository{}
		n.On("Store", mock.Anything, mock.Anything).Return(cause).Once()
		err := seeder.NewCredentialNotificationSeeder(r, n).Seed(ctx)
		require.ErrorIs(t, err, cause)
		assert.ErrorContains(t, err, "store notifications")
		n.AssertExpectations(t)
	})
}

func TestCredentialNotificationSeeder_Registry(t *testing.T) {
	s := seedCredentials(t)
	ctx := context.Background()
	notifications := seeder.NewCredentialNotificationSeeder(s.credRepo, s.notificationRepo)
	var ran []string
	credentialStep := &testSeeder{name: "credential", run: func() error { ran = append(ran, "credential"); return nil }}
	notificationStep := &testSeeder{name: notifications.Name(), run: func() error {
		ran = append(ran, notifications.Name())
		return notifications.Seed(ctx)
	}}
	r := seeder.NewRegistry(credentialStep, notificationStep)
	require.NoError(t, r.Run(ctx, "credential-notification"))
	assert.Equal(t, []string{"credential-notification"}, ran)
	ran = nil
	require.NoError(t, r.Run(ctx))
	assert.Equal(t, []string{"credential", "credential-notification"}, ran)
	ran = nil
	require.NoError(t, r.Run(ctx, "credential-notification", "credential"))
	assert.Equal(t, []string{"credential-notification", "credential"}, ran)
	assert.ErrorContains(t, r.Run(ctx, "unknown"), "seeder not found")
}

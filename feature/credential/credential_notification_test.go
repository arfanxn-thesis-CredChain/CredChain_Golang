package credential

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/jobs"
)

type mockNotificationEnqueuer struct {
	mock.Mock
}

func (m *mockNotificationEnqueuer) EnqueueExtract(ctx context.Context, args jobs.CredentialExtractArgs) error {
	return m.Called(ctx, args).Error(0)
}

func (m *mockNotificationEnqueuer) EnqueueCredentialNotifications(ctx context.Context, args []jobs.CredentialNotificationArgs) error {
	return m.Called(ctx, args).Error(0)
}

func TestEnqueueLifecycleNotifications_DisabledMail_Noop(t *testing.T) {
	cfg := &config.Config{MailHost: nil}
	enq := new(mockNotificationEnqueuer)

	svc := &credentialService{
		cfg:      cfg,
		enqueuer: enq,
		logger:   zap.NewNop(),
	}

	creds := []domain.Credential{
		{ID: "c1", HolderUserID: "u1"},
	}

	svc.enqueueLifecycleNotifications(context.Background(), creds, jobs.NotificationEventApproved)
	enq.AssertNotCalled(t, "EnqueueCredentialNotifications")
}

func TestEnqueueLifecycleNotifications_DeduplicatesAndIgnoresEmpty(t *testing.T) {
	host := "smtp.example.com"
	cfg := &config.Config{MailHost: &host}
	enq := new(mockNotificationEnqueuer)

	svc := &credentialService{
		cfg:      cfg,
		enqueuer: enq,
		logger:   zap.NewNop(),
	}

	number := "CERT-123"
	creds := []domain.Credential{
		{ID: "c1", HolderUserID: "u1", Number: &number},
		{ID: "c1", HolderUserID: "u1"}, // duplicate
		{ID: "", HolderUserID: "u2"},   // empty ID
		{ID: "c2", HolderUserID: ""},   // empty holder
		{ID: "c3", HolderUserID: "u3"},
	}

	enq.On("EnqueueCredentialNotifications", mock.Anything, mock.MatchedBy(func(args []jobs.CredentialNotificationArgs) bool {
		assert.Len(t, args, 2)
		assert.Equal(t, "c1", args[0].CredentialID)
		assert.Equal(t, "u1", args[0].HolderUserID)
		assert.Equal(t, "CERT-123", args[0].CredentialNumber)
		assert.Equal(t, jobs.NotificationEventRegistered, args[0].Event)
		assert.Equal(t, "c3", args[1].CredentialID)
		assert.Equal(t, "u3", args[1].HolderUserID)
		assert.Empty(t, args[1].CredentialNumber)
		assert.Equal(t, jobs.NotificationEventRegistered, args[1].Event)
		return true
	})).Return(nil)

	svc.enqueueLifecycleNotifications(context.Background(), creds, jobs.NotificationEventRegistered)
	enq.AssertExpectations(t)
}

func TestEnqueueLifecycleNotifications_EnqueueError_Swallowed(t *testing.T) {
	host := "smtp.example.com"
	cfg := &config.Config{MailHost: &host}
	enq := new(mockNotificationEnqueuer)

	svc := &credentialService{
		cfg:      cfg,
		enqueuer: enq,
		logger:   zap.NewNop(),
	}

	creds := []domain.Credential{
		{ID: "c1", HolderUserID: "u1"},
	}

	enq.On("EnqueueCredentialNotifications", mock.Anything, mock.Anything).Return(errors.New("db connection lost"))

	// Must not panic or return error
	assert.NotPanics(t, func() {
		svc.enqueueLifecycleNotifications(context.Background(), creds, jobs.NotificationEventApproved)
	})
	enq.AssertExpectations(t)
}

func TestEnqueueLifecycleNotifications_ContextWithoutCancel(t *testing.T) {
	host := "smtp.example.com"
	cfg := &config.Config{MailHost: &host}
	enq := new(mockNotificationEnqueuer)

	svc := &credentialService{
		cfg:      cfg,
		enqueuer: enq,
		logger:   zap.NewNop(),
	}

	creds := []domain.Credential{
		{ID: "c1", HolderUserID: "u1"},
	}

	// Create canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var capturedCtx context.Context
	var capturedErr error
	var capturedHasDeadline bool
	var capturedDeadline time.Time

	enq.On("EnqueueCredentialNotifications", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			capturedCtx = args.Get(0).(context.Context)
			capturedErr = capturedCtx.Err()
			capturedDeadline, capturedHasDeadline = capturedCtx.Deadline()
		}).Return(nil)

	svc.enqueueLifecycleNotifications(ctx, creds, jobs.NotificationEventRevoked)
	enq.AssertExpectations(t)

	assert.NotNil(t, capturedCtx)
	assert.NoError(t, capturedErr)
	assert.True(t, capturedHasDeadline)
	assert.True(t, capturedDeadline.After(time.Now()))
}

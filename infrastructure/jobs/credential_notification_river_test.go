package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"golang.org/x/text/language"
	"gorm.io/gorm"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/mail"
	"CredChain_Golang/tests/gintest"
	"CredChain_Golang/tests/mocks"
)

type mockMailer struct {
	mock.Mock
}

func (m *mockMailer) SendEmail(ctx context.Context, to string, data mail.EmailData) error {
	args := m.Called(ctx, to, data)
	return args.Error(0)
}

func TestCredentialNotificationWorker_Work_DisabledMail(t *testing.T) {
	cfg := &config.Config{MailHost: nil}
	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		Config: cfg,
		Logger: zap.NewNop(),
	})

	job := &river.Job[CredentialNotificationArgs]{
		Args: CredentialNotificationArgs{
			CredentialID: "cred-1",
			HolderUserID: "user-1",
			Event:        NotificationEventApproved,
		},
	}

	err := w.Work(context.Background(), job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestCredentialNotificationWorker_Work_UserNotFound_CancelsJob(t *testing.T) {
	host := "smtp.example.com"
	baseURL := "http://localhost:5173"
	cfg := &config.Config{MailHost: &host, AppBaseURL: &baseURL}
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(nil, gorm.ErrRecordNotFound)

	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo,
		Config:   cfg,
		Logger:   zap.NewNop(),
	})

	job := &river.Job[CredentialNotificationArgs]{
		Args: CredentialNotificationArgs{
			CredentialID: "cred-1",
			HolderUserID: "user-1",
			Event:        NotificationEventApproved,
		},
	}

	err := w.Work(context.Background(), job)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, river.JobCancel(err)))
}

func TestCredentialNotificationWorker_Work_TransientLookup_ReturnsRetryableError(t *testing.T) {
	host := "smtp.example.com"
	lookupErr := errors.New("database unavailable")
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(nil, lookupErr).Once()

	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo,
		Config:   &config.Config{MailHost: &host},
		Logger:   zap.NewNop(),
	})
	job := &river.Job[CredentialNotificationArgs]{Args: CredentialNotificationArgs{
		CredentialID: "cred-1", HolderUserID: "user-1", Event: NotificationEventApproved,
	}}

	err := w.Work(context.Background(), job)
	assert.ErrorIs(t, err, lookupErr)
	assert.ErrorContains(t, err, "load recipient user")
	assert.False(t, errors.Is(err, river.JobCancel(err)))
	userRepo.AssertExpectations(t)
}

func TestCredentialNotificationWorker_Work_NilUser_CancelsJob(t *testing.T) {
	host := "smtp.example.com"
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(nil, nil).Once()

	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo,
		Config:   &config.Config{MailHost: &host},
		Logger:   zap.NewNop(),
	})
	job := &river.Job[CredentialNotificationArgs]{Args: CredentialNotificationArgs{
		CredentialID: "cred-1", HolderUserID: "user-1", Event: NotificationEventApproved,
	}}

	err := w.Work(context.Background(), job)
	assert.True(t, errors.Is(err, river.JobCancel(err)))
	userRepo.AssertExpectations(t)
}

func TestCredentialNotificationWorker_Work_UserDeleted_CancelsJob(t *testing.T) {
	host := "smtp.example.com"
	baseURL := "http://localhost:5173"
	cfg := &config.Config{MailHost: &host, AppBaseURL: &baseURL}
	userRepo := new(mocks.MockUserRepository)
	deletedAt := time.Now()
	userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
		Id:        "user-1",
		Email:     "user@example.com",
		DeletedAt: &deletedAt,
	}, nil)

	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo,
		Config:   cfg,
		Logger:   zap.NewNop(),
	})

	job := &river.Job[CredentialNotificationArgs]{
		Args: CredentialNotificationArgs{
			CredentialID: "cred-1",
			HolderUserID: "user-1",
			Event:        NotificationEventApproved,
		},
	}

	err := w.Work(context.Background(), job)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, river.JobCancel(err)))
}

func TestCredentialNotificationWorker_Work_EmptyEmail_CancelsJob(t *testing.T) {
	host := "smtp.example.com"
	baseURL := "http://localhost:5173"
	cfg := &config.Config{MailHost: &host, AppBaseURL: &baseURL}
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
		Id:    "user-1",
		Email: "   ",
	}, nil)

	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo,
		Config:   cfg,
		Logger:   zap.NewNop(),
	})

	job := &river.Job[CredentialNotificationArgs]{
		Args: CredentialNotificationArgs{
			CredentialID: "cred-1",
			HolderUserID: "user-1",
			Event:        NotificationEventApproved,
		},
	}

	err := w.Work(context.Background(), job)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, river.JobCancel(err)))
}

func TestCredentialNotificationWorker_Work_AllEvents_Success(t *testing.T) {
	events := []struct {
		event      CredentialNotificationEvent
		expectedEN string
		expectedID string
	}{
		{
			event:      NotificationEventApproved,
			expectedEN: "approved",
			expectedID: "disetujui",
		},
		{
			event:      NotificationEventRejected,
			expectedEN: "rejected",
			expectedID: "ditolak",
		},
		{
			event:      NotificationEventRegistered,
			expectedEN: "registered",
			expectedID: "didaftarkan",
		},
		{
			event:      NotificationEventRevoked,
			expectedEN: "revoked",
			expectedID: "dicabut",
		},
	}

	for _, tt := range events {
		t.Run(string(tt.event), func(t *testing.T) {
			host := "smtp.example.com"
			baseURL := "http://localhost:5173"
			cfg := &config.Config{MailHost: &host, AppBaseURL: &baseURL}
			userRepo := new(mocks.MockUserRepository)
			name := "Jane Doe"
			userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
				Id: "user-1", Name: &name, Email: "jane@example.com", Language: domain.LanguageEN,
			}, nil)

			mailer := new(mockMailer)
			mailer.On("SendEmail", mock.Anything, "jane@example.com", mock.MatchedBy(func(data mail.EmailData) bool {
				assert.Equal(t, "en", data.Language)
				assert.Contains(t, data.Greeting, "Jane Doe")
				assert.Equal(t, tt.expectedEN, strings.ToLower(strings.Split(data.Title, " ")[len(strings.Split(data.Title, " "))-1]))
				assert.Contains(t, data.FallbackURL, "http://localhost:5173/credentials/cred-1")
				assert.Len(t, data.Buttons, 1)
				assert.Equal(t, "http://localhost:5173/credentials/cred-1", data.Buttons[0].URL)
				assert.Equal(t, []mail.EmailDetail{
					{Key: "Credential Number", Value: "CERT-123"},
					{Key: "Event Time", Value: "Jun 25, 2025, 07:00 AM"},
				}, data.Details)
				return true
			})).Return(nil)

			w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
				UserRepo: userRepo,
				Mailer:   mailer,
				Config:   cfg,
				Logger:   zap.NewNop(),
				Bundle:   gintest.LoadTestI18nBundle(t),
			})

			job := &river.Job[CredentialNotificationArgs]{
				Args: CredentialNotificationArgs{
					CredentialID:     "cred-1",
					CredentialNumber: "CERT-123",
					HolderUserID:     "user-1",
					Event:            tt.event,
					EventTime:        time.Date(2025, time.June, 25, 0, 0, 0, 0, time.UTC),
				},
			}

			err := w.Work(context.Background(), job)
			assert.NoError(t, err)
			mailer.AssertExpectations(t)
		})
	}
}

func TestCredentialNotificationWorker_Work_LatestRecipientLanguage(t *testing.T) {
	host := "smtp.example.com"
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
		Id: "user-1", Email: "jane@example.com", Language: domain.LanguageID,
	}, nil).Once()
	userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
		Id: "user-1", Email: "jane@example.com", Language: domain.LanguageEN,
	}, nil).Once()
	mailer := new(mockMailer)
	mailer.On("SendEmail", mock.Anything, "jane@example.com", mock.MatchedBy(func(data mail.EmailData) bool {
		return data.Language == "id" && data.Title == "Pengajuan Kredensial Disetujui"
	})).Return(nil).Once()
	mailer.On("SendEmail", mock.Anything, "jane@example.com", mock.MatchedBy(func(data mail.EmailData) bool {
		return data.Language == "en" && data.Title == "Credential Submission Approved"
	})).Return(nil).Once()
	worker := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo, Mailer: mailer, Config: &config.Config{MailHost: &host},
		Logger: zap.NewNop(), Bundle: gintest.LoadTestI18nBundle(t),
	})
	job := &river.Job[CredentialNotificationArgs]{Args: CredentialNotificationArgs{
		CredentialID: "cred-1", HolderUserID: "user-1", Event: NotificationEventApproved,
	}}
	assert.NoError(t, worker.Work(context.Background(), job))
	assert.NoError(t, worker.Work(context.Background(), job))
	userRepo.AssertExpectations(t)
	mailer.AssertExpectations(t)
}

func TestCredentialNotificationWorker_Work_MissingTranslationRetries(t *testing.T) {
	host := "smtp.example.com"
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
		Id: "user-1", Email: "jane@example.com", Language: domain.LanguageEN,
	}, nil)
	bundle := i18n.NewBundle(language.English)
	bundle.AddMessages(language.English,
		&i18n.Message{ID: "mail_credential_default_name", Other: "User"},
		&i18n.Message{ID: "mail_credential_approved_title", Other: "Title"},
	)
	mailer := new(mockMailer)
	worker := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo, Mailer: mailer, Config: &config.Config{MailHost: &host},
		Logger: zap.NewNop(), Bundle: bundle,
	})
	job := &river.Job[CredentialNotificationArgs]{Args: CredentialNotificationArgs{
		CredentialID: "cred-1", HolderUserID: "user-1", Event: NotificationEventApproved,
	}}
	err := worker.Work(context.Background(), job)
	assert.ErrorContains(t, err, "mail_credential_approved_message")
	assert.False(t, errors.Is(err, river.JobCancel(err)))
	mailer.AssertNotCalled(t, "SendEmail", mock.Anything, mock.Anything, mock.Anything)
}

func TestCredentialNotificationWorker_Work_IndonesianAllEvents(t *testing.T) {
	for _, test := range []struct {
		event CredentialNotificationEvent
		title string
	}{
		{NotificationEventApproved, "Pengajuan Kredensial Disetujui"},
		{NotificationEventRejected, "Pengajuan Kredensial Ditolak"},
		{NotificationEventRegistered, "Kredensial Didaftarkan"},
		{NotificationEventRevoked, "Kredensial Dicabut"},
	} {
		t.Run(string(test.event), func(t *testing.T) {
			host := "smtp.example.com"
			name := "Jane Doe"
			userRepo := new(mocks.MockUserRepository)
			userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
				Id: "user-1", Name: &name, Email: "jane@example.com", Language: domain.LanguageID,
			}, nil)
			mailer := new(mockMailer)
			mailer.On("SendEmail", mock.Anything, "jane@example.com", mock.MatchedBy(func(data mail.EmailData) bool {
				assert.Equal(t, "id", data.Language)
				assert.Equal(t, test.title, data.Title)
				assert.Equal(t, "Halo Jane Doe", data.Greeting)
				assert.Equal(t, "Lihat Kredensial", data.Buttons[0].Label)
				assert.Equal(t, []mail.EmailDetail{
					{Key: "Nomor Kredensial", Value: "CERT-123"},
					{Key: "Waktu Peristiwa", Value: "25 Juni 2025, 07.00"},
				}, data.Details)
				assert.NotContains(t, data.Message, "Your credential")
				return true
			})).Return(nil)
			worker := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
				UserRepo: userRepo, Mailer: mailer, Config: &config.Config{MailHost: &host},
				Logger: zap.NewNop(), Bundle: gintest.LoadTestI18nBundle(t),
			})
			job := &river.Job[CredentialNotificationArgs]{Args: CredentialNotificationArgs{
				CredentialID: "cred-1", CredentialNumber: "CERT-123", HolderUserID: "user-1",
				Event: test.event, EventTime: time.Date(2025, time.June, 25, 0, 0, 0, 0, time.UTC),
			}}
			assert.NoError(t, worker.Work(context.Background(), job))
			mailer.AssertExpectations(t)
		})
	}
}

func TestCredentialNotificationWorker_Work_MissingNumberOmitsDetail(t *testing.T) {
	host := "smtp.example.com"
	baseURL := "http://localhost:5173"
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
		Id: "user-1", Email: "jane@example.com",
	}, nil)

	mailer := new(mockMailer)
	mailer.On("SendEmail", mock.Anything, "jane@example.com", mock.MatchedBy(func(data mail.EmailData) bool {
		assert.Equal(t, []mail.EmailDetail{
			{Key: "Waktu Peristiwa", Value: "1 Oktober 2026, 06.54"},
		}, data.Details)
		assert.Equal(t, "http://localhost:5173/credentials/cred-1", data.Buttons[0].URL)
		return true
	})).Return(nil)

	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo, Mailer: mailer,
		Config: &config.Config{MailHost: &host, AppBaseURL: &baseURL}, Logger: zap.NewNop(),
		Bundle: gintest.LoadTestI18nBundle(t),
	})
	job := &river.Job[CredentialNotificationArgs]{Args: CredentialNotificationArgs{
		CredentialID: "cred-1", HolderUserID: "user-1", Event: NotificationEventRejected,
		EventTime: time.Date(2026, time.September, 30, 23, 54, 0, 0, time.UTC),
	}}

	assert.NoError(t, w.Work(context.Background(), job))
	mailer.AssertExpectations(t)
}

func TestCredentialNotificationWorker_Work_MailerError_ReturnsRetryableError(t *testing.T) {
	host := "smtp.example.com"
	baseURL := "http://localhost:5173"
	cfg := &config.Config{MailHost: &host, AppBaseURL: &baseURL}
	userRepo := new(mocks.MockUserRepository)
	userRepo.On("Find", mock.Anything, "user-1").Return(&domain.User{
		Id:    "user-1",
		Email: "jane@example.com",
	}, nil)

	mailer := new(mockMailer)
	mailer.On("SendEmail", mock.Anything, "jane@example.com", mock.Anything).Return(errors.New("smtp connection failed"))

	w := NewCredentialNotificationWorker(CredentialNotificationWorkerParams{
		UserRepo: userRepo,
		Mailer:   mailer,
		Config:   cfg,
		Logger:   zap.NewNop(),
		Bundle:   gintest.LoadTestI18nBundle(t),
	})

	job := &river.Job[CredentialNotificationArgs]{
		Args: CredentialNotificationArgs{
			CredentialID: "cred-1",
			HolderUserID: "user-1",
			Event:        NotificationEventApproved,
		},
	}

	err := w.Work(context.Background(), job)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "smtp connection failed")
	// Must NOT be a JobCancel error
	assert.False(t, errors.Is(err, river.JobCancel(err)))
}

func TestRiverEnqueuer_EnqueueCredentialNotifications_EmptyBatch(t *testing.T) {
	enq := &RiverEnqueuer{}
	err := enq.EnqueueCredentialNotifications(context.Background(), nil)
	assert.NoError(t, err)
}

package jobs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/mail"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/riverqueue/river"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CredentialNotificationEvent string

const (
	NotificationEventApproved   CredentialNotificationEvent = "approved"
	NotificationEventRejected   CredentialNotificationEvent = "rejected"
	NotificationEventRegistered CredentialNotificationEvent = "registered"
	NotificationEventRevoked    CredentialNotificationEvent = "revoked"
)

type CredentialNotificationArgs struct {
	CredentialID     string                      `json:"credential_id" river:"unique"`
	CredentialNumber string                      `json:"credential_number,omitempty"`
	HolderUserID     string                      `json:"holder_user_id"`
	Event            CredentialNotificationEvent `json:"event" river:"unique"`
	EventTime        time.Time                   `json:"event_time"`
}

func (CredentialNotificationArgs) Kind() string { return "credential_notification" }

type CredentialNotificationWorker struct {
	river.WorkerDefaults[CredentialNotificationArgs]
	userRepo domain.UserRepository
	mailer   mail.Mailer
	cfg      *config.Config
	logger   *zap.Logger
	bundle   *i18n.Bundle
}

type CredentialNotificationWorkerParams struct {
	fx.In
	UserRepo domain.UserRepository
	Mailer   mail.Mailer
	Config   *config.Config
	Logger   *zap.Logger
	Bundle   *i18n.Bundle
}

func NewCredentialNotificationWorker(p CredentialNotificationWorkerParams) *CredentialNotificationWorker {
	return &CredentialNotificationWorker{
		userRepo: p.UserRepo,
		mailer:   p.Mailer,
		cfg:      p.Config,
		logger:   p.Logger,
		bundle:   p.Bundle,
	}
}

func (w *CredentialNotificationWorker) Work(ctx context.Context, job *river.Job[CredentialNotificationArgs]) error {
	args := job.Args

	if !w.cfg.IsMailEnabled() {
		w.logger.Warn("mail is not enabled, holding notification job for retry",
			zap.String("credential_id", args.CredentialID),
			zap.String("event", string(args.Event)))
		return fmt.Errorf("mail delivery is disabled in configuration")
	}

	u, err := w.userRepo.Find(ctx, args.HolderUserID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		w.logger.Error("failed to load notification recipient",
			zap.Error(err),
			zap.String("credential_id", args.CredentialID),
			zap.String("holder_user_id", args.HolderUserID))
		return fmt.Errorf("load recipient user %q: %w", args.HolderUserID, err)
	}
	if errors.Is(err, gorm.ErrRecordNotFound) || u == nil {
		w.logger.Warn("recipient user not found, cancelling notification job",
			zap.String("credential_id", args.CredentialID),
			zap.String("holder_user_id", args.HolderUserID))
		return river.JobCancel(fmt.Errorf("recipient user %q not found", args.HolderUserID))
	}

	if u.DeletedAt != nil {
		w.logger.Warn("recipient user is deleted, cancelling notification job",
			zap.String("credential_id", args.CredentialID),
			zap.String("holder_user_id", args.HolderUserID))
		return river.JobCancel(fmt.Errorf("recipient user %q is deleted", args.HolderUserID))
	}

	trimmedEmail := strings.TrimSpace(u.Email)
	if trimmedEmail == "" {
		w.logger.Warn("recipient email is empty, cancelling notification job",
			zap.String("credential_id", args.CredentialID),
			zap.String("holder_user_id", args.HolderUserID))
		return river.JobCancel(fmt.Errorf("recipient user %q has empty email", args.HolderUserID))
	}

	switch args.Event {
	case NotificationEventApproved, NotificationEventRejected, NotificationEventRegistered, NotificationEventRevoked:
	default:
		w.logger.Warn("unrecognized notification event, cancelling job", zap.String("event", string(args.Event)))
		return river.JobCancel(fmt.Errorf("unrecognized notification event: %s", args.Event))
	}

	language := u.Language
	if language == "" {
		language = domain.LanguageID
	}
	if language != domain.LanguageID && language != domain.LanguageEN {
		return fmt.Errorf("unsupported recipient language %q", language)
	}
	if w.bundle == nil {
		return fmt.Errorf("notification translations are unavailable")
	}
	localizer := i18n.NewLocalizer(w.bundle, string(language))
	translate := func(key string, data map[string]interface{}) (string, error) {
		value, err := localizer.Localize(&i18n.LocalizeConfig{MessageID: key, TemplateData: data})
		if err != nil || strings.TrimSpace(value) == "" {
			return "", fmt.Errorf("translate %q for %s: %v", key, language, err)
		}
		return value, nil
	}

	name := ""
	if u.Name != nil {
		name = strings.TrimSpace(*u.Name)
	}
	if name == "" {
		var err error
		name, err = translate("mail_credential_default_name", nil)
		if err != nil {
			return err
		}
	}
	copyKeys := []string{
		"mail_credential_" + string(args.Event) + "_title",
		"mail_credential_" + string(args.Event) + "_message",
		"mail_credential_greeting", "mail_credential_number", "mail_credential_event_time",
		"mail_credential_view", "mail_fallback", "mail_footer",
	}
	copy := make([]string, len(copyKeys))
	for index, key := range copyKeys {
		var err error
		copy[index], err = translate(key, map[string]interface{}{"Name": name})
		if err != nil {
			return err
		}
	}

	baseURL := "http://localhost:5173"
	if w.cfg.AppBaseURL != nil && *w.cfg.AppBaseURL != "" {
		baseURL = *w.cfg.AppBaseURL
	}
	credURL := fmt.Sprintf("%s/credentials/%s", strings.TrimRight(baseURL, "/"), args.CredentialID)
	wib := args.EventTime.In(time.FixedZone("WIB", 7*60*60))
	eventTimeStr := wib.Format("Jan 2, 2006, 03:04 PM")
	if language == domain.LanguageID {
		months := [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}
		eventTimeStr = fmt.Sprintf("%d %s %d, %02d.%02d", wib.Day(), months[int(wib.Month())-1], wib.Year(), wib.Hour(), wib.Minute())
	}
	details := make([]mail.EmailDetail, 0, 2)
	if number := strings.TrimSpace(args.CredentialNumber); number != "" {
		details = append(details, mail.EmailDetail{Key: copy[3], Value: number})
	}
	details = append(details, mail.EmailDetail{Key: copy[4], Value: eventTimeStr})
	emailData := mail.EmailData{
		Language: string(language), Title: copy[0], Message: copy[1], Greeting: copy[2],
		Details: details, Buttons: []mail.EmailButton{{Label: copy[5], URL: credURL}},
		FallbackURL: credURL, FallbackText: copy[6], Footer: copy[7],
	}

	if err := w.mailer.SendEmail(ctx, trimmedEmail, emailData); err != nil {
		w.logger.Error("failed to send credential notification email",
			zap.Error(err),
			zap.String("credential_id", args.CredentialID),
			zap.String("event", string(args.Event)))
		return fmt.Errorf("send email: %w", err)
	}

	w.logger.Info("credential notification email sent successfully",
		zap.String("credential_id", args.CredentialID),
		zap.String("event", string(args.Event)))
	return nil
}

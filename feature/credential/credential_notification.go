package credential

import (
	"context"
	"time"

	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/jobs"

	"go.uber.org/zap"
)

func (s *credentialService) enqueueLifecycleNotifications(
	ctx context.Context,
	creds []domain.Credential,
	event jobs.CredentialNotificationEvent,
) {
	if s.enqueuer == nil || len(creds) == 0 {
		return
	}
	if s.cfg != nil && !s.cfg.IsMailEnabled() {
		return
	}

	now := time.Now()
	seen := make(map[string]bool, len(creds))
	args := make([]jobs.CredentialNotificationArgs, 0, len(creds))

	for _, c := range creds {
		if c.ID == "" || c.HolderUserID == "" {
			continue
		}
		if seen[c.ID] {
			continue
		}
		seen[c.ID] = true

		number := ""
		if c.Number != nil {
			number = *c.Number
		}
		args = append(args, jobs.CredentialNotificationArgs{
			CredentialID:     c.ID,
			CredentialNumber: number,
			HolderUserID:     c.HolderUserID,
			Event:            event,
			EventTime:        now,
		})
	}

	if len(args) == 0 {
		return
	}

	bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()

	if err := s.enqueuer.EnqueueCredentialNotifications(bgCtx, args); err != nil {
		s.logger.Warn("failed to enqueue credential lifecycle notification jobs",
			zap.Error(err),
			zap.String("event", string(event)),
			zap.Int("count", len(args)))
	}
}

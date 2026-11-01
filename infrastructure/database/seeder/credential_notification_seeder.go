package seeder

import (
	"context"
	"fmt"
	"time"

	"CredChain_Golang/domain"
)

type CredentialNotificationSeeder struct {
	credRepo         domain.CredentialRepository
	notificationRepo domain.CredentialNotificationRepository
}

func NewCredentialNotificationSeeder(credRepo domain.CredentialRepository, notificationRepo domain.CredentialNotificationRepository) *CredentialNotificationSeeder {
	return &CredentialNotificationSeeder{credRepo: credRepo, notificationRepo: notificationRepo}
}

func (s *CredentialNotificationSeeder) Name() string { return "credential-notification" }

func (s *CredentialNotificationSeeder) Seed(ctx context.Context) error {
	ids := seedCredIDs()
	credentials, err := s.credRepo.FindByIds(ctx, ids, nil)
	if err != nil {
		return fmt.Errorf("credential notification seeder: fetch credentials: %w", err)
	}
	byID := make(map[string]domain.Credential, len(credentials))
	for _, c := range credentials {
		byID[c.ID] = c
	}

	var notifications []domain.CredentialNotification
	found := false
	for slot, id := range ids {
		c, ok := byID[id]
		if !ok {
			continue
		}
		found = true
		appendEvent := func(kind domain.CredentialNotificationType, ordinal int, at time.Time) {
			n := domain.CredentialNotification{
				ID:             deterministicULID(uint32(10000 + 4*slot + ordinal)),
				NotifiedUserID: c.HolderUserID,
				CredentialID:   c.ID,
				Type:           kind,
				CreatedAt:      at,
			}
			if (slot+ordinal)%2 == 0 {
				readAt := at.Add(time.Hour)
				n.ReadAt = &readAt
			}
			notifications = append(notifications, n)
		}
		if c.SubmitterUserID != c.HolderUserID {
			appendEvent(domain.CredentialNotificationTypeRegistered, 0, c.CreatedAt)
		} else if c.ActivatedAt != nil {
			appendEvent(domain.CredentialNotificationTypeApproved, 1, *c.ActivatedAt)
		}
		if c.RejectedAt != nil {
			appendEvent(domain.CredentialNotificationTypeRejected, 2, *c.RejectedAt)
		}
		if c.RevokedAt != nil {
			appendEvent(domain.CredentialNotificationTypeRevoked, 3, *c.RevokedAt)
		}
	}
	if !found {
		return fmt.Errorf("credential notification seeder: no seeded credentials found; run credential seeder first")
	}
	if len(notifications) == 0 {
		return nil
	}
	if err := s.notificationRepo.Store(ctx, notifications...); err != nil {
		return fmt.Errorf("credential notification seeder: store notifications: %w", err)
	}
	return nil
}

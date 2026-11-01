package response

import (
	"time"

	"CredChain_Golang/domain"
)

type CredentialNotificationSummaryCredential struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Number *string `json:"number"`
}

type CredentialNotificationResponse struct {
	ID             string                                   `json:"id"`
	NotifiedUserID string                                   `json:"notified_user_id"`
	CredentialID   string                                   `json:"credential_id"`
	Type           domain.CredentialNotificationType        `json:"type"`
	CreatedAt      time.Time                                `json:"created_at"`
	ReadAt         *time.Time                               `json:"read_at"`
	Credential     *CredentialNotificationSummaryCredential `json:"credential"`
}

type CredentialNotificationUnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}

type CredentialNotificationReadCountResponse struct {
	UpdatedCount int64 `json:"updated_count"`
}

type CredentialNotificationPageResponse struct {
	Items      []CredentialNotificationResponse `json:"items"`
	Total      int64                            `json:"total"`
	NextCursor *string                          `json:"next_cursor"`
}

func FromDomainCredentialNotification(n domain.CredentialNotification) CredentialNotificationResponse {
	res := CredentialNotificationResponse{
		ID:             n.ID,
		NotifiedUserID: n.NotifiedUserID,
		CredentialID:   n.CredentialID,
		Type:           n.Type,
		CreatedAt:      n.CreatedAt,
		ReadAt:         n.ReadAt,
	}
	if n.Credential != nil {
		res.Credential = &CredentialNotificationSummaryCredential{
			ID:     n.Credential.ID,
			Name:   n.Credential.Name,
			Number: n.Credential.Number,
		}
	}
	return res
}

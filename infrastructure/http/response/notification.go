package response

import (
	"CredChain_Golang/domain"
	"time"
)

type NotificationSubject struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Number *string `json:"number"`
	Status *string `json:"status"`
}

type NotificationResponse struct {
	ID             string                          `json:"id"`
	NotifiedUserID string                          `json:"notified_user_id"`
	SubjectType    *domain.NotificationSubjectType `json:"subject_type"`
	SubjectID      *string                         `json:"subject_id"`
	Type           domain.NotificationType         `json:"type"`
	Data           map[string]any                  `json:"data"`
	CreatedAt      time.Time                       `json:"created_at"`
	UpdatedAt      *time.Time                      `json:"updated_at"`
	ReadAt         *time.Time                      `json:"read_at"`
	Subject        *NotificationSubject            `json:"subject"`
}

type NotificationUnreadCountResponse struct {
	UnreadCount int64 `json:"unread_count"`
}
type NotificationReadCountResponse struct {
	UpdatedCount int64 `json:"updated_count"`
}
type NotificationPageResponse struct {
	Items      []NotificationResponse `json:"items"`
	Total      int64                  `json:"total"`
	NextCursor *string                `json:"next_cursor"`
}

func FromDomainNotification(n domain.Notification) NotificationResponse {
	r := NotificationResponse{ID: n.ID, NotifiedUserID: n.NotifiedUserID, SubjectType: n.SubjectType, SubjectID: n.SubjectID, Type: n.Type, Data: n.Data, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt, ReadAt: n.ReadAt}
	if n.Subject != nil {
		r.Subject = &NotificationSubject{ID: n.Subject.ID, Name: n.Subject.Name, Number: n.Subject.Number, Status: n.Subject.Status}
	}
	return r
}

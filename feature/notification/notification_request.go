package notification

import (
	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type NotificationListRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Cursor string `form:"cursor"`
}

func (r NotificationListRequest) Validate() error {
	return validation.ValidateStruct(&r, validation.Field(&r.Status, validation.In("", "all", "unread")), validation.Field(&r.Limit, validation.Min(0), validation.Max(100)))
}
func (r NotificationListRequest) ToDomain() domain.NotificationListOptions {
	status, limit := domain.NotificationStatus(r.Status), r.Limit
	if status == "" {
		status = domain.NotificationStatusAll
	}
	if limit == 0 {
		limit = 20
	}
	return domain.NotificationListOptions{Status: status, Limit: limit, Cursor: r.Cursor}
}

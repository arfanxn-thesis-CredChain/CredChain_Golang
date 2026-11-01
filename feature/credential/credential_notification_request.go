package credential

import (
	"CredChain_Golang/domain"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type CredentialNotificationListRequest struct {
	Status string `form:"status"`
	Limit  int    `form:"limit"`
	Cursor string `form:"cursor"`
}

func (r CredentialNotificationListRequest) Validate() error {
	return validation.ValidateStruct(&r,
		validation.Field(&r.Status, validation.In("", "all", "unread")),
		validation.Field(&r.Limit, validation.Min(0), validation.Max(100)),
	)
}

func (r CredentialNotificationListRequest) ToDomain() domain.CredentialNotificationListOptions {
	limit := r.Limit
	if limit <= 0 {
		limit = 20
	}
	opts := domain.CredentialNotificationListOptions{
		Limit:  limit,
		Cursor: r.Cursor,
	}
	if r.Status == string(domain.CredentialNotificationStatusUnread) {
		opts.Status = domain.CredentialNotificationStatusUnread
	} else {
		opts.Status = domain.CredentialNotificationStatusAll
	}
	return opts
}

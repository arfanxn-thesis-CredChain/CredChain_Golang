package notification

import (
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type NotificationHandler interface {
	List(*gin.Context)
	UnreadCount(*gin.Context)
	MarkRead(*gin.Context)
	MarkAllRead(*gin.Context)
}
type notificationHandler struct{ svc NotificationService }
type NotificationHandlerParams struct {
	fx.In
	Svc NotificationService
}

func NewNotificationHandler(p NotificationHandlerParams) NotificationHandler {
	return &notificationHandler{svc: p.Svc}
}

func (h *notificationHandler) List(c *gin.Context) {
	var req NotificationListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	page, err := h.svc.List(c.Request.Context(), httpContext.MustGetUser(c.Request.Context()).Id, req.ToDomain())
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	items := make([]response.NotificationResponse, len(page.Items))
	for i, n := range page.Items {
		items[i] = response.FromDomainNotification(n)
	}
	responder.Send(c, domain.CodeNotificationFetchSuccess, response.NotificationPageResponse{Items: items, Total: page.Total, NextCursor: page.NextCursor})
}
func (h *notificationHandler) UnreadCount(c *gin.Context) {
	count, err := h.svc.UnreadCount(c.Request.Context(), httpContext.MustGetUser(c.Request.Context()).Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeNotificationUnreadCountSuccess, response.NotificationUnreadCountResponse{UnreadCount: count})
}
func (h *notificationHandler) MarkRead(c *gin.Context) {
	n, err := h.svc.MarkRead(c.Request.Context(), httpContext.MustGetUser(c.Request.Context()).Id, c.Param("id"))
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeNotificationMarkReadSuccess, response.FromDomainNotification(*n))
}
func (h *notificationHandler) MarkAllRead(c *gin.Context) {
	count, err := h.svc.MarkAllRead(c.Request.Context(), httpContext.MustGetUser(c.Request.Context()).Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeNotificationMarkAllReadSuccess, response.NotificationReadCountResponse{UpdatedCount: count})
}

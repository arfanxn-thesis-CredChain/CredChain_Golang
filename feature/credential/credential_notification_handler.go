package credential

import (
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type CredentialNotificationHandler interface {
	List(c *gin.Context)
	UnreadCount(c *gin.Context)
	MarkRead(c *gin.Context)
	MarkAllRead(c *gin.Context)
}

type credentialNotificationHandler struct {
	svc CredentialNotificationService
}

type CredentialNotificationHandlerParams struct {
	fx.In
	Svc CredentialNotificationService
}

func NewCredentialNotificationHandler(p CredentialNotificationHandlerParams) CredentialNotificationHandler {
	return &credentialNotificationHandler{
		svc: p.Svc,
	}
}

func (h *credentialNotificationHandler) List(c *gin.Context) {
	var req CredentialNotificationListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	user := httpContext.MustGetUser(c.Request.Context())
	opts := req.ToDomain()

	page, err := h.svc.List(c.Request.Context(), user.Id, opts)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	items := make([]response.CredentialNotificationResponse, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, response.FromDomainCredentialNotification(item))
	}

	responder.Send(c, domain.CodeCredentialNotificationFetchSuccess, response.CredentialNotificationPageResponse{
		Items:      items,
		Total:      page.Total,
		NextCursor: page.NextCursor,
	})
}

func (h *credentialNotificationHandler) UnreadCount(c *gin.Context) {
	user := httpContext.MustGetUser(c.Request.Context())
	count, err := h.svc.UnreadCount(c.Request.Context(), user.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeCredentialNotificationUnreadCountSuccess, response.CredentialNotificationUnreadCountResponse{
		UnreadCount: count,
	})
}

func (h *credentialNotificationHandler) MarkRead(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		responder.SendError(c, domain.NewError(domain.CodeCredentialNotificationNotFound, domain.WithMetadata("id", id)))
		return
	}

	user := httpContext.MustGetUser(c.Request.Context())
	notif, err := h.svc.MarkRead(c.Request.Context(), user.Id, id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeCredentialNotificationMarkReadSuccess, response.FromDomainCredentialNotification(*notif))
}

func (h *credentialNotificationHandler) MarkAllRead(c *gin.Context) {
	user := httpContext.MustGetUser(c.Request.Context())
	count, err := h.svc.MarkAllRead(c.Request.Context(), user.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeCredentialNotificationMarkAllReadSuccess, response.CredentialNotificationReadCountResponse{
		UpdatedCount: count,
	})
}

package overview

import (
	"CredChain_Golang/domain"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type OverviewHandler interface {
	Get(c *gin.Context)
	CredentialRecap(c *gin.Context)
}

type overviewHandler struct {
	svc OverviewService
}

type OverviewHandlerParams struct {
	fx.In
	Svc OverviewService
}

func NewOverviewHandler(p OverviewHandlerParams) OverviewHandler {
	return &overviewHandler{svc: p.Svc}
}

func (h *overviewHandler) CredentialRecap(c *gin.Context) {
	recap, err := h.svc.CredentialRecap(c.Request.Context())
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeOverviewSuccess, recap)
}

func (h *overviewHandler) Get(c *gin.Context) {
	var req queryRequest.QueryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	query, err := req.ToDomain()
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	ov, err := h.svc.Get(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	responder.Send(c, domain.CodeOverviewSuccess, ov)
}

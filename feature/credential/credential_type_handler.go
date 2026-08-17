package credential

import (
	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

// lookupDefaultPageSize is the handler-side default for lookup-table lists:
// small reference tables, one request should fill a dropdown.
const lookupDefaultPageSize = 100

type CredentialTypeHandler interface {
	Paginate(c *gin.Context)
	Store(c *gin.Context)
	Update(c *gin.Context)
	Destroy(c *gin.Context)
}

type credentialTypeHandler struct {
	credentialTypeSvc CredentialTypeService
}

type CredentialTypeHandlerParams struct {
	fx.In
	CredentialTypeSvc CredentialTypeService
}

func NewCredentialTypeHandler(p CredentialTypeHandlerParams) CredentialTypeHandler {
	return &credentialTypeHandler{credentialTypeSvc: p.CredentialTypeSvc}
}

func (h *credentialTypeHandler) Paginate(c *gin.Context) {
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
	if query == nil {
		query = &domainQuery.Query{}
	}
	if req.Page == 0 && req.Limit == 0 {
		query.Limit = lookupDefaultPageSize
	}
	types, err := h.credentialTypeSvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := make([]response.CredentialType, len(types))
	for i, t := range types {
		out[i] = response.FromDomainCredentialType(t)
	}
	responder.Send(c, domain.CodeCredentialTypeFetchSuccess, out)
}

func (h *credentialTypeHandler) Store(c *gin.Context) {
	var req CredentialTypeStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	created, err := h.credentialTypeSvc.Store(c.Request.Context(), req.Name, req.Active)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCredentialTypeStoreSuccess, response.FromDomainCredentialType(*created))
}

func (h *credentialTypeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req CredentialTypeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	updated, err := h.credentialTypeSvc.Update(c.Request.Context(), id, req.Name, req.Active)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCredentialTypeUpdateSuccess, response.FromDomainCredentialType(*updated))
}

func (h *credentialTypeHandler) Destroy(c *gin.Context) {
	id := c.Param("id")
	destroyed, err := h.credentialTypeSvc.Destroy(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCredentialTypeDestroySuccess, gin.H{"destroyed_count": destroyed})
}

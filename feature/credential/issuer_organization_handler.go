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

type CredentialIssuerOrganizationHandler interface {
	Paginate(c *gin.Context)
	Store(c *gin.Context)
	Update(c *gin.Context)
	Destroy(c *gin.Context)
}

type credentialIssuerOrganizationHandler struct {
	issuerOrganizationSvc CredentialIssuerOrganizationService
}

type CredentialIssuerOrganizationHandlerParams struct {
	fx.In
	IssuerOrganizationSvc CredentialIssuerOrganizationService
}

func NewCredentialIssuerOrganizationHandler(p CredentialIssuerOrganizationHandlerParams) CredentialIssuerOrganizationHandler {
	return &credentialIssuerOrganizationHandler{issuerOrganizationSvc: p.IssuerOrganizationSvc}
}

func (h *credentialIssuerOrganizationHandler) Paginate(c *gin.Context) {
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
	orgs, total, err := h.issuerOrganizationSvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := make([]response.IssuerOrganization, len(orgs))
	for i, o := range orgs {
		out[i] = response.FromDomainIssuerOrganization(o)
	}
	responder.SendPaginationWithPageLimit(c, domain.CodeIssuerOrganizationFetchSuccess, out, total, query.Page, query.Limit)
}

func (h *credentialIssuerOrganizationHandler) Store(c *gin.Context) {
	var req IssuerOrganizationStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	created, err := h.issuerOrganizationSvc.Store(c.Request.Context(), req.Name, req.Active)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeIssuerOrganizationStoreSuccess, response.FromDomainIssuerOrganization(*created))
}

func (h *credentialIssuerOrganizationHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req IssuerOrganizationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	updated, err := h.issuerOrganizationSvc.Update(c.Request.Context(), id, req.Name, req.Active)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeIssuerOrganizationUpdateSuccess, response.FromDomainIssuerOrganization(*updated))
}

func (h *credentialIssuerOrganizationHandler) Destroy(c *gin.Context) {
	id := c.Param("id")
	destroyed, err := h.issuerOrganizationSvc.Destroy(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeIssuerOrganizationDestroySuccess, gin.H{"destroyed_count": destroyed})
}

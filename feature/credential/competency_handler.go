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

type CompetencyHandler interface {
	Paginate(c *gin.Context)
	Store(c *gin.Context)
	Update(c *gin.Context)
	Destroy(c *gin.Context)
}

type competencyHandler struct {
	competencySvc CompetencyService
}

type CompetencyHandlerParams struct {
	fx.In
	CompetencySvc CompetencyService
}

func NewCompetencyHandler(p CompetencyHandlerParams) CompetencyHandler {
	return &competencyHandler{competencySvc: p.CompetencySvc}
}

func (h *competencyHandler) Paginate(c *gin.Context) {
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
	competencies, err := h.competencySvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := make([]response.Competency, len(competencies))
	for i, cpt := range competencies {
		out[i] = response.FromDomainCompetency(cpt)
	}
	responder.Send(c, domain.CodeCompetencyFetchSuccess, out)
}

func (h *competencyHandler) Store(c *gin.Context) {
	var req CompetencyStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	created, err := h.competencySvc.Store(c.Request.Context(), req.Name)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCompetencyStoreSuccess, response.FromDomainCompetency(*created))
}

func (h *competencyHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req CompetencyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	updated, err := h.competencySvc.Update(c.Request.Context(), id, req.Name)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCompetencyUpdateSuccess, response.FromDomainCompetency(*updated))
}

func (h *competencyHandler) Destroy(c *gin.Context) {
	id := c.Param("id")
	destroyed, err := h.competencySvc.Destroy(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeCompetencyDestroySuccess, gin.H{"destroyed_count": destroyed})
}

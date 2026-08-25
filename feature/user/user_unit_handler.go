package user

import (
	"encoding/json"

	"CredChain_Golang/domain"
	domainQuery "CredChain_Golang/domain/query"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.uber.org/fx"
)

// lookupDefaultPageSize is the handler-side default when the client sends
// neither page nor limit. Units render as a tree, so a truncated response is
// structurally broken rather than merely short — the default is high enough to
// return whole org charts in one request instead of silently cutting them off
// at the old 100.
const lookupDefaultPageSize = 1000

type UserUnitHandler interface {
	Paginate(c *gin.Context)
	Store(c *gin.Context)
	Update(c *gin.Context)
	Destroy(c *gin.Context)
}

type userUnitHandler struct {
	userUnitSvc UserUnitService
}

type UserUnitHandlerParams struct {
	fx.In
	UserUnitSvc UserUnitService
}

func NewUserUnitHandler(p UserUnitHandlerParams) UserUnitHandler {
	return &userUnitHandler{userUnitSvc: p.UserUnitSvc}
}

func (h *userUnitHandler) Paginate(c *gin.Context) {
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
	units, err := h.userUnitSvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	out := make([]response.UserUnit, len(units))
	for i, u := range units {
		out[i] = response.FromDomainUserUnit(u)
	}
	responder.Send(c, domain.CodeUserUnitFetchSuccess, out)
}

func (h *userUnitHandler) Store(c *gin.Context) {
	var req UserUnitStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	created, err := h.userUnitSvc.Store(c.Request.Context(), req.Name, req.ParentID)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserUnitStoreSuccess, response.FromDomainUserUnit(*created))
}

func (h *userUnitHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req UserUnitUpdateRequest
	if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	// Detect whether parent_id was present in the body so a null value clears the
	// parent (move to root) — distinct from omitting the field, which leaves the
	// parent untouched. A *string alone can't tell JSON null from an absent key.
	var raw map[string]json.RawMessage
	_ = c.ShouldBindBodyWith(&raw, binding.JSON)
	_, setParent := raw["parent_id"]
	updated, err := h.userUnitSvc.Update(c.Request.Context(), id, req.Name, req.ParentID, setParent)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserUnitUpdateSuccess, response.FromDomainUserUnit(*updated))
}

func (h *userUnitHandler) Destroy(c *gin.Context) {
	id := c.Param("id")
	destroyed, err := h.userUnitSvc.Destroy(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserUnitDestroySuccess, gin.H{"destroyed_count": destroyed})
}

package user

import (
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	queryRequest "CredChain_Golang/infrastructure/http/request/query"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"

	"github.com/gin-gonic/gin"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"go.uber.org/fx"
)

type UserHandler interface {
	Paginate(c *gin.Context)
	Self(c *gin.Context)
	Find(c *gin.Context)
	Store(c *gin.Context)
	Update(c *gin.Context)
	UpdateSelfEmail(c *gin.Context)
	UpdateRole(c *gin.Context)
	Delete(c *gin.Context)
	TransferSuperAdmin(c *gin.Context)
	Restore(c *gin.Context)
}

type userHandler struct {
	userSvc UserService
}

type UserHandlerParams struct {
	fx.In
	UserSvc UserService
}

func NewUserHandler(p UserHandlerParams) UserHandler {
	return &userHandler{userSvc: p.UserSvc}
}

func (h *userHandler) Paginate(c *gin.Context) {
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
	users, total, err := h.userSvc.Paginate(c.Request.Context(), query)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responseUsers := make([]response.User, len(users))
	for i, u := range users {
		responseUsers[i] = response.FromDomainUser(u)
	}
	responder.SendPagination(c, domain.CodeUserFetchSuccess, responseUsers, total)
}

func (h *userHandler) Self(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())
	user, err := h.userSvc.Find(c.Request.Context(), authUser.Id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, response.FromDomainUser(*user))
}

func (h *userHandler) Find(c *gin.Context) {
	id := c.Param("id")
	user, err := h.userSvc.Find(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserFetchSuccess, response.FromDomainUser(*user))
}

func (h *userHandler) Store(c *gin.Context) {
	var req UserStoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	domainUsers := req.ToDomain()
	created, err := h.userSvc.Store(c.Request.Context(), domainUsers...)
	if err != nil {
		c.Error(err)
		if verrs, ok := err.(validation.Errors); ok {
			responder.SendValidationError(c, verrs)
			return
		}
		responder.SendError(c, err)
		return
	}
	users := make([]response.User, len(created))
	for i, u := range created {
		users[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserStoreSuccess, users)
}

func (h *userHandler) UpdateSelfEmail(c *gin.Context) {
	authUser := httpContext.MustGetUser(c.Request.Context())
	var req UserUpdateSelfEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	newEmail, err := h.userSvc.UpdateEmail(c.Request.Context(), authUser.Id, req.Email, req.IdToken)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserEmailSuccess, gin.H{"email": newEmail})
}

func (h *userHandler) UpdateRole(c *gin.Context) {
	var req UserUpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	var domainUpdates []domain.UserRoleUpdate
	for _, u := range req.UserRoles {
		if err := u.Validate(); err != nil {
			responder.SendValidationError(c, err)
			return
		}
		domainUpdates = append(domainUpdates, domain.UserRoleUpdate{UserID: u.UserID, Role: u.Role})
	}
	updatedUsers, _, err := h.userSvc.UpdateRole(c.Request.Context(), domainUpdates...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responseUsers := make([]response.User, len(updatedUsers))
	for i, u := range updatedUsers {
		responseUsers[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserRoleSuccess, responseUsers)
}

func (h *userHandler) Delete(c *gin.Context) {
	var req UserDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if validationErr := req.Validate(); validationErr != nil {
		responder.SendValidationError(c, validationErr)
		return
	}
	deletedCount, err := h.userSvc.Delete(c.Request.Context(), req.Ids...)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserBatchDeleteSuccess, gin.H{"deleted_count": deletedCount})
}

func (h *userHandler) TransferSuperAdmin(c *gin.Context) {
	var req UserTransferSuperAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	if err := h.userSvc.TransferSuperAdmin(c.Request.Context(), req.Id); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responder.Send(c, domain.CodeUserTransferSuperAdminSuccess, gin.H{})
}

func (h *userHandler) Update(c *gin.Context) {
	var req UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	users := req.ToDomain()
	updated, err := h.userSvc.Update(c.Request.Context(), users...)
	if err != nil {
		c.Error(err)
		if verrs, ok := err.(validation.Errors); ok {
			responder.SendValidationError(c, verrs)
			return
		}
		responder.SendError(c, err)
		return
	}
	out := make([]response.User, len(updated))
	for i, u := range updated {
		out[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserUpdateSuccess, out)
}

func (h *userHandler) Restore(c *gin.Context) {
	var req UserRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}
	users, count, err := h.userSvc.Restore(c.Request.Context(), req.IDs)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}
	responseUsers := make([]response.User, len(users))
	for i, u := range users {
		responseUsers[i] = response.FromDomainUser(u)
	}
	responder.Send(c, domain.CodeUserRestoreSuccess, gin.H{
		"users":          responseUsers,
		"restored_count": count,
	})
}

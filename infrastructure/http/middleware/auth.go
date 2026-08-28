package middleware

import (
	"context"
	"errors"
	"strings"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/infrastructure/chain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/security"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

type AuthMiddlewareParams struct {
	fx.In
	Config   *config.Config
	UserRepo domain.UserRepository
}

type RoleMiddlewareParams struct {
	fx.In
	AuthorityService chain.AuthorityService
}

// Named wrapper types for role-based middlewares
type AdminRoleMiddleware gin.HandlerFunc
type IssuerRoleMiddleware gin.HandlerFunc
type SuperAdminRoleMiddleware gin.HandlerFunc

// NewAuthMiddleware validates JWT token and stores full user in context.
// Reads token from access_token cookie first; falls back to Authorization: Bearer header.
func NewAuthMiddleware(p AuthMiddlewareParams) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// Prefer cookie (frontend uses httpOnly cookies)
		if cookie, err := c.Cookie("access_token"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// Fall back to Authorization: Bearer (CLI tooling, Postman, init flows)
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" {
				c.Abort()
				responder.SendError(c, domain.NewError(domain.CodeAuthUnauthorized))
				return
			}
			trimmed := strings.TrimPrefix(authHeader, "Bearer ")
			if trimmed == authHeader {
				c.Abort()
				responder.SendError(c, domain.NewError(domain.CodeAuthUnauthorized))
				return
			}
			tokenString = trimmed
		}

		claims, err := security.ValiparseJWT(tokenString, []byte(*p.Config.JWTSecret))
		if err != nil {
			c.Abort()
			responder.SendError(c, domain.NewError(domain.CodeAuthInvalidToken))
			return
		}

		// Fetch full user from DB (unscoped — returns trashed users too)
		user, err := p.UserRepo.Find(c.Request.Context(), claims.UserId)
		if err != nil {
			c.Abort()
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// A token referencing a user no longer in the DB is a stale
				// session, not a 404 — treat it like the soft-delete case below.
				responder.SendError(c, domain.NewError(domain.CodeAuthUnauthorized))
			} else {
				responder.SendError(c, domain.NewError(domain.CodeSystemInternal, domain.WithError(err)))
			}
			return
		}

		// Reject soft-deleted users: their refresh tokens are already revoked at
		// delete-time, but a valid access token may still exist until natural expiry.
		if user.DeletedAt != nil {
			c.Abort()
			responder.SendError(c, domain.NewError(domain.CodeAuthUnauthorized))
			return
		}

		// Store in both Gin context and Go context with UserKey
		c.Set(httpContext.UserKey, user)
		ctx := context.WithValue(c.Request.Context(), httpContext.UserKey, user)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// NewAdminRoleMiddleware enforces Admin or higher role requirement
func NewAdminRoleMiddleware(p RoleMiddlewareParams) AdminRoleMiddleware {
	return AdminRoleMiddleware(func(c *gin.Context) {
		user, err := httpContext.GetUser(c.Request.Context())
		if err != nil {
			responder.SendError(c, domain.NewError(domain.CodeAuthUnauthorized))
			c.Abort()
			return
		}
		if !p.AuthorityService.HasRoleOrAbove(c.Request.Context(), user.WalletAddress, domain.RoleAdmin) {
			responder.SendError(c, domain.NewError(domain.CodeAuthForbidden))
			c.Abort()
			return
		}
		c.Next()
	})
}

// NewIssuerRoleMiddleware enforces Issuer or higher role requirement
func NewIssuerRoleMiddleware(p RoleMiddlewareParams) IssuerRoleMiddleware {
	return IssuerRoleMiddleware(func(c *gin.Context) {
		user, err := httpContext.GetUser(c.Request.Context())
		if err != nil {
			responder.SendError(c, domain.NewError(domain.CodeAuthUnauthorized))
			c.Abort()
			return
		}
		if !p.AuthorityService.HasRoleOrAbove(c.Request.Context(), user.WalletAddress, domain.RoleIssuer) {
			responder.SendError(c, domain.NewError(domain.CodeAuthForbidden))
			c.Abort()
			return
		}
		c.Next()
	})
}

// NewSuperAdminRoleMiddleware enforces SuperAdmin role requirement
func NewSuperAdminRoleMiddleware(p RoleMiddlewareParams) SuperAdminRoleMiddleware {
	return SuperAdminRoleMiddleware(func(c *gin.Context) {
		user, err := httpContext.GetUser(c.Request.Context())
		if err != nil {
			responder.SendError(c, domain.NewError(domain.CodeAuthUnauthorized))
			c.Abort()
			return
		}
		if !p.AuthorityService.HasRoleOrAbove(c.Request.Context(), user.WalletAddress, domain.RoleSuperAdmin) {
			responder.SendError(c, domain.NewError(domain.CodeAuthForbidden))
			c.Abort()
			return
		}
		c.Next()
	})
}

package auth

import (
	"net/http"
	"strings"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/http/responder"
	"CredChain_Golang/infrastructure/http/response"
	"CredChain_Golang/infrastructure/security"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
)

type AuthHandler interface {
	GoogleLogin(c *gin.Context)
	Refresh(c *gin.Context)
	Logout(c *gin.Context)
}

type authHandler struct {
	service AuthService
	config  *config.Config
}

type AuthHandlerParams struct {
	fx.In
	Service AuthService
	Config  *config.Config
}

func NewAuthHandler(p AuthHandlerParams) AuthHandler {
	return &authHandler{service: p.Service, config: p.Config}
}

const (
	CookieAccessToken  = "access_token"
	CookieRefreshToken = "refresh_token"
)

func parseSameSite(s string) http.SameSite {
	switch strings.ToLower(s) {
	case "lax":
		return http.SameSiteLaxMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteStrictMode
	}
}

func (h *authHandler) setAuthCookie(c *gin.Context, name, value, path string, ttlSec int) {
	domain := ""
	if h.config.CookieDomain != nil {
		domain = *h.config.CookieDomain
	}
	secure := false
	if h.config.CookieSecure != nil {
		secure = *h.config.CookieSecure
	}
	sameSite := http.SameSiteStrictMode
	if h.config.CookieSameSite != nil {
		sameSite = parseSameSite(*h.config.CookieSameSite)
	}
	c.SetSameSite(sameSite)
	c.SetCookie(name, value, ttlSec, path, domain, secure, true)
}

func (h *authHandler) clearAuthCookie(c *gin.Context, name, path string) {
	domain := ""
	if h.config.CookieDomain != nil {
		domain = *h.config.CookieDomain
	}
	secure := false
	if h.config.CookieSecure != nil {
		secure = *h.config.CookieSecure
	}
	sameSite := http.SameSiteStrictMode
	if h.config.CookieSameSite != nil {
		sameSite = parseSameSite(*h.config.CookieSameSite)
	}
	c.SetSameSite(sameSite)
	c.SetCookie(name, "", -1, path, domain, secure, true)
}

func (h *authHandler) setAuthCookies(c *gin.Context, accessToken, refreshToken string) {
	accessTTL := *h.config.JWTAccessExpiryMinutes * int(time.Minute.Seconds())
	refreshTTL := *h.config.JWTRefreshExpiryHours * int(time.Hour.Seconds())
	accessPath := "/api"
	if h.config.CookieAccessPath != nil {
		accessPath = *h.config.CookieAccessPath
	}
	refreshPath := "/api/auth"
	if h.config.CookieRefreshPath != nil {
		refreshPath = *h.config.CookieRefreshPath
	}
	h.setAuthCookie(c, CookieAccessToken, accessToken, accessPath, accessTTL)
	h.setAuthCookie(c, CookieRefreshToken, refreshToken, refreshPath, refreshTTL)
}

func (h *authHandler) clearAuthCookies(c *gin.Context) {
	accessPath := "/api"
	if h.config.CookieAccessPath != nil {
		accessPath = *h.config.CookieAccessPath
	}
	refreshPath := "/api/auth"
	if h.config.CookieRefreshPath != nil {
		refreshPath = *h.config.CookieRefreshPath
	}
	h.clearAuthCookie(c, CookieAccessToken, accessPath)
	h.clearAuthCookie(c, CookieRefreshToken, refreshPath)
}

func (h *authHandler) sendAuthResponse(c *gin.Context, code int, user domain.User, refreshToken domain.UserToken, accessToken string) {
	refreshExpirySec := *h.config.JWTRefreshExpiryHours * int(time.Hour.Seconds())
	accessExpirySec := *h.config.JWTAccessExpiryMinutes * int(time.Minute.Seconds())
	h.setAuthCookies(c, accessToken, refreshToken.Token)
	responder.Send(c, code, response.NewAuth(
		response.FromDomainUser(user),
		accessToken,
		refreshToken.Token,
		accessExpirySec,
		refreshExpirySec,
	))
}

func (h *authHandler) GoogleLogin(c *gin.Context) {
	var req AuthGoogleLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	if err := req.Validate(); err != nil {
		responder.SendValidationError(c, err)
		return
	}

	user, refreshToken, accessToken, err := h.service.GoogleLogin(c.Request.Context(), req.IdToken)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	h.sendAuthResponse(c, domain.CodeAuthGoogleLoginSuccess, user, refreshToken, accessToken)
}

func (h *authHandler) Refresh(c *gin.Context) {
	var refreshToken string

	// Prefer cookie (frontend)
	if cookie, err := c.Cookie(CookieRefreshToken); err == nil && cookie != "" {
		refreshToken = cookie
	}

	// Fall back to JSON body (CLI/Postman/back-compat)
	if refreshToken == "" {
		var req AuthRefreshRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			refreshToken = req.RefreshToken
		}
	}

	if refreshToken == "" {
		responder.SendError(c, domain.NewError(domain.CodeAuthRefreshInvalidToken))
		return
	}

	user, newRefreshToken, accessToken, err := h.service.Refresh(c.Request.Context(), refreshToken)
	if err != nil {
		c.Error(err)
		responder.SendError(c, err)
		return
	}

	h.sendAuthResponse(c, domain.CodeAuthRefreshSuccess, user, newRefreshToken, accessToken)
}

func (h *authHandler) Logout(c *gin.Context) {
	// Clearing this browser's cookies never requires a live access token.
	h.clearAuthCookies(c)
	ctx := c.Request.Context()
	var err error
	if authUser, _ := httpContext.GetUser(ctx); authUser != nil {
		err = h.service.Logout(ctx, authUser.Id)
	} else {
		accessToken, _ := c.Cookie(CookieAccessToken)
		if accessToken == "" {
			accessToken = strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		}
		if claims, parseErr := security.ValiparseJWT(accessToken, []byte(*h.config.JWTSecret)); parseErr == nil {
			err = h.service.Logout(ctx, claims.UserId)
		} else if refreshToken, _ := c.Cookie(CookieRefreshToken); refreshToken != "" {
			err = h.service.LogoutByToken(ctx, refreshToken)
		}
	}
	if err != nil {
		wrapped := domain.NewError(domain.CodeSystemInternal, domain.WithError(err))
		c.Error(wrapped)
		responder.SendError(c, wrapped)
		return
	}
	responder.Send[any](c, domain.CodeAuthLogoutSuccess, nil)
}

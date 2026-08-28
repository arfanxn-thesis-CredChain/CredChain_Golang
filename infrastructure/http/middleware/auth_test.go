package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/infrastructure/security"
	"CredChain_Golang/tests/fixtures"
	"CredChain_Golang/tests/mocks"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func mkAuthCfg() *config.Config {
	s := "test-jwt-secret"
	return &config.Config{JWTSecret: &s}
}

func TestAuthMiddleware_NoHeader_401(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	mw := NewAuthMiddleware(AuthMiddlewareParams{Config: mkAuthCfg(), UserRepo: repo})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	mw(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_MalformedBearer_401(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	mw := NewAuthMiddleware(AuthMiddlewareParams{Config: mkAuthCfg(), UserRepo: repo})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Token abc")
	mw(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_InvalidJWT_401(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	mw := NewAuthMiddleware(AuthMiddlewareParams{Config: mkAuthCfg(), UserRepo: repo})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer not-a-jwt")
	mw(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_UserNotFound_401(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	repo.On("Find", mock.Anything, "u1").Return(nil, gorm.ErrRecordNotFound)

	mw := NewAuthMiddleware(AuthMiddlewareParams{Config: mkAuthCfg(), UserRepo: repo})

	tok, _ := security.GenerateJWT("u1", []byte(*mkAuthCfg().JWTSecret), time.Hour)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tok)
	mw(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthMiddleware_FindError_500(t *testing.T) {
	repo := &mocks.MockUserRepository{}
	repo.On("Find", mock.Anything, "u1").Return(nil, errors.New("db down"))

	mw := NewAuthMiddleware(AuthMiddlewareParams{Config: mkAuthCfg(), UserRepo: repo})

	tok, _ := security.GenerateJWT("u1", []byte(*mkAuthCfg().JWTSecret), time.Hour)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tok)
	mw(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAuthMiddleware_Success_SetsUserInContext(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithID("u1"))
	repo := &mocks.MockUserRepository{}
	repo.On("Find", mock.Anything, "u1").Return(&user, nil)

	mw := NewAuthMiddleware(AuthMiddlewareParams{Config: mkAuthCfg(), UserRepo: repo})

	tok, _ := security.GenerateJWT("u1", []byte(*mkAuthCfg().JWTSecret), time.Hour)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tok)
	mw(c)

	got, err := httpContext.GetUser(c.Request.Context())
	assert.NoError(t, err)
	assert.Equal(t, "u1", got.Id)
}

func TestAuthMiddleware_TrashedUser_401(t *testing.T) {
	deletedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	user := fixtures.NewDomainUser(fixtures.WithID("u1"))
	user.DeletedAt = &deletedAt
	repo := &mocks.MockUserRepository{}
	repo.On("Find", mock.Anything, "u1").Return(&user, nil)

	mw := NewAuthMiddleware(AuthMiddlewareParams{Config: mkAuthCfg(), UserRepo: repo})

	tok, _ := security.GenerateJWT("u1", []byte(*mkAuthCfg().JWTSecret), time.Hour)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("Authorization", "Bearer "+tok)
	mw(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code, "trashed users must not authenticate even with a valid JWT")
}

func TestAdminRoleMiddleware_NoUser_401(t *testing.T) {
	auth := &mocks.MockAuthorityService{}
	mw := gin.HandlerFunc(NewAdminRoleMiddleware(RoleMiddlewareParams{AuthorityService: auth}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	mw(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAdminRoleMiddleware_BelowRank_403(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	auth := &mocks.MockAuthorityService{}
	auth.On("HasRoleOrAbove", mock.Anything, user.WalletAddress, domain.RoleAdmin).Return(false)

	mw := gin.HandlerFunc(NewAdminRoleMiddleware(RoleMiddlewareParams{AuthorityService: auth}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), httpContext.UserKey, &user)
	c.Request = req.WithContext(ctx)
	mw(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestAdminRoleMiddleware_Success(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	auth := &mocks.MockAuthorityService{}
	auth.On("HasRoleOrAbove", mock.Anything, user.WalletAddress, domain.RoleAdmin).Return(true)

	mw := gin.HandlerFunc(NewAdminRoleMiddleware(RoleMiddlewareParams{AuthorityService: auth}))

	w := httptest.NewRecorder()
	_, engine := gin.CreateTestContext(w)
	called := false
	engine.Use(mw)
	engine.GET("/", func(c *gin.Context) {
		called = true
		c.Status(200)
	})
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), httpContext.UserKey, &user)
	req = req.WithContext(ctx)
	engine.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, 200, w.Code)
}

func TestIssuerRoleMiddleware_BelowRank_403(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleHolder))
	auth := &mocks.MockAuthorityService{}
	auth.On("HasRoleOrAbove", mock.Anything, user.WalletAddress, domain.RoleIssuer).Return(false)

	mw := gin.HandlerFunc(NewIssuerRoleMiddleware(RoleMiddlewareParams{AuthorityService: auth}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), httpContext.UserKey, &user)
	c.Request = req.WithContext(ctx)
	mw(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestSuperAdminRoleMiddleware_BelowRank_403(t *testing.T) {
	user := fixtures.NewDomainUser(fixtures.WithRole(domain.RoleAdmin))
	auth := &mocks.MockAuthorityService{}
	auth.On("HasRoleOrAbove", mock.Anything, user.WalletAddress, domain.RoleSuperAdmin).Return(false)

	mw := gin.HandlerFunc(NewSuperAdminRoleMiddleware(RoleMiddlewareParams{AuthorityService: auth}))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	ctx := context.WithValue(req.Context(), httpContext.UserKey, &user)
	c.Request = req.WithContext(ctx)
	mw(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

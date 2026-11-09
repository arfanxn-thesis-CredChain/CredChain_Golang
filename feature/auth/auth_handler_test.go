package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"CredChain_Golang/infrastructure/security"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	httpContext "CredChain_Golang/infrastructure/http/context"
	"CredChain_Golang/tests/fixtures"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockAuthService implements AuthService for handler tests.
type mockAuthService struct {
	mock.Mock
}

func (m *mockAuthService) GoogleLogin(ctx context.Context, idToken string) (domain.User, domain.UserToken, string, error) {
	args := m.Called(ctx, idToken)
	user, _ := args.Get(0).(domain.User)
	token, _ := args.Get(1).(domain.UserToken)
	return user, token, args.String(2), args.Error(3)
}

func (m *mockAuthService) Refresh(ctx context.Context, refreshToken string) (domain.User, domain.UserToken, string, error) {
	args := m.Called(ctx, refreshToken)
	user, _ := args.Get(0).(domain.User)
	token, _ := args.Get(1).(domain.UserToken)
	return user, token, args.String(2), args.Error(3)
}

func (m *mockAuthService) Logout(ctx context.Context, userId string) error {
	return m.Called(ctx, userId).Error(0)
}

func (m *mockAuthService) LogoutByToken(ctx context.Context, token string) error {
	return m.Called(ctx, token).Error(0)
}

func mkHandlerCfg() *config.Config {
	jwtSecret := "test-secret"
	accessMin := 15
	refreshHours := 168
	cookieDomain := ""
	cookieSecure := false
	cookieSameSite := "strict"
	cookieAccessPath := "/api"
	cookieRefreshPath := "/api/auth"
	return &config.Config{
		JWTSecret:              &jwtSecret,
		JWTAccessExpiryMinutes: &accessMin,
		JWTRefreshExpiryHours:  &refreshHours,
		CookieDomain:           &cookieDomain,
		CookieSecure:           &cookieSecure,
		CookieSameSite:         &cookieSameSite,
		CookieAccessPath:       &cookieAccessPath,
		CookieRefreshPath:      &cookieRefreshPath,
	}
}

func newGoogleLoginRequest(idToken string) *http.Request {
	body, _ := json.Marshal(AuthGoogleLoginRequest{IdToken: idToken})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/google", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func findCookie(cookies []*http.Cookie, name string) *http.Cookie {
	for _, c := range cookies {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestGoogleLogin_SetsAuthCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := fixtures.NewDomainUser(fixtures.WithID("u1"))
	expiresAt := time.Now().Add(time.Hour)
	refreshToken := domain.UserToken{
		Id:        "tok_1",
		UserId:    user.Id,
		Type:      domain.UserTokenTypeRefresh,
		Token:     "refresh-secret-value",
		ExpiresAt: &expiresAt,
	}

	svc := &mockAuthService{}
	svc.On("GoogleLogin", mock.Anything, "google-id-token").
		Return(user, refreshToken, "access-jwt-value", nil)

	h := NewAuthHandler(AuthHandlerParams{Service: svc, Config: mkHandlerCfg()})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = newGoogleLoginRequest("google-id-token")
	h.GoogleLogin(c)

	assert.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	access := findCookie(cookies, CookieAccessToken)
	refresh := findCookie(cookies, CookieRefreshToken)

	if assert.NotNil(t, access, "access_token cookie must be set") {
		assert.Equal(t, "access-jwt-value", access.Value)
		assert.True(t, access.HttpOnly, "access_token must be HttpOnly")
		assert.Equal(t, "/api", access.Path)
		assert.Equal(t, http.SameSiteStrictMode, access.SameSite)
	}

	if assert.NotNil(t, refresh, "refresh_token cookie must be set") {
		assert.Equal(t, "refresh-secret-value", refresh.Value)
		assert.True(t, refresh.HttpOnly, "refresh_token must be HttpOnly")
		assert.Equal(t, "/api/auth", refresh.Path)
		assert.Equal(t, http.SameSiteStrictMode, refresh.SameSite)
	}
}

func TestRefresh_ReadsFromCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := fixtures.NewDomainUser(fixtures.WithID("u1"))
	expiresAt := time.Now().Add(time.Hour)
	newRefresh := domain.UserToken{
		Id:        "tok_new",
		UserId:    user.Id,
		Type:      domain.UserTokenTypeRefresh,
		Token:     "new-refresh",
		ExpiresAt: &expiresAt,
	}

	svc := &mockAuthService{}
	svc.On("Refresh", mock.Anything, "old-refresh-from-cookie").
		Return(user, newRefresh, "new-access", nil)

	h := NewAuthHandler(AuthHandlerParams{Service: svc, Config: mkHandlerCfg()})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	c.Request.AddCookie(&http.Cookie{Name: CookieRefreshToken, Value: "old-refresh-from-cookie"})
	h.Refresh(c)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertCalled(t, "Refresh", mock.Anything, "old-refresh-from-cookie")

	cookies := w.Result().Cookies()
	refresh := findCookie(cookies, CookieRefreshToken)
	if assert.NotNil(t, refresh) {
		assert.Equal(t, "new-refresh", refresh.Value)
	}
}

func TestRefresh_FallbackToJSONBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := fixtures.NewDomainUser(fixtures.WithID("u1"))
	expiresAt := time.Now().Add(time.Hour)
	newRefresh := domain.UserToken{
		Id:        "tok_new",
		UserId:    user.Id,
		Type:      domain.UserTokenTypeRefresh,
		Token:     "new-refresh",
		ExpiresAt: &expiresAt,
	}

	svc := &mockAuthService{}
	svc.On("Refresh", mock.Anything, "body-refresh").
		Return(user, newRefresh, "new-access", nil)

	h := NewAuthHandler(AuthHandlerParams{Service: svc, Config: mkHandlerCfg()})

	body, _ := json.Marshal(AuthRefreshRequest{RefreshToken: "body-refresh"})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	h.Refresh(c)

	assert.Equal(t, http.StatusOK, w.Code)
	svc.AssertCalled(t, "Refresh", mock.Anything, "body-refresh")
}

func TestRefresh_NoTokenAtAll_ReturnsError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &mockAuthService{}
	h := NewAuthHandler(AuthHandlerParams{Service: svc, Config: mkHandlerCfg()})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	h.Refresh(c)

	assert.NotEqual(t, http.StatusOK, w.Code)
	svc.AssertNotCalled(t, "Refresh", mock.Anything, mock.Anything)
}

func TestLogout_WithoutAccessTokenClearsCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewAuthHandler(AuthHandlerParams{Service: &mockAuthService{}, Config: mkHandlerCfg()})
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	assert.NotPanics(t, func() { h.Logout(c) })
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotNil(t, findCookie(w.Result().Cookies(), CookieRefreshToken))
}

func TestLogout_ExpiredAccessUsesRefreshAndReportsDatabaseFailure(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			svc := &mockAuthService{}
			var revokeErr error
			if failed {
				revokeErr = errors.New("database unavailable")
			}
			svc.On("LogoutByToken", mock.Anything, "refresh").Return(revokeErr)
			h := NewAuthHandler(AuthHandlerParams{Service: svc, Config: mkHandlerCfg()})
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
			expired, err := security.GenerateJWT("u1", []byte("test-secret"), -time.Hour)
			assert.NoError(t, err)
			c.Request.AddCookie(&http.Cookie{Name: CookieAccessToken, Value: expired})
			c.Request.AddCookie(&http.Cookie{Name: CookieRefreshToken, Value: "refresh"})
			h.Logout(c)
			wantStatus := http.StatusOK
			if failed {
				wantStatus = http.StatusInternalServerError
			}
			assert.Equal(t, wantStatus, w.Code)
			for _, name := range []string{CookieAccessToken, CookieRefreshToken} {
				cookie := findCookie(w.Result().Cookies(), name)
				if assert.NotNil(t, cookie) {
					assert.Negative(t, cookie.MaxAge)
				}
			}
			svc.AssertExpectations(t)
		})
	}
}

func TestLogout_ClearsAuthCookies(t *testing.T) {
	gin.SetMode(gin.TestMode)
	user := fixtures.NewDomainUser(fixtures.WithID("u1"))
	svc := &mockAuthService{}
	svc.On("Logout", mock.Anything, user.Id).Return(nil)

	h := NewAuthHandler(AuthHandlerParams{Service: svc, Config: mkHandlerCfg()})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	ctx := context.WithValue(req.Context(), httpContext.UserKey, &user)
	c.Request = req.WithContext(ctx)
	c.Set(httpContext.UserKey, &user)
	h.Logout(c)

	assert.Equal(t, http.StatusOK, w.Code)

	cookies := w.Result().Cookies()
	access := findCookie(cookies, CookieAccessToken)
	refresh := findCookie(cookies, CookieRefreshToken)

	if assert.NotNil(t, access, "access_token must be cleared") {
		assert.Equal(t, "", access.Value)
		assert.True(t, access.MaxAge < 0, "MaxAge must be negative to expire cookie")
	}
	if assert.NotNil(t, refresh, "refresh_token must be cleared") {
		assert.Equal(t, "", refresh.Value)
		assert.True(t, refresh.MaxAge < 0, "MaxAge must be negative to expire cookie")
	}
}

package http

import (
	"context"
	"net/http"
	"time"

	"CredChain_Golang/config"
	"CredChain_Golang/domain"
	"CredChain_Golang/feature/auth"
	"CredChain_Golang/feature/credential"
	"CredChain_Golang/feature/meta"
	"CredChain_Golang/feature/overview"
	"CredChain_Golang/feature/user"
	"CredChain_Golang/infrastructure/http/middleware"
	"CredChain_Golang/infrastructure/http/responder"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type RouterParams struct {
	fx.In
	Config *config.Config
}

func NewGinRouter(p RouterParams) *gin.Engine {
	// Browsers reject Access-Control-Allow-Origin: * when credentials are sent.
	// Detect the misconfiguration early so cookie auth doesn't silently break.
	if p.Config.GinCorsAllowCredentials != nil && *p.Config.GinCorsAllowCredentials {
		if lo.Contains(p.Config.GinCorsAllowOrigins, "*") {
			panic("GIN_CORS_ALLOW_ORIGINS cannot contain \"*\" when GIN_CORS_ALLOW_CREDENTIALS=true; specify explicit origins")
		}
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     p.Config.GinCorsAllowOrigins,
		AllowMethods:     p.Config.GinCorsAllowMethods,
		AllowHeaders:     p.Config.GinCorsAllowHeaders,
		ExposeHeaders:    p.Config.GinCorsExposeHeaders,
		AllowCredentials: *p.Config.GinCorsAllowCredentials,
		MaxAge:           p.Config.GinCorsMaxAge,
	}))
	return r
}

type RouteParams struct {
	fx.In
	Lifecycle                           fx.Lifecycle
	Router                              *gin.Engine
	Config                              *config.Config
	I18nMiddleware                      middleware.I18nMiddleware
	AuthHandler                         auth.AuthHandler
	UserHandler                         user.UserHandler
	UserUnitHandler                     user.UserUnitHandler
	CredentialHandler                   credential.CredentialHandler
	CredentialTypeHandler               credential.CredentialTypeHandler
	CredentialIssuerOrganizationHandler credential.CredentialIssuerOrganizationHandler
	CompetencyHandler                   credential.CompetencyHandler
	MetaHandler                         meta.MetaHandler
	OverviewHandler                     overview.OverviewHandler
	Logger                              *zap.Logger
	AuthMiddleware                      gin.HandlerFunc
	AdminRoleMiddleware                 middleware.AdminRoleMiddleware
	IssuerRoleMiddleware                middleware.IssuerRoleMiddleware
	SuperAdminRoleMiddleware            middleware.SuperAdminRoleMiddleware
	LoginRateLimitMiddleware            middleware.LoginRateLimitMiddleware
	RefreshRateLimitMiddleware          middleware.RefreshRateLimitMiddleware
	LogoutRateLimitMiddleware           middleware.LogoutRateLimitMiddleware
	ApiRateLimitMiddleware              middleware.ApiRateLimitMiddleware
	ErrorLoggerMiddleware               middleware.ErrorLoggerMiddleware
}

func RegisterRoutes(p RouteParams) {
	p.Router.Use(gin.HandlerFunc(p.ErrorLoggerMiddleware))
	p.Router.Use(gin.HandlerFunc(p.I18nMiddleware))
	api := p.Router.Group("/api")
	api.Use(gin.HandlerFunc(p.ApiRateLimitMiddleware))
	{
		api.GET("/health", func(c *gin.Context) {
			responder.Send(c, domain.CodeSystemSuccess, gin.H{"status": "ok"})
		})
		api.GET("/meta", p.MetaHandler.Get)
		api.POST("/auth/google", gin.HandlerFunc(p.LoginRateLimitMiddleware), p.AuthHandler.GoogleLogin)
		api.POST("/auth/refresh", gin.HandlerFunc(p.RefreshRateLimitMiddleware), p.AuthHandler.Refresh)
		// Public credential verification — used by external verifiers (HR, employers).
		// No auth required; rate-limited by the global ApiRateLimitMiddleware.
		api.POST("/credentials/verify", p.CredentialHandler.Verify)
		secure := api.Group("/")
		secure.Use(p.AuthMiddleware)
		{
			secure.POST("/auth/logout", gin.HandlerFunc(p.LogoutRateLimitMiddleware), p.AuthHandler.Logout)
			secure.GET("/overview", p.OverviewHandler.Get)
			users := secure.Group("/users")
			{
				users.GET("", gin.HandlerFunc(p.IssuerRoleMiddleware), p.UserHandler.Paginate)
				users.GET("/self", p.UserHandler.Self)
				users.PUT("/self/email", p.UserHandler.UpdateSelfEmail)
				users.GET("/self/credentials", p.CredentialHandler.SelfPaginate)
				users.GET("/self/credentials/:id", p.CredentialHandler.SelfFind)
				users.POST("/self/transfer-super-admin", gin.HandlerFunc(p.SuperAdminRoleMiddleware), p.UserHandler.TransferSuperAdmin)
				users.GET("/:id", gin.HandlerFunc(p.IssuerRoleMiddleware), p.UserHandler.Find)
				users.POST("/batch", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Store)
				users.PUT("/batch", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Update)
				users.PUT("/batch/role", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.UpdateRole)
				users.DELETE("/batch", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Delete)
				users.PUT("/batch/restore", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserHandler.Restore)
			}
			creds := secure.Group("/credentials")
			{
				creds.GET("", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredentialHandler.Paginate)
				creds.GET("/:id", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredentialHandler.Find)
				creds.POST("/batch/issue", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredentialHandler.Issue)
				creds.POST("/batch/submit", p.CredentialHandler.Submit)
				creds.POST("/batch/revoke", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredentialHandler.Revoke)
				creds.POST("/batch/reextract", gin.HandlerFunc(p.IssuerRoleMiddleware), p.CredentialHandler.ReExtract)
			}
			credentialTypes := secure.Group("/credential-types")
			{
				credentialTypes.GET("", p.CredentialTypeHandler.Paginate)
				credentialTypes.POST("", p.CredentialTypeHandler.Store)
				credentialTypes.PUT("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.CredentialTypeHandler.Update)
				credentialTypes.DELETE("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.CredentialTypeHandler.Destroy)
			}
			issuerOrganizations := secure.Group("/issuer-organizations")
			{
				issuerOrganizations.GET("", p.CredentialIssuerOrganizationHandler.Paginate)
				issuerOrganizations.POST("", p.CredentialIssuerOrganizationHandler.Store)
				issuerOrganizations.PUT("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.CredentialIssuerOrganizationHandler.Update)
				issuerOrganizations.DELETE("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.CredentialIssuerOrganizationHandler.Destroy)
			}
			competencies := secure.Group("/competencies")
			{
				competencies.GET("", p.CompetencyHandler.Paginate)
				competencies.POST("", p.CompetencyHandler.Store)
				competencies.PUT("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.CompetencyHandler.Update)
				competencies.DELETE("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.CompetencyHandler.Destroy)
			}
			userUnits := secure.Group("/user-units")
			{
				userUnits.GET("", p.UserUnitHandler.Paginate)
				userUnits.POST("", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserUnitHandler.Store)
				userUnits.PUT("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserUnitHandler.Update)
				userUnits.DELETE("/:id", gin.HandlerFunc(p.AdminRoleMiddleware), p.UserUnitHandler.Destroy)
			}
			secure.GET("/credentials/:id/file", p.CredentialHandler.DownloadFile)
		}
	}
	srv := &http.Server{
		Addr:         ":" + *p.Config.GinPort,
		Handler:      p.Router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				p.Logger.Info("credchain golang backend starting", zap.String("port", *p.Config.GinPort))
				if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					p.Logger.Error("server failed to start", zap.Error(err))
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("stopping server")
			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			return srv.Shutdown(shutdownCtx)
		},
	})
}

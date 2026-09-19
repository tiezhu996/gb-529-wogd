package router

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/handler"
	"lng-boiloff-gas-balance/backend/internal/middleware"
	"lng-boiloff-gas-balance/backend/internal/service"
)

type Handlers struct {
	Support     *handler.SupportHandler
	Tank        *handler.TankHandler
	Measurement *handler.MeasurementHandler
	Transfer    *handler.TransferHandler
	Balance     *handler.BalanceHandler
}

func New(log *slog.Logger, authService *service.AuthService, handlers Handlers, corsOrigins []string) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.AccessLog(log))
	engine.Use(middleware.Recovery(log))
	engine.Use(cors(corsOrigins))

	engine.GET("/healthz", handlers.Support.Health)
	engine.GET("/readyz", handlers.Support.Ready)

	loginLimiter := middleware.NewRateLimiter(8, 0.5)
	runLimiter := middleware.NewRateLimiter(8, 0.25)
	apiV1 := engine.Group("/api/v1")
	apiV1.POST("/auth/login", loginLimiter.Middleware("login"), handlers.Support.Login)

	protected := apiV1.Group("")
	protected.Use(middleware.Auth(authService))
	protected.GET("/auth/me", handlers.Support.Me)

	protected.GET("/tanks", handlers.Tank.List)
	protected.GET("/tanks/:id", handlers.Tank.Get)
	protected.GET("/tanks/:id/measurement-quality", handlers.Tank.MeasurementQuality)
	protected.POST("/tanks", middleware.RBAC(constants.RoleProcessAnalyst, constants.RoleAdmin), handlers.Tank.Create)
	protected.PUT("/tanks/:id", middleware.RBAC(constants.RoleProcessAnalyst, constants.RoleAdmin), handlers.Tank.Update)

	protected.GET("/measurements", handlers.Measurement.List)
	protected.GET("/measurements/:id", handlers.Measurement.Get)
	protected.POST("/measurements", middleware.RBAC(constants.RoleProcessAnalyst, constants.RoleAdmin), handlers.Measurement.Create)

	protected.GET("/transfers", handlers.Transfer.List)
	protected.GET("/transfers/:id", handlers.Transfer.Get)
	protected.POST("/transfers", middleware.RBAC(constants.RoleProcessAnalyst, constants.RoleAdmin), handlers.Transfer.Create)
	protected.POST("/transfers/:id/status", middleware.RBAC(constants.RoleProcessAnalyst, constants.RoleAdmin), handlers.Transfer.Transition)

	protected.GET("/balances", handlers.Balance.List)
	protected.GET("/balances/:id", handlers.Balance.Get)
	protected.GET("/balances/:id/uncertainty", handlers.Balance.Uncertainty)
	protected.POST("/balances/run", runLimiter.Middleware("balance-run"), middleware.RBAC(constants.RoleProcessAnalyst, constants.RoleAdmin), handlers.Balance.Run)
	protected.POST("/balances/:id/submit", middleware.RBAC(constants.RoleProcessAnalyst, constants.RoleAdmin), handlers.Balance.Submit)
	protected.POST("/balances/:id/review", middleware.RBAC(constants.RoleReviewer, constants.RoleAdmin), handlers.Balance.Review)
	protected.POST("/balances/:id/invalidate", middleware.RBAC(constants.RoleAdmin), handlers.Balance.Invalidate)

	protected.GET("/audits", middleware.RBAC(constants.RoleReviewer, constants.RoleAdmin), handlers.Support.ListAudits)
	return engine
}

func cors(origins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		allowed[strings.TrimSpace(origin)] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok && origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

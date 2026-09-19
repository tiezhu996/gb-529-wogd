package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/internal/service"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

func Auth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			api.Fail(c, api.NewError(401, "AUTH_REQUIRED", "请先登录后再访问该资源"))
			return
		}
		claims, err := authService.ParseToken(c.Request.Context(), strings.TrimSpace(parts[1]))
		if err != nil {
			api.Fail(c, err)
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

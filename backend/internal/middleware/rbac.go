package middleware

import (
	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/pkg/api"
)

func RBAC(allowed ...string) gin.HandlerFunc {
	roleSet := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		roleSet[role] = struct{}{}
	}
	return func(c *gin.Context) {
		roleValue, exists := c.Get("role")
		role, ok := roleValue.(string)
		if !exists || !ok {
			api.Fail(c, api.NewError(401, "AUTH_REQUIRED", "请先登录后再访问该资源"))
			return
		}
		if _, permitted := roleSet[role]; !permitted {
			api.Fail(c, api.ErrForbidden)
			return
		}
		c.Next()
	}
}

package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"lng-boiloff-gas-balance/backend/pkg/api"
)

func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("panic_recovered",
					"request_id", requestIDFromContext(c),
					"panic", fmt.Sprint(recovered),
					"stack", string(debug.Stack()),
				)
				api.Fail(c, api.NewError(500, "INTERNAL_ERROR", "服务暂时无法完成请求"))
			}
		}()
		c.Next()
	}
}

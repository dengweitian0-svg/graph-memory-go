package middleware

import (
	"net/http"

	"github.com/example/graph-memory/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Recovery 恢复中间件
func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Error("Panic recovered",
					"error", err,
					"path", c.Request.URL.Path,
					"method", c.Request.Method,
				)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

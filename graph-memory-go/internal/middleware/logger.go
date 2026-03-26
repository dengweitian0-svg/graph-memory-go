package middleware

import (
	"time"

	"github.com/example/graph-memory/pkg/logger"
	"github.com/gin-gonic/gin"
)

// Logger 日志中间件
func Logger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		statusCode := c.Writer.Status()

		log.Info("HTTP Request",
			"method", method,
			"path", path,
			"status", statusCode,
			"duration", latency,
			"remote_addr", c.ClientIP(),
		)
	}
}

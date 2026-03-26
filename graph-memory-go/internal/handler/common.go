package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// RespondWithError 返回错误响应
func RespondWithError(c *gin.Context, code int, message string) {
	c.JSON(code, ErrorResponse{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
	})
}

// HealthCheck 健康检查
// @Summary 健康检查
// @Description 检查服务是否正常运行
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
	})
}

// ReadinessCheck 就绪检查
// @Summary 就绪检查
// @Description 检查服务是否就绪
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /ready [get]
func ReadinessCheck(c *gin.Context) {
	// TODO: 检查数据库连接等依赖
	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

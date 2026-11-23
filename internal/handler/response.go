package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"homepage/internal/service"

	"github.com/gin-gonic/gin"
)

func success(c *gin.Context, data any) {
	if c.Request.Method == http.MethodHead {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Status(http.StatusOK)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func fail(c *gin.Context, status int, code, message string) {
	if c.Request.Method == http.MethodHead {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.AbortWithStatus(status)
		return
	}
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}

func handleError(c *gin.Context, err error) {
	var serviceErr *service.Error
	if errors.As(err, &serviceErr) {
		fail(c, serviceErr.Status, serviceErr.Code, serviceErr.Message)
		return
	}
	slog.Error("请求处理失败", "error", err, "path", c.Request.URL.Path)
	fail(c, http.StatusInternalServerError, "internal_error", "服务器内部错误")
}

func bindJSON(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			fail(c, http.StatusRequestEntityTooLarge, "request_too_large", "请求体超过大小限制")
		} else {
			fail(c, http.StatusBadRequest, "invalid_json", "请求 JSON 无效")
		}
		return false
	}
	return true
}

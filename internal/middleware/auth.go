package middleware

import (
	"net/http"

	"homepage/internal/service"

	"github.com/gin-gonic/gin"
)

const SessionKey = "admin_session"

func RequireAuth(auth *service.AuthService, cookieName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "unauthorized", "message": "请先登录"}})
			return
		}
		session, err := auth.Session(token)
		if err != nil {
			if serviceErr, ok := err.(*service.Error); ok {
				c.AbortWithStatusJSON(serviceErr.Status, gin.H{"error": gin.H{"code": serviceErr.Code, "message": serviceErr.Message}})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "internal_error", "message": "服务器内部错误"}})
			return
		}
		c.Set(SessionKey, session)
		c.Next()
	}
}

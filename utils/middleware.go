package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"github.com/gin-contrib/sessions"
)

func CheckLoginIn() gin.HandlerFunc {
	return func(context *gin.Context) {
		session := sessions.Default(context)
		login := session.Get("login")
		if login != "true" {
			session.Clear()
			if strings.ToUpper(context.Request.Method) == "GET" {
				context.Abort()
				context.Redirect(http.StatusTemporaryRedirect, "/login")
			} else if strings.ToUpper(context.Request.Method) == "POST" {
				context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			} else {
				context.AbortWithStatus(http.StatusUnauthorized)
			}
		} else {
			context.Next()
		}
	}
}

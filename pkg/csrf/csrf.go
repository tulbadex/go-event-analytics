package csrf

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF for auth routes and logout
		if c.Request.URL.Path == "/auth/login" || c.Request.URL.Path == "/auth/register" || c.Request.URL.Path == "/user/logout" {
			c.Next()
			return
		}

		if c.Request.Method == "GET" {
			token := generateToken()
			c.SetCookie("csrf_token", token, 3600, "/", "", false, true)
			c.Set("csrf_token", token)
		} else if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "DELETE" {
			cookieToken, _ := c.Cookie("csrf_token")
			formToken := c.PostForm("csrf_token")
			if cookieToken == "" || formToken == "" || cookieToken != formToken {
				c.HTML(http.StatusForbidden, "login.html", gin.H{"error": "Invalid CSRF token. Please try again."})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

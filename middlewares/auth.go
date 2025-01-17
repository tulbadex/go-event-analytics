package middlewares

import (
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"

	"event-analytics/models"
	"event-analytics/utils"
	// "log"
)

func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, _ := c.Cookie("session_token")
		if sessionToken == "" {
			location := url.URL{
                Path:     "/auth/login",
                RawQuery: url.Values{"error": {"auth_required"}}.Encode(),
            }
            c.Redirect(http.StatusFound, location.String())
            c.Abort()
			return
		}
		c.Next()
	}
}

func PreventAuthenticatedAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, _ := c.Cookie("session_token")
		if sessionToken != "" {
			c.Redirect(http.StatusFound, "/user/dashboard")
			c.Abort()
			return
		}
		c.Next()
	}
}

func UserMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        user, err := utils.GetUserFromSession(c)
        if err != nil {
            c.Set("user", nil)
        } else {
            sanitizedUser := &models.User{
                ID:        user.ID,
                Username:  user.Username,
                Email:     user.Email,
                FirstName: user.FirstName,
                LastName:  user.LastName,
                Address:   user.Address,
            }
            c.Set("user", sanitizedUser)
        }
        c.Next()
    }
}

func FlashMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        flash, err := c.Cookie("flash")
        if err == nil {
            c.Set("flash", flash)
            // Clear the flash cookie
            c.SetCookie("flash", "", -1, "/", "", false, true)
        }
        c.Next()
    }
}

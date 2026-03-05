package middlewares

import (
	"context"
	"event-analytics/config"
	"event-analytics/models"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
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

		userID, err := config.SessionStore.Get(context.Background(), sessionToken)
		if err != nil {
			location := url.URL{
				Path:     "/auth/login",
				RawQuery: url.Values{"error": {"session_expired"}}.Encode(),
			}
			c.Redirect(http.StatusFound, location.String())
			c.Abort()
			return
		}

		var user models.User
		if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			c.Redirect(http.StatusFound, "/auth/login")
			c.Abort()
			return
		}

		c.Set("user", &user)
		c.Next()
	}
}

func PreventAuthenticatedAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, _ := c.Cookie("session_token")
		if sessionToken != "" {
			_, err := config.SessionStore.Get(context.Background(), sessionToken)
			if err == nil {
				c.Redirect(http.StatusFound, "/user/dashboard")
				c.Abort()
				return
			}
			c.SetCookie("session_token", "", -1, "/", "", false, true)
		}
		c.Next()
	}
}

func UserMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionToken, _ := c.Cookie("session_token")
		if sessionToken == "" {
			c.Set("user", nil)
			c.Next()
			return
		}

		userID, err := config.SessionStore.Get(context.Background(), sessionToken)
		if err != nil {
			c.Set("user", nil)
			c.Next()
			return
		}

		var user models.User
		if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			c.Set("user", nil)
			c.Next()
			return
		}

		sanitizedUser := &models.User{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Address:   user.Address,
		}
		c.Set("user", sanitizedUser)
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

package handler

import (
    "event-analytics/render"
    "github.com/gin-gonic/gin"

    // "log"
    "net/http"
	// "event-analytics/config"
	// "event-analytics/models"
	// "event-analytics/utils"

    // "gorm.io/gorm"
    // "errors"
)

func ShowCreateEventPage(c *gin.Context) {
    user, exists := c.Get("user")
    if !exists || user == nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        c.Abort()
        return
    }

    render.Render(c, gin.H{
		"title": "Create New Event",
        "user":  user,
    }, "new_event.html")
}
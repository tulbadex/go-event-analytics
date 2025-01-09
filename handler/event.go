package handler

import (
	"event-analytics/render"

	"github.com/gin-gonic/gin"

	"encoding/json"
	"event-analytics/config"
	"event-analytics/models"
	"event-analytics/utils"
	"log"
	"net/http"
	// "gorm.io/gorm"
	// "errors"
)

type EventInput struct {
    Title           string `form:"title" binding:"required"`
    Description     string `form:"description" binding:"required"`
    StartTime       string `form:"start_time" binding:"required"`
    EndTime         string `form:"end_time" binding:"required"`
    Location        string `form:"location" binding:"required"`
    Status          string `form:"status" binding:"required,oneof=draft published"`
    PublishedDate   string `form:"published_date"`
}

// ShowCreateEventPage renders the create event page
func ShowCreateEventPage(c *gin.Context) {
    user, err := utils.GetUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        return
    }

    // Get error from query parameter
    errorMsg := c.Query("error")

    // Get form data from cookie if it exists
    var formData EventInput
    if cookie, err := c.Cookie("form_data"); err == nil {
        json.Unmarshal([]byte(cookie), &formData)
        // Clear the cookie
        c.SetCookie("form_data", "", -1, "/", "", false, true)
    }

    // If no form data, set default status
    if formData.Status == "" {
        formData.Status = "draft"
    }

    render.Render(c, gin.H{
        "title":    "Create Event",
        "user":     user,
        "error":    errorMsg,
        "formData": formData,
    }, "event_new.html")
}

func ShowEventDetails(c *gin.Context) {
	user, err := utils.GetUserFromSession(c)
	if err != nil {
		c.Redirect(http.StatusFound, "/auth/login")
		return
	}

	// Parse event ID from URL
	eventID := c.Param("id")

	var event models.Event
	if err := config.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		log.Printf("Error fetching event details: %v", err)
		c.HTML(http.StatusNotFound, "event_details.html", gin.H{
			"error": "Event not found",
			"title": "Event Details",
			"user":  user,
		})
		return
	}

	c.HTML(http.StatusOK, "event_details.html", gin.H{
		"title": "Event Details",
		"user":  user,
		"event": event,
	})
}

// ShowEditEventPage renders the edit event page
func ShowEditEventPage(c *gin.Context) {
    user, err := utils.GetUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        return
    }

    eventID := c.Param("id")
	log.Println("Event ID from URL:", eventID)

    var event models.Event
    if err := config.DB.First(&event, "id = ?", eventID).Error; err != nil {
		log.Println("Error fetching event:", err)
		c.Redirect(http.StatusFound, "/user/dashboard?error=Event not found")
		return
	}	

    // Check permission
    if !utils.IsAdminOrOwner(user, event) {
        c.Redirect(http.StatusFound, "/user/dashboard?error=Permission denied")
        return
    }

    // Get error from query parameter
    errorMsg := c.Query("error")

    render.Render(c, gin.H{
        "title":    "Edit Event",
        "user":     user,
        "event":    event,
        "error":    errorMsg,
    }, "event_edit.html")
}
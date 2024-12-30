package controllers

import (
	"net/http"
	"time"
	"fmt"
	"os"
	"event-analytics/models"
	// "event-analytics/utils"
	"github.com/gin-gonic/gin"
	// "gorm.io/gorm"
	"event-analytics/config"
)

// GetEvents retrieves all events and renders the dashboard
func GetEvents(c *gin.Context) {
	var events []models.Event
	if err := config.DB.Find(&events).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to fetch events"})
		return
	}

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"Events": events,
	})
}

// NewEvent handles event creation
func CreateEvent(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists || user == nil {
		c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
		c.Abort()
		return
	}

	// currentUser, ok := user.(*models.User)
    // if !ok {
    //     c.Redirect(http.StatusFound, "/auth/login?error=invalid_user")
    //     c.Abort()
    //     return
    // }

	var input struct {
		Title       string    `form:"title" binding:"required"`
		Description string    `form:"description" binding:"required"`
		StartTime   time.Time `form:"start_time" binding:"required"`
		EndTime     time.Time `form:"end_time" binding:"required"`
		Location    string    `form:"location" binding:"required"`
		Status      string    `form:"status" binding:"required"`
	}

	if err := c.ShouldBind(&input); err != nil {
		c.HTML(http.StatusBadRequest, "new_event.html", gin.H{
			"error": "Please fill all required fields correctly",
			"title": "Create Event",
		})
		return
	}

	// Ensure uploads directory exists
	if err := os.MkdirAll("uploads", 0755); err != nil {
        c.HTML(http.StatusInternalServerError, "new_event.html", gin.H{
            "error": "Server configuration error",
            "title": "Create Event",
            "user":  user,
        })
        return
    }

	file, _ := c.FormFile("image")
	imagePath := ""
	if file != nil {
		filename := fmt.Sprintf("%d_%s", time.Now().Unix(), file.Filename)
		imagePath = "/uploads/" + filename
		if err := c.SaveUploadedFile(file, "."+imagePath); err != nil {
			c.HTML(http.StatusInternalServerError, "new_event.html", gin.H{
				"error": "Failed to upload image",
				"title": "Create Event",
				"user":  user,
			})
			return
		}
	}

	event := models.Event{
		Title:       input.Title,
		Description: input.Description,
		StartTime:   input.StartTime,
		EndTime:     input.EndTime,
		Location:    input.Location,
		Image:       imagePath,
		Status:      input.Status,
		CreatedBy:   user.(models.User).ID,
	}

	if err := config.DB.Create(&event).Error; err != nil {
		c.HTML(http.StatusInternalServerError, "new_event.html", gin.H{
			"error": "Failed to create event",
			"title": "Create Event",
			"user":  user,
		})
		return
	}

	c.Redirect(http.StatusFound, "/user/dashboard?success=Event created successfully")
}

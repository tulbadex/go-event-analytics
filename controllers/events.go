package controllers

import (
	"encoding/json"
	"errors"
	"event-analytics/models"
	"fmt"
	// "log"
	"net/http"
	"os"
	// "path/filepath"
	"time"

	"event-analytics/config"
	"event-analytics/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EventInput struct {
    Title       string `form:"title" binding:"required"`
    Description string `form:"description" binding:"required"`
    StartTime   string `form:"start_time" binding:"required"`
    EndTime     string `form:"end_time" binding:"required"`
    Location    string `form:"location" binding:"required"`
    Status      string `form:"status" binding:"required,oneof=draft published"`
}

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


func CreateEvent(c *gin.Context) {
    user, err := utils.GetUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        return
    }

    var input EventInput
    if err := c.ShouldBind(&input); err != nil {
        // Store form data in session temporarily
        formData, _ := json.Marshal(input)
        c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
        c.Redirect(http.StatusFound, "/events/new?error=Please fill all required fields correctly")
        return
    }

    // Check title uniqueness
    var count int64
    config.DB.Model(&models.Event{}).Where("title = ?", input.Title).Count(&count)
    if count > 0 {
        formData, _ := json.Marshal(input)
        c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
        c.Redirect(http.StatusFound, "/events/new?error=Event title must be unique")
        return
    }

    const datetimeFormat = "2006-01-02T15:04"
    startTime, err := time.Parse(datetimeFormat, input.StartTime)
    if err != nil {
        formData, _ := json.Marshal(input)
        c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
        c.Redirect(http.StatusFound, "/events/new?error=Invalid start datetime format")
        return
    }

    endTime, err := time.Parse(datetimeFormat, input.EndTime)
    if err != nil {
        formData, _ := json.Marshal(input)
        c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
        c.Redirect(http.StatusFound, "/events/new?error=Invalid end datetime format")
        return
    }

    if endTime.Before(startTime) {
        formData, _ := json.Marshal(input)
        c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
        c.Redirect(http.StatusFound, "/events/new?error=End datetime must be after start datetime")
        return
    }

    // Handle file upload
    file, _ := c.FormFile("image")
    imagePath := ""
    if file != nil {
        if err := os.MkdirAll("uploads", 0755); err != nil {
            formData, _ := json.Marshal(input)
            c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
            c.Redirect(http.StatusFound, "/events/new?error=Server configuration error")
            return
        }

        // filename := fmt.Sprintf("%d%s", time.Now().Unix(), file.Filename)
        filename    := utils.GenerateSecureFileName(file.Filename)
        imagePath   = "/uploads/" + filename
        if err := c.SaveUploadedFile(file, "."+imagePath); err != nil {
            formData, _ := json.Marshal(input)
            c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
            c.Redirect(http.StatusFound, "/events/new?error=Failed to upload image")
            return
        }
    }

    // Create event
    event := models.Event{
        Title:       input.Title,
        Description: input.Description,
        StartTime:   startTime,
        EndTime:     endTime,
        Location:    input.Location,
        Image:       imagePath,
        Status:      input.Status,
        CreatedBy:   user.ID,
    }

    if err := config.DB.Create(&event).Error; err != nil {
        formData, _ := json.Marshal(input)
        c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
        c.Redirect(http.StatusFound, "/events/new?error=Failed to create event")
        return
    }

    // Successful creation
    c.SetCookie("flash", "Event created successfully", 300, "/", "", false, true)
    c.Redirect(http.StatusFound, "/user/dashboard")
}

func UpdateEvent(c *gin.Context) {
    user, err := utils.GetUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        return
    }

    eventID := c.Param("id")
    var existingEvent models.Event
    if err := config.DB.First(&existingEvent, "id = ?", eventID).Error; err != nil {
		c.Redirect(http.StatusFound, "/user/dashboard?error=Event not found")
		return
	}

    // Check permission
    if !utils.IsAdminOrOwner(user, existingEvent) {
        c.Redirect(http.StatusFound, "/user/dashboard?error=Permission denied")
        return
    }

    var input EventInput
    if err := c.ShouldBind(&input); err != nil {
        c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Please fill all required fields correctly", eventID))
        return
    }

    // Check title uniqueness (excluding current event)
    var count int64
    config.DB.Model(&models.Event{}).Where("title = ? AND id != ?", input.Title, eventID).Count(&count)
    if count > 0 {
        c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Event title must be unique", eventID))
        return
    }

    // Parse dates
    startTime, err := time.Parse("2006-01-02T15:04", input.StartTime)
    if err != nil {
        c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Invalid start datetime format", eventID))
        return
    }

    endTime, err := time.Parse("2006-01-02T15:04", input.EndTime)
    if err != nil {
        c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Invalid end datetime format", eventID))
        return
    }

    if endTime.Before(startTime) {
        c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=End datetime must be after start datetime", eventID))
        return
    }

    // Handle image upload
    file, _ := c.FormFile("image")
    if file != nil {
        // Delete old image if exists
        if existingEvent.Image != "" {
            oldImagePath := "." + existingEvent.Image
            os.Remove(oldImagePath)
        }

        // Save new image
        if err := os.MkdirAll("uploads", 0755); err != nil {
            c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Server configuration error", eventID))
            return
        }

        // filename := fmt.Sprintf("%d%s", time.Now().Unix(), filepath.Ext(file.Filename))
        filename    := utils.GenerateSecureFileName(file.Filename)
        imagePath   := "/uploads/" + filename
        if err := c.SaveUploadedFile(file, "."+imagePath); err != nil {
            c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Failed to upload image", eventID))
            return
        }
        existingEvent.Image = imagePath
    }

    // Update event
    existingEvent.Title = input.Title
    existingEvent.Description = input.Description
    existingEvent.StartTime = startTime
    existingEvent.EndTime = endTime
    existingEvent.Location = input.Location
    existingEvent.Status = input.Status

    if err := config.DB.Save(&existingEvent).Error; err != nil {
        c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Failed to update event", eventID))
        return
    }

    c.SetCookie("flash", "Event updated successfully", 300, "/", "", false, true)
    c.Redirect(http.StatusFound, "/user/dashboard")
}

func DeleteEvent(c *gin.Context) {
    user, err := utils.GetUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        return
    }

    // Get the event ID from the route parameter
    eventID := c.Param("id")

    // Fetch the event from the database
    var event models.Event
    if err := config.DB.First(&event, "id = ?", eventID).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            c.Redirect(http.StatusFound, "/user/dashboard?error=Event not found")
        } else {
            c.Redirect(http.StatusFound, "/user/dashboard?error=Error fetching event")
        }
        return
    }

    // Check permission using utility function
    if !utils.IsAdminOrOwner(user, event) {
        c.Redirect(http.StatusFound, fmt.Sprintf("/user/dashboard?error=Permission denied for event"))
        return
    }

    // Delete image file if it exists
    /* if event.Image != "" {
        imagePath := "." + event.Image // Convert relative path to file system path
        err := os.Remove(imagePath)
        if err != nil {
            // Log the error but continue with event deletion
            log.Printf("Error deleting image file: %v", err)
        }
    } */

    // Delete the event
    if err := config.DB.Delete(&event).Error; err != nil {
        c.Redirect(http.StatusFound, fmt.Sprintf("/user/dashboard?error=Failed to delete event"))
        return
    }

    // Success message via flash cookie
    c.SetCookie("flash", "Event deleted successfully", 300, "/", "", false, true)
    c.Redirect(http.StatusFound, "/user/dashboard")
}
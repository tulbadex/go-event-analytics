package controllers

import (
	"encoding/json"
	"errors"
	"log"

	// "event-analytics/handler"
	"event-analytics/models"
	"fmt"

	// "log"
	"net/http"
	"net/url"
	"os"

	// "path/filepath"
	"time"

	"event-analytics/config"
	"event-analytics/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
        log.Printf("Failed to retrieve user from session: %v", err)
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        return
    }

    var input EventInput
    if err := c.ShouldBind(&input); err != nil {
        handleRedirectWithFormData(c, input, "Please fill all required fields correctly")
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

    startTime, err := parseDateTime(input.StartTime)
    if err != nil {
        handleRedirectWithFormData(c, input, "Invalid start datetime format")
        return
    }

    endTime, err := parseDateTime(input.EndTime)
    if err != nil {
        handleRedirectWithFormData(c, input, "Invalid end datetime format")
        return
    }

    if endTime.Before(startTime) {
        handleRedirectWithFormData(c, input, "End datetime must be after start datetime")
        return
    }

    var publishedDate *time.Time
    if input.Status == "published" {
        now := time.Now()
        publishedDate = &now
    } else if input.Status == "draft" && input.PublishedDate != "" {
        parsedDate, err := parseDateTime(input.PublishedDate)
        if err != nil {
            handleRedirectWithFormData(c, input, "Invalid publish date format")
            return
        }
        publishedDate = &parsedDate
    }

    file, _ := c.FormFile("image")
    imagePath := ""
    if file != nil {
        if err := os.MkdirAll("uploads/events/", 0755); err != nil {
            formData, _ := json.Marshal(input)
            c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
            c.Redirect(http.StatusFound, "/events/new?error=Server configuration error")
            return
        }

        // filename := fmt.Sprintf("%d%s", time.Now().Unix(), file.Filename)
        filename    := utils.GenerateSecureFileName(file.Filename)
        imagePath   = "/uploads/events/" + filename
        if err := c.SaveUploadedFile(file, "."+imagePath); err != nil {
            formData, _ := json.Marshal(input)
            c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
            c.Redirect(http.StatusFound, "/events/new?error=Failed to upload image")
            return
        }
    }

    event := models.Event{
        Title:        input.Title,
        Description:  input.Description,
        StartTime:    startTime,
        EndTime:      endTime,
        Location:     input.Location,
        Image:        imagePath,
        Status:       input.Status,
        CreatedBy:    user.ID,
        PublishedDate: publishedDate,
    }

    if err := config.DB.Create(&event).Error; err != nil {
        handleRedirectWithFormData(c, input, "Failed to create event")
        return
    }

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

    if !utils.IsAdminOrOwner(user, existingEvent) {
        c.Redirect(http.StatusFound, "/user/dashboard?error=Permission denied")
        return
    }

    var input EventInput
    if err := c.ShouldBind(&input); err != nil {
        handleRedirectWithFormData(c, input, fmt.Sprintf("/events/edit/%s?error=Invalid input", eventID))
        return
    }

    var count int64
    config.DB.Model(&models.Event{}).Where("title = ? AND id != ?", input.Title, eventID).Count(&count)
    if count > 0 {
        c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Event title must be unique", eventID))
        return
    }

    startTime, err := parseDateTime(input.StartTime)
    if err != nil {
        handleRedirectWithFormData(c, input, "Invalid start datetime format")
        return
    }

    endTime, err := parseDateTime(input.EndTime)
    if err != nil {
        handleRedirectWithFormData(c, input, "Invalid end datetime format")
        return
    }

    if endTime.Before(startTime) {
        handleRedirectWithFormData(c, input, "End datetime must be after start datetime")
        return
    }

    if input.Status == "published" {
        now := time.Now()
        existingEvent.PublishedDate = &now
    } else if input.Status == "draft" && input.PublishedDate != "" {
        parsedDate, err := parseDateTime(input.PublishedDate)
        if err != nil {
            handleRedirectWithFormData(c, input, "Invalid publish date format")
            return
        }
        existingEvent.PublishedDate = &parsedDate
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
        if err := os.MkdirAll("uploads/events/", 0755); err != nil {
            c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Server configuration error", eventID))
            return
        }

        filename    := utils.GenerateSecureFileName(file.Filename)
        imagePath   := "/uploads/events/" + filename
        if err := c.SaveUploadedFile(file, "."+imagePath); err != nil {
            c.Redirect(http.StatusFound, fmt.Sprintf("/events/edit/%s?error=Failed to upload image", eventID))
            return
        }
        existingEvent.Image = imagePath
    }

    existingEvent.Title = input.Title
    existingEvent.Description = input.Description
    existingEvent.StartTime = startTime
    existingEvent.EndTime = endTime
    existingEvent.Location = input.Location
    existingEvent.Status = input.Status

    if err := config.DB.Save(&existingEvent).Error; err != nil {
        handleRedirectWithFormData(c, input, "Failed to update event")
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

// Helper function to handle redirects with form data
func handleRedirectWithFormData(c *gin.Context, input EventInput, errorMessage string) {
    formData, _ := json.Marshal(input)
    c.SetCookie("form_data", string(formData), 300, "/", "", false, true)
    c.Redirect(http.StatusFound, "/events/new?error="+url.QueryEscape(errorMessage))
}

// Function to parse datetime strings into time.Time objects
func parseDateTime(datetimeStr string) (time.Time, error) {
    const datetimeFormat = "2006-01-02T15:04"
    return time.Parse(datetimeFormat, datetimeStr)
}
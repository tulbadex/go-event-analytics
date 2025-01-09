package handler

import (
	"event-analytics/render"
	"event-analytics/utils"

	"github.com/gin-gonic/gin"

	"event-analytics/config"
	"event-analytics/models"
	"event-analytics/services"
	"log"
	"net/http"

	"errors"
	"strconv"

	"gorm.io/gorm"
)

func ShowLoginPage(c *gin.Context) {
    var message struct {
        Error   string
        Success string
    }

    // Handle error messages
    if errType := c.Query("error"); errType != "" {
        switch errType {
        case "invalid_reset_token":
            message.Error = "Invalid or expired reset token"
        case "user_not_found":
            message.Error = "User account not found"
        case "invalid_token":
            message.Error = "Invalid reset verification token"
        case "auth_required":
            message.Error = "Please login to access the page"
        case "invalid_user":
            message.Error = "User data is invalid"
        default:
            message.Error = "An error occurred"
        }
    }

    // Handle success messages
    if successType := c.Query("success"); successType != "" {
        switch successType {
        case "password_reset":
            message.Success = "Password successfully reset. Please log in."
        }
    }

    render.Render(c, gin.H{
        "title":    "Login",
        "error":    message.Error,
        "success":  message.Success,
    }, "login.html")
}

func ShowRegistrationPage(c *gin.Context) {
    render.Render(c, gin.H{
        "title": "Registeration",
        "error": "",
    }, "register.html")
}

func ShowForgotPasswordPage(c *gin.Context) {
    render.Render(c, gin.H{
        "title": "Forgot Password",
        "error": "",
    }, "forgot_password.html")
}

func ShowResetPasswordPage(c *gin.Context) {
    token := c.Query("token")

    // Check if token is provided
    if token == "" {
        log.Printf("Reset password attempt with missing token")
        c.Redirect(http.StatusFound, "/auth/login?error=invalid_token")
        return
    }

    // Check if the token exists in the database (using Find)
    var passwordReset models.PasswordReset
    if err := config.DB.Where("token = ?", token).First(&passwordReset).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) { 
            log.Printf("Reset password: Invalid or expired token")
            c.Redirect(http.StatusFound, "/auth/login?error=invalid_token")
            return
        }
        log.Printf("Reset password: Error checking token: %v", err)
        c.Redirect(http.StatusFound, "/auth/login?error=invalid_token")
        return
    }

    // If token exists, render the reset password page
    render.Render(c, gin.H{
        "title": "Reset Password",
        "error": "",
        "token": token,
    }, "reset_password.html")
}

func Dashboard(c *gin.Context) {
    user, exists := c.Get("user")
    if !exists || user == nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        c.Abort()
        return
    }

    currentUser, ok := user.(*models.User)
    if !ok {
        c.Redirect(http.StatusFound, "/auth/login?error=invalid_user")
        c.Abort()
        return
    }

    error_message := c.Query("error")
    flashMessage, _ := c.Cookie("flash")
    c.SetCookie("flash", "", -1, "/", "", false, true)

    page := c.DefaultQuery("page", "1")
    limit := 4
    pageNum, err := strconv.Atoi(page)
    if err != nil {
        pageNum = 1
    }
    offset := (pageNum - 1) * limit

    var events []models.Event

    // Fetch events based on visibility rules
    query := config.DB.Offset(offset).Limit(limit)
    if !utils.IsAdmin(currentUser) {
        query = query.Where("status = ? OR created_by = ?", "published", currentUser.ID)
    }
    result := query.Find(&events)
    if result.Error != nil {
        // c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
        c.Redirect(http.StatusFound, "/user/dashboard?error=Failed to fetch events")
        c.Abort()
        return
    }

    // Apply edit permissions and truncate descriptions
    for i := range events {
        events[i].IsEditable = services.CheckEventEditPermission(&events[i], currentUser)
        events[i].Description = utils.Truncate(events[i].Description, 50) // Truncate to 50 characters
    }

    // HTMX handling
    if c.GetHeader("HX-Request") != "" {
        c.HTML(http.StatusOK, "event_cards.html", gin.H{
            "content":  events,
            "title":    "Dashboard",
        })
        return
    }

    // Count total events based on visibility rules for pagination
    var totalEvents int64
    countQuery := config.DB.Model(&models.Event{})
    if !utils.IsAdmin(currentUser) {
        countQuery = countQuery.Where("status = ? OR created_by = ?", "published", currentUser.ID)
    }
    countQuery.Count(&totalEvents)

    hasMore := offset+limit < int(totalEvents)

    render.Render(c, gin.H{
        "title":    "Dashboard",
        "user":     currentUser,
        "content":  events,
        "hasMore":  hasMore,
        "nextPage": pageNum + 1,
        "flash":    flashMessage,
        "error":    error_message,
    }, "dashboard.html")
}


func ShowProfilePage(c *gin.Context) {
    user, exists := c.Get("user")
    if !exists || user == nil {
        log.Printf("Dashboard: No user in context")
        c.Redirect(http.StatusFound, "/auth/login")
        return
    }

    render.Render(c, gin.H{
        "user":  user,
		"title": "Profile",
    }, "profile.html")
}

func ShowChangePasswordPage(c *gin.Context) {
    user, exists := c.Get("user")
    if !exists || user == nil {
        c.Redirect(http.StatusFound, "/auth/login")
        return
    }

    render.Render(c, gin.H{
		"title": "Change Password",
        "user":  user,
    }, "change_password.html")
}

/* func Dashboard(c *gin.Context) {
    user, exists := c.Get("user")
    if !exists || user == nil {
        c.Redirect(http.StatusFound, "/auth/login?error=auth_required")
        c.Abort()
        return
    }

    currentUser, ok := user.(*models.User)
    if !ok {
        c.Redirect(http.StatusFound, "/auth/login?error=invalid_user")
        c.Abort()
        return
    }

    error_message := c.Query("error")

    flashMessage, _ := c.Cookie("flash")
    c.SetCookie("flash", "", -1, "/", "", false, true)

    page := c.DefaultQuery("page", "1")
    limit := 4
    pageNum, err := strconv.Atoi(page)
    if err != nil {
        pageNum = 1
    }
    offset := (pageNum - 1) * limit

    var events []models.Event
    result := config.DB.Offset(offset).Limit(limit).Find(&events)

    if result.Error != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
        return
    }

    // formattedEvents := make([]gin.H, len(events))
    // for i, event := range events {
    //     formattedEvents[i] = gin.H{
    //         "ID":           event.ID,
    //         "Title":        event.Title,
    //         "Location":     event.Location,
    //         "Description":  event.Description,
    //         "Status":       event.Status,
    //         "Image":        event.Image,
    //         "StartTime":    utils.FormatDate(event.StartTime),
    //         "EndTime":      utils.FormatDate(event.EndTime),
    //         "IsEditable":   services.CheckEventEditPermission(&event, currentUser),
    //     }
    // }

    for i := range events {
        events[i].IsEditable = services.CheckEventEditPermission(&events[i], currentUser)
        events[i].Description = utils.Truncate(events[i].Description, 50) // Truncate to 50 characters
    }

    // HTMX handling
    if c.GetHeader("HX-Request") != "" {
        c.HTML(http.StatusOK, "event_cards.html", gin.H{
            "content": events,
        })
        return
    }

    // Full page rendering
    var totalEvents int64
    config.DB.Model(&models.Event{}).Count(&totalEvents)
    hasMore := offset+limit < int(totalEvents)

    render.Render(c, gin.H{
        "title":    "Dashboard",
        "user":     user,
        "content":  events,
        "hasMore":  hasMore,
        "nextPage": pageNum + 1,
        "flash":    flashMessage,
        "error":    error_message,
    }, "dashboard.html")
} */
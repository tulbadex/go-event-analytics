package handler

import (
    "event-analytics/render"
    "github.com/gin-gonic/gin"

    "log"
    "net/http"
	"event-analytics/config"
	"event-analytics/models"
	"event-analytics/utils"

    "gorm.io/gorm"
    "errors"
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
    user, err := utils.GetUserFromSession(c)
    if err != nil {
        c.Redirect(http.StatusFound, "/auth/login")
        return
    }

    var events []models.Event
    if err := config.DB.Find(&events).Error; err != nil {
        render.Render(c, gin.H{
            "error":   "Failed to fetch events",
            "title":   "Dashboard",
            "user":    user,
            "content": nil,
        }, "dashboard.html")
        return
    }

    // Debug logging
    log.Printf("Dashboard: User data: %+v", user)
    log.Printf("Dashboard: Events count: %d", len(events))

    render.Render(c, gin.H{
        "title":   "Dashboard",
        "user":    user,
        "content": events,
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
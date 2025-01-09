package router

import (
	"event-analytics/controllers"
	"event-analytics/handler"
	"event-analytics/middlewares"
	"fmt"
	"html/template"
	"time"

	"github.com/gin-gonic/gin"
)

// Format functions for templates
func formatAsDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	year, month, day := t.Date()
	return fmt.Sprintf("%d/%02d/%02d", year, month, day)
}

func formatDatetime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02T15:04")
}

func formatForDisplay(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Jan 2, 2006 3:04 PM")
}

func SetupRouter() *gin.Engine {
	r := gin.Default()
	
	// Important: First set the template functions
	r.SetFuncMap(template.FuncMap{
		"formatDate":     formatAsDate,      // for basic date formatting
		"formatDatetime": formatDatetime,    // for datetime-local inputs
		"formatDisplay":  formatForDisplay,  // for user-friendly display
	})

	// Then load the templates
	r.LoadHTMLGlob("../templates/*")

	// Set template delimiters
	r.Delims("{{", "}}")

	// Serve static files
	r.Static("/static", "./static")
	r.Static("/uploads", "./uploads")


	// Use middleware
	r.Use(middlewares.UserMiddleware())
	r.Use(middlewares.FlashMiddleware())

	// Routes
	r.GET("/", middlewares.PreventAuthenticatedAccess(), handler.ShowLoginPage)

	userRoutes := r.Group("/auth")
	userRoutes.Use(middlewares.PreventAuthenticatedAccess())
	{
		userRoutes.GET("/login", handler.ShowLoginPage)
		userRoutes.POST("/login", controllers.Login)
		userRoutes.GET("/register", handler.ShowRegistrationPage)
		userRoutes.POST("/register", controllers.Register)
		userRoutes.GET("/verify", controllers.Verify)
		userRoutes.GET("/forgot-password", handler.ShowForgotPasswordPage)
		userRoutes.POST("/forgot-password", controllers.ForgotPassword)
		userRoutes.GET("/reset-password", handler.ShowResetPasswordPage)
		userRoutes.POST("/reset-password", controllers.ResetPassword)
	}

	protected := r.Group("/user")
	protected.Use(middlewares.AuthRequired())
	{
		protected.GET("/dashboard", handler.Dashboard)
		protected.POST("/logout", controllers.Logout)
		protected.GET("/profile", handler.ShowProfilePage)
		protected.POST("/profile", controllers.EditProfile)
		protected.GET("/change-password", handler.ShowChangePasswordPage)
		protected.POST("/change-password", controllers.ChangePassword)
	}

	protected_event := r.Group("/events")
	protected_event.Use(middlewares.AuthRequired())
	{
		protected_event.GET("/new", handler.ShowCreateEventPage)
		protected_event.POST("/create", controllers.CreateEvent)
		protected_event.GET("/:id", handler.ShowEventDetails)
		protected_event.GET("/edit/:id", handler.ShowEditEventPage)
		protected_event.POST("/update/:id", controllers.UpdateEvent)
		protected_event.POST("/delete/:id", controllers.DeleteEvent)
	}

	r.GET("/ws", handler.WebSocketHandler)
	return r
}
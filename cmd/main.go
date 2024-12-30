package main

import (
	"event-analytics/config"
	"event-analytics/controllers"
	"event-analytics/handler"
	"event-analytics/middlewares"
	"event-analytics/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize database and Redis
	// config.InitDB()
	// config.InitRedis()
	config.Init()
	utils.InitializeRoles()

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	// Important: Set template delimiters to default Go template delimiters
    r.Delims("{{", "}}")

	// Serve static files
	r.Static("/static", "./static")

	// Use middleware
	r.Use(middlewares.UserMiddleware())

	// Routes
	r.GET("/", middlewares.PreventAuthenticatedAccess(), handler.ShowLoginPage)

	userRoutes := r.Group("/auth")
	userRoutes.Use(middlewares.PreventAuthenticatedAccess())
	{
		userRoutes.GET("/login", handler.ShowLoginPage)
		userRoutes.POST("/login", controllers.Login)

		userRoutes.GET("/register", handler.ShowRegistrationPage)
		userRoutes.POST("/register", controllers.Register)

		// Email verification route
		userRoutes.GET("/verify", controllers.Verify)

		// forgot password
		userRoutes.GET("/forgot-password", handler.ShowForgotPasswordPage)
		userRoutes.POST("/forgot-password", controllers.ForgotPassword)

		// reset password
		userRoutes.GET("/reset-password", handler.ShowResetPasswordPage)
		userRoutes.POST("/reset-password", controllers.ResetPassword)
	}

	protected := r.Group("/user")
	protected.Use(middlewares.AuthRequired())
	{
		// protected.GET("/dashboard", controllers.Dashboard)
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

		// protected_event.GET("/events/:id", handler.ShowEventDetails)
		// protected_event.GET("/events/edit/:id", handler.ShowEditEventPage)
		// protected_event.POST("/events/edit/:id", controllers.UpdateEvent)
		// protected_event.POST("/events/delete/:id", controllers.DeleteEvent)
	}

	// Add this to serve uploaded files
	r.Static("/uploads", "./uploads")

	r.Run(":8080")
}

package main

import (
	"event-analytics/config"
	"event-analytics/controllers"
	"event-analytics/cron"
	"event-analytics/handler"
	"event-analytics/middlewares"
	"event-analytics/pkg/csrf"
	"event-analytics/pkg/ratelimit"
	"event-analytics/utils"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

func main() {
	// Initialize database and Redis
	config.Init()
	utils.InitializeRoles()

	r := gin.Default()

	// Set template functions
	r.SetFuncMap(template.FuncMap{
		"formatDate":     formatAsDate,
		"formatDatetime": formatDatetime,
		"formatDisplay":  formatForDisplay,
	})

	// Start the cron jobs
	go cron.StartCronJobs()

	// Load all templates
	r.LoadHTMLGlob("templates/*")

	// Serve static files
	r.Static("/static", "./static")
	r.Static("/uploads", "./uploads")

	// Use middleware
	r.Use(middlewares.Recovery())
	r.Use(middlewares.Logger())
	r.Use(middlewares.ErrorHandler())
	r.Use(ratelimit.Middleware(config.RedisClient, 100, time.Minute))
	r.Use(middlewares.UserMiddleware())
	r.Use(middlewares.FlashMiddleware())
	r.Use(csrf.Middleware())

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

	// Start the WebSocket hub
	go handler.Hub.Run()

	// Graceful shutdown
	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}
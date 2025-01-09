package tests

import (
	"context"
	"event-analytics/config"
	"event-analytics/controllers"
	"event-analytics/handler"
	"event-analytics/middlewares"
	"event-analytics/models"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	TestDB *gorm.DB // Exported for use in other test files
)

var testDB *gorm.DB

func TestingSuite(m *testing.M) {
	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Get main database URL
	mainDBURL := os.Getenv("DATABASE_URL")
	if mainDBURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Create test database name
	testDBName := "event_analytics_test"

	// Connect to default postgres database to create/drop test database
	defaultDB, err := gorm.Open(postgres.Open(mainDBURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to default database: %v", err)
	}

	sqlDB, err := defaultDB.DB()
	if err != nil {
		log.Fatalf("Failed to get underlying SQL DB: %v", err)
	}

	// Drop test database if it exists
	sqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE);", testDBName))

	// Create test database
	_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", testDBName))
	if err != nil {
		log.Fatalf("Failed to create test database: %v", err)
	}
	if err != nil {
		log.Fatalf("Failed to create test database: %v", err)
	}

	// Close connection to default database
	sqlDB.Close()

	// Create DSN for test database
	testDBURL := getTestDatabaseURL(mainDBURL, testDBName)

	// Connect to test database
	testDB, err = gorm.Open(postgres.Open(testDBURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	err = testDB.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.VerificationToken{},
		&models.PasswordReset{},
		&models.Role{},
		&models.UserRole{},
		&models.PasswordHistory{},
		&models.UserLog{},
	)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Replace the main DB with test DB
	config.DB = testDB

	// Run tests
	code := m.Run()

	// Cleanup: Close connection and drop test database
	sqlDB, err = testDB.DB()
	if err != nil {
		log.Printf("Warning: failed to get underlying SQL DB: %v", err)
	} else {
		sqlDB.Close()
	}

	// Reconnect to default database to drop test database
	defaultDB, err = gorm.Open(postgres.Open(mainDBURL), &gorm.Config{})
	if err != nil {
		log.Printf("Warning: failed to connect to default database for cleanup: %v", err)
	} else {
		defaultDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE);", testDBName))
	}

	ClearTestData(testDB)

	if config.RedisClient != nil {
        config.RedisClient.FlushAll(context.Background())
        config.RedisClient.Close()
    }

	os.Exit(code)
}

// ClearTestData removes all data from the test database
func ClearTestData(db *gorm.DB) {
	// PostgreSQL specific cleanup
	db.Exec("DO $$ DECLARE r RECORD; BEGIN FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP EXECUTE 'TRUNCATE TABLE ' || quote_ident(r.tablename) || ' CASCADE'; END LOOP; END $$;")
}

func getTestDatabaseURL(originalURL, testDBName string) string {
	// Find the last '/' and replace the database name
	lastSlash := strings.LastIndex(originalURL, "/")
	if lastSlash == -1 {
		db_name := originalURL + "/" + testDBName + "?sslmode=disable"
		return db_name
	}
	db_name := originalURL[:lastSlash+1] + testDBName + "?sslmode=disable"
	return db_name
}

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

func SetupTestRouter() *gin.Engine {

	config.RedisClient = redis.NewClient(&redis.Options{
		// Addr: "localhost:6379", // Adjust to your Redis server
		Addr: os.Getenv("REDIS_ADDR"),
		Password: "",
        DB:       0,
	})
	
	gin.SetMode(gin.TestMode)

	r := gin.Default()

	// Important: First set the template functions
	r.SetFuncMap(template.FuncMap{
		"formatDate":     formatAsDate,      // for basic date formatting
		"formatDatetime": formatDatetime,    // for datetime-local inputs
		"formatDisplay":  formatForDisplay,  // for user-friendly display
	})
	
	r.LoadHTMLGlob("../templates/*.html")

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
	return r
}

// Utility to simulate logged-in user
func MockSession(c *gin.Context, user *models.User) {
	c.Set("user", user)
}

// Helper to create a test user
func CreateTestUser(t *testing.T) *models.User {
	user := &models.User{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}
	user.HashPassword()
	assert.NoError(t, config.DB.Create(user).Error)
	return user
}

// Helper to create a test user
func CreateSecondTestUser(t *testing.T) *models.User {
	user := &models.User{
		Username: "testuser1",
		Email:    "test1@example.com",
		Password: "password123",
	}
	user.HashPassword()
	assert.NoError(t, config.DB.Create(user).Error)
	return user
}

func CreateTestEvent(t *testing.T, userID uuid.UUID) *models.Event {
	event := &models.Event{
		Title:       "Test Event",
		Description: "Test Description",
		StartTime:   time.Now(),
		EndTime:     time.Now().Add(time.Hour * 2),
		Location:    "Test Location",
		CreatedBy:   userID,
		Status:      "published",
	}
	result := config.DB.Create(event)
	assert.NoError(t, result.Error)
	return event
}

// SetupTestDB initializes the test database
func SetupTestDB() (*gorm.DB, error) {
	// Load environment variables
	err := godotenv.Load("../.env")
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	// Get main database URL
	mainDBURL := os.Getenv("DATABASE_URL")
	if mainDBURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable is not set")
	}

	// Create test database name
	testDBName := "event_analytics_test"

	// Connect to default postgres database to create/drop test database
	defaultDB, err := gorm.Open(postgres.Open(mainDBURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to default database: %v", err)
	}

	sqlDB, err := defaultDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %v", err)
	}
	defer sqlDB.Close()

	// Drop test database if it exists
	sqlDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE);", testDBName))

	// Create test database
	_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s;", testDBName))
	if err != nil {
		return nil, fmt.Errorf("failed to create test database: %v", err)
	}

	// Create DSN for test database
	testDBURL := getTestDatabaseURL(mainDBURL, testDBName)

	// Connect to test database
	testDB, err := gorm.Open(postgres.Open(testDBURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %v", err)
	}

	return testDB, nil
}

// SetupTestRedis initializes the test Redis client
func SetupTestRedis() (*redis.Client, error) {
	return redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: "",
		DB:       0,
	}), nil
}

// TestMainHelper is a helper function that implements the common TestMain logic
func TestMainHelper(m *testing.M) int {
	var err error
	
	// Setup test database
	TestDB, err = SetupTestDB()
	if err != nil {
		log.Fatalf("Failed to setup test database: %v", err)
	}

	// Run migrations
	err = TestDB.AutoMigrate(
		&models.User{},
		&models.Event{},
		&models.VerificationToken{},
		&models.PasswordReset{},
		&models.Role{},
		&models.UserRole{},
		&models.PasswordHistory{},
		&models.UserLog{},
	)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Set the global DB instance
	config.DB = TestDB

	// Setup Redis
	config.RedisClient, err = SetupTestRedis()
	if err != nil {
		log.Fatalf("Failed to setup test Redis: %v", err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	CleanupTests()

	return code
}

// CleanupTests handles all cleanup after tests
func CleanupTests() {
	if TestDB != nil {
		sqlDB, err := TestDB.DB()
		if err == nil {
			sqlDB.Close()
		}
	}

	if config.RedisClient != nil {
		config.RedisClient.FlushAll(context.Background())
		config.RedisClient.Close()
	}
}
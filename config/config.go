package config

import (
	"log"
	"os"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/joho/godotenv"
	"event-analytics/models"
)

var DB *gorm.DB
var RedisClient *redis.Client

// Initialize the database connection and run migrations
func InitDB() {
	log.Printf("DATABASE_URL: %s", os.Getenv("DATABASE_URL"))
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = DB.AutoMigrate(
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
}

// Initialize Redis
func InitRedis() {
	RedisClient = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})
}

func Init() {
    err := godotenv.Load()
    if err != nil {
        log.Printf("Error loading .env file: %v", err)
    } else {
        log.Println(".env file loaded successfully")
    }
    InitDB()
    InitRedis()
}

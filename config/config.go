package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"event-analytics/models"
	"event-analytics/pkg/session"
	"github.com/go-redis/redis/v8"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB
var RedisClient *redis.Client
var SessionStore *session.Store

// Initialize the database connection and run migrations
func InitDB() {
	log.Printf("DATABASE_URL: %s", os.Getenv("DATABASE_URL"))
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	// Extract database name from DSN
	var dbName string
	if strings.Contains(dsn, "database=") {
		parts := strings.Split(dsn, "database=")
		if len(parts) > 1 {
			dbName = strings.Split(parts[1], " ")[0]
		}
	} else if strings.Contains(dsn, "/") {
		parts := strings.Split(dsn, "/")
		if len(parts) > 0 {
			dbName = strings.Split(parts[len(parts)-1], "?")[0]
		}
	}

	// Try to connect, if database doesn't exist, create it
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil && strings.Contains(err.Error(), "does not exist") {
		// Connect to postgres database to create the target database
		adminDSN := strings.Replace(dsn, dbName, "postgres", 1)
		sqlDB, err := sql.Open("postgres", adminDSN)
		if err != nil {
			log.Fatalf("Failed to connect to postgres: %v", err)
		}
		defer sqlDB.Close()

		_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			log.Fatalf("Failed to create database: %v", err)
		}
		log.Printf("Database %s created successfully", dbName)

		// Now connect to the newly created database
		DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}
	} else if err != nil {
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
	SessionStore = session.NewStore(RedisClient, 24*time.Hour)
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

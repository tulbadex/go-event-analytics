package utils

import (
	"math/rand"
	"time"

	"gopkg.in/gomail.v2"
	"os"
	"fmt"
	"net/http"

	"event-analytics/config"
	"event-analytics/models"
	"errors"
	"github.com/gin-gonic/gin"
	"context"

	"bytes"
	"html/template"
	"log"

	"strings"
)

func GenerateRandomToken() string {
	// rand.Seed(time.Now().UnixNano())
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	token := make([]rune, 32)
	for i := range token {
		token[i] = letters[rand.Intn(len(letters))]
	}
	return string(token)
}

// SendEmailAsync sends an email asynchronously
func SendEmailAsync(c *http.Request, to, subject, body string) {
	go func() {
		err := SendEmail(c, to, subject, body)
		if err != nil {
			fmt.Println("Failed to send email:", err)
		}
	}()
}

func SendEmail(c *http.Request, to, subject, body string) error {
    m := gomail.NewMessage()
    // Set From address with angle brackets
    m.SetHeader("From", fmt.Sprintf("%s <%s>", os.Getenv("MAIL_SENDER"), os.Getenv("MAIL_USERNAME")))
    m.SetHeader("To", to)
    m.SetHeader("Subject", subject)
    m.SetBody("text/html", body)

    d := gomail.NewDialer(os.Getenv("MAIL_HOST"), 587, os.Getenv("MAIL_USERNAME"), os.Getenv("MAIL_PASSWORD"))

    return d.DialAndSend(m)
}

func GetUserFromSession(c *gin.Context) (*models.User, error) {
    sessionToken, err := c.Cookie("session_token")
    if err != nil {
        log.Printf("GetUserFromSession: No session token found: %v", err)
        return nil, errors.New("session token not found")
    }

    userID, err := config.RedisClient.Get(c, sessionToken).Result()
    if err != nil {
        log.Printf("GetUserFromSession: Failed to get user ID from Redis: %v", err)
        return nil, errors.New("invalid or expired session token")
    }

    var user models.User
    if err := config.DB.Where("id = ?", userID).First(&user).Error; err != nil {
        log.Printf("GetUserFromSession: Failed to find user in DB: %v", err)
        return nil, fmt.Errorf("user not found: %w", err)
    }

    // Sanitize the user object before returning
    sanitizedUser := &models.User{
        ID:        		user.ID,
        Username:  		user.Username,
        FirstName:  	user.FirstName,
        LastName:  		user.LastName,
        Email:     		user.Email,
        Address:   		user.Address,
        IsVerified:   	user.IsVerified,
        CreatedAt: 		user.CreatedAt,
        UpdatedAt: 		user.UpdatedAt,
        DeletedAt:		user.DeletedAt,
    }

    log.Printf("GetUserFromSession: Successfully found sanitized user: %+v", sanitizedUser)
    return sanitizedUser, nil
}

func SaveSession(token string, user models.User) error {
    // Convert UUID to string if it's not already a string
    userIDString := user.ID.String()
    return config.RedisClient.Set(context.Background(), token, userIDString, time.Hour).Err()
}

func ValidateSessionToken(c *gin.Context, token string) (string, error) {
    userID, err := config.RedisClient.Get(c, token).Result()
    if err != nil {
        return "", errors.New("invalid or expired session token")
    }
    return userID, nil
}

func RenderTemplate(filePath string, data interface{}) string {
	tmpl, err := template.ParseFiles(filePath)
	if err != nil {
		log.Printf("Failed to parse template: %v", err)
		return ""
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		log.Printf("Failed to execute template: %v", err)
		return ""
	}

	return buf.String()
}

// GetBaseURL extracts the base URL from the request
func GetBaseURL(r *http.Request) string {
    scheme := "http"
    if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
        scheme = "https"
    }
    
    // Get the host without any path
    host := r.Host
    if strings.Contains(host, ":") {
        hostParts := strings.Split(host, ":")
        if len(hostParts) > 0 {
            host = hostParts[0] + ":" + hostParts[1]
        }
    }
    
    return fmt.Sprintf("%s://%s", scheme, host)
}

func SetUserInSession(c *gin.Context, user *models.User) error {
	if user == nil {
		return errors.New("invalid user")
	}

	// Retrieve session token from cookie
	sessionToken, err := c.Cookie("session_token")
	if err != nil {
		log.Printf("SetUserInSession: No session token found: %v", err)
		return errors.New("session token not found")
	}

	// Save the user's ID in Redis using the session token as the key
	userIDString := user.ID.String()
	if err := config.RedisClient.Set(context.Background(), sessionToken, userIDString, time.Hour).Err(); err != nil {
		log.Printf("SetUserInSession: Failed to set user in session: %v", err)
		return errors.New("failed to update session")
	}

	log.Printf("SetUserInSession: Successfully updated user in session: %v", userIDString)
	return nil
}

// UpdateUserProfile updates the user profile while ensuring unique constraints on username and email
func UpdateUserProfile(userID string, updates map[string]interface{}) error {
	// Check if username exists for another user
	if username, ok := updates["username"].(string); ok {
		var existingUser models.User
		if err := config.DB.Where("username = ? AND id != ?", username, userID).First(&existingUser).Error; err == nil {
			return errors.New("username already exists")
		}
	}

	// Check if email exists for another user
	if email, ok := updates["email"].(string); ok {
		var existingUser models.User
		if err := config.DB.Where("email = ? AND id != ?", email, userID).First(&existingUser).Error; err == nil {
			return errors.New("email already exists")
		}
	}

	// Perform the update operation
	if err := config.DB.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		log.Printf("UpdateUserProfile: Failed to update user profile: %v", err)
		return errors.New("failed to update profile")
	}

	log.Printf("UpdateUserProfile: Successfully updated profile for user ID %v", userID)
	return nil
}

func InitializeRoles() {
	roles := []string{"admin", "user", "moderator"} // Define your roles here

	for _, role := range roles {
		var existingRole models.Role
		result := config.DB.Where("name = ?", role).First(&existingRole)

		if result.Error != nil && result.Error.Error() == "record not found" {
			newRole := models.Role{Name: role}
			if err := config.DB.Create(&newRole).Error; err != nil {
				log.Fatalf("Failed to create role %s: %v", role, err)
			} else {
				log.Printf("Role %s created successfully.", role)
			}
		} else {
			log.Printf("Role %s already exists.", role)
		}
	}
}

func Truncate(input string, length int) string {
    if len(input) > length {
        return input[:length] + "..."
    }
    return input
}

func IsAdminOrOwner(user interface{}, event models.Event) bool {
    currentUser, ok := user.(*models.User)
    if !ok || currentUser == nil {
        return false
    }

    if event.CreatedBy == currentUser.ID {
        return true
    }

    var userRoles []models.Role
    if err := config.DB.Model(&models.Role{}).
        Joins("JOIN user_roles ON user_roles.role_id = roles.id").
        Where("user_roles.user_id = ?", currentUser.ID).
        Find(&userRoles).Error; err != nil {
        return false
    }

    for _, role := range userRoles {
        if role.Name == "admin" {
            return true
        }
    }

    return false
}
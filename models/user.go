package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

type User struct {
	ID        	uuid.UUID      	`gorm:"type:uuid;primaryKey"`
	Username  	string         	`gorm:"size:100;unique;not null"`
	FirstName 	string         	`gorm:"size:100"`
	LastName  	string         	`gorm:"size:100"`
	Email     	string         	`gorm:"size:150;unique;not null"`
	Password  	string         	`gorm:"not null"`
	Address   	string         	`gorm:"size:255"`
	IsVerified 	bool 			`gorm:"default:false"`
	CreatedAt 	time.Time 	 	`gorm:"autoCreateTime"`
	UpdatedAt 	time.Time 	 	`gorm:"autoUpdateTime"`
	DeletedAt 	gorm.DeletedAt 	`gorm:"index"`
}

// BeforeCreate generates a UUID for new users.
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = uuid.New()
	return
}

// HashPassword hashes a user's password.
func (u *User) HashPassword() error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashed)
	return nil
}

// CheckPassword verifies a user's password.
func (u *User) CheckPassword(providedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(providedPassword))
}

// func FindUserByUsername(db *sql.DB, username string) (*User, error) {
// 	user := &User{}
// 	err := db.QueryRow("SELECT id, username, password FROM users WHERE username=$1", username).
// 		Scan(&user.ID, &user.Username, &user.Password)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return user, nil
// }

// FindUserByUsername retrieves a user by their username
func FindUserByUsername(db *gorm.DB, username string) (*User, error) {
	var user User
	if err := db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// func RegisterUser(db *sql.DB, username, password string) error {
// 	_, err := db.Exec("INSERT INTO users (username, password) VALUES ($1, $2)", username, password)
// 	return err
// }
// RegisterUser saves a new user to the database
func RegisterUser(db *gorm.DB, user *User) error {
	return db.Create(user).Error
}

// IsUsernameAvailable checks if the supplied username is available
// func IsUsernameAvailable(username string) (bool, error) {
//     db := repository.GetDB()
//     var exists bool

//     // Query to check if the username exists
//     err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)", username).Scan(&exists)
//     if err != nil {
//         return false, err // Return false and the error if the query fails
//     }

//     return !exists, nil // Return true if username is available (not exists)
// }
// IsUsernameAvailable checks if the supplied username is available
func IsUsernameAvailable(db *gorm.DB, username string) (bool, error) {
	var count int64
	if err := db.Model(&User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err // Return false and the error if the query fails
	}
	return count == 0, nil // Return true if username is available (count should be 0)
}

// IsPasswordUsed checks if a given plain password matches any password in the history.
func (u *User) IsPasswordUsed(plainPassword string, passwordHistories []PasswordHistory) bool {
	for _, history := range passwordHistories {
		err := bcrypt.CompareHashAndPassword([]byte(history.Password), []byte(plainPassword))
		if err == nil {
			return true // Password matches one in the history
		}
	}
	return false
}
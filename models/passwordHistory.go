package models

import (
    "time"
	"github.com/google/uuid"
)

type PasswordHistory struct {
	ID       	uint      	`gorm:"primaryKey"`
	UserID    	uuid.UUID 	`gorm:"type:uuid;not null;index;references:ID;constraint:OnDelete:CASCADE"`
	Password 	string    	`gorm:"not null"`
	CreatedAt   time.Time 	`gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time 	`gorm:"autoUpdateTime" json:"updated_at"`
}
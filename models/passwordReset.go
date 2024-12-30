package models

import (
    "time"
)

type PasswordReset struct {
    ID          uint      `gorm:"primaryKey"`
    Email       string    `gorm:"type:varchar(255);not null;index"`
    Token       string    `gorm:"type:varchar(255);not null"`
    CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

package models

import (
    "time"
    "github.com/google/uuid"
)

type VerificationToken struct {
    ID        uint      `gorm:"primaryKey"`
    UserID    uuid.UUID `gorm:"type:uuid;not null;index;references:ID;constraint:OnDelete:CASCADE"`
    Token     string    `gorm:"type:varchar(255);not null"`
    CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

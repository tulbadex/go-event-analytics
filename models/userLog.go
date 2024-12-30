package models

import (
	"time"

	"github.com/google/uuid"
)

type UserLog struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;references:ID;constraint:OnDelete:CASCADE"`
	Action    string    `gorm:"size:255;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

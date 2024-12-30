package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Event represents the structure of an event
type Event struct {
    ID          uuid.UUID       `gorm:"type:uuid;primaryKey" json:"id"`
    Title       string          `gorm:"size:255;not null" json:"title"`
    Description string          `gorm:"type:text;not null" json:"description"`
    StartTime   time.Time       `json:"start_time"`
    EndTime     time.Time       `json:"end_time"`
    Location    string          `gorm:"size:255" json:"location"`
    Image       string          `gorm:"size:255" json:"image"`
    Status      string          `gorm:"size:50;default:'draft'" json:"status"` // draft or published
    CreatedBy   uuid.UUID       `gorm:"not null" json:"created_by"`
    CreatedAt   time.Time 	    `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time 	    `gorm:"autoUpdateTime" json:"updated_at"`
    DeletedAt   gorm.DeletedAt  `gorm:"index" json:"deleted_at"`
    IsEditable  bool            `gorm:"-" json:"is_editable"` // Virtual field
}

func (e *Event) BeforeCreate(tx *gorm.DB) (err error) {
    e.ID = uuid.New()
    return
}
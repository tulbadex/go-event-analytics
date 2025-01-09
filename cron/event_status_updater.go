package cron

import (
	"log"
	"time"

	"event-analytics/config"
	"event-analytics/models"
)

func UpdateEventStatuses() {
	currentTime := time.Now()

	// Update draft events to published if their published_date is today
	err := config.DB.Model(&models.Event{}).
		Where("status = ? AND published_date <= ?", "draft", currentTime).
		Update("status", "published").Error
	if err != nil {
		log.Printf("Failed to update draft events to published: %v", err)
	}

	// Update events to expired if their end_time is in the past
	err = config.DB.Model(&models.Event{}).
		Where("status != ? AND end_time <= ?", "expired", currentTime).
		Update("status", "expired").Error
	if err != nil {
		log.Printf("Failed to update events to expired: %v", err)
	}
}

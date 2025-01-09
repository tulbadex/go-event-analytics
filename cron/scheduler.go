package cron

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

func StartCronJobs() {
	scheduler := gocron.NewScheduler(time.Local)

	// Schedule the event status updater every minute
	_, err := scheduler.Every(1).Minute().Do(UpdateEventStatuses)
	if err != nil {
		log.Fatalf("Failed to schedule event status updater: %v", err)
	}

	// Start the scheduler
	scheduler.StartAsync()
}
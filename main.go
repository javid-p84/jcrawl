package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/restaurant"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/scheduler"
)

func main() {
	log.Println("jcrawl - Restaurant Availability Monitoring and Auto-Booking Service")

	// Example restaurant preference
	pref := &models.RestaurantPreference{
		ID:            "rest-001",
		GoogleLink:    "https://www.google.com/maps/...", // TODO: Replace with actual link
		DateRangeFrom: time.Now(),
		DateRangeTo:   time.Now().AddDate(0, 0, 30), // 30 days from now
		DayPreference: []int{5, 6}, // Friday and Saturday
		PartySize:     2,
		CreatedAt:     time.Now(),
	}

	// Initialize checker and scheduler
	checker := restaurant.NewChecker()
	sched := scheduler.NewScheduler(checker)

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down gracefully...")
		cancel()
	}()

	// Start monitoring every 5 minutes
	sched.Start(ctx, pref, 5*time.Minute)
}

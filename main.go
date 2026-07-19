package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/api"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/booker"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/crypto"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/restaurant"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/worker"
)

func main() {
	log.Println("jcrawl - Restaurant Availability Monitoring and Auto-Booking Service")

	// Load environment variables
	godotenv.Load()

	// Initialize crypto manager
	cryptoMgr, err := crypto.NewCryptoManager()
	if err != nil {
		log.Fatalf("Failed to initialize crypto: %v", err)
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/jcrawl?sslmode=disable"
	}

	database, err := db.Connect(dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize schema
	if err := db.InitializeSchema(database); err != nil {
		log.Fatalf("Failed to initialize schema: %v", err)
	}

	// Initialize repositories
	userRepo := db.NewUserRepository(database)
	prefRepo := db.NewPreferenceRepository(database)
	bookRepo := db.NewBookingRepository(database)
	notifRepo := db.NewNotificationRepository(database)

	// Initialize services
	checker := restaurant.NewChecker()
	bookr, err := booker.NewBooker()
	if err != nil {
		log.Printf("Warning: Failed to initialize booker: %v\n", err)
	}
	checkWorker := worker.NewCheckWorker(prefRepo, bookRepo, checker, bookr, 5*time.Minute)
	apiHandler := api.NewHandler(userRepo, prefRepo, bookRepo, notifRepo)

	// Setup routes
	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/health", apiHandler.Health).Methods("GET")

	// Auth endpoints
	router.HandleFunc("/api/v1/auth/register", apiHandler.Register).Methods("POST")
	router.HandleFunc("/api/v1/auth/login", apiHandler.Login).Methods("POST")

	// Preference endpoints
	router.HandleFunc("/api/v1/preferences", apiHandler.CreatePreference).Methods("POST")
	router.HandleFunc("/api/v1/preferences", apiHandler.GetPreferences).Methods("GET")

	// Booking endpoints
	router.HandleFunc("/api/v1/bookings", apiHandler.GetBookings).Methods("GET")

	// Notification endpoints
	router.HandleFunc("/api/v1/notifications", apiHandler.GetNotifications).Methods("GET")
	router.HandleFunc("/api/v1/notifications/unread-count", apiHandler.GetUnreadNotificationCount).Methods("GET")
	router.HandleFunc("/api/v1/notifications/mark-as-read", apiHandler.MarkNotificationAsRead).Methods("POST")
	router.HandleFunc("/api/v1/notifications/mark-all-as-read", apiHandler.MarkAllNotificationsAsRead).Methods("POST")

	// Setup HTTP server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start worker in background
	go checkWorker.Start(ctx)

	// Start server in goroutine
	go func() {
		log.Printf("Starting server on %s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down gracefully...")
	cancel()

	// Shutdown server with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}

package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("Connected to database successfully")
	return db, nil
}

// InitializeSchema creates all necessary tables
func InitializeSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS user_preferences (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		google_link TEXT NOT NULL,
		restaurant_name VARCHAR(255),
		date_range_from DATE NOT NULL,
		date_range_to DATE NOT NULL,
		day_preference INTEGER[] DEFAULT ARRAY[]::INTEGER[],
		party_size INTEGER NOT NULL,
		auto_book BOOLEAN DEFAULT true,
		active BOOLEAN DEFAULT true,
		guest_name VARCHAR(255),
		guest_email VARCHAR(255),
		guest_phone VARCHAR(20),
		special_notes TEXT,
		last_checked_at TIMESTAMP,
		last_booked_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS booking_history (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		preference_id UUID NOT NULL REFERENCES user_preferences(id) ON DELETE CASCADE,
		user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		booking_date DATE NOT NULL,
		booking_time VARCHAR(10),
		party_size INTEGER NOT NULL,
		status VARCHAR(50) NOT NULL,
		confirmation_id VARCHAR(255),
		notes TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id ON user_preferences(user_id);
	CREATE INDEX IF NOT EXISTS idx_booking_history_user_id ON booking_history(user_id);
	CREATE INDEX IF NOT EXISTS idx_booking_history_preference_id ON booking_history(preference_id);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Schema initialized successfully")
	return nil
}

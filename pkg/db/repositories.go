package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) CreateUser(user *models.User) error {
	err := r.db.QueryRow(
		"INSERT INTO users (email, password) VALUES ($1, $2) RETURNING id, created_at, updated_at",
		user.Email, user.Password,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	return err
}

func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		"SELECT id, email, password, created_at, updated_at FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Email, &user.Password, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

type PreferenceRepository struct {
	db *sql.DB
}

func NewPreferenceRepository(db *sql.DB) *PreferenceRepository {
	return &PreferenceRepository{db: db}
}

func (r *PreferenceRepository) CreatePreference(pref *models.UserPreference) error {
	err := r.db.QueryRow(
		`INSERT INTO user_preferences
		(user_id, google_link, restaurant_name, date_range_from, date_range_to,
		 day_preference, party_size, auto_book, active, guest_name, guest_email, guest_phone, special_notes,
		 recreation_gov_username, recreation_gov_password)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at`,
		pref.UserID, pref.GoogleLink, pref.RestaurantName, pref.DateRangeFrom, pref.DateRangeTo,
		pref.DayPreference, pref.PartySize, pref.AutoBook, pref.Active,
		pref.GuestName, pref.GuestEmail, pref.GuestPhone, pref.SpecialNotes,
		pref.RecreationGovUsername, pref.RecreationGovPassword,
	).Scan(&pref.ID, &pref.CreatedAt, &pref.UpdatedAt)
	return err
}

func (r *PreferenceRepository) GetPreferencesByUserID(userID string) ([]models.UserPreference, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, google_link, restaurant_name, date_range_from, date_range_to,
		 day_preference, party_size, auto_book, active, guest_name, guest_email, guest_phone,
		 special_notes, recreation_gov_username, recreation_gov_password, last_checked_at, last_booked_at, created_at, updated_at
		 FROM user_preferences WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []models.UserPreference
	for rows.Next() {
		var pref models.UserPreference
		err := rows.Scan(
			&pref.ID, &pref.UserID, &pref.GoogleLink, &pref.RestaurantName,
			&pref.DateRangeFrom, &pref.DateRangeTo, &pref.DayPreference,
			&pref.PartySize, &pref.AutoBook, &pref.Active, &pref.GuestName, &pref.GuestEmail,
			&pref.GuestPhone, &pref.SpecialNotes, &pref.RecreationGovUsername, &pref.RecreationGovPassword,
			&pref.LastCheckedAt, &pref.LastBookedAt, &pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Don't return passwords in API responses
		pref.RecreationGovPassword = ""
		prefs = append(prefs, pref)
	}
	return prefs, rows.Err()
}

func (r *PreferenceRepository) GetActivePreferences() ([]models.UserPreference, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, google_link, restaurant_name, date_range_from, date_range_to,
		 day_preference, party_size, auto_book, active, guest_name, guest_email, guest_phone,
		 special_notes, recreation_gov_username, recreation_gov_password, last_checked_at, last_booked_at, created_at, updated_at
		 FROM user_preferences WHERE active = true`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prefs []models.UserPreference
	for rows.Next() {
		var pref models.UserPreference
		err := rows.Scan(
			&pref.ID, &pref.UserID, &pref.GoogleLink, &pref.RestaurantName,
			&pref.DateRangeFrom, &pref.DateRangeTo, &pref.DayPreference,
			&pref.PartySize, &pref.AutoBook, &pref.Active, &pref.GuestName, &pref.GuestEmail,
			&pref.GuestPhone, &pref.SpecialNotes, &pref.RecreationGovUsername, &pref.RecreationGovPassword,
			&pref.LastCheckedAt, &pref.LastBookedAt, &pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		prefs = append(prefs, pref)
	}
	return prefs, rows.Err()
}

func (r *PreferenceRepository) UpdateLastChecked(preferenceID string) error {
	now := time.Now()
	_, err := r.db.Exec(
		"UPDATE user_preferences SET last_checked_at = $1, updated_at = $1 WHERE id = $2",
		now, preferenceID,
	)
	return err
}

type BookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

func (r *BookingRepository) CreateBooking(booking *models.BookingHistory) error {
	err := r.db.QueryRow(
		`INSERT INTO booking_history
		(preference_id, user_id, booking_date, booking_time, party_size, status, confirmation_id, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`,
		booking.PreferenceID, booking.UserID, booking.BookingDate, booking.BookingTime,
		booking.PartySize, booking.Status, booking.ConfirmationID, booking.Notes,
	).Scan(&booking.ID, &booking.CreatedAt, &booking.UpdatedAt)
	return err
}

func (r *BookingRepository) GetBookingsByUserID(userID string) ([]models.BookingHistory, error) {
	rows, err := r.db.Query(
		`SELECT id, preference_id, user_id, booking_date, booking_time, party_size,
		 status, confirmation_id, notes, created_at, updated_at
		 FROM booking_history WHERE user_id = $1 ORDER BY booking_date DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []models.BookingHistory
	for rows.Next() {
		var booking models.BookingHistory
		err := rows.Scan(
			&booking.ID, &booking.PreferenceID, &booking.UserID, &booking.BookingDate,
			&booking.BookingTime, &booking.PartySize, &booking.Status, &booking.ConfirmationID,
			&booking.Notes, &booking.CreatedAt, &booking.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		bookings = append(bookings, booking)
	}
	return bookings, rows.Err()
}

type NotificationRepository struct {
	db *sql.DB
}

func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) CreateNotification(notif *models.Notification) error {
	var dataJSON sql.NullString
	if notif.Data != nil {
		// Convert map to JSON - simplified for this example
		dataJSON = sql.NullString{String: "{}", Valid: true}
	}

	err := r.db.QueryRow(
		`INSERT INTO notifications
		(user_id, preference_id, booking_id, type, title, message, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		notif.UserID, notif.PreferenceID, notif.BookingID,
		notif.Type, notif.Title, notif.Message, dataJSON,
	).Scan(&notif.ID, &notif.CreatedAt, &notif.UpdatedAt)
	return err
}

func (r *NotificationRepository) GetNotificationsByUserID(userID string, limit int, offset int) ([]models.Notification, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, preference_id, booking_id, type, title, message,
		 read, read_at, created_at, updated_at
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []models.Notification
	for rows.Next() {
		var notif models.Notification
		err := rows.Scan(
			&notif.ID, &notif.UserID, &notif.PreferenceID, &notif.BookingID,
			&notif.Type, &notif.Title, &notif.Message, &notif.Read, &notif.ReadAt,
			&notif.CreatedAt, &notif.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notifs = append(notifs, notif)
	}
	return notifs, rows.Err()
}

func (r *NotificationRepository) GetUnreadCount(userID string) (int, error) {
	var count int
	err := r.db.QueryRow(
		"SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read = false",
		userID,
	).Scan(&count)
	return count, err
}

func (r *NotificationRepository) MarkAsRead(notificationID string) error {
	now := time.Now()
	_, err := r.db.Exec(
		"UPDATE notifications SET read = true, read_at = $1, updated_at = $1 WHERE id = $2",
		now, notificationID,
	)
	return err
}

func (r *NotificationRepository) MarkAllAsRead(userID string) error {
	now := time.Now()
	_, err := r.db.Exec(
		"UPDATE notifications SET read = true, read_at = $1, updated_at = $1 WHERE user_id = $2 AND read = false",
		now, userID,
	)
	return err
}

package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/lib/pq"

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

func (r *UserRepository) GetUserByID(id string) (*models.User, error) {
	user := &models.User{}
	err := r.db.QueryRow(
		"SELECT id, email, password, created_at, updated_at FROM users WHERE id = $1",
		id,
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
		 day_preference, party_size, auto_book, notify_only, active, guest_name, guest_email, guest_phone, special_notes,
		 recreation_gov_username, recreation_gov_password, recreation_gov_oauth_token, recreation_gov_oauth_provider,
		 recreation_gov_oauth_refresh, recreation_gov_oauth_expiry)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id, created_at, updated_at`,
		pref.UserID, pref.GoogleLink, pref.RestaurantName, pref.DateRangeFrom, pref.DateRangeTo,
		pq.Array(pref.DayPreference), pref.PartySize, pref.AutoBook, pref.NotifyOnly, pref.Active,
		pref.GuestName, pref.GuestEmail, pref.GuestPhone, pref.SpecialNotes,
		pref.RecreationGovUsername, pref.RecreationGovPassword, pref.RecreationGovOAuthToken,
		pref.RecreationGovOAuthProvider, pref.RecreationGovOAuthRefresh, pref.RecreationGovOAuthExpiry,
	).Scan(&pref.ID, &pref.CreatedAt, &pref.UpdatedAt)
	return err
}

func (r *PreferenceRepository) GetPreferencesByUserID(userID string) ([]models.UserPreference, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, google_link, restaurant_name, date_range_from, date_range_to,
		 day_preference, party_size, auto_book, notify_only, active, guest_name, guest_email, guest_phone,
		 special_notes, recreation_gov_username, recreation_gov_password, recreation_gov_oauth_token,
		 recreation_gov_oauth_provider, recreation_gov_oauth_refresh, recreation_gov_oauth_expiry,
		 last_checked_at, last_booked_at, created_at, updated_at
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
			&pref.DateRangeFrom, &pref.DateRangeTo, pq.Array(&pref.DayPreference),
			&pref.PartySize, &pref.AutoBook, &pref.NotifyOnly, &pref.Active, &pref.GuestName, &pref.GuestEmail,
			&pref.GuestPhone, &pref.SpecialNotes, &pref.RecreationGovUsername, &pref.RecreationGovPassword,
			&pref.RecreationGovOAuthToken, &pref.RecreationGovOAuthProvider, &pref.RecreationGovOAuthRefresh,
			&pref.RecreationGovOAuthExpiry, &pref.LastCheckedAt, &pref.LastBookedAt, &pref.CreatedAt, &pref.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		// Don't return sensitive data in API responses
		pref.RecreationGovPassword = ""
		pref.RecreationGovOAuthToken = ""
		pref.RecreationGovOAuthRefresh = ""
		prefs = append(prefs, pref)
	}
	return prefs, rows.Err()
}

func (r *PreferenceRepository) GetActivePreferences() ([]models.UserPreference, error) {
	rows, err := r.db.Query(
		`SELECT id, user_id, google_link, restaurant_name, date_range_from, date_range_to,
		 day_preference, party_size, auto_book, notify_only, active, guest_name, guest_email, guest_phone,
		 special_notes, recreation_gov_username, recreation_gov_password, recreation_gov_oauth_token,
		 recreation_gov_oauth_provider, recreation_gov_oauth_refresh, recreation_gov_oauth_expiry,
		 last_checked_at, last_booked_at, created_at, updated_at
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
			&pref.DateRangeFrom, &pref.DateRangeTo, pq.Array(&pref.DayPreference),
			&pref.PartySize, &pref.AutoBook, &pref.NotifyOnly, &pref.Active, &pref.GuestName, &pref.GuestEmail,
			&pref.GuestPhone, &pref.SpecialNotes, &pref.RecreationGovUsername, &pref.RecreationGovPassword,
			&pref.RecreationGovOAuthToken, &pref.RecreationGovOAuthProvider, &pref.RecreationGovOAuthRefresh,
			&pref.RecreationGovOAuthExpiry, &pref.LastCheckedAt, &pref.LastBookedAt, &pref.CreatedAt, &pref.UpdatedAt,
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

// UpdateRecreationGovCredentials stores (already encrypted) username/password for a preference.
// Scoped by user ID so one user cannot overwrite another user's preference.
func (r *PreferenceRepository) UpdateRecreationGovCredentials(preferenceID, userID, username, encryptedPassword string) error {
	res, err := r.db.Exec(
		`UPDATE user_preferences
		 SET recreation_gov_username = $1, recreation_gov_password = $2, updated_at = $3
		 WHERE id = $4 AND user_id = $5`,
		username, encryptedPassword, time.Now(), preferenceID, userID,
	)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// UpdateRecreationGovOAuth stores (already encrypted) OAuth token details for a preference.
func (r *PreferenceRepository) UpdateRecreationGovOAuth(preferenceID, userID, encryptedToken, provider, encryptedRefresh string, expiry *time.Time) error {
	res, err := r.db.Exec(
		`UPDATE user_preferences
		 SET recreation_gov_oauth_token = $1, recreation_gov_oauth_provider = $2,
		     recreation_gov_oauth_refresh = $3, recreation_gov_oauth_expiry = $4, updated_at = $5
		 WHERE id = $6 AND user_id = $7`,
		encryptedToken, provider, encryptedRefresh, expiry, time.Now(), preferenceID, userID,
	)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// DeactivatePreference stops monitoring a preference (e.g. after a successful booking)
func (r *PreferenceRepository) DeactivatePreference(preferenceID string) error {
	_, err := r.db.Exec(
		"UPDATE user_preferences SET active = false, last_booked_at = $1, updated_at = $1 WHERE id = $2",
		time.Now(), preferenceID,
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

// nullIfEmpty converts an empty string to NULL for nullable UUID columns
func nullIfEmpty(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func (r *NotificationRepository) CreateNotification(notif *models.Notification) error {
	var dataJSON sql.NullString
	if notif.Data != nil {
		if b, err := json.Marshal(notif.Data); err == nil {
			dataJSON = sql.NullString{String: string(b), Valid: true}
		}
	}

	err := r.db.QueryRow(
		`INSERT INTO notifications
		(user_id, preference_id, booking_id, type, title, message, data)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at`,
		notif.UserID, nullIfEmpty(notif.PreferenceID), nullIfEmpty(notif.BookingID),
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
		var prefID, bookingID sql.NullString
		err := rows.Scan(
			&notif.ID, &notif.UserID, &prefID, &bookingID,
			&notif.Type, &notif.Title, &notif.Message, &notif.Read, &notif.ReadAt,
			&notif.CreatedAt, &notif.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		notif.PreferenceID = prefID.String
		notif.BookingID = bookingID.String
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

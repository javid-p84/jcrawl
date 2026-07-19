package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"database/sql"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/crypto"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	userRepo  *db.UserRepository
	prefRepo  *db.PreferenceRepository
	bookRepo  *db.BookingRepository
	notifRepo *db.NotificationRepository
	crypto    *crypto.Manager
}

func NewHandler(userRepo *db.UserRepository, prefRepo *db.PreferenceRepository, bookRepo *db.BookingRepository, notifRepo *db.NotificationRepository, cryptoMgr *crypto.Manager) *Handler {
	return &Handler{
		userRepo:  userRepo,
		prefRepo:  prefRepo,
		bookRepo:  bookRepo,
		notifRepo: notifRepo,
		crypto:    cryptoMgr,
	}
}

// parseDate accepts both date-only (2006-01-02) and RFC3339 formats
func parseDate(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

// RegisterRequest defines the registration payload
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest defines the login payload
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Register creates a new user account
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	user := &models.User{
		Email:    req.Email,
		Password: string(hashedPwd),
	}

	if err := h.userRepo.CreateUser(user); err != nil {
		http.Error(w, "Email already exists", http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": user.ID, "email": user.Email})
}

// Login authenticates a user
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// TODO: Generate JWT token and return it
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Login successful", "id": user.ID})
}

// CreatePreferenceRequest accepts human-friendly date strings (YYYY-MM-DD or RFC3339)
type CreatePreferenceRequest struct {
	GoogleLink     string `json:"google_link"`
	RestaurantName string `json:"restaurant_name"`
	DateRangeFrom  string `json:"date_range_from"`
	DateRangeTo    string `json:"date_range_to"`
	DayPreference  []int  `json:"day_preference"`
	PartySize      int    `json:"party_size"`
	AutoBook       *bool  `json:"auto_book"`
	NotifyOnly     bool   `json:"notify_only"`
	GuestName      string `json:"guest_name"`
	GuestEmail     string `json:"guest_email"`
	GuestPhone     string `json:"guest_phone"`
	SpecialNotes   string `json:"special_notes"`
}

// CreatePreference creates a new monitoring preference
func (h *Handler) CreatePreference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token in Authorization header
	userID := r.Header.Get("X-User-ID") // Placeholder
	if userID == "" {
		http.Error(w, "X-User-ID header required", http.StatusUnauthorized)
		return
	}

	var req CreatePreferenceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.GoogleLink == "" {
		http.Error(w, "google_link is required", http.StatusBadRequest)
		return
	}

	dateFrom, err := parseDate(req.DateRangeFrom)
	if err != nil {
		http.Error(w, "date_range_from must be YYYY-MM-DD or RFC3339", http.StatusBadRequest)
		return
	}
	dateTo, err := parseDate(req.DateRangeTo)
	if err != nil {
		http.Error(w, "date_range_to must be YYYY-MM-DD or RFC3339", http.StatusBadRequest)
		return
	}
	if dateTo.Before(dateFrom) {
		http.Error(w, "date_range_to must not be before date_range_from", http.StatusBadRequest)
		return
	}

	autoBook := true
	if req.AutoBook != nil {
		autoBook = *req.AutoBook
	}
	if req.NotifyOnly {
		autoBook = false
	}

	pref := models.UserPreference{
		UserID:         userID,
		GoogleLink:     req.GoogleLink,
		RestaurantName: req.RestaurantName,
		DateRangeFrom:  dateFrom,
		DateRangeTo:    dateTo,
		DayPreference:  req.DayPreference,
		PartySize:      req.PartySize,
		AutoBook:       autoBook,
		NotifyOnly:     req.NotifyOnly,
		Active:         true,
		GuestName:      req.GuestName,
		GuestEmail:     req.GuestEmail,
		GuestPhone:     req.GuestPhone,
		SpecialNotes:   req.SpecialNotes,
	}

	if err := h.prefRepo.CreatePreference(&pref); err != nil {
		http.Error(w, "Failed to create preference", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(pref)
}

// GetPreferences retrieves all preferences for a user
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.Header.Get("X-User-ID") // Placeholder

	prefs, err := h.prefRepo.GetPreferencesByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to fetch preferences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(prefs)
}

// GetBookings retrieves all bookings for a user
func (h *Handler) GetBookings(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.Header.Get("X-User-ID") // Placeholder

	bookings, err := h.bookRepo.GetBookingsByUserID(userID)
	if err != nil {
		http.Error(w, "Failed to fetch bookings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookings)
}

// GetNotifications retrieves notifications for a user
func (h *Handler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.Header.Get("X-User-ID") // Placeholder

	// Get query parameters for pagination
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsedL, err := strconv.Atoi(l); err == nil && parsedL > 0 && parsedL <= 100 {
			limit = parsedL
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsedO, err := strconv.Atoi(o); err == nil && parsedO >= 0 {
			offset = parsedO
		}
	}

	notifs, err := h.notifRepo.GetNotificationsByUserID(userID, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifs)
}

// GetUnreadNotificationCount returns count of unread notifications
func (h *Handler) GetUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.Header.Get("X-User-ID") // Placeholder

	count, err := h.notifRepo.GetUnreadCount(userID)
	if err != nil {
		http.Error(w, "Failed to fetch unread count", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"unread_count": count})
}

// MarkNotificationAsRead marks a notification as read
func (h *Handler) MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	notifID := r.URL.Query().Get("id")
	if notifID == "" {
		http.Error(w, "Notification ID required", http.StatusBadRequest)
		return
	}

	if err := h.notifRepo.MarkAsRead(notifID); err != nil {
		http.Error(w, "Failed to update notification", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// MarkAllNotificationsAsRead marks all notifications as read
func (h *Handler) MarkAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.Header.Get("X-User-ID") // Placeholder

	if err := h.notifRepo.MarkAllAsRead(userID); err != nil {
		http.Error(w, "Failed to update notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// UpdateRecreationGovCredentials updates recreation.gov login credentials (Option 1: password)
func (h *Handler) UpdateRecreationGovCredentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.Header.Get("X-User-ID") // Placeholder
	prefID := r.URL.Query().Get("preference_id")

	if userID == "" {
		http.Error(w, "X-User-ID header required", http.StatusUnauthorized)
		return
	}
	if prefID == "" {
		http.Error(w, "Preference ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Password == "" {
		http.Error(w, "Username and password required", http.StatusBadRequest)
		return
	}

	encryptedPassword, err := h.crypto.Encrypt(req.Password)
	if err != nil {
		http.Error(w, "Failed to secure credentials", http.StatusInternalServerError)
		return
	}

	if err := h.prefRepo.UpdateRecreationGovCredentials(prefID, userID, req.Username, encryptedPassword); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Preference not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to store credentials", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Credentials stored (encrypted). Please ensure your recreation.gov username and password are correct.",
	})
}

// UpdateRecreationGovOAuthToken updates recreation.gov OAuth token (Option 2: token)
func (h *Handler) UpdateRecreationGovOAuthToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token
	userID := r.Header.Get("X-User-ID") // Placeholder
	prefID := r.URL.Query().Get("preference_id")

	if userID == "" {
		http.Error(w, "X-User-ID header required", http.StatusUnauthorized)
		return
	}
	if prefID == "" {
		http.Error(w, "Preference ID required", http.StatusBadRequest)
		return
	}

	var req struct {
		OAuthToken    string `json:"oauth_token"`
		OAuthProvider string `json:"oauth_provider"` // google, facebook, recreation.gov
		OAuthRefresh  string `json:"oauth_refresh,omitempty"`
		OAuthExpiry   string `json:"oauth_expiry,omitempty"` // RFC3339
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.OAuthToken == "" {
		http.Error(w, "OAuth token required", http.StatusBadRequest)
		return
	}

	if req.OAuthProvider == "" {
		req.OAuthProvider = "recreation.gov"
	}

	var expiry *time.Time
	if req.OAuthExpiry != "" {
		t, err := time.Parse(time.RFC3339, req.OAuthExpiry)
		if err != nil {
			http.Error(w, "oauth_expiry must be RFC3339", http.StatusBadRequest)
			return
		}
		expiry = &t
	}

	encryptedToken, err := h.crypto.Encrypt(req.OAuthToken)
	if err != nil {
		http.Error(w, "Failed to secure token", http.StatusInternalServerError)
		return
	}

	encryptedRefresh := ""
	if req.OAuthRefresh != "" {
		encryptedRefresh, err = h.crypto.Encrypt(req.OAuthRefresh)
		if err != nil {
			http.Error(w, "Failed to secure refresh token", http.StatusInternalServerError)
			return
		}
	}

	if err := h.prefRepo.UpdateRecreationGovOAuth(prefID, userID, encryptedToken, req.OAuthProvider, encryptedRefresh, expiry); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Preference not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to store token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": fmt.Sprintf("OAuth token stored (encrypted) for provider: %s", req.OAuthProvider),
	})
}

// Health check endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now(),
	})
}

package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jaavvviiiiddddd/jcrawl/pkg/db"
	"github.com/jaavvviiiiddddd/jcrawl/pkg/models"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	userRepo  *db.UserRepository
	prefRepo  *db.PreferenceRepository
	bookRepo  *db.BookingRepository
}

func NewHandler(userRepo *db.UserRepository, prefRepo *db.PreferenceRepository, bookRepo *db.BookingRepository) *Handler {
	return &Handler{
		userRepo: userRepo,
		prefRepo: prefRepo,
		bookRepo: bookRepo,
	}
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

// CreatePreference creates a new monitoring preference
func (h *Handler) CreatePreference(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Extract user ID from JWT token in Authorization header
	userID := r.Header.Get("X-User-ID") // Placeholder

	var pref models.UserPreference
	if err := json.NewDecoder(r.Body).Decode(&pref); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	pref.UserID = userID
	pref.Active = true
	pref.AutoBook = true

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

// Health check endpoint
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"time":   time.Now(),
	})
}

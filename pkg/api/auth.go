package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDContextKey contextKey = "userID"

// AuthService issues and validates JWTs for API authentication
type AuthService struct {
	secret []byte
	ttl    time.Duration
}

// NewAuthService reads JWT_SECRET from the environment. If unset, a random
// ephemeral secret is generated: the server still works, but all tokens
// become invalid on restart.
func NewAuthService() *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			log.Fatalf("Failed to generate ephemeral JWT secret: %v", err)
		}
		secret = hex.EncodeToString(buf)
		log.Println("WARNING: JWT_SECRET not set; using a random ephemeral secret. All tokens become invalid when the server restarts.")
	}
	return &AuthService{
		secret: []byte(secret),
		ttl:    24 * time.Hour,
	}
}

// GenerateToken creates a signed JWT for the given user ID
func (a *AuthService) GenerateToken(userID string) (string, time.Time, error) {
	expiresAt := time.Now().Add(a.ttl)
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    "jcrawl",
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
	return signed, expiresAt, err
}

// ValidateToken verifies a JWT and returns the user ID it was issued for
func (a *AuthService) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return a.secret, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid || claims.Subject == "" {
		return "", fmt.Errorf("invalid token")
	}
	return claims.Subject, nil
}

// Middleware rejects requests without a valid Bearer token and stores the
// authenticated user ID in the request context.
func (a *AuthService) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Authorization: Bearer <token> header required", http.StatusUnauthorized)
			return
		}
		userID, err := a.ValidateToken(strings.TrimPrefix(authHeader, "Bearer "))
		if err != nil {
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDContextKey, userID)))
	})
}

// UserIDFromContext returns the authenticated user ID set by Middleware,
// or "" if the request was not authenticated.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDContextKey).(string)
	return id
}

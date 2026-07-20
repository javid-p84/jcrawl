package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func newTestAuth(t *testing.T) *AuthService {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret")
	defer os.Unsetenv("JWT_SECRET")
	return NewAuthService()
}

func TestTokenRoundTrip(t *testing.T) {
	auth := newTestAuth(t)

	token, expiresAt, err := auth.GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if !expiresAt.After(time.Now()) {
		t.Error("expiry should be in the future")
	}

	userID, err := auth.ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("got user %q, want user-123", userID)
	}
}

func TestValidateRejectsBadTokens(t *testing.T) {
	auth := newTestAuth(t)

	if _, err := auth.ValidateToken("not-a-jwt"); err == nil {
		t.Error("expected error for malformed token")
	}

	// Token signed with a different secret must be rejected
	os.Setenv("JWT_SECRET", "other-secret")
	otherAuth := NewAuthService()
	os.Unsetenv("JWT_SECRET")
	token, _, err := otherAuth.GenerateToken("user-123")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if _, err := auth.ValidateToken(token); err == nil {
		t.Error("expected error for token signed with wrong secret")
	}
}

func TestMiddleware(t *testing.T) {
	auth := newTestAuth(t)

	var gotUserID string
	handler := auth.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// No token -> 401
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("no token: got status %d, want 401", rec.Code)
	}

	// Garbage token -> 401
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer garbage")
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("bad token: got status %d, want 401", rec.Code)
	}

	// Valid token -> 200 with user ID in context
	token, _, err := auth.GenerateToken("user-456")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("valid token: got status %d, want 200", rec.Code)
	}
	if gotUserID != "user-456" {
		t.Errorf("context user: got %q, want user-456", gotUserID)
	}
}

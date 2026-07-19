package recreation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type OAuthManager struct {
	client *http.Client
	token  string
	expiry *time.Time
}

// NewOAuthManager creates a new OAuth token manager
func NewOAuthManager(token string) *OAuthManager {
	return &OAuthManager{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		token: token,
	}
}

// SetExpiry sets the token expiration time
func (om *OAuthManager) SetExpiry(expiry *time.Time) {
	om.expiry = expiry
}

// IsExpired checks if the token has expired
func (om *OAuthManager) IsExpired() bool {
	if om.expiry == nil {
		return false // No expiry set, assume valid
	}
	return time.Now().After(*om.expiry)
}

// ValidateToken validates the OAuth token with recreation.gov
func (om *OAuthManager) ValidateToken(ctx context.Context) (bool, error) {
	if om.token == "" {
		return false, fmt.Errorf("no token provided")
	}

	// Check expiry first
	if om.IsExpired() {
		log.Println("Token has expired")
		return false, fmt.Errorf("token expired")
	}

	// Make a test request to validate token
	url := "https://www.recreation.gov/api/camps/availability/campgrounds/1/month?start_date=2024-01-01T00:00:00.000Z"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	// Add token to Authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", om.token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := om.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("validation request failed: %w", err)
	}
	defer resp.Body.Close()

	// Token is valid if we get a successful response
	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	// Token is invalid if we get 401 or 403
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return false, fmt.Errorf("token invalid or expired (status: %d)", resp.StatusCode)
	}

	return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
}

// GetAuthorizedClient returns an HTTP client with token authentication
func (om *OAuthManager) GetAuthorizedClient() *http.Client {
	// Create a new client with a wrapper transport that adds the token
	transport := &TokenTransport{
		token:     om.token,
		transport: http.DefaultTransport,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
	}
}

// TokenTransport adds OAuth token to all requests
type TokenTransport struct {
	token     string
	transport http.RoundTripper
}

func (t *TokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add authorization header
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", t.token))
	return t.transport.RoundTrip(req)
}

// MakeAuthenticatedRequest makes an HTTP request with token authentication
func (om *OAuthManager) MakeAuthenticatedRequest(ctx context.Context, method, url string, body []byte) ([]byte, error) {
	if om.IsExpired() {
		return nil, fmt.Errorf("token has expired")
	}

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", om.token))
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := om.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// GetMeInfo fetches the current user's profile information using OAuth token
func (om *OAuthManager) GetMeInfo(ctx context.Context) (map[string]interface{}, error) {
	respBody, err := om.MakeAuthenticatedRequest(ctx, "GET", "https://www.recreation.gov/api/user", nil)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("failed to parse user info: %w", err)
	}

	return data, nil
}

// OAuthProvider represents different OAuth providers
type OAuthProvider string

const (
	OAuthProviderGoogle   OAuthProvider = "google"
	OAuthProviderFacebook OAuthProvider = "facebook"
	OAuthProviderRecGov   OAuthProvider = "recreation.gov"
)

// ExchangeCodeForToken exchanges an authorization code for an access token
// This is used when implementing full OAuth flow
func ExchangeCodeForToken(ctx context.Context, provider OAuthProvider, code string, clientID string, clientSecret string) (string, *time.Time, error) {
	// Recreation.gov OAuth token exchange would happen here
	// For now, this is a placeholder for future OAuth implementation
	log.Printf("Token exchange not yet fully implemented for provider: %s\n", provider)
	return "", nil, fmt.Errorf("token exchange not implemented for this provider")
}

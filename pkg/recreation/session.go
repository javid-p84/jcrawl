package recreation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/chromedp/chromedp"
)

type SessionManager struct {
	client     *http.Client
	username   string
	password   string
	oauthToken string
	oauthMgr   *OAuthManager
	authMethod AuthMethod
	isLoggedIn bool
}

type AuthMethod string

const (
	AuthMethodPassword AuthMethod = "password"
	AuthMethodOAuth    AuthMethod = "oauth"
)

// NewSessionManager creates a new recreation.gov session manager with password auth
func NewSessionManager(username, password string) *SessionManager {
	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
	}

	return &SessionManager{
		client:     client,
		username:   username,
		password:   password,
		authMethod: AuthMethodPassword,
		isLoggedIn: false,
	}
}

// NewOAuthSessionManager creates a new recreation.gov session manager with OAuth token
func NewOAuthSessionManager(token string) *SessionManager {
	oauthMgr := NewOAuthManager(token)

	return &SessionManager{
		client:     oauthMgr.client,
		oauthToken: token,
		oauthMgr:   oauthMgr,
		authMethod: AuthMethodOAuth,
		isLoggedIn: false,
	}
}

// Login authenticates with recreation.gov (password or OAuth)
// Returns error if login fails
func (sm *SessionManager) Login(ctx context.Context) error {
	if sm.authMethod == AuthMethodOAuth {
		return sm.LoginWithOAuth(ctx)
	}
	return sm.LoginWithPassword(ctx)
}

// LoginWithPassword authenticates using username/password
func (sm *SessionManager) LoginWithPassword(ctx context.Context) error {
	log.Printf("Logging into recreation.gov as %s\n", sm.username)

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	loginCtx, cancel := context.WithTimeout(browserCtx, 30*time.Second)
	defer cancel()

	var loginSuccess bool

	err := chromedp.Run(loginCtx,
		// Navigate to login page
		chromedp.Navigate("https://www.recreation.gov/auth/login"),
		chromedp.WaitVisible("form", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),

		// Fill in email/username
		chromedp.SetValue("input[type='email']", sm.username, chromedp.ByQuery),

		// Fill in password
		chromedp.SetValue("input[type='password']", sm.password, chromedp.ByQuery),

		// Click login button
		chromedp.Click("button[type='submit']", chromedp.ByQuery),

		// Wait for either success or error
		chromedp.Sleep(3*time.Second),

		// Check if we're logged in by checking for dashboard or error message
		chromedp.Evaluate(`window.location.pathname.includes('/account') || document.body.innerText.includes('Account')`, &loginSuccess),
	)

	if err != nil {
		return fmt.Errorf("login navigation failed: %w", err)
	}

	if !loginSuccess {
		return fmt.Errorf("login failed: invalid credentials or login page not loaded")
	}

	sm.isLoggedIn = true
	log.Println("Successfully logged into recreation.gov")
	return nil
}

// APILogin attempts to login via recreation.gov API (alternative method)
func (sm *SessionManager) APILogin(ctx context.Context) error {
	log.Printf("Attempting API login to recreation.gov as %s\n", sm.username)

	loginURL := "https://www.recreation.gov/auth/login"

	payload := map[string]string{
		"email":    sm.username,
		"password": sm.password,
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", loginURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := sm.client.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("login failed with status %d", resp.StatusCode)
	}

	var loginResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Check if login was successful
	if _, ok := loginResp["error"]; ok {
		return fmt.Errorf("login error: %v", loginResp["error"])
	}

	sm.isLoggedIn = true
	log.Println("Successfully authenticated via API")
	return nil
}

// IsLoggedIn returns whether the session is authenticated
func (sm *SessionManager) IsLoggedIn() bool {
	return sm.isLoggedIn
}

// GetClient returns the HTTP client with session cookies
func (sm *SessionManager) GetClient() *http.Client {
	return sm.client
}

// ValidateCredentials checks if the credentials are valid without full login
func (sm *SessionManager) ValidateCredentials(ctx context.Context) error {
	if sm.username == "" || sm.password == "" {
		return fmt.Errorf("missing username or password")
	}

	if len(sm.username) < 3 {
		return fmt.Errorf("username too short")
	}

	if len(sm.password) < 6 {
		return fmt.Errorf("password too short")
	}

	return nil
}

// LoginWithOAuth authenticates using OAuth token
func (sm *SessionManager) LoginWithOAuth(ctx context.Context) error {
	if sm.oauthMgr == nil {
		return fmt.Errorf("oauth manager not initialized")
	}

	log.Println("Validating OAuth token...")

	isValid, err := sm.oauthMgr.ValidateToken(ctx)
	if err != nil {
		return fmt.Errorf("oauth token validation failed: %w", err)
	}

	if !isValid {
		return fmt.Errorf("oauth token is invalid")
	}

	sm.isLoggedIn = true
	log.Println("Successfully authenticated with OAuth token")
	return nil
}

// GetOAuthManager returns the OAuth manager for token-based operations
func (sm *SessionManager) GetOAuthManager() *OAuthManager {
	return sm.oauthMgr
}

// IsUsingOAuth returns whether this session is using OAuth authentication
func (sm *SessionManager) IsUsingOAuth() bool {
	return sm.authMethod == AuthMethodOAuth
}

// Logout clears the session
func (sm *SessionManager) Logout() {
	sm.isLoggedIn = false
	// cookiejar has no way to clear cookies, so replace the jar entirely
	if sm.authMethod == AuthMethodPassword {
		jar, _ := cookiejar.New(nil)
		sm.client.Jar = jar
	}
	log.Println("Logged out from recreation.gov")
}

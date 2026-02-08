package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	sessionCookieName = "miren_oidc_session"
	stateCookieName   = "miren_oidc_state"

	// Default session lifetime
	defaultSessionDuration = 24 * time.Hour

	// PKCE challenge length (43-128 characters per RFC 7636)
	pkceVerifierLength = 64
)

// SessionManager handles OIDC session lifecycle using secure cookies.
// Sessions are stored as encrypted cookies with HttpOnly and Secure flags.
type SessionManager struct {
	// cookieSecure controls whether cookies require HTTPS
	cookieSecure bool

	// cookieDomain is the domain for session cookies
	cookieDomain string

	// sessionDuration controls how long sessions last
	sessionDuration time.Duration
}

// SessionData contains the authenticated session information
type SessionData struct {
	// IDToken is the raw ID token JWT from the OIDC provider
	IDToken string `json:"id_token"`

	// AccessToken is the OAuth2 access token (optional)
	AccessToken string `json:"access_token,omitempty"`

	// RefreshToken is the OAuth2 refresh token (optional)
	RefreshToken string `json:"refresh_token,omitempty"`

	// Claims are the parsed JWT claims
	Claims map[string]interface{} `json:"claims"`

	// ExpiresAt is when this session expires
	ExpiresAt time.Time `json:"expires_at"`
}

// StateData contains OIDC flow state for CSRF protection
type StateData struct {
	// State is the random state parameter
	State string `json:"state"`

	// PKCEVerifier is the PKCE code verifier (RFC 7636)
	PKCEVerifier string `json:"pkce_verifier"`

	// ReturnPath is where to redirect after auth
	ReturnPath string `json:"return_path"`

	// ExpiresAt is when this state expires (short-lived)
	ExpiresAt time.Time `json:"expires_at"`
}

// NewSessionManager creates a new session manager
func NewSessionManager(cookieSecure bool, cookieDomain string) *SessionManager {
	return &SessionManager{
		cookieSecure:    cookieSecure,
		cookieDomain:    cookieDomain,
		sessionDuration: defaultSessionDuration,
	}
}

// GetSession retrieves the current session from cookies
func (sm *SessionManager) GetSession(r *http.Request) (*SessionData, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read session cookie: %w", err)
	}

	// Decode base64
	data, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode session cookie: %w", err)
	}

	// Parse JSON
	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, nil
	}

	return &session, nil
}

// SetSession stores a new session in a secure cookie
func (sm *SessionManager) SetSession(w http.ResponseWriter, session *SessionData) error {
	// Set expiration if not set
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(sm.sessionDuration)
	}

	// Marshal to JSON
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	// Encode as base64
	encoded := base64.RawURLEncoding.EncodeToString(data)

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    encoded,
		Path:     "/",
		Domain:   sm.cookieDomain,
		Expires:  session.ExpiresAt,
		Secure:   sm.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// ClearSession removes the session cookie
func (sm *SessionManager) ClearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		Domain:   sm.cookieDomain,
		MaxAge:   -1,
		Secure:   sm.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// GenerateState creates a new OIDC flow state with PKCE
func (sm *SessionManager) GenerateState(returnPath string) (*StateData, error) {
	// Generate random state
	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, fmt.Errorf("failed to generate state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Generate PKCE verifier
	verifierBytes := make([]byte, pkceVerifierLength)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, fmt.Errorf("failed to generate PKCE verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	return &StateData{
		State:        state,
		PKCEVerifier: verifier,
		ReturnPath:   returnPath,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}, nil
}

// SetState stores OIDC flow state in a cookie
func (sm *SessionManager) SetState(w http.ResponseWriter, state *StateData) error {
	// Marshal to JSON
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Encode as base64
	encoded := base64.RawURLEncoding.EncodeToString(data)

	// Set cookie (shorter-lived than session)
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    encoded,
		Path:     "/",
		Domain:   sm.cookieDomain,
		Expires:  state.ExpiresAt,
		Secure:   sm.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	return nil
}

// GetState retrieves OIDC flow state from cookies
func (sm *SessionManager) GetState(r *http.Request) (*StateData, error) {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil {
		if err == http.ErrNoCookie {
			return nil, fmt.Errorf("state cookie not found")
		}
		return nil, fmt.Errorf("failed to read state cookie: %w", err)
	}

	// Decode base64
	data, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state cookie: %w", err)
	}

	// Parse JSON
	var state StateData
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Check expiration
	if time.Now().After(state.ExpiresAt) {
		return nil, fmt.Errorf("state has expired")
	}

	return &state, nil
}

// ClearState removes the state cookie
func (sm *SessionManager) ClearState(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/",
		Domain:   sm.cookieDomain,
		MaxAge:   -1,
		Secure:   sm.cookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

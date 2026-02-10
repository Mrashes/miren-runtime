package oidc

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
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

// SessionManager handles OIDC session lifecycle using HMAC-signed cookies.
type SessionManager struct {
	cookieSecure    bool
	cookieDomain    string
	sessionDuration time.Duration
	signingKey      []byte
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

// SetSecure updates whether cookies should be marked Secure.
func (sm *SessionManager) SetSecure(secure bool) {
	sm.cookieSecure = secure
}

// NewSessionManager creates a new session manager.
// If signingKey is nil, a random 32-byte key is generated. This means sessions
// won't survive server restarts; pass a persistent key for durable sessions.
func NewSessionManager(cookieSecure bool, cookieDomain string, signingKey []byte) *SessionManager {
	if signingKey == nil {
		signingKey = make([]byte, 32)
		rand.Read(signingKey)
	}
	return &SessionManager{
		cookieSecure:    cookieSecure,
		cookieDomain:    cookieDomain,
		sessionDuration: defaultSessionDuration,
		signingKey:      signingKey,
	}
}

// signCookie produces "base64(json).base64(hmac-sha256)" from data.
func (sm *SessionManager) signCookie(data []byte) string {
	encoded := base64.RawURLEncoding.EncodeToString(data)
	mac := hmac.New(sha256.New, sm.signingKey)
	mac.Write([]byte(encoded))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return encoded + "." + sig
}

// verifyCookie splits "payload.sig", verifies the HMAC, and returns the decoded payload.
func (sm *SessionManager) verifyCookie(value string) ([]byte, error) {
	parts := splitCookieValue(value)
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed signed cookie")
	}

	mac := hmac.New(sha256.New, sm.signingKey)
	mac.Write([]byte(parts[0]))
	expectedSig := mac.Sum(nil)

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	if !hmac.Equal(sig, expectedSig) {
		return nil, fmt.Errorf("cookie signature verification failed")
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("failed to decode cookie payload: %w", err)
	}

	return data, nil
}

// splitCookieValue splits on the last '.' to separate payload from signature.
func splitCookieValue(s string) []string {
	idx := len(s) - 1
	for idx >= 0 && s[idx] != '.' {
		idx--
	}
	if idx <= 0 {
		return nil
	}
	return []string{s[:idx], s[idx+1:]}
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

	data, err := sm.verifyCookie(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid session cookie: %w", err)
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, nil
	}

	return &session, nil
}

// SetSession stores a new session in an HMAC-signed cookie
func (sm *SessionManager) SetSession(w http.ResponseWriter, session *SessionData) error {
	if session.ExpiresAt.IsZero() {
		session.ExpiresAt = time.Now().Add(sm.sessionDuration)
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    sm.signCookie(data),
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

// SetState stores OIDC flow state in an HMAC-signed cookie
func (sm *SessionManager) SetState(w http.ResponseWriter, state *StateData) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    sm.signCookie(data),
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

	data, err := sm.verifyCookie(cookie.Value)
	if err != nil {
		return nil, fmt.Errorf("invalid state cookie: %w", err)
	}

	var state StateData
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

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

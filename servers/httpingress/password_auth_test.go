package httpingress

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/ingress/ingress_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/oidc"
	"miren.dev/runtime/pkg/rpc"
	"miren.dev/runtime/servers/entityserver"
)

func storePasswordProvider(store *entity.MockStore, ident, hash string) {
	store.AddEntity(entity.Id(ident), entity.New([]entity.Attr{
		{ID: entity.Ident, Value: entity.KeywordValue(ident)},
		entity.String(ingress_v1alpha.PasswordProviderPasswordHashId, hash),
	}))
}

func newPasswordTestServer(store *entity.MockStore) *Server {
	esrv := &entityserver.EntityServer{
		Log:   slog.Default(),
		Store: store,
	}
	eac := &entityserver_v1alpha.EntityAccessClient{
		Client: rpc.LocalClient(entityserver_v1alpha.AdaptEntityAccess(esrv)),
	}

	signingKey := make([]byte, 32)

	return &Server{
		Log:                slog.Default(),
		eac:                eac,
		oidcSessionManager: oidc.NewSessionManager(false, "", signingKey),
		oidcHandlers:       make(map[string]*oidcHandler),
		passwordHandlers:   make(map[string]*passwordHandler),
	}
}

func TestPasswordProviderMatches(t *testing.T) {
	base := &ingress_v1alpha.PasswordProvider{
		ID:           "provider-1",
		PasswordHash: "$2a$10$test",
	}

	handler := &passwordHandler{provider: base}

	t.Run("identical provider matches", func(t *testing.T) {
		same := &ingress_v1alpha.PasswordProvider{
			ID:           "provider-1",
			PasswordHash: "$2a$10$test",
		}
		if !passwordProviderMatches(handler, same) {
			t.Error("expected match for identical provider")
		}
	})

	t.Run("different hash does not match", func(t *testing.T) {
		different := &ingress_v1alpha.PasswordProvider{
			ID:           "provider-1",
			PasswordHash: "$2a$10$other",
		}
		if passwordProviderMatches(handler, different) {
			t.Error("expected mismatch for different hash")
		}
	})

	t.Run("different ID does not match", func(t *testing.T) {
		different := &ingress_v1alpha.PasswordProvider{
			ID:           "provider-2",
			PasswordHash: "$2a$10$test",
		}
		if passwordProviderMatches(handler, different) {
			t.Error("expected mismatch for different ID")
		}
	})
}

func TestGetOrCreatePasswordHandlerCacheInvalidation(t *testing.T) {
	store := entity.NewMockStore()
	srv := newPasswordTestServer(store)

	providerIdent := "test/pw-provider"
	hash1, _ := bcrypt.GenerateFromPassword([]byte("secret1"), bcrypt.MinCost)
	storePasswordProvider(store, providerIdent, string(hash1))

	route := &ingress_v1alpha.HttpRoute{
		Host:             "app.example.com",
		PasswordProvider: entity.Id(providerIdent),
	}

	h1, err := srv.getOrCreatePasswordHandler(context.Background(), route, "https://app.example.com")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if h1.provider.PasswordHash != string(hash1) {
		t.Fatalf("expected hash=%s, got %s", string(hash1), h1.provider.PasswordHash)
	}

	// Same call should return cached handler
	h2, err := srv.getOrCreatePasswordHandler(context.Background(), route, "https://app.example.com")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if h1 != h2 {
		t.Error("expected same handler instance on cache hit")
	}

	// Update the password hash
	hash2, _ := bcrypt.GenerateFromPassword([]byte("secret2"), bcrypt.MinCost)
	storePasswordProvider(store, providerIdent, string(hash2))

	h3, err := srv.getOrCreatePasswordHandler(context.Background(), route, "https://app.example.com")
	if err != nil {
		t.Fatalf("third call: %v", err)
	}
	if h3.provider.PasswordHash != string(hash2) {
		t.Fatalf("expected updated hash after provider change")
	}
	if h1 == h3 {
		t.Error("expected different handler instance after provider change")
	}
}

func TestGetOrCreatePasswordHandlerFailClosed(t *testing.T) {
	store := entity.NewMockStore()
	srv := newPasswordTestServer(store)

	providerIdent := "test/pw-provider"
	hash, _ := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	storePasswordProvider(store, providerIdent, string(hash))

	route := &ingress_v1alpha.HttpRoute{
		Host:             "app.example.com",
		PasswordProvider: entity.Id(providerIdent),
	}

	// Warm the cache
	h1, err := srv.getOrCreatePasswordHandler(context.Background(), route, "https://app.example.com")
	if err != nil {
		t.Fatalf("initial call: %v", err)
	}

	// Remove provider to simulate unavailability
	store.RemoveEntity(entity.Id(providerIdent))

	// Should return cached handler
	h2, err := srv.getOrCreatePasswordHandler(context.Background(), route, "https://app.example.com")
	if err != nil {
		t.Fatalf("expected cached handler, got error: %v", err)
	}
	if h1 != h2 {
		t.Error("expected same cached handler on entity store failure")
	}

	// No cache + no entity store should error
	routeNew := &ingress_v1alpha.HttpRoute{
		Host:             "new.example.com",
		PasswordProvider: entity.Id(providerIdent),
	}
	_, err = srv.getOrCreatePasswordHandler(context.Background(), routeNew, "https://new.example.com")
	if err == nil {
		t.Error("expected error when entity store fails and no cached handler exists")
	}
}

func TestPasswordSessionRoundtrip(t *testing.T) {
	signingKey := make([]byte, 32)
	sm := oidc.NewSessionManager(false, "", signingKey)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.MinCost)
	handler := &passwordHandler{
		route: &ingress_v1alpha.HttpRoute{
			Host: "app.example.com",
		},
		provider: &ingress_v1alpha.PasswordProvider{
			PasswordHash: string(hash),
		},
		sm:     sm,
		logger: slog.Default(),
	}

	t.Run("no cookie returns unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		if handler.checkSession(req) {
			t.Error("expected unauthenticated with no cookie")
		}
	})

	t.Run("set and check session", func(t *testing.T) {
		w := httptest.NewRecorder()
		if err := handler.setSession(w); err != nil {
			t.Fatalf("setSession: %v", err)
		}

		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == pwSessionCookieName {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("expected session cookie to be set")
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(sessionCookie)
		if !handler.checkSession(req) {
			t.Error("expected authenticated with valid session cookie")
		}
	})

	t.Run("wrong route host is rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		if err := handler.setSession(w); err != nil {
			t.Fatalf("setSession: %v", err)
		}

		cookies := w.Result().Cookies()
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == pwSessionCookieName {
				sessionCookie = c
				break
			}
		}

		otherHandler := &passwordHandler{
			route: &ingress_v1alpha.HttpRoute{
				Host: "other.example.com",
			},
			sm: sm,
		}

		req := httptest.NewRequest("GET", "/", nil)
		req.AddCookie(sessionCookie)
		if otherHandler.checkSession(req) {
			t.Error("expected session to be rejected for wrong route host")
		}
	})
}

func TestPasswordLoginFlow(t *testing.T) {
	signingKey := make([]byte, 32)
	sm := oidc.NewSessionManager(false, "", signingKey)

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctpassword"), bcrypt.MinCost)
	handler := &passwordHandler{
		route: &ingress_v1alpha.HttpRoute{
			Host: "app.example.com",
		},
		provider: &ingress_v1alpha.PasswordProvider{
			PasswordHash: string(hash),
		},
		sm:     sm,
		logger: slog.Default(),
	}

	t.Run("GET login shows form", func(t *testing.T) {
		req := httptest.NewRequest("GET", passwordLoginPath, nil)
		w := httptest.NewRecorder()
		handler.handleLogin(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Password Required") {
			t.Error("expected login form HTML")
		}
	})

	t.Run("wrong password shows error", func(t *testing.T) {
		form := url.Values{"password": {"wrong"}, "return": {"/"}}
		req := httptest.NewRequest("POST", passwordLoginPath, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.handleLogin(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected 200 (form re-render), got %d", w.Code)
		}
		body := w.Body.String()
		if !strings.Contains(body, "Incorrect password") {
			t.Error("expected error message in form")
		}
	})

	t.Run("correct password sets cookie and redirects", func(t *testing.T) {
		form := url.Values{"password": {"correctpassword"}, "return": {"/dashboard"}}
		req := httptest.NewRequest("POST", passwordLoginPath, strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()
		handler.handleLogin(w, req)

		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}

		loc := w.Header().Get("Location")
		if loc != "/dashboard" {
			t.Errorf("expected redirect to /dashboard, got %s", loc)
		}

		var foundCookie bool
		for _, c := range w.Result().Cookies() {
			if c.Name == pwSessionCookieName {
				foundCookie = true
				break
			}
		}
		if !foundCookie {
			t.Error("expected session cookie to be set after successful login")
		}
	})

	t.Run("logout clears cookie and redirects", func(t *testing.T) {
		req := httptest.NewRequest("GET", passwordLogoutPath, nil)
		w := httptest.NewRecorder()
		handler.handleLogout(w, req)

		if w.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", w.Code)
		}

		loc := w.Header().Get("Location")
		if loc != "/" {
			t.Errorf("expected redirect to /, got %s", loc)
		}

		for _, c := range w.Result().Cookies() {
			if c.Name == pwSessionCookieName && c.MaxAge == -1 {
				return
			}
		}
		t.Error("expected session cookie to be cleared")
	})
}

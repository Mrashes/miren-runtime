package httpingress

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"miren.dev/runtime/api/ingress/ingress_v1alpha"
	"miren.dev/runtime/pkg/waf"
)

func newTestWAFServer() *Server {
	return &Server{
		Log:       slog.Default(),
		wafEngine: waf.NewEngine(slog.Default()),
	}
}

func TestWafMiddlewareDisabledWhenLevelZero(t *testing.T) {
	s := newTestWAFServer()

	route := &ingress_v1alpha.HttpRoute{WafLevel: 0}
	called := false
	next := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	handler := s.wafMiddleware(route, next)

	req := httptest.NewRequest("GET", "http://example.com/?id=1%20OR%201=1--", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.True(t, called, "next handler should be called when WAF is disabled")
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestWafMiddlewareBlocksSQLInjection(t *testing.T) {
	s := newTestWAFServer()

	route := &ingress_v1alpha.HttpRoute{WafLevel: 1}
	called := false
	next := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	handler := s.wafMiddleware(route, next)

	req := httptest.NewRequest("GET", "http://example.com/?id=1%20OR%201=1--", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.False(t, called, "next handler should not be called when WAF blocks")
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWafMiddlewareBlocksXSS(t *testing.T) {
	s := newTestWAFServer()

	route := &ingress_v1alpha.HttpRoute{WafLevel: 1}
	next := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	handler := s.wafMiddleware(route, next)

	req := httptest.NewRequest("GET", "http://example.com/?q=%3Cscript%3Ealert(1)%3C/script%3E", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestWafMiddlewareAllowsCleanRequest(t *testing.T) {
	s := newTestWAFServer()

	route := &ingress_v1alpha.HttpRoute{WafLevel: 1}
	next := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}

	handler := s.wafMiddleware(route, next)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestWafMiddlewareRespectsParanoiaLevel(t *testing.T) {
	s := newTestWAFServer()

	for _, level := range []int64{1, 2, 3, 4} {
		t.Run("level"+string(rune('0'+level)), func(t *testing.T) {
			route := &ingress_v1alpha.HttpRoute{WafLevel: level}
			next := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}

			handler := s.wafMiddleware(route, next)

			// Clean request should pass at any level
			req := httptest.NewRequest("GET", "http://example.com/", nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)

			// SQL injection should be blocked at any level
			req = httptest.NewRequest("GET", "http://example.com/?id=1%20OR%201=1--", nil)
			rec = httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusForbidden, rec.Code)
		})
	}
}

func TestWafMiddlewareInvalidLevel(t *testing.T) {
	s := newTestWAFServer()

	route := &ingress_v1alpha.HttpRoute{WafLevel: 99}
	called := false
	next := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	handler := s.wafMiddleware(route, next)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	// On invalid level, falls through to next handler
	assert.True(t, called)
}

func TestWafMiddlewareNegativeLevel(t *testing.T) {
	s := newTestWAFServer()

	route := &ingress_v1alpha.HttpRoute{WafLevel: -1}
	called := false
	next := func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}

	handler := s.wafMiddleware(route, next)

	req := httptest.NewRequest("GET", "http://example.com/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	assert.True(t, called, "negative WAF level should be treated as disabled")
}

func TestWafEngineInitialized(t *testing.T) {
	s := newTestWAFServer()
	require.NotNil(t, s.wafEngine)
}

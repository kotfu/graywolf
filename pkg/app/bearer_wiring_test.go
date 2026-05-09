package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestBearerWiring_AppliedWhenSet exercises the wiring decision in
// isolation: when cfg.BearerToken is non-empty the middleware wraps
// the handler chain.
func TestBearerWiring_AppliedWhenSet(t *testing.T) {
	cfg := Config{BearerToken: "abc"}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	h := wrapWithBearerIfSet(cfg, inner)
	req := httptest.NewRequest("GET", "/api/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("want 401 (no auth header) got %d", rec.Code)
	}
}

func TestBearerWiring_NoOpWhenEmpty(t *testing.T) {
	cfg := Config{BearerToken: ""}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	h := wrapWithBearerIfSet(cfg, inner)
	req := httptest.NewRequest("GET", "/api/x", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("want 200 (no middleware) got %d", rec.Code)
	}
}

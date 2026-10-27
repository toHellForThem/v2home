package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"v2home/internal/status"
)

func newTestHandler() http.Handler {
	return New(Options{
		Status:     status.New(),
		WSInterval: time.Second,
	})
}

func TestStatusEndpoint(t *testing.T) {
	handler := newTestHandler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/status", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("content type = %q, want application/json", contentType)
	}

	var snapshot status.Snapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &snapshot); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if snapshot.App != "v2home" {
		t.Errorf("app = %q, want v2home", snapshot.App)
	}
	if snapshot.Version == "" {
		t.Error("version must not be empty")
	}
	if snapshot.UptimeSeconds < 0 {
		t.Errorf("uptime = %d, want >= 0", snapshot.UptimeSeconds)
	}
}

func TestHealthz(t *testing.T) {
	handler := newTestHandler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/healthz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status = %v, want ok", body["status"])
	}
}

func TestAPINotFound(t *testing.T) {
	handler := newTestHandler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/nope", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestWebUIServesHTML(t *testing.T) {
	handler := newTestHandler()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("content type = %q, want text/html", contentType)
	}
}

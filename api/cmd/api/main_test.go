package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/subhayan-bhattacharya/events/api/internal/testdb"
)

type fakePinger struct {
	err error
}

func (p fakePinger) Ping(context.Context) error {
	return p.err
}

func TestHealthHandlerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test requires Docker")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	database, err := testdb.Start(ctx)
	if err != nil {
		t.Fatalf("start test database: %v", err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		if err := database.Close(cleanupCtx); err != nil {
			t.Errorf("close test database: %v", err)
		}
	})

	request := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
	response := httptest.NewRecorder()

	healthHandler(database.Pool).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status code: got %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "{\"status\":\"ok\"}\n" {
		t.Errorf("body: got %q, want %q", body, "{\"status\":\"ok\"}\n")
	}
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantStatus int
		wantBody   string
	}{
		{
			name:       "database available",
			wantStatus: http.StatusOK,
			wantBody:   "{\"status\":\"ok\"}\n",
		},
		{
			name:       "database unavailable",
			pingErr:    errors.New("database unavailable"),
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "{\"status\":\"unavailable\"}\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/api/healthz", nil)
			response := httptest.NewRecorder()

			healthHandler(fakePinger{err: tt.pingErr}).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Errorf("status code: got %d, want %d", response.Code, tt.wantStatus)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
				t.Errorf("Content-Type: got %q, want %q", contentType, "application/json")
			}
			if body := response.Body.String(); body != tt.wantBody {
				t.Errorf("body: got %q, want %q", body, tt.wantBody)
			}
		})
	}
}

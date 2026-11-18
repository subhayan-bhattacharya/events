package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireOrigin(t *testing.T) {
	tests := []struct {
		name           string
		allowed        string
		origin         string
		wantStatus     int
		wantNextCalled bool
	}{
		{
			name:           "exact match",
			allowed:        "http://localhost:8080",
			origin:         "http://localhost:8080",
			wantStatus:     http.StatusNoContent,
			wantNextCalled: true,
		},
		{
			name:           "wrong port",
			allowed:        "http://localhost:8080",
			origin:         "http://localhost:3000",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "header missing",
			allowed:        "http://localhost:8080",
			origin:         "",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "other host",
			allowed:        "http://localhost:8080",
			origin:         "http://127.0.0.1:8080",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "other scheme",
			allowed:        "http://localhost:8080",
			origin:         "https://localhost:8080",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "trailing slash",
			allowed:        "http://localhost:8080",
			origin:         "http://localhost:8080/",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "with a path",
			allowed:        "http://localhost:8080",
			origin:         "http://localhost:8080/api",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "opaque origin",
			allowed:        "http://localhost:8080",
			origin:         "null",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "prefix trick",
			allowed:        "http://localhost:8080",
			origin:         "http://localhost:8080.evil.com",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "different case (exact compare)",
			allowed:        "http://localhost:8080",
			origin:         "HTTP://LOCALHOST:8080",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "both empty",
			allowed:        "",
			origin:         "",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "empty allowed with a real origin",
			allowed:        "",
			origin:         "http://localhost:8080",
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			})

			req := httptest.NewRequest(http.MethodPost, "/", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}

			rec := httptest.NewRecorder()
			requireOrigin(tt.allowed)(next).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
			if called != tt.wantNextCalled {
				t.Errorf("got next called = %v, want %v", called, tt.wantNextCalled)
			}
		})
	}
}

//go:build dev

package httpapi_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/subhayan-bhattacharya/events/api/internal/auth"
	"github.com/subhayan-bhattacharya/events/api/internal/httpapi"
)

type fakeDevLoginService struct {
	calls     int
	context   context.Context
	config    auth.DevUserConfig
	tokenData auth.TokenData
	err       error
}

var _ auth.DevLoginService = (*fakeDevLoginService)(nil)

func (f *fakeDevLoginService) DevLogin(
	ctx context.Context, config auth.DevUserConfig,
) (auth.TokenData, error) {
	f.calls++
	f.context = ctx
	f.config = config
	return f.tokenData, f.err
}

func devLoginRequest(body io.Reader) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/dev-login", body)
	request.Header.Set("Origin", "http://localhost:8080")
	return request
}

func TestDevLoginHandlerSuccess(t *testing.T) {
	expiresAt := time.Date(2030, time.May, 2, 12, 34, 56, 0, time.UTC)
	service := &fakeDevLoginService{tokenData: auth.TokenData{
		Token:      "raw-session-token",
		Expiration: expiresAt,
	}}
	config := auth.DevUserConfig{
		Email:       "developer@example.com",
		DisplayName: "Local Developer",
		TimeZone:    "Europe/Berlin",
	}
	request := devLoginRequest(nil)
	recorder := httptest.NewRecorder()

	httpapi.DevLoginHandler(service, config).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if got := recorder.Body.String(); got != "" {
		t.Errorf("body: got %q, want empty", got)
	}
	if service.calls != 1 {
		t.Fatalf("service calls: got %d, want 1", service.calls)
	}
	if service.context != request.Context() {
		t.Error("service did not receive the request context")
	}
	if service.config != config {
		t.Errorf("service config: got %+v, want %+v", service.config, config)
	}

	if got := len(recorder.Header().Values("Set-Cookie")); got != 1 {
		t.Fatalf("Set-Cookie headers: got %d, want 1", got)
	}
	response := recorder.Result()
	defer response.Body.Close()
	cookies := response.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("response cookies: got %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if cookie.Name != "__Host-session" {
		t.Errorf("cookie name: got %q, want %q", cookie.Name, "__Host-session")
	}
	if cookie.Value != service.tokenData.Token {
		t.Errorf("cookie value: got %q, want raw service token", cookie.Value)
	}
	if cookie.Path != "/" {
		t.Errorf("cookie path: got %q, want /", cookie.Path)
	}
	if cookie.Domain != "" {
		t.Errorf("cookie domain: got %q, want empty", cookie.Domain)
	}
	if !cookie.Expires.Equal(expiresAt) {
		t.Errorf("cookie expiry: got %s, want %s", cookie.Expires, expiresAt)
	}
	if !cookie.Secure {
		t.Error("cookie is not Secure")
	}
	if !cookie.HttpOnly {
		t.Error("cookie is not HttpOnly")
	}
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Errorf("cookie SameSite: got %v, want Lax", cookie.SameSite)
	}
}

func TestDevLoginHandlerRejectsOriginBeforeCallingService(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		body   string
	}{
		{name: "missing Origin"},
		{name: "wrong Origin", origin: "http://localhost:3000"},
		{name: "Origin checked before body", origin: "http://localhost:3000", body: "payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeDevLoginService{}
			request := devLoginRequest(strings.NewReader(tt.body))
			request.Header.Del("Origin")
			if tt.origin != "" {
				request.Header.Set("Origin", tt.origin)
			}
			recorder := httptest.NewRecorder()

			httpapi.DevLoginHandler(service, auth.DevUserConfig{}).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusForbidden {
				t.Errorf("status: got %d, want %d", recorder.Code, http.StatusForbidden)
			}
			if service.calls != 0 {
				t.Errorf("service calls: got %d, want 0", service.calls)
			}
			if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
				t.Errorf("Set-Cookie headers: got %q, want none", got)
			}
		})
	}
}

func TestDevLoginHandlerRejectsNonemptyBody(t *testing.T) {
	tests := []struct {
		name          string
		unknownLength bool
	}{
		{name: "known length"},
		{name: "unknown length", unknownLength: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeDevLoginService{}
			request := devLoginRequest(strings.NewReader("payload"))
			if tt.unknownLength {
				request.ContentLength = -1
			}
			recorder := httptest.NewRecorder()

			httpapi.DevLoginHandler(service, auth.DevUserConfig{}).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Errorf("status: got %d, want %d", recorder.Code, http.StatusBadRequest)
			}
			if service.calls != 0 {
				t.Errorf("service calls: got %d, want 0", service.calls)
			}
			if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
				t.Errorf("Set-Cookie headers: got %q, want none", got)
			}
		})
	}
}

func TestDevLoginHandlerAcceptsEmptyUnknownLengthBody(t *testing.T) {
	service := &fakeDevLoginService{tokenData: auth.TokenData{
		Token:      "raw-session-token",
		Expiration: time.Date(2030, time.May, 2, 12, 34, 56, 0, time.UTC),
	}}
	request := devLoginRequest(strings.NewReader(""))
	request.ContentLength = -1
	recorder := httptest.NewRecorder()

	httpapi.DevLoginHandler(service, auth.DevUserConfig{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if service.calls != 1 {
		t.Errorf("service calls: got %d, want 1", service.calls)
	}
}

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("body read failed") }
func (failingBody) Close() error             { return nil }

func TestDevLoginHandlerRejectsBodyReadError(t *testing.T) {
	service := &fakeDevLoginService{}
	request := devLoginRequest(nil)
	request.Body = failingBody{}
	request.ContentLength = -1
	recorder := httptest.NewRecorder()

	httpapi.DevLoginHandler(service, auth.DevUserConfig{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", recorder.Code, http.StatusBadRequest)
	}
	if service.calls != 0 {
		t.Errorf("service calls: got %d, want 0", service.calls)
	}
	if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("Set-Cookie headers: got %q, want none", got)
	}
}

func TestDevLoginHandlerServiceFailure(t *testing.T) {
	serviceErr := errors.New("private database detail")
	service := &fakeDevLoginService{err: serviceErr}
	request := devLoginRequest(nil)
	recorder := httptest.NewRecorder()

	httpapi.DevLoginHandler(service, auth.DevUserConfig{}).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	if service.calls != 1 {
		t.Errorf("service calls: got %d, want 1", service.calls)
	}
	if got := recorder.Header().Values("Set-Cookie"); len(got) != 0 {
		t.Errorf("Set-Cookie headers: got %q, want none", got)
	}
	if got := recorder.Body.String(); strings.Contains(got, serviceErr.Error()) {
		t.Errorf("response exposed service error: %q", got)
	}
}

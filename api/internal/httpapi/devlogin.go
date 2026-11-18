//go:build dev

// Package httpapi holds the HTTP handlers of the API: request parsing, the
// Origin check and cookies, with business rules left to the feature services.
package httpapi

import (
	"errors"
	"io"
	"net/http"

	"github.com/subhayan-bhattacharya/events/api/internal/auth"
)

const devAllowedOrigin = "http://localhost:8080"

func hasPayload(r *http.Request) (bool, error) {
	switch {
	case r.ContentLength > 0:
		return true, nil
	case r.ContentLength == 0:
		return false, nil
	}

	// ContentLength == -1: unknown length, so peek one byte
	var buf [1]byte
	_, err := io.ReadFull(r.Body, buf[:])
	switch {
	case err == nil:
		return true, nil // we got a byte, so there is a payload
	case errors.Is(err, io.EOF):
		return false, nil // stream was empty
	default:
		return false, err // a real read error
	}
}

func DevLoginHandler(
	service auth.DevLoginService, devUserDetails auth.DevUserConfig,
) http.Handler {
	return requireOrigin(devAllowedOrigin)(
		http.HandlerFunc(
			func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				ok, err := hasPayload(r)
				if err != nil {
					http.Error(
						w, "could not read request", http.StatusBadRequest,
					)
					return
				}
				if ok {
					http.Error(
						w, "this endpoint does not accept a request body",
						http.StatusBadRequest,
					)
					return
				}
				tokenData, err := service.DevLogin(r.Context(), devUserDetails)
				if err != nil {
					http.Error(
						w, "could not do dev login",
						http.StatusInternalServerError,
					)
					return
				}
				http.SetCookie(
					w, &http.Cookie{
						Name:     "__Host-session",
						Value:    tokenData.Token,
						Path:     "/",
						Domain:   "",
						Expires:  tokenData.Expiration,
						Secure:   true,
						HttpOnly: true,
						SameSite: http.SameSiteLaxMode,
					},
				)
				w.WriteHeader(http.StatusNoContent)
			},
		),
	)
}

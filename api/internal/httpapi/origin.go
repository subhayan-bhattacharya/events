package httpapi

import "net/http"

func requireOrigin(allowed string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				requestOriginHeader := r.Header.Get("Origin")
				if requestOriginHeader == "" || requestOriginHeader != allowed {
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
				next.ServeHTTP(w, r)
			},
		)
	}
}

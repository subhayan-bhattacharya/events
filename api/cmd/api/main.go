package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/subhayan-bhattacharya/events/api/internal/postgres"
)

type HealthResponse struct {
	Status string `json:"status"`
}

type Pinger interface {
	Ping(ctx context.Context) error
}

func healthHandler(
	pool Pinger,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		resp := HealthResponse{
			Status: "ok",
		}
		statusCode := http.StatusOK
		err := pool.Ping(ctx)
		if err != nil {
			resp = HealthResponse{Status: "unavailable"}
			log.Println(err)
			statusCode = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("the database url is not set.")
	}
	ctx := context.Background()
	pool, err := postgres.Connect(ctx, dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/healthz", healthHandler(pool))

	// 3. Define the server configuration
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	// 4. Start listening and serving requests
	log.Println("Server listening on http://localhost:8080/api/healtz")
	if err := server.ListenAndServe(); err != nil && !errors.Is(
		err, http.ErrServerClosed,
	) {
		log.Fatalf("Server failed: %v", err)
	}
}

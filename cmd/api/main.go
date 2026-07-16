package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kohei321dev/spot-diggz/internal/httpapi"
	"github.com/kohei321dev/spot-diggz/internal/postgres"
	"github.com/kohei321dev/spot-diggz/internal/spot"
)

func main() {
	ctx := context.Background()
	store, closeStore, err := buildStore(ctx)
	if err != nil {
		log.Fatalf("initialize spot store: %v", err)
	}
	defer closeStore()

	router := httpapi.NewSdzRouter(store)
	addr := ":" + port()

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("spotdiggz api listening on http://0.0.0.0%s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("spotdiggz api stopped: %v", err)
	}
}

func buildStore(ctx context.Context) (spot.SdzStore, func(), error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Print("DATABASE_URL is not set; using in-memory store")
		return spot.NewSdzMemoryStore(), func() {}, nil
	}

	connectCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	store, err := postgres.NewSdzStore(connectCtx, databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("postgres store: %w", err)
	}
	log.Print("using postgres store")
	return store, store.Close, nil
}

func port() string {
	if value := os.Getenv("PORT"); value != "" {
		return value
	}
	return "8080"
}

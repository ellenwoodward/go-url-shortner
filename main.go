package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"go-url-shortner/handler"
	"go-url-shortner/repository"
	"go-url-shortner/service"
)

// config holds runtime configuration loaded from environment variables.
type config struct {
	DatabaseURL string
	BaseURL     string
	Addr        string
}

// loadConfig reads config from environment variables, falling back to defaults.
func loadConfig() config {
	return config{
		DatabaseURL: envOrDefault("DATABASE_URL", "postgres://shortener:shortener@localhost:5432/shortener"),
		BaseURL:     envOrDefault("BASE_URL", "http://localhost:8080"),
		Addr:        envOrDefault("ADDR", ":8080"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	cfg := loadConfig()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("pinging database: %v", err)
	}
	log.Println("database connection established")

	repo := repository.New(pool)
	if err := repo.CreateSchema(ctx); err != nil {
		log.Fatalf("creating schema: %v", err)
	}

	svc := service.New(repo, cfg.BaseURL)
	h := handler.New(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /shorten", h.Shorten)
	mux.HandleFunc("GET /{code}", h.Redirect)

	srv := &http.Server{
		Addr:         cfg.Addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server listening on %s", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}

// Package handler contains the HTTP handlers for the URL shortener.
package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"go-url-shortner/repository"
	"go-url-shortner/service"
)

// URLService is the interface the handler requires from the service layer.
type URLService interface {
	Shorten(ctx context.Context, longURL string) (service.ShortenResult, error)
	Resolve(ctx context.Context, shortCode string) (string, error)
}

// Handler holds the HTTP handler methods.
type Handler struct {
	svc URLService
}

// New creates a Handler with the given service.
func New(svc URLService) *Handler {
	return &Handler{svc: svc}
}

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
}

// Shorten handles POST /shorten and returns a generated short URL.
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: must be JSON", http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, `"url" field is required`, http.StatusBadRequest)
		return
	}

	result, err := h.svc.Shorten(r.Context(), req.URL)
	if err != nil {
		log.Printf("shorten error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(shortenResponse{
		ShortCode: result.ShortCode,
		ShortURL:  result.ShortURL,
	})
}

// Redirect handles GET /{code} and redirects to the original URL.
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	longURL, err := h.svc.Resolve(r.Context(), code)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		log.Printf("redirect error for code %q: %v", code, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longURL, http.StatusFound)
}

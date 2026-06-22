// Package service contains the URL shortening business logic.
package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"

	"go-url-shortner/repository"
)

// URLRepository is the interface the service requires from the data layer.
type URLRepository interface {
	Save(ctx context.Context, shortCode, longURL string) (repository.URL, error)
	FindByCode(ctx context.Context, shortCode string) (repository.URL, error)
}

// ShortenResult is returned by Shorten.
type ShortenResult struct {
	ShortCode string
	ShortURL  string
}

// Service implements the URL shortening business logic.
type Service struct {
	repo    URLRepository
	baseURL string
}

// New creates a Service with the given repository and base URL.
func New(repo URLRepository, baseURL string) *Service {
	return &Service{repo: repo, baseURL: baseURL}
}

// Shorten generates a short code for the given URL and persists it.
func (s *Service) Shorten(ctx context.Context, longURL string) (ShortenResult, error) {
	const maxAttempts = 5

	for attempt := range maxAttempts {
		code, err := generateCode(7)
		if err != nil {
			return ShortenResult{}, fmt.Errorf("generating code: %w", err)
		}

		url, err := s.repo.Save(ctx, code, longURL)
		if err == nil {
			return ShortenResult{
				ShortCode: url.ShortCode,
				ShortURL:  s.baseURL + "/" + url.ShortCode,
			}, nil
		}

		if errors.Is(err, repository.ErrConflict) {
			continue
		}
		return ShortenResult{}, fmt.Errorf("saving url (attempt %d): %w", attempt+1, err)
	}

	return ShortenResult{}, errors.New("failed to generate a unique short code after max attempts")
}

// Resolve returns the original URL for the given short code.
func (s *Service) Resolve(ctx context.Context, shortCode string) (string, error) {
	url, err := s.repo.FindByCode(ctx, shortCode)
	if err != nil {
		return "", fmt.Errorf("resolving code %q: %w", shortCode, err)
	}
	return url.LongURL, nil
}

// generateCode returns a random 7-char base62 string.
// Uses crypto/rand so codes can't be predicted or enumerated.
func generateCode(n int) (string, error) {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	for i := range b {
		b[i] = alphabet[b[i]%62]
	}
	return string(b), nil
}

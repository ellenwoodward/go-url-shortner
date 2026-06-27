package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-url-shortner/repository"
	"go-url-shortner/service"
)

type stubURLService struct {
	shorten func(context.Context, string) (service.ShortenResult, error)
	resolve func(context.Context, string) (string, error)
}

func (s stubURLService) Shorten(ctx context.Context, longURL string) (service.ShortenResult, error) {
	return s.shorten(ctx, longURL)
}

func (s stubURLService) Resolve(ctx context.Context, shortCode string) (string, error) {
	return s.resolve(ctx, shortCode)
}

func TestShorten(t *testing.T) {
	t.Run("creates a short URL", func(t *testing.T) {
		called := false
		h := New(stubURLService{
			shorten: func(_ context.Context, longURL string) (service.ShortenResult, error) {
				called = true
				if longURL != "https://example.com/a" {
					t.Fatalf("long URL = %q", longURL)
				}
				return service.ShortenResult{ShortCode: "abc1234", ShortURL: "https://sho.rt/abc1234"}, nil
			},
		})

		req := httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(`{"url":"https://example.com/a"}`))
		res := httptest.NewRecorder()
		h.Shorten(res, req)

		if !called {
			t.Fatal("service was not called")
		}
		if res.Code != http.StatusCreated {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusCreated)
		}
		if got := res.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}

		var body shortenResponse
		if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.ShortCode != "abc1234" || body.ShortURL != "https://sho.rt/abc1234" {
			t.Errorf("response = %+v", body)
		}
	})

	tests := []struct {
		name       string
		body       string
		serviceErr error
		wantStatus int
		wantBody   string
		wantCalls  int
	}{
		{name: "rejects malformed JSON", body: `{`, wantStatus: http.StatusBadRequest, wantBody: "invalid request body", wantCalls: 0},
		{name: "requires URL", body: `{}`, wantStatus: http.StatusBadRequest, wantBody: `"url" field is required`, wantCalls: 0},
		{name: "handles service error", body: `{"url":"https://example.com"}`, serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError, wantBody: "internal server error", wantCalls: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			h := New(stubURLService{
				shorten: func(context.Context, string) (service.ShortenResult, error) {
					calls++
					return service.ShortenResult{}, tt.serviceErr
				},
			})
			res := httptest.NewRecorder()
			h.Shorten(res, httptest.NewRequest(http.MethodPost, "/shorten", strings.NewReader(tt.body)))

			if res.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.Code, tt.wantStatus)
			}
			if !strings.Contains(res.Body.String(), tt.wantBody) {
				t.Errorf("body = %q, want it to contain %q", res.Body.String(), tt.wantBody)
			}
			if calls != tt.wantCalls {
				t.Errorf("service calls = %d, want %d", calls, tt.wantCalls)
			}
		})
	}
}

func TestRedirect(t *testing.T) {
	tests := []struct {
		name         string
		serviceURL   string
		serviceErr   error
		wantStatus   int
		wantLocation string
	}{
		{name: "redirects to original URL", serviceURL: "https://example.com/path?q=1", wantStatus: http.StatusFound, wantLocation: "https://example.com/path?q=1"},
		{name: "returns not found", serviceErr: repository.ErrNotFound, wantStatus: http.StatusNotFound},
		{name: "recognizes wrapped not found", serviceErr: errors.Join(errors.New("resolve failed"), repository.ErrNotFound), wantStatus: http.StatusNotFound},
		{name: "handles service error", serviceErr: errors.New("database unavailable"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(stubURLService{
				resolve: func(_ context.Context, code string) (string, error) {
					if code != "abc1234" {
						t.Errorf("code = %q, want abc1234", code)
					}
					return tt.serviceURL, tt.serviceErr
				},
			})
			mux := http.NewServeMux()
			mux.HandleFunc("GET /{code}", h.Redirect)
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/abc1234", nil))

			if res.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.Code, tt.wantStatus)
			}
			if got := res.Header().Get("Location"); got != tt.wantLocation {
				t.Errorf("Location = %q, want %q", got, tt.wantLocation)
			}
		})
	}
}

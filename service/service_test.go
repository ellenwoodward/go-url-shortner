package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-url-shortner/repository"
)

type stubURLRepository struct {
	save       func(context.Context, string, string) (repository.URL, error)
	findByCode func(context.Context, string) (repository.URL, error)
}

func (r stubURLRepository) Save(ctx context.Context, code, longURL string) (repository.URL, error) {
	return r.save(ctx, code, longURL)
}

func (r stubURLRepository) FindByCode(ctx context.Context, code string) (repository.URL, error) {
	return r.findByCode(ctx, code)
}

func TestShorten(t *testing.T) {
	t.Run("saves URL and builds result", func(t *testing.T) {
		repo := stubURLRepository{
			save: func(_ context.Context, code, longURL string) (repository.URL, error) {
				assertValidCode(t, code, 7)
				if longURL != "https://example.com" {
					t.Errorf("long URL = %q", longURL)
				}
				return repository.URL{ShortCode: code, LongURL: longURL}, nil
			},
		}

		result, err := New(repo, "https://sho.rt").Shorten(context.Background(), "https://example.com")
		if err != nil {
			t.Fatalf("Shorten() error = %v", err)
		}
		assertValidCode(t, result.ShortCode, 7)
		if result.ShortURL != "https://sho.rt/"+result.ShortCode {
			t.Errorf("short URL = %q", result.ShortURL)
		}
	})

	t.Run("retries code conflicts", func(t *testing.T) {
		calls := 0
		repo := stubURLRepository{
			save: func(_ context.Context, code, longURL string) (repository.URL, error) {
				calls++
				if calls < 3 {
					return repository.URL{}, repository.ErrConflict
				}
				return repository.URL{ShortCode: code, LongURL: longURL}, nil
			},
		}

		if _, err := New(repo, "https://sho.rt").Shorten(context.Background(), "https://example.com"); err != nil {
			t.Fatalf("Shorten() error = %v", err)
		}
		if calls != 3 {
			t.Errorf("Save() calls = %d, want 3", calls)
		}
	})

	t.Run("stops after maximum conflicts", func(t *testing.T) {
		calls := 0
		repo := stubURLRepository{
			save: func(context.Context, string, string) (repository.URL, error) {
				calls++
				return repository.URL{}, repository.ErrConflict
			},
		}

		_, err := New(repo, "https://sho.rt").Shorten(context.Background(), "https://example.com")
		if err == nil || !strings.Contains(err.Error(), "after max attempts") {
			t.Fatalf("Shorten() error = %v, want max-attempts error", err)
		}
		if calls != 5 {
			t.Errorf("Save() calls = %d, want 5", calls)
		}
	})

	t.Run("returns non-conflict save error", func(t *testing.T) {
		saveErr := errors.New("connection lost")
		calls := 0
		repo := stubURLRepository{
			save: func(context.Context, string, string) (repository.URL, error) {
				calls++
				return repository.URL{}, saveErr
			},
		}

		_, err := New(repo, "https://sho.rt").Shorten(context.Background(), "https://example.com")
		if !errors.Is(err, saveErr) {
			t.Fatalf("Shorten() error = %v, want wrapped save error", err)
		}
		if calls != 1 {
			t.Errorf("Save() calls = %d, want 1", calls)
		}
	})
}

func TestResolve(t *testing.T) {
	t.Run("returns original URL", func(t *testing.T) {
		repo := stubURLRepository{
			findByCode: func(_ context.Context, code string) (repository.URL, error) {
				if code != "abc1234" {
					t.Errorf("code = %q", code)
				}
				return repository.URL{LongURL: "https://example.com"}, nil
			},
		}

		got, err := New(repo, "").Resolve(context.Background(), "abc1234")
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if got != "https://example.com" {
			t.Errorf("Resolve() = %q", got)
		}
	})

	t.Run("wraps repository error", func(t *testing.T) {
		repo := stubURLRepository{
			findByCode: func(context.Context, string) (repository.URL, error) {
				return repository.URL{}, repository.ErrNotFound
			},
		}

		_, err := New(repo, "").Resolve(context.Background(), "missing")
		if !errors.Is(err, repository.ErrNotFound) {
			t.Fatalf("Resolve() error = %v, want wrapped ErrNotFound", err)
		}
	})
}

func TestGenerateCode(t *testing.T) {
	for _, length := range []int{0, 1, 7, 32} {
		code, err := generateCode(length)
		if err != nil {
			t.Fatalf("generateCode(%d) error = %v", length, err)
		}
		assertValidCode(t, code, length)
	}
}

func assertValidCode(t *testing.T, code string, length int) {
	t.Helper()
	if len(code) != length {
		t.Errorf("code length = %d, want %d", len(code), length)
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	for _, char := range code {
		if !strings.ContainsRune(alphabet, char) {
			t.Errorf("code %q contains non-base62 character %q", code, char)
		}
	}
}

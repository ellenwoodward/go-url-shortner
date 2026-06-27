package repository

import (
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestTranslateError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   error
		wantOp bool
	}{
		{name: "no rows", err: pgx.ErrNoRows, want: ErrNotFound},
		{name: "wrapped no rows", err: errors.Join(errors.New("scan"), pgx.ErrNoRows), want: ErrNotFound},
		{name: "unique violation", err: &pgconn.PgError{Code: "23505"}, want: ErrConflict},
		{name: "other postgres error", err: &pgconn.PgError{Code: "23503"}, wantOp: true},
		{name: "generic error", err: errors.New("connection lost"), wantOp: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateError(tt.err, "testing operation")
			if tt.want != nil && !errors.Is(got, tt.want) {
				t.Errorf("translateError() = %v, want %v", got, tt.want)
			}
			if tt.wantOp && !strings.HasPrefix(got.Error(), "testing operation") {
				t.Errorf("translateError() = %q, want operation context", got)
			}
		})
	}
}

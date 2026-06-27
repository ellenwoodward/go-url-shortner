package main

import "testing"

func TestEnvOrDefault(t *testing.T) {
	const key = "GO_URL_SHORTENER_TEST_VALUE"

	t.Run("uses default when unset", func(t *testing.T) {
		t.Setenv(key, "")
		if got := envOrDefault(key, "fallback"); got != "fallback" {
			t.Errorf("envOrDefault() = %q, want fallback", got)
		}
	})

	t.Run("uses environment value", func(t *testing.T) {
		t.Setenv(key, "configured")
		if got := envOrDefault(key, "fallback"); got != "configured" {
			t.Errorf("envOrDefault() = %q, want configured", got)
		}
	})
}

func TestLoadConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test/db")
	t.Setenv("BASE_URL", "https://sho.rt")
	t.Setenv("ADDR", ":9090")

	got := loadConfig()
	want := config{DatabaseURL: "postgres://test/db", BaseURL: "https://sho.rt", Addr: ":9090"}
	if got != want {
		t.Errorf("loadConfig() = %+v, want %+v", got, want)
	}
}

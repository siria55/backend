package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_HOST", "")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("DATABASE_URL", "")

	cfg := Load()

	if cfg.Environment != "development" {
		t.Fatalf("expected environment to default to development, got %q", cfg.Environment)
	}
	if cfg.HTTP.Host != "0.0.0.0" || cfg.HTTP.Port != "8080" {
		t.Fatalf("expected default HTTP host:port 0.0.0.0:8080, got %s:%s", cfg.HTTP.Host, cfg.HTTP.Port)
	}
	expectedURL := "postgres://postgres:z13547842355@localhost:5432/mars?sslmode=disable"
	if cfg.Database.URL != expectedURL {
		t.Fatalf("expected default database url %q, got %q", expectedURL, cfg.Database.URL)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("HTTP_HOST", "127.0.0.1")
	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://user:pass@db.example/mars?sslmode=require")

	cfg := Load()

	if cfg.Environment != "test" {
		t.Fatalf("expected environment override, got %q", cfg.Environment)
	}
	if cfg.HTTP.Host != "127.0.0.1" || cfg.HTTP.Port != "9090" {
		t.Fatalf("expected HTTP override 127.0.0.1:9090, got %s:%s", cfg.HTTP.Host, cfg.HTTP.Port)
	}
	if cfg.Database.URL != "postgres://user:pass@db.example/mars?sslmode=require" {
		t.Fatalf("expected database override, got %q", cfg.Database.URL)
	}
}

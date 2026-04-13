package config

import (
	"testing"
)

func TestGetEnv_Present(t *testing.T) {
	t.Setenv("TEST_GET_ENV_KEY", "myvalue")
	got := getEnv("TEST_GET_ENV_KEY", "default")
	if got != "myvalue" {
		t.Errorf("getEnv() = %q, want %q", got, "myvalue")
	}
}

func TestGetEnv_Absent(t *testing.T) {
	got := getEnv("TEST_GET_ENV_NONEXISTENT_KEY_12345", "fallback")
	if got != "fallback" {
		t.Errorf("getEnv() = %q, want %q", got, "fallback")
	}
}

func TestGetEnv_EmptyReturnsDefault(t *testing.T) {
	t.Setenv("TEST_GET_ENV_EMPTY", "")
	got := getEnv("TEST_GET_ENV_EMPTY", "default")
	if got != "default" {
		t.Errorf("getEnv() = %q, want %q for empty env var", got, "default")
	}
}

func TestGetCORSOrigins_Explicit(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.com,https://b.com")
	t.Setenv("DOMAIN", "") // should be ignored

	origins := getCORSOrigins()
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
	if origins[0] != "https://a.com" || origins[1] != "https://b.com" {
		t.Errorf("unexpected origins: %v", origins)
	}
}

func TestGetCORSOrigins_DomainFallback(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("DOMAIN", "example.com")

	origins := getCORSOrigins()
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins, got %d", len(origins))
	}
	if origins[0] != "https://example.com" {
		t.Errorf("origins[0] = %q, want %q", origins[0], "https://example.com")
	}
	if origins[1] != "http://example.com" {
		t.Errorf("origins[1] = %q, want %q", origins[1], "http://example.com")
	}
}

func TestGetCORSOrigins_Default(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("DOMAIN", "")

	origins := getCORSOrigins()
	if len(origins) != 1 || origins[0] != "http://localhost:3000" {
		t.Errorf("expected default [http://localhost:3000], got %v", origins)
	}
}

func TestGetDBPath_DBDir(t *testing.T) {
	t.Setenv("DB_DIR", "/data/mydir")
	t.Setenv("DB_PATH", "/other/path.db")

	got := getDBPath()
	want := "/data/mydir/portfolio_manager.db"
	if got != want {
		t.Errorf("getDBPath() = %q, want %q", got, want)
	}
}

func TestGetDBPath_DBPath(t *testing.T) {
	t.Setenv("DB_DIR", "")
	t.Setenv("DB_PATH", "/custom/path.db")

	got := getDBPath()
	if got != "/custom/path.db" {
		t.Errorf("getDBPath() = %q, want %q", got, "/custom/path.db")
	}
}

func TestGetDBPath_Default(t *testing.T) {
	t.Setenv("DB_DIR", "")
	t.Setenv("DB_PATH", "")

	got := getDBPath()
	if got != "./data/portfolio_manager.db" {
		t.Errorf("getDBPath() = %q, want %q", got, "./data/portfolio_manager.db")
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Clear relevant env vars so defaults are used.
	t.Setenv("SERVER_PORT", "")
	t.Setenv("SERVER_HOST", "")
	t.Setenv("DB_DIR", "")
	t.Setenv("DB_PATH", "")
	t.Setenv("LOG_DIR", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("DOMAIN", "")
	t.Setenv("IBKR_ENCRYPTION_KEY", "")
	t.Setenv("INTERNAL_API_KEY", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Port != "5000" {
		t.Errorf("Server.Port = %q, want %q", cfg.Server.Port, "5000")
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Addr != "0.0.0.0:5000" {
		t.Errorf("Server.Addr = %q, want %q", cfg.Server.Addr, "0.0.0.0:5000")
	}
	if cfg.Database.Path != "./data/portfolio_manager.db" {
		t.Errorf("Database.Path = %q, want default", cfg.Database.Path)
	}
	if cfg.Log.Dir != "./data/logs" {
		t.Errorf("Log.Dir = %q, want %q", cfg.Log.Dir, "./data/logs")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("DB_DIR", "")
	t.Setenv("DB_PATH", "/tmp/test.db")
	t.Setenv("LOG_DIR", "/var/log/app")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://mysite.com")
	t.Setenv("DOMAIN", "")
	t.Setenv("IBKR_ENCRYPTION_KEY", "secret123")
	t.Setenv("INTERNAL_API_KEY", "apikey456")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Server.Addr != "127.0.0.1:8080" {
		t.Errorf("Server.Addr = %q, want %q", cfg.Server.Addr, "127.0.0.1:8080")
	}
	if cfg.Database.Path != "/tmp/test.db" {
		t.Errorf("Database.Path = %q", cfg.Database.Path)
	}
	if cfg.Log.Dir != "/var/log/app" {
		t.Errorf("Log.Dir = %q", cfg.Log.Dir)
	}
	if cfg.EncryptionKey != "secret123" {
		t.Errorf("EncryptionKey = %q", cfg.EncryptionKey)
	}
	if cfg.InternalAPIKey != "apikey456" {
		t.Errorf("InternalAPIKey = %q", cfg.InternalAPIKey)
	}
	if len(cfg.CORS.AllowedOrigins) != 1 || cfg.CORS.AllowedOrigins[0] != "https://mysite.com" {
		t.Errorf("CORS.AllowedOrigins = %v", cfg.CORS.AllowedOrigins)
	}
}

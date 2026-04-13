// Package config loads and exposes application configuration from environment variables and .env files.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application.
type Config struct {
	Server         ServerConfig
	Database       DatabaseConfig
	Log            LogConfig
	CORS           CORSConfig
	EncryptionKey  string // IBKR_ENCRYPTION_KEY (fernet, base64-encoded)
	InternalAPIKey string // INTERNAL_API_KEY
}

// ServerConfig holds server-specific configuration.
type ServerConfig struct {
	Port string
	Host string
	Addr string // Combined host:port for convenience
}

// DatabaseConfig holds database-specific configuration.
type DatabaseConfig struct {
	Path string
}

// LogConfig holds log-specific configuration.
type LogConfig struct {
	Dir string
}

// CORSConfig holds CORS-specific configuration.
type CORSConfig struct {
	AllowedOrigins []string
}

// getCORSOrigins returns the allowed CORS origins from environment variables.
func getCORSOrigins() []string {
	// Check for explicit CORS config first
	if origins := os.Getenv("CORS_ALLOWED_ORIGINS"); origins != "" {
		return strings.Split(origins, ",")
	}

	// Check DOMAIN variable (from Docker) second
	if domain := os.Getenv("DOMAIN"); domain != "" {
		return []string{
			fmt.Sprintf("https://%s", domain),
			fmt.Sprintf("http://%s", domain),
		}
	}
	// If nether, return development setting
	return []string{"http://localhost:3000"}
}

// Load reads configuration from environment variables and .env file.
func Load() (*Config, error) {
	// Try to load .env file (ignore error if it doesn't exist)
	if err := godotenv.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: .env file not loaded: %v (this is OK if using env vars)\n", err)
	}

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "5000"),
			Host: getEnv("SERVER_HOST", "0.0.0.0"),
		},
		Database: DatabaseConfig{
			Path: getDBPath(),
		},
		Log: LogConfig{
			Dir: getEnv("LOG_DIR", "./data/logs"),
		},
		CORS: CORSConfig{
			AllowedOrigins: getCORSOrigins(),
		},
		EncryptionKey:  getEnv("IBKR_ENCRYPTION_KEY", ""),
		InternalAPIKey: getEnv("INTERNAL_API_KEY", ""),
	}

	// Combine host and port
	config.Server.Addr = fmt.Sprintf("%s:%s", config.Server.Host, config.Server.Port)

	return config, nil
}

// getDBPath resolves the database file path.
// DB_DIR (set by Docker) takes precedence: DB_DIR/portfolio_manager.db.
// Falls back to DB_PATH, then the default local path.
func getDBPath() string {
	if dir := os.Getenv("DB_DIR"); dir != "" {
		return filepath.Join(dir, "portfolio_manager.db")
	}
	return getEnv("DB_PATH", "./data/portfolio_manager.db")
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

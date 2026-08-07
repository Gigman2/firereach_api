package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	DatabaseURL  string
	ClaudeAPIKey string
	JWTSecret    string
	Environment  string
}

func Load() (*Config, error) {
	// Load .env file if present (local dev only — ignored in production)
	_ = godotenv.Load()

	cfg := &Config{
		Port:         getEnv("PORT", "9000"),
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		ClaudeAPIKey: os.Getenv("CLAUDE_API_KEY"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		Environment:  getEnv("ENVIRONMENT", "development"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

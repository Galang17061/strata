package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTKey           string
	JWTIssuer        string
	JWTAudience      string
	JWTExpiryMinutes int
	PasswordKey      string
	UploadDir        string
	MigrationsDir    string
}

func Load() (Config, error) {
	_ = godotenv.Load()
	cfg := Config{
		Port:             envOr("STRATA_PORT", "5000"),
		DatabaseURL:      os.Getenv("STRATA_DB_CONNECTION"),
		JWTKey:           os.Getenv("STRATA_JWT_KEY"),
		JWTIssuer:        envOr("STRATA_JWT_ISSUER", "issuer"),
		JWTAudience:      envOr("STRATA_JWT_AUDIENCE", "audience"),
		JWTExpiryMinutes: 10080,
		PasswordKey:      os.Getenv("STRATA_PASSWORD_KEY"),
		UploadDir:        envOr("STRATA_UPLOAD_DIR", "./upload"),
		MigrationsDir:    envOr("STRATA_MIGRATIONS_DIR", "./migrations"),
	}
	if raw := os.Getenv("STRATA_JWT_EXPIRY_MINUTES"); raw != "" {
		minutes, err := strconv.Atoi(raw)
		if err != nil {
			return cfg, errors.New("STRATA_JWT_EXPIRY_MINUTES must be a whole number")
		}
		cfg.JWTExpiryMinutes = minutes
	}
	if cfg.DatabaseURL == "" {
		return cfg, errors.New("STRATA_DB_CONNECTION is required")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

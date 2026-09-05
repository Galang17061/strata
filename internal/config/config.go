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
	WebURL           string
	MailHost         string
	MailPort         string
	MailUsername     string
	MailPassword     string
	MailFrom         string
	FeedbackEmail    string
	GAConcurrency    int
	GAQueueWait      int
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
		WebURL:           envOr("STRATA_WEB_URL", "http://localhost:3000"),
		MailHost:         os.Getenv("STRATA_MAIL_HOST"),
		MailPort:         envOr("STRATA_MAIL_PORT", "587"),
		MailUsername:     os.Getenv("STRATA_MAIL_USERNAME"),
		MailPassword:     os.Getenv("STRATA_MAIL_PASSWORD"),
		MailFrom:         os.Getenv("STRATA_MAIL_FROM"),
		FeedbackEmail:    os.Getenv("STRATA_FEEDBACK_EMAIL"),
		GAConcurrency:    2,
		GAQueueWait:      10,
	}
	if raw := os.Getenv("STRATA_GA_CONCURRENCY"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return cfg, errors.New("STRATA_GA_CONCURRENCY must be a whole number of at least 1")
		}
		cfg.GAConcurrency = parsed
	}
	if raw := os.Getenv("STRATA_GA_QUEUE_WAIT"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			return cfg, errors.New("STRATA_GA_QUEUE_WAIT must be zero or more seconds")
		}
		cfg.GAQueueWait = parsed
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

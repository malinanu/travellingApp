// Package config loads server settings from environment variables
// (optionally from a .env file next to the working directory).
package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	Port              string
	DatabaseURL       string
	JWTSecret         string
	CORSOrigins       []string
	TOTPEncryptionKey []byte // 32 raw bytes, decoded from TOTP_ENCRYPTION_KEY (base64)

	// Mail and Google OAuth are optional — unset means the mailer logs
	// instead of sending, and the Google endpoints report themselves as
	// unconfigured rather than failing startup. Neither requires a paid
	// account: any SMTP relay works for mail, and Google Cloud OAuth
	// clients are free.
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPass           string
	EmailFrom          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	// GoogleCalendarAPIKey enables live public-holiday lookups (see
	// internal/holidays) — a plain API key (not an OAuth client), since
	// Google's public holiday calendars are readable without user consent.
	// Unset means the holiday scheduler falls back to a bundled static list
	// instead of failing.
	GoogleCalendarAPIKey string
	// VendorAppURL is the base URL of the vendor Next.js app, used to build
	// links embedded in emails (invite accept, password reset).
	VendorAppURL string
	// AdminAppURL is the base URL of the admin Next.js app — used the same
	// way as VendorAppURL, but for admin-role accounts (e.g. the
	// forgot-password reset link must point at the admin app, not vendor's).
	AdminAppURL string
	// SuperAdminEmails lists admin accounts (by email, case-insensitive)
	// granted the "super-admin" tier — e.g. viewing the cross-admin audit
	// log. Deliberately env-var-driven rather than a DB role/invite system:
	// there's no need yet for admin accounts to be self-service provisioned.
	SuperAdminEmails []string
}

// minSecretLen guards against trivially short/guessable secrets — anyone
// setting JWT_SECRET to "abc" defeats the point of requiring it at all.
const minSecretLen = 16

func Load() Config {
	loadDotEnv(".env")

	jwtSecret := mustGetenv("JWT_SECRET")
	if len(jwtSecret) < minSecretLen {
		log.Fatalf("JWT_SECRET must be at least %d characters — refusing to start with a weak secret", minSecretLen)
	}

	totpKey, err := decodeTOTPKey(mustGetenv("TOTP_ENCRYPTION_KEY"))
	if err != nil {
		log.Fatalf("TOTP_ENCRYPTION_KEY: %v", err)
	}

	return Config{
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       mustGetenv("DATABASE_URL"),
		JWTSecret:         jwtSecret,
		CORSOrigins:       strings.Split(mustGetenv("CORS_ORIGINS"), ","),
		TOTPEncryptionKey: totpKey,

		SMTPHost:             getenv("SMTP_HOST", ""),
		SMTPPort:             getenv("SMTP_PORT", ""),
		SMTPUser:             getenv("SMTP_USER", ""),
		SMTPPass:             getenv("SMTP_PASS", ""),
		EmailFrom:            getenv("EMAIL_FROM", ""),
		GoogleClientID:       getenv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getenv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    getenv("GOOGLE_REDIRECT_URL", ""),
		GoogleCalendarAPIKey: getenv("GOOGLE_CALENDAR_API_KEY", ""),
		VendorAppURL:         getenv("VENDOR_APP_URL", "http://localhost:3000"),
		AdminAppURL:          getenv("ADMIN_APP_URL", "http://localhost:3001"),
		SuperAdminEmails:     splitNonEmpty(getenv("SUPER_ADMIN_EMAILS", "")),
	}
}

// splitNonEmpty splits a comma-separated env value, trimming whitespace and
// dropping empty segments — so a trailing comma or extra spaces don't
// silently produce a spurious empty-string entry.
func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func decodeTOTPKey(b64 string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("must be valid base64 (generate one with: openssl rand -base64 32): %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("must decode to exactly 32 bytes for AES-256 (got %d) — generate one with: openssl rand -base64 32", len(key))
	}
	return key, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// mustGetenv fails startup rather than silently falling back to a hardcoded
// default — a missing secret should be a loud, immediate error, not a quiet
// downgrade to a value that (for a public repo like this one) is effectively
// public knowledge.
func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("%s must be set (see server/.env.example) — refusing to start without it", key)
	}
	return v
}

// loadDotEnv sets KEY=VALUE lines from the file if the keys are unset.
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if os.Getenv(key) == "" {
			os.Setenv(key, strings.TrimSpace(value))
		}
	}
}

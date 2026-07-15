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
	}
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

// Package config loads server settings from environment variables
// (optionally from a .env file next to the working directory).
package config

import (
	"bufio"
	"os"
	"strings"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	CORSOrigins []string
}

func Load() Config {
	loadDotEnv(".env")
	return Config{
		Port:        getenv("PORT", "8080"),
		DatabaseURL: getenv("DATABASE_URL", "postgres://postgres:admin123@localhost:5432/spotto?sslmode=disable"),
		JWTSecret:   getenv("JWT_SECRET", "spotto-dev-secret-change-me"),
		CORSOrigins: strings.Split(getenv("CORS_ORIGINS", "http://localhost:3000,http://localhost:3001"), ","),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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

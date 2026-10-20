// Package config loads server settings from environment variables
// (optionally from a .env file next to the working directory).
package config

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strconv"
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
	// AppleBundleID is the iOS bundle id Sign in with Apple identity tokens
	// must be issued to (the token's audience). Defaults to the app's own id.
	AppleBundleID string
	// GoogleCalendarAPIKey enables live public-holiday lookups (see
	// internal/holidays) and, since the same Cloud project/key also has the
	// Maps JavaScript + Places APIs enabled (it's the same value as the
	// vendor portal's NEXT_PUBLIC_GOOGLE_MAPS_API_KEY and the Flutter app's
	// MAPS_API_KEY — one key, several config files), also powers live
	// Places Details lookups (see internal/placedetails) that backfill
	// phone/website/wheelchair-accessibility/price-level for imported
	// spots. Unset means the holiday scheduler falls back to a bundled
	// static list, and GET /places/{id} just skips the Places enrichment —
	// neither fails startup.
	GoogleCalendarAPIKey string
	// TomorrowIOAPIKey enables live weather (see internal/weather) for the
	// app's Weather screen. Unset means GET /weather reports itself
	// unconfigured (501) rather than failing startup — same convention as
	// GoogleCalendarAPIKey above.
	TomorrowIOAPIKey string
	// GeminiAPIKey enables BR-certificate OCR pre-fill (see
	// internal/gemini) and powers the Local Guide chat/embeddings
	// (see internal/http/guide.go, internal/recommend). Unset means
	// these features are unconfigured (no OCR, 501 for guide endpoints).
	GeminiAPIKey string
	// GeminiChatModel / GeminiEmbedModel override the default model ids in
	// internal/gemini — Google retires ids on its own schedule, so a
	// retirement is an env change rather than a code change.
	GeminiChatModel  string
	GeminiEmbedModel string
	// Inspiration Board link extraction (see internal/http/inspiration_worker.go).
	// GeminiVideoModel watches reels (defaults to GeminiChatModel).
	// GooglePlacesAPIKey needs Places API (New); it falls back to
	// GoogleCalendarAPIKey, the shared Maps/Places key, when unset.
	// YtdlpCookiesFile is an optional Netscape cookies file for yt-dlp when
	// Instagram refuses anonymous downloads. InspirationDailyLimit caps links
	// one traveller can add per 24 hours.
	GeminiVideoModel      string
	GooglePlacesAPIKey    string
	YtdlpCookiesFile      string
	InspirationDailyLimit int
	// VendorAppURL is the base URL of the vendor Next.js app, used to build
	// links embedded in emails (invite accept, password reset).
	VendorAppURL string
	// AdminAppURL is the base URL of the admin Next.js app — used the same
	// way as VendorAppURL, but for admin-role accounts (e.g. the
	// forgot-password reset link must point at the admin app, not vendor's).
	AdminAppURL string
	// PublicAPIURL is this API's public base URL (e.g.
	// https://spottoapi.threelogiq.com). Traveller reset emails link to the
	// /reset-password page the API serves itself, since travellers have no
	// web app. Taken from config, never the request's Host header, so a
	// spoofed Host can't redirect reset links.
	PublicAPIURL string
	// SuperAdminEmails lists admin accounts (by email, case-insensitive)
	// granted the "super-admin" tier — e.g. viewing the cross-admin audit
	// log. Deliberately env-var-driven rather than a DB role/invite system:
	// there's no need yet for admin accounts to be self-service provisioned.
	SuperAdminEmails []string

	// AppEnv is "production" on the live deployment and anything else
	// (default "development") on local and staging. In production the demo
	// seeders refuse to run and the API locks the public demo logins at
	// boot (see seed.LockDemoAccounts).
	AppEnv string
	// AdminEmail/AdminPassword define the deployment's admin account. The
	// API creates or updates it at every boot (seed.EnsureAdmin), so the env
	// stays the source of truth and rotating it means changing the env and
	// redeploying. AdminEmail is also a super-admin.
	AdminEmail    string
	AdminPassword string

	// FirebaseServiceAccount is the JSON string of a Firebase service account
	// key, used for sending FCM push notifications to the Flutter app.
	FirebaseServiceAccount string

	// R2 (Cloudflare) powers image/video uploads. All four are optional —
	// unset means the media endpoints report themselves unconfigured (501)
	// rather than failing startup, exactly like Google OAuth. The bucket is
	// kept private: reads are proxied through our own API (see
	// internal/http/media.go's handleMediaRaw), not served straight from a
	// public R2/custom domain.
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2Bucket          string
}

// R2Configured reports whether all R2 settings needed to sign uploads and
// proxy reads are present.
func (c Config) R2Configured() bool {
	return c.R2AccountID != "" && c.R2AccessKeyID != "" && c.R2SecretAccessKey != "" && c.R2Bucket != ""
}

// IsProduction reports whether APP_ENV is "production".
func (c Config) IsProduction() bool { return c.AppEnv == "production" }

// minAdminPasswordLen is longer than the user-facing minimum because this
// account can see everything, and nobody types it often.
const minAdminPasswordLen = 12

// minSecretLen guards against trivially short/guessable secrets — anyone
// setting JWT_SECRET to "abc" defeats the point of requiring it at all.
const minSecretLen = 16

func Load() Config {
	loadDotEnv(".env")

	jwtSecret := mustGetenv("JWT_SECRET")
	if len(jwtSecret) < minSecretLen {
		log.Fatalf("JWT_SECRET must be at least %d characters — refusing to start with a weak secret", minSecretLen)
	}

	adminEmail := strings.ToLower(strings.TrimSpace(getenv("ADMIN_EMAIL", "")))
	adminPassword := getenv("ADMIN_PASSWORD", "")
	if (adminEmail == "") != (adminPassword == "") {
		log.Fatalf("ADMIN_EMAIL and ADMIN_PASSWORD must be set together")
	}
	if adminPassword != "" && len(adminPassword) < minAdminPasswordLen {
		log.Fatalf("ADMIN_PASSWORD must be at least %d characters — refusing to start with a weak admin password", minAdminPasswordLen)
	}
	superAdmins := splitNonEmpty(getenv("SUPER_ADMIN_EMAILS", ""))
	if adminEmail != "" {
		superAdmins = append(superAdmins, adminEmail)
	}

	totpKey, err := decodeTOTPKey(mustGetenv("TOTP_ENCRYPTION_KEY"))
	if err != nil {
		log.Fatalf("TOTP_ENCRYPTION_KEY: %v", err)
	}

	return Config{
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       mustGetenv("DATABASE_URL"),
		JWTSecret:         jwtSecret,
		CORSOrigins:       splitNonEmpty(mustGetenv("CORS_ORIGINS")),
		TOTPEncryptionKey: totpKey,

		SMTPHost:             getenv("SMTP_HOST", ""),
		SMTPPort:             getenv("SMTP_PORT", ""),
		SMTPUser:             getenv("SMTP_USER", ""),
		SMTPPass:             getenv("SMTP_PASS", ""),
		EmailFrom:            getenv("EMAIL_FROM", ""),
		GoogleClientID:       getenv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getenv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    getenv("GOOGLE_REDIRECT_URL", ""),
		AppleBundleID:        getenv("APPLE_BUNDLE_ID", "com.spotto.spotto"),
		GoogleCalendarAPIKey: getenv("GOOGLE_CALENDAR_API_KEY", ""),
		TomorrowIOAPIKey:     getenv("TOMORROW_IO_API_KEY", ""),
		GeminiAPIKey:         getenv("GEMINI_API_KEY", ""),
		GeminiChatModel:      getenv("GEMINI_CHAT_MODEL", ""),
		GeminiEmbedModel:     getenv("GEMINI_EMBED_MODEL", ""),
		VendorAppURL:         getenv("VENDOR_APP_URL", "http://localhost:3000"),
		AdminAppURL:          getenv("ADMIN_APP_URL", "http://localhost:3001"),
		PublicAPIURL:         strings.TrimRight(getenv("PUBLIC_API_URL", ""), "/"),
		GeminiVideoModel:      getenv("GEMINI_VIDEO_MODEL", ""),
		GooglePlacesAPIKey:    getenv("GOOGLE_PLACES_API_KEY", getenv("GOOGLE_CALENDAR_API_KEY", "")),
		YtdlpCookiesFile:      getenv("YTDLP_COOKIES_FILE", ""),
		InspirationDailyLimit: getenvInt("INSPIRATION_DAILY_LIMIT", 20),
		SuperAdminEmails:     superAdmins,
		AppEnv:               getenv("APP_ENV", "development"),
		AdminEmail:           adminEmail,
		AdminPassword:        adminPassword,
		FirebaseServiceAccount: getenv("FIREBASE_SERVICE_ACCOUNT_JSON", ""),

		R2AccountID:       sanitizedR2AccountID(getenv("R2_ACCOUNT_ID", "")),
		R2AccessKeyID:     getenv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getenv("R2_SECRET_ACCESS_KEY", ""),
		R2Bucket:          getenv("R2_BUCKET", ""),
	}
}

// sanitizedR2AccountID guards against the R2 endpoint silently becoming
// unreachable. A Cloudflare R2 account ID is always a bare 32-character hex
// string; the most common way to break it is pasting in the dashboard's
// example endpoint ("<ACCOUNT_ID>.r2.cloudflarestorage.com") instead of just
// the ID, which — once media.NewR2 appends ".r2.cloudflarestorage.com" of
// its own — builds a doubled, non-existent host and every upload fails with
// an opaque "tls: handshake failure" instead of a message that says what's
// actually wrong. Treating a malformed value as unset (rather than passing
// it through) keeps R2 in its existing "unconfigured" path.
func sanitizedR2AccountID(id string) string {
	if id == "" || !strings.ContainsAny(id, "./:") {
		return id
	}
	log.Printf("R2_ACCOUNT_ID looks malformed (%q) — it should be just the 32-character hex account ID, not the full endpoint hostname or URL; treating R2 as unconfigured", id)
	return ""
}

// splitNonEmpty splits a comma-separated env value, trimming whitespace and
// quotes, and dropping empty segments — so a trailing comma, extra spaces,
// or spurious quotation marks don't silently produce malformed entries.
func splitNonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	// Many users deploy via Docker Compose / Portainer where they might
	// inadvertently wrap the whole list in quotes. Strip them from the
	// entire string before splitting.
	s = strings.Trim(strings.TrimSpace(s), "\"'")

	var out []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(strings.Trim(strings.TrimSpace(p), "\"'")); t != "" {
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

// getenvInt reads a positive integer, logging and falling back on a bad value.
func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		log.Printf("config: %s=%q is not a positive integer, using %d", key, v, fallback)
		return fallback
	}
	return n
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

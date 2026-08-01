// Package auth provides password hashing, JWT issue/verify and chi middleware.
package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/spotto/server/internal/domain"
)

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

type Claims struct {
	Role     domain.Role `json:"role"`
	VendorID string      `json:"vendorId,omitempty"`
	// VendorRole distinguishes a vendor account's own owner from invited
	// staff — checked by RequireVendorOwner for team-management endpoints.
	VendorRole string `json:"vendorRole,omitempty"`
	// Purpose distinguishes a real session token (empty) from a short-lived
	// pending-2FA token (set to twoFactorPurpose) — Authenticated rejects
	// anything but a real session, so a pending token can't be used as one.
	Purpose string `json:"purpose,omitempty"`
	// TokenVersion is compared against the user's current token_version in
	// the database on every request; a mismatch means the token was issued
	// before a revoking event (e.g. 2FA disable) and is rejected even
	// though its signature is still valid.
	TokenVersion int `json:"tokenVersion"`
	jwt.RegisteredClaims
}

const twoFactorPurpose = "2fa-pending"

// sessionLifetime bounds how long a stolen/leaked token stays valid. There's
// no refresh-token flow yet, so this is a straight tradeoff against
// re-login frequency — 24h was chosen over the previous 7 days to shrink
// the exposure window significantly without requiring users to sign in
// more than once a day.
const sessionLifetime = 24 * time.Hour

type Tokens struct {
	secret []byte
}

func NewTokens(secret string) *Tokens { return &Tokens{secret: []byte(secret)} }

func (t *Tokens) Issue(u domain.User) (string, error) {
	c := Claims{
		Role:         u.Role,
		TokenVersion: u.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(sessionLifetime)),
		},
	}
	if u.VendorID != nil {
		c.VendorID = *u.VendorID
		c.VendorRole = u.VendorRole
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
}

func (t *Tokens) Verify(token string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(token, c, func(tok *jwt.Token) (any, error) {
		// Pin to the exact algorithm used at issuance (HS256), not just
		// "any HMAC variant" — narrows the trusted signing surface to
		// precisely what Issue/IssueTwoFactorPending produce.
		if tok.Method != jwt.SigningMethodHS256 {
			return nil, jwt.ErrSignatureInvalid
		}
		return t.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}

// IssueTwoFactorPending issues a short-lived token identifying a login that
// passed password auth but still needs a TOTP code. It cannot be used as a
// Bearer session token (Authenticated rejects any non-empty Purpose).
func (t *Tokens) IssueTwoFactorPending(userID string) (string, error) {
	c := Claims{
		Purpose: twoFactorPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
}

// VerifyTwoFactorPending validates a token from IssueTwoFactorPending and
// returns the pending user's ID.
func (t *Tokens) VerifyTwoFactorPending(token string) (string, error) {
	c, err := t.Verify(token)
	if err != nil {
		return "", err
	}
	if c.Purpose != twoFactorPurpose {
		return "", jwt.ErrTokenInvalidClaims
	}
	return c.Subject, nil
}

const mediaViewPurpose = "media-view"

// mediaViewLifetime bounds how long a signed /media/raw URL stays valid.
// Callers re-sign on every response (see Server.signMediaPath), so in
// practice a client never sees an expired one during normal use — this only
// bounds how long a leaked/shared URL (browser history, referrer, logs)
// keeps working.
const mediaViewLifetime = 24 * time.Hour

// IssueMediaView issues a short-lived, single-purpose token scoped to
// reading exactly one R2 object key — embedded as a query param on
// /media/raw URLs instead of a bearer session token, since plain <img>/
// <video>/<a> tags can't attach an Authorization header.
func (t *Tokens) IssueMediaView(key string) (string, error) {
	c := Claims{
		Purpose: mediaViewPurpose,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   key,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(mediaViewLifetime)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
}

// VerifyMediaView validates a token from IssueMediaView and returns the R2
// object key it authorises reading.
func (t *Tokens) VerifyMediaView(token string) (string, error) {
	c, err := t.Verify(token)
	if err != nil {
		return "", err
	}
	if c.Purpose != mediaViewPurpose {
		return "", jwt.ErrTokenInvalidClaims
	}
	return c.Subject, nil
}

// GenerateTOTPSecret creates a new secret + otpauth:// provisioning URI for
// an authenticator app (Google Authenticator/Authy/1Password-compatible).
func GenerateTOTPSecret(accountEmail string) (secret string, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Spotto",
		AccountName: accountEmail,
	})
	if err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// EncryptSecret seals plaintext (a TOTP secret) with AES-256-GCM under key,
// returning a base64 string safe to store in a TEXT column. A leaked
// database alone can't be used to derive live 2FA codes — the encryption
// key is only ever held in the server's environment, never in the DB.
func EncryptSecret(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// DecryptSecret reverses EncryptSecret.
func DecryptSecret(key []byte, encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	sealed, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(sealed) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// ValidateTOTP checks a 6-digit code against a secret (30s step, per RFC 6238 default).
func ValidateTOTP(secret, code string) bool {
	if secret == "" || code == "" {
		return false
	}
	valid, _ := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})
	return valid
}

type ctxKey struct{}

// FromContext returns the verified claims set by Authenticated.
func FromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(ctxKey{}).(*Claims)
	return c
}

// WithClaims stores verified claims in ctx, exported for callers (such as
// Server.Authenticated in package httpapi) that need to layer additional
// checks — e.g. a token_version lookup — on top of plain signature
// verification before claims become visible to handlers via FromContext.
func WithClaims(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// ParseBearer extracts and verifies a session token (rejecting pending-2FA
// tokens) from an Authorization header. Callers that also need to enforce
// token_version revocation (see Server.Authenticated in package httpapi)
// use this instead of a self-contained middleware.
func (t *Tokens) ParseBearer(header string) (*Claims, error) {
	raw, ok := strings.CutPrefix(header, "Bearer ")
	if !ok || raw == "" {
		return nil, jwt.ErrTokenMalformed
	}
	c, err := t.Verify(raw)
	if err != nil {
		return nil, err
	}
	if c.Purpose != "" {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return c, nil
}

// RequireRole gates a subtree to one role (use after Authenticated).
func RequireRole(role domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c := FromContext(r.Context())
			if c == nil || c.Role != role {
				writeErr(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireVendorOwner further restricts a vendor-role route (use after
// RequireRole(domain.RoleVendor)) to the account's own owner, not invited
// staff or managers — for team-management endpoints (invite/remove/revoke).
// Accounts predating vendor_role (or its zero value) are treated as
// owners, matching the migration's DEFAULT 'owner' for existing rows —
// this denylists staff/manager explicitly rather than allowlisting
// "owner" so that empty-string legacy rows still pass.
func RequireVendorOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := FromContext(r.Context())
		if c == nil || c.VendorRole == domain.VendorRoleStaff || c.VendorRole == domain.VendorRoleManager {
			writeErr(w, http.StatusForbidden, "owner only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireVendorManagerOrAbove further restricts a vendor-role route to
// owner and manager tiers, blocking staff — for endpoints staff has zero
// access to (offer/menu mutations, growth, reports). Same denylist shape
// as RequireVendorOwner so legacy empty-VendorRole rows still pass.
func RequireVendorManagerOrAbove(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := FromContext(r.Context())
		if c == nil || c.VendorRole == domain.VendorRoleStaff {
			writeErr(w, http.StatusForbidden, "manager or owner only")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Package auth provides password hashing, JWT issue/verify and chi middleware.
package auth

import (
	"context"
	"encoding/json"
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
	// Purpose distinguishes a real session token (empty) from a short-lived
	// pending-2FA token (set to twoFactorPurpose) — Authenticated rejects
	// anything but a real session, so a pending token can't be used as one.
	Purpose string `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

const twoFactorPurpose = "2fa-pending"

type Tokens struct {
	secret []byte
}

func NewTokens(secret string) *Tokens { return &Tokens{secret: []byte(secret)} }

func (t *Tokens) Issue(u domain.User) (string, error) {
	c := Claims{
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   u.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		},
	}
	if u.VendorID != nil {
		c.VendorID = *u.VendorID
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
}

func (t *Tokens) Verify(token string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(token, c, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
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

func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// Authenticated requires a valid Bearer token and stores claims in context.
func (t *Tokens) Authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := r.Header.Get("Authorization")
		raw, ok := strings.CutPrefix(h, "Bearer ")
		if !ok || raw == "" {
			writeErr(w, http.StatusUnauthorized, "missing token")
			return
		}
		c, err := t.Verify(raw)
		if err != nil || c.Purpose != "" {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, c)))
	})
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

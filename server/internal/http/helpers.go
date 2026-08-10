package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/spotto/server/internal/store/postgres"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// storeErr maps ErrNotFound to 404, everything else to 500.
func storeErr(w http.ResponseWriter, err error) {
	if errors.Is(err, postgres.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "The requested item could not be found.")
		return
	}
	writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeErr(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}
		writeErr(w, http.StatusBadRequest, "invalid json body")
		return false
	}
	return true
}

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I

// newID returns a random 12-char lowercase hex id with a prefix.
func newID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("%s_%x", prefix, b)
}

// newClaimCode returns SPOTTO-XXXX using the unambiguous alphabet.
func newClaimCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	out := make([]byte, 4)
	for i, v := range b {
		out[i] = codeAlphabet[int(v)%len(codeAlphabet)]
	}
	return "SPOTTO-" + string(out)
}

// newBookingCode returns BOOK-XXXX â€” a distinct namespace from
// newClaimCode's SPOTTO-XXXX so handleValidate can dispatch on prefix alone.
func newBookingCode() string {
	b := make([]byte, 4)
	rand.Read(b)
	out := make([]byte, 4)
	for i, v := range b {
		out[i] = codeAlphabet[int(v)%len(codeAlphabet)]
	}
	return "BOOK-" + string(out)
}

// sanitizeText trims whitespace and caps string length to maxLen if maxLen > 0.
func sanitizeText(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if maxLen > 0 {
		runes := []rune(s)
		if len(runes) > maxLen {
			return string(runes[:maxLen])
		}
	}
	return s
}

// validateTextLength returns 400 Bad Request if string length exceeds maxLen.
func validateTextLength(w http.ResponseWriter, fieldName, val string, maxLen int) bool {
	if maxLen > 0 && len([]rune(val)) > maxLen {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("%s cannot exceed %d characters", fieldName, maxLen))
		return false
	}
	return true
}
// randomHex returns a random hex string of n bytes (2*n hex chars).
func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

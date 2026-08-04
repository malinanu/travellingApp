package httpapi

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

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
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	writeErr(w, http.StatusInternalServerError, "internal error")
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
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

// newBookingCode returns BOOK-XXXX — a distinct namespace from
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

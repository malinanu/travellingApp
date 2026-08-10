package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeText(t *testing.T) {
	if got := sanitizeText("  hello world  ", 100); got != "hello world" {
		t.Errorf("expected 'hello world', got %q", got)
	}
	if got := sanitizeText("abcdefghij", 5); got != "abcde" {
		t.Errorf("expected 'abcde', got %q", got)
	}
}

func TestValidateTextLength(t *testing.T) {
	w := httptest.NewRecorder()
	if !validateTextLength(w, "Field", "short", 10) {
		t.Errorf("expected true for short string")
	}

	w2 := httptest.NewRecorder()
	longStr := strings.Repeat("a", 15)
	if validateTextLength(w2, "Field", longStr, 10) {
		t.Errorf("expected false for long string")
	}
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w2.Code)
	}
}

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeviceLabel(t *testing.T) {
	// X-Spotto-Device takes priority
	r1 := httptest.NewRequest(http.MethodGet, "/", nil)
	r1.Header.Set("X-Spotto-Device", "Google Pixel 9 Pro")
	r1.Header.Set("User-Agent", "Mozilla/5.0")
	if got := deviceLabel(r1); got != "Google Pixel 9 Pro" {
		t.Errorf("expected 'Google Pixel 9 Pro', got %q", got)
	}

	// Falls back to UserAgent
	r2 := httptest.NewRequest(http.MethodGet, "/", nil)
	r2.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)")
	if got := deviceLabel(r2); got != "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)" {
		t.Errorf("expected User-Agent fallback, got %q", got)
	}

	// Truncates long device label to 120 characters
	r3 := httptest.NewRequest(http.MethodGet, "/", nil)
	r3.Header.Set("X-Spotto-Device", strings.Repeat("A", 150))
	if got := deviceLabel(r3); len(got) != 120 || got != strings.Repeat("A", 120) {
		t.Errorf("expected 120 chars, got len %d: %q", len(got), got)
	}
}

func TestSessionClient(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := sessionClient(r); got != "app" {
		t.Errorf("expected default 'app', got %q", got)
	}

	r.Header.Set("X-Spotto-Client", "admin")
	if got := sessionClient(r); got != "admin" {
		t.Errorf("expected 'admin', got %q", got)
	}

	r.Header.Set("X-Spotto-Client", "vendor")
	if got := sessionClient(r); got != "vendor" {
		t.Errorf("expected 'vendor', got %q", got)
	}

	r.Header.Set("X-Spotto-Client", "unknown")
	if got := sessionClient(r); got != "app" {
		t.Errorf("expected 'app' for unknown client, got %q", got)
	}
}

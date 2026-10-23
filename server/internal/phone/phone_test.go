package phone

import (
	"strings"
	"testing"
)

func TestSriLankanSpellingsShareAKey(t *testing.T) {
	for _, raw := range []string{"+94 76 123 4567", "0761234567", "761234567", "94761234567", "+94-761-234-567", "(076) 123 4567"} {
		if got := Key(raw); got != "761234567" {
			t.Errorf("Key(%q) = %q, want 761234567", raw, got)
		}
		if got := Normalize(raw); got != "+94 761234567" {
			t.Errorf("Normalize(%q) = %q, want +94 761234567", raw, got)
		}
		if got := E164(raw); got != "+94761234567" {
			t.Errorf("E164(%q) = %q, want +94761234567", raw, got)
		}
		if !IsSriLankan(raw) {
			t.Errorf("IsSriLankan(%q) = false", raw)
		}
	}
}

func TestForeignNumbersKeepTheirCountryCode(t *testing.T) {
	cases := map[string]string{
		"+44 7700 900123":   "+447700900123",
		"+1 (415) 555-0100": "+14155550100",
		"+61 412 345 678":   "+61412345678",
	}
	for raw, want := range cases {
		if IsSriLankan(raw) {
			t.Errorf("IsSriLankan(%q) = true", raw)
		}
		if got := E164(raw); got != want {
			t.Errorf("E164(%q) = %q, want %q", raw, got, want)
		}
		if Key(raw) == Key("+94 761234567") {
			t.Errorf("Key(%q) collides with a Sri Lankan number", raw)
		}
	}
}

func TestEmpty(t *testing.T) {
	for _, raw := range []string{"", "   ", "+", "n/a"} {
		if Key(raw) != "" || Normalize(raw) != "" || IsSriLankan(raw) {
			t.Errorf("%q should normalise to nothing", raw)
		}
	}
}

func TestMasked(t *testing.T) {
	if got := Masked("0761234567"); got != "+94 •••• 4567" {
		t.Errorf("Masked = %q", got)
	}
	if got := Masked("+44 7700 900123"); got != "+•••• 0123" {
		t.Errorf("Masked = %q", got)
	}
}

func TestAmbiguousNumbersAreKeptAsTyped(t *testing.T) {
	// e.g. the Account page autosaving a half-typed number
	for _, raw := range []string{"0761", "12345", " 077 12 "} {
		if got := Normalize(raw); got != strings.TrimSpace(raw) {
			t.Errorf("Normalize(%q) = %q, want it kept as typed", raw, got)
		}
		if got := E164(raw); got != "" {
			t.Errorf("E164(%q) = %q, want no sendable number", raw, got)
		}
	}
}

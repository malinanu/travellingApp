package httpapi

import (
	"testing"
	"time"
)

func TestClampVibe(t *testing.T) {
	ptr := func(f float64) *float64 { return &f }
	cases := []struct {
		name string
		in   *float64
		want *float64
	}{
		{"nil stays nil", nil, nil},
		{"in range unchanged", ptr(3), ptr(3)},
		{"above range clamps to 5", ptr(10), ptr(5)},
		{"below range clamps to 1", ptr(-3), ptr(1)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := clampVibe(c.in)
			if (got == nil) != (c.want == nil) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			if got != nil && *got != *c.want {
				t.Fatalf("got %v, want %v", *got, *c.want)
			}
		})
	}
}

func TestGuideRateLimiting(t *testing.T) {
	limiter := newRateLimiter(3, 1*time.Minute)
	key := "user_test_123"

	for i := 1; i <= 3; i++ {
		if !limiter.allowInc(key) {
			t.Errorf("request %d should be allowed", i)
		}
	}

	if limiter.allowInc(key) {
		t.Errorf("request 4 should be rate limited")
	}
}


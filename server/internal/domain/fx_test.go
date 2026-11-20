package domain

import (
	"testing"
	"time"
)

func TestConvertFromLKR(t *testing.T) {
	cases := []struct {
		name   string
		lkr    int64
		rate   int64
		margin int
		want   int64
	}{
		// LKR 3,000.00 at 300 LKR/USD = USD 10.00
		{"no margin", 300000, 300_000_000, 0, 1000},
		// plus 3% = USD 10.30
		{"with margin", 300000, 300_000_000, 300, 1030},
		// LKR 4,500.00 at 302.5 = 14.876... → 14.88 (half up)
		{"rounds half up", 450000, 302_500_000, 0, 1488},
		// LKR 1.50 at 300 = 0.005 → 0.01
		{"half cent rounds up", 150, 300_000_000, 0, 1},
		{"zero amount", 0, 300_000_000, 300, 0},
		{"no rate", 300000, 0, 300, 0},
		// Large amounts don't overflow: LKR 50 billion.
		{"large", 5_000_000_000_000, 300_000_000, 300, 17_166_666_667},
	}
	for _, c := range cases {
		if got := ConvertFromLKR(c.lkr, c.rate, c.margin); got != c.want {
			t.Errorf("%s: ConvertFromLKR(%d, %d, %d) = %d, want %d", c.name, c.lkr, c.rate, c.margin, got, c.want)
		}
	}
}

func TestProportionalRefund(t *testing.T) {
	cases := []struct {
		name                  string
		refund, paid, charged int64
		want                  int64
	}{
		{"full refund returns the exact charge", 450000, 450000, 1533, 1533},
		{"half refund", 225000, 450000, 1533, 766}, // 766.5 rounds down
		{"refund above paid is capped", 500000, 450000, 1533, 1533},
		{"nothing to refund", 0, 450000, 1533, 0},
		{"same currency", 100000, 450000, 450000, 100000},
	}
	for _, c := range cases {
		if got := ProportionalRefund(c.refund, c.paid, c.charged); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestFxRateStale(t *testing.T) {
	now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	if (FxRate{SetAt: now.Add(-6 * 24 * time.Hour)}).Stale(now) {
		t.Error("a 6-day-old rate should still be usable")
	}
	if !(FxRate{SetAt: now.Add(-8 * 24 * time.Hour)}).Stale(now) {
		t.Error("an 8-day-old rate should be stale")
	}
}

package payments

import "testing"

func TestAmountRoundTrip(t *testing.T) {
	for _, c := range []struct {
		minor int64
		text  string
	}{{0, "0.00"}, {5, "0.05"}, {150000, "1500.00"}, {123456, "1234.56"}} {
		if got := FormatAmount(c.minor); got != c.text {
			t.Errorf("FormatAmount(%d) = %q, want %q", c.minor, got, c.text)
		}
		if got, err := ParseAmount(c.text); err != nil || got != c.minor {
			t.Errorf("ParseAmount(%q) = %d, %v", c.text, got, err)
		}
	}
	if got, _ := ParseAmount("1500.5"); got != 150050 {
		t.Errorf("one decimal: got %d", got)
	}
	if got, _ := ParseAmount("1500"); got != 150000 {
		t.Errorf("no decimals: got %d", got)
	}
	for _, bad := range []string{"", "1.234", "12a", "-5", "1.2.3"} {
		if _, err := ParseAmount(bad); err == nil {
			t.Errorf("ParseAmount(%q) should fail", bad)
		}
	}
}

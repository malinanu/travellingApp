package mail

import "testing"

func TestEnvelopeAddress(t *testing.T) {
	cases := map[string]string{
		"Spotto <mjjayaweera@gmail.com>": "mjjayaweera@gmail.com",
		"no-reply@spotto.dev":            "no-reply@spotto.dev",
		`"Spotto Team" <a@b.dev>`:        "a@b.dev",
		"not an address":                 "not an address",
	}
	for in, want := range cases {
		if got := envelopeAddress(in); got != want {
			t.Errorf("envelopeAddress(%q) = %q, want %q", in, got, want)
		}
	}
}

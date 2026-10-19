package mail

import (
	"strings"
	"testing"
)

func TestRender(t *testing.T) {
	out, err := Render(Email{
		Preheader: "Your code is inside",
		Title:     `<script>alert(1)</script> Booking`,
		Intro:     []string{"Hi there"},
		Code:      "AB12",
		Button:    Button{Label: "Open", URL: "https://spotto.example/x?token=abc"},
		Details:   []Detail{{Label: "Place", Value: `Café "<b>"`}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Your code is inside", "Hi there", "AB12", `href="https://spotto.example/x?token=abc"`, "&lt;script&gt;", "&lt;b&gt;"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered email missing %q", want)
		}
	}
	if strings.Contains(out, "<script>") || strings.Contains(out, "<b>") {
		t.Error("user text was not escaped")
	}

	bare, err := Render(Email{Title: "Only a title"})
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{"Button not working", "letter-spacing:4px", "display:none;max-height:0"} {
		if strings.Contains(bare, absent) {
			t.Errorf("empty section rendered: %q", absent)
		}
	}
}

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

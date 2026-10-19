package gemini

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseVideoPlaces(t *testing.T) {
	got, err := ParseVideoPlaces("```json\n" + `{"regionHint":"Phuket, Thailand","places":[
		{"name":" Kata Noodle House ","kind":"eat","city":"Phuket","evidence":"on_screen","atSec":12,"confidence":0.9},
		{"name":"kata noodle house","kind":"eat","evidence":"speech","confidence":0.8},
		{"name":"","kind":"see","evidence":"caption","confidence":0.5},
		{"name":"Big Buddha","kind":"temple","evidence":"subtitles","atSec":-3,"confidence":1.7}
	]}` + "\n```")
	if err != nil {
		t.Fatal(err)
	}
	if got.RegionHint != "Phuket, Thailand" || len(got.Places) != 2 {
		t.Fatalf("got %+v", got)
	}
	if p := got.Places[0]; p.Name != "Kata Noodle House" || *p.AtSec != 12 {
		t.Errorf("first place = %+v", p)
	}
	if p := got.Places[1]; p.Kind != "see" || p.Evidence != "caption" || p.AtSec != nil || p.Confidence != 1 {
		t.Errorf("normalised place = %+v", p)
	}

	var many []string
	for i := 0; i < 40; i++ {
		many = append(many, fmt.Sprintf(`{"name":"Place %d","kind":"see","evidence":"caption","confidence":0.5}`, i))
	}
	got, err = ParseVideoPlaces(`{"places":[` + strings.Join(many, ",") + `]}`)
	if err != nil || len(got.Places) != MaxVideoPlaces {
		t.Errorf("cap: %d places, err %v", len(got.Places), err)
	}

	if _, err := ParseVideoPlaces("Sorry, I can't watch videos."); err == nil {
		t.Error("expected an error for a non-JSON answer")
	}
}

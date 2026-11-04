package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
)

func TestUserPrivacyFieldsJSON(t *testing.T) {
	u := domain.User{
		ID:                 "usr_test_123",
		Email:              "test@example.com",
		VisibleToGroup:     true,
		CountVibeVotes:     false,
		PersonalisedOffers: true,
		CreatedAt:          time.Now(),
	}

	data, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal map error: %v", err)
	}

	if m["visibleToGroup"] != true {
		t.Errorf("expected visibleToGroup=true, got %v", m["visibleToGroup"])
	}
	if m["countVibeVotes"] != false {
		t.Errorf("expected countVibeVotes=false, got %v", m["countVibeVotes"])
	}
	if m["personalisedOffers"] != true {
		t.Errorf("expected personalisedOffers=true, got %v", m["personalisedOffers"])
	}

	var u2 domain.User
	if err := json.Unmarshal(data, &u2); err != nil {
		t.Fatalf("unmarshal user error: %v", err)
	}
	if !u2.VisibleToGroup || u2.CountVibeVotes || !u2.PersonalisedOffers {
		t.Errorf("roundtrip mismatch: %+v", u2)
	}
}

func TestCheckinJSON(t *testing.T) {
	now := time.Now().UTC()
	c := domain.Checkin{
		ID:          42,
		PlaceID:     "plc_galle_fort",
		TravellerID: "usr_alice",
		CreatedAt:   now,
	}
	b, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal Checkin: %v", err)
	}
	var c2 domain.Checkin
	if err := json.Unmarshal(b, &c2); err != nil {
		t.Fatalf("unmarshal Checkin: %v", err)
	}
	if c2.ID != 42 || c2.PlaceID != "plc_galle_fort" || c2.TravellerID != "usr_alice" {
		t.Fatalf("Checkin mismatch: %+v", c2)
	}
}

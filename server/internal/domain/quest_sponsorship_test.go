package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
)

func TestVendorSponsorshipSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	s := domain.VendorSponsorship{
		ID:                "spon_test_123",
		VendorID:          "ven_cafe_1",
		VendorName:        "Artisan Cafe",
		VendorCategory:    "Cafe",
		VendorTown:        "Ahangama",
		Title:             "Special Afternoon Tea",
		RewardDescription: "Free Pot of Ceylon Tea & Homemade Cake",
		RewardLimit:       15,
		IsActive:          true,
		Status:            "approved",
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed domain.VendorSponsorship
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.ID != "spon_test_123" {
		t.Errorf("expected ID 'spon_test_123', got %q", parsed.ID)
	}
	if parsed.Title != "Special Afternoon Tea" {
		t.Errorf("expected Title 'Special Afternoon Tea', got %q", parsed.Title)
	}
	if parsed.RewardDescription != "Free Pot of Ceylon Tea & Homemade Cake" {
		t.Errorf("expected RewardDescription, got %q", parsed.RewardDescription)
	}
	if parsed.RewardLimit != 15 {
		t.Errorf("expected RewardLimit 15, got %d", parsed.RewardLimit)
	}
	if !parsed.IsActive {
		t.Errorf("expected IsActive to be true")
	}
	if parsed.Status != "approved" {
		t.Errorf("expected Status 'approved', got %q", parsed.Status)
	}
}

func TestQuestScheduleSerialization(t *testing.T) {
	q := domain.QuestSchedule{
		Date:              "2026-12-25",
		VendorID:          "ven_surf_1",
		SponsorshipID:     "spon_surf_special",
		Goal:              5,
		SponsorName:       "Ahangama Surf Camp",
		RewardTitle:       "Free Board Wax & Espresso",
		RewardDescription: "Complimentary board wax bar and single espresso",
		VendorCategory:    "Surfing",
		VendorTown:        "Ahangama",
		Completions:       12,
		Redemptions:       8,
	}

	data, err := json.Marshal(q)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed domain.QuestSchedule
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Date != "2026-12-25" {
		t.Errorf("expected Date '2026-12-25', got %q", parsed.Date)
	}
	if parsed.SponsorshipID != "spon_surf_special" {
		t.Errorf("expected SponsorshipID 'spon_surf_special', got %q", parsed.SponsorshipID)
	}
	if parsed.RewardTitle != "Free Board Wax & Espresso" {
		t.Errorf("expected RewardTitle, got %q", parsed.RewardTitle)
	}
	if parsed.Completions != 12 || parsed.Redemptions != 8 {
		t.Errorf("expected 12 completions and 8 redemptions, got %d and %d", parsed.Completions, parsed.Redemptions)
	}
}

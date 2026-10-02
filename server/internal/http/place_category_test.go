package httpapi

import (
	"testing"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/seed"
)

func TestNormalizePlaceCategory(t *testing.T) {
	prev := domain.VendorCategories
	t.Cleanup(func() { domain.InitVendorCategories(prev) })
	domain.InitVendorCategories(seed.InitialCategories)

	// Legacy imported row: no categoryId, default teal color.
	m := map[string]any{"type": "food", "tags": []any{"Coffee Shop"}, "color": "teal"}
	normalizePlaceCategory(m)
	if m["categoryId"] != "coffee" {
		t.Errorf("categoryId = %v, want coffee", m["categoryId"])
	}
	if m["color"] != "coral" {
		t.Errorf("color = %v, want coral (food & drink)", m["color"])
	}

	// Already-categorised row keeps its id and a custom color.
	m = map[string]any{"categoryId": "hotels", "color": "sun"}
	normalizePlaceCategory(m)
	if m["categoryId"] != "hotels" || m["color"] != "sun" {
		t.Errorf("existing values were overwritten: %v", m)
	}
}

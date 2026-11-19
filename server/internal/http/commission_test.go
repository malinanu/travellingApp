package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
)

func TestCommissionRulePrecedence(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	past := time.Now().Add(-time.Hour)
	ruleID := func(s string) string { return "cr_test_" + s + "_" + f.suffix }

	resolve := func() int {
		r, err := f.store.ResolveCommissionRule(f.ctx, domain.CommissionPercent, f.vendor.ID, "gyms", domain.BookingTypeSession, time.Now())
		must(t, err)
		if r == nil {
			t.Fatal("no rule resolved")
		}
		return r.RateBps
	}
	base := resolve() // the seeded default (or whatever the database has)

	must(t, f.store.CreateCommissionRule(f.ctx, domain.CommissionRule{ID: ruleID("type"), Scope: domain.ScopeBookingType, ScopeKey: "session", Mode: domain.CommissionPercent, RateBps: 1100, ActiveFrom: past}))
	t.Cleanup(func() { f.store.Pool.Exec(f.ctx, `DELETE FROM commission_rules WHERE id LIKE 'cr_test_%'`) })
	if got := resolve(); got != 1100 {
		t.Errorf("booking type beats default: want 1100, got %d (default %d)", got, base)
	}
	must(t, f.store.CreateCommissionRule(f.ctx, domain.CommissionRule{ID: ruleID("cat"), Scope: domain.ScopeCategory, ScopeKey: "gyms", Mode: domain.CommissionPercent, RateBps: 1200, ActiveFrom: past}))
	if got := resolve(); got != 1200 {
		t.Errorf("category beats booking type: want 1200, got %d", got)
	}
	must(t, f.store.CreateCommissionRule(f.ctx, domain.CommissionRule{ID: ruleID("ven"), Scope: domain.ScopeVendor, ScopeKey: f.vendor.ID, Mode: domain.CommissionPercent, RateBps: 800, ActiveFrom: past}))
	if got := resolve(); got != 800 {
		t.Errorf("vendor beats category: want 800, got %d", got)
	}
	// A newer vendor rule that only starts tomorrow doesn't apply yet.
	must(t, f.store.CreateCommissionRule(f.ctx, domain.CommissionRule{ID: ruleID("ven_future"), Scope: domain.ScopeVendor, ScopeKey: f.vendor.ID, Mode: domain.CommissionPercent, RateBps: 500, ActiveFrom: time.Now().Add(24 * time.Hour)}))
	if got := resolve(); got != 800 {
		t.Errorf("future rule must not apply yet: want 800, got %d", got)
	}
	// ...and can be deleted, unlike one already in effect.
	must(t, f.store.DeleteCommissionRule(f.ctx, ruleID("ven_future")))
	if err := f.store.DeleteCommissionRule(f.ctx, ruleID("ven")); err == nil {
		t.Error("deleting a rule in effect should fail")
	}

	// A paid booking snapshots the vendor rate onto the booking.
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 5,
		PartySizeMin: 1, PartySizeMax: 4, PricingMode: domain.PricingPerPerson, PriceMinor: 250000, Currency: "LKR"})
	status, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(2), "partySize": 2})
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, out)
	}
	booking, _ := out["booking"].(map[string]any)
	if booking == nil {
		booking = out
	}
	if booking["amountMinor"].(float64) != 500000 || booking["commissionMinor"].(float64) != 40000 ||
		booking["vendorNetMinor"].(float64) != 460000 || booking["commissionMode"] != "percent" {
		t.Errorf("money snapshot: %v", booking)
	}
	if booking["status"] != "awaiting_payment" || booking["paymentStatus"] != "pending" {
		t.Errorf("a paid booking starts awaiting payment: %v", booking)
	}
}

func TestCommissionFreeTableGetsPerGuestFee(t *testing.T) {
	f := newBookingFixture(t, "restaurants", nil)
	svc := f.service(domain.BookableService{Type: domain.BookingTypeTable, DurationMins: 90, Capacity: 5, PartySizeMin: 1, PartySizeMax: 8})
	status, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(2), "partySize": 3})
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, out)
	}
	if out["status"] != "confirmed" || out["paymentStatus"] != "not_required" {
		t.Errorf("a free table is confirmed straight away: %v", out)
	}
	if out["commissionMode"] != "per_guest_fee" || out["feePerGuestMinor"].(float64) <= 0 || out["commissionMinor"].(float64) != 0 {
		t.Errorf("free table should carry the per-guest fee, charged only at check-in: %v", out)
	}
}

func TestCommissionServicePriceValidation(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	create := func(body map[string]any) int {
		body["name"], body["type"] = "Class", "session"
		code, _ := doJSON(t, http.MethodPost, f.url("/vendor/services"), f.vendorTk, body)
		return code
	}
	if got := create(map[string]any{"pricingMode": "per_person", "priceMinor": 0}); got != http.StatusBadRequest {
		t.Errorf("paid mode with no price: want 400, got %d", got)
	}
	if got := create(map[string]any{"pricingMode": "per_night", "priceMinor": 1000}); got != http.StatusBadRequest {
		t.Errorf("per-night on a session: want 400, got %d", got)
	}
	if got := create(map[string]any{"pricingMode": "per_person", "priceMinor": 1000, "currency": "USD"}); got != http.StatusBadRequest {
		t.Errorf("non-LKR currency: want 400, got %d", got)
	}
	if got := create(map[string]any{"pricingMode": "per_person", "priceMinor": 150000, "lateCancelRefundPct": 50}); got != http.StatusCreated {
		t.Errorf("valid paid service: want 201, got %d", got)
	}
}

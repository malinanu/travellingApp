package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
)

func TestExperienceOccurrenceBooking(t *testing.T) {
	f := newBookingFixture(t, "whale-watching", nil)
	// A host has no opening hours: an experience must not depend on them.
	must(t, f.store.UpdateVendorProfile(f.ctx, f.vendor.ID, []byte(`{}`)))

	status, out := doJSON(t, http.MethodPost, f.url("/vendor/services"), f.vendorTk, map[string]any{
		"name": "Dawn whale watch", "type": "experience", "scheduling": "occurrences", "durationMins": 240,
		"capacity": 12, "partySizeMin": 1, "partySizeMax": 6, "pricingMode": "per_person", "priceMinor": 1500000,
		"meetingPoint": map[string]any{"note": "Mirissa harbour, blue gate"}})
	if status != http.StatusCreated {
		t.Fatalf("create service: %d %v", status, out)
	}
	svcID := out["id"].(string)
	if out["capacityUnit"] != "guests" {
		t.Errorf("experiences default to counting guests, got %v", out["capacityUnit"])
	}

	start := time.Now().Add(96 * time.Hour).Truncate(time.Hour)
	status, out = doJSON(t, http.MethodPost, f.url("/vendor/services/"+svcID+"/occurrences"), f.vendorTk, map[string]any{
		"startAt": start.Format(time.RFC3339), "capacity": 5})
	if status != http.StatusCreated {
		t.Fatalf("create occurrence: %d %v", status, out)
	}
	occID := out["id"].(string)
	if end, _ := time.Parse(time.RFC3339, out["endAt"].(string)); !end.Equal(start.Add(240 * time.Minute)) {
		t.Errorf("end defaults to start + duration, got %v", out["endAt"])
	}
	if code, _ := doJSON(t, http.MethodPost, f.url("/vendor/services/"+svcID+"/occurrences"), f.vendorTk, map[string]any{
		"startAt": time.Now().Add(-time.Hour).Format(time.RFC3339)}); code != http.StatusBadRequest {
		t.Errorf("a past date: want 400, got %d", code)
	}

	book := func(party int, occ string) (int, map[string]any) {
		body := map[string]any{"serviceId": svcID, "partySize": party}
		if occ != "" {
			body["occurrenceId"] = occ
		}
		return doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, body)
	}
	if code, _ := book(2, ""); code != http.StatusBadRequest {
		t.Errorf("no date picked: want 400, got %d", code)
	}
	code, booking := book(3, occID)
	if code != http.StatusCreated {
		t.Fatalf("book 3 of 5 (no placeId, uses the host's place): %d %v", code, booking)
	}
	if booking["placeId"] == "" || booking["occurrenceId"] != occID || booking["amountMinor"].(float64) != 4500000 {
		t.Errorf("booking: %v", booking)
	}
	if got, _ := time.Parse(time.RFC3339, booking["startAt"].(string)); !got.Equal(start) {
		t.Errorf("booking takes the occurrence's time: %v", booking["startAt"])
	}
	if code, _ := book(3, occID); code != http.StatusConflict {
		t.Errorf("3+3 > occurrence capacity 5: want 409, got %d", code)
	}
	if code, _ := book(2, occID); code != http.StatusCreated {
		t.Errorf("3+2 = 5: want 201, got %d", code)
	}

	// Listing: the occurrence shows 0 remaining (holds count until expiry).
	_, list := doJSON(t, http.MethodGet, f.url("/places/"+f.placeID+"/services/"+svcID+"/availability"), f.travATk, nil)
	occs, _ := list["occurrences"].([]any)
	if len(occs) != 1 || occs[0].(map[string]any)["remaining"].(float64) != 0 {
		t.Errorf("availability: %v", list)
	}

	// Time can't move once booked; cancelling the date cancels its bookings.
	if code, _ := doJSON(t, http.MethodPut, f.url("/vendor/occurrences/"+occID), f.vendorTk, map[string]any{
		"startAt": start.Add(time.Hour).Format(time.RFC3339)}); code != http.StatusConflict {
		t.Errorf("moving a booked date: want 409, got %d", code)
	}
	status, out = doJSON(t, http.MethodPost, f.url("/vendor/occurrences/"+occID+"/cancel"), f.vendorTk, nil)
	if status != http.StatusOK || out["bookingsCancelled"].(float64) != 2 {
		t.Errorf("cancel date: %d %v", status, out)
	}
	if code, _ := book(1, occID); code == http.StatusCreated {
		t.Error("a cancelled date must not take bookings")
	}
}

func TestExperienceServiceTypesPerCategory(t *testing.T) {
	f := newBookingFixture(t, "restaurants", nil)
	create := func(typ string) int {
		code, _ := doJSON(t, http.MethodPost, f.url("/vendor/services"), f.vendorTk, map[string]any{
			"name": "x", "type": typ, "capacity": 10})
		return code
	}
	if got := create("table"); got != http.StatusCreated {
		t.Errorf("restaurant table: want 201, got %d", got)
	}
	if got := create("event"); got != http.StatusCreated {
		t.Errorf("restaurant event night: want 201, got %d", got)
	}
	if got := create("stay"); got != http.StatusBadRequest {
		t.Errorf("restaurant stay: want 400, got %d", got)
	}
}

func TestExperienceListingAndBlackout(t *testing.T) {
	f := newBookingFixture(t, "surf", nil)
	must(t, f.store.UpdateVendorProfile(f.ctx, f.vendor.ID, []byte(`{}`)))
	colombo, _ := time.LoadLocation("Asia/Colombo")
	blackout := time.Now().In(colombo).Add(48 * time.Hour).Format("2006-01-02")
	svc := f.service(domain.BookableService{Type: domain.BookingTypeExperience, Name: "Beginner surf lesson",
		DurationMins: 90, Capacity: 8, CapacityUnit: domain.CapacityGuests, PartySizeMin: 1, PartySizeMax: 4,
		PricingMode: domain.PricingPerPerson, PriceMinor: 600000, Currency: "LKR", BlackoutDates: []string{blackout}})

	_, out := doJSON(t, http.MethodGet, f.url("/places/"+f.placeID+"/services/"+svc.ID+"/availability?date="+blackout), f.travATk, nil)
	if slots, _ := out["slots"].([]any); len(slots) != 0 {
		t.Errorf("blackout day should have no slots, got %d", len(slots))
	}
	other := time.Now().In(colombo).Add(72 * time.Hour).Format("2006-01-02")
	_, out = doJSON(t, http.MethodGet, f.url("/places/"+f.placeID+"/services/"+svc.ID+"/availability?date="+other), f.travATk, nil)
	if slots, _ := out["slots"].([]any); len(slots) == 0 {
		t.Error("a host with no opening hours still offers its experience's slots")
	}

	req, _ := http.NewRequest(http.MethodGet, f.url("/experiences?category=surf"), nil)
	req.Header.Set("Authorization", "Bearer "+f.travATk)
	res, err := http.DefaultClient.Do(req)
	must(t, err)
	defer res.Body.Close()
	var listings []map[string]any
	json.NewDecoder(res.Body).Decode(&listings)
	found := false
	for _, l := range listings {
		if l["service"].(map[string]any)["id"] == svc.ID {
			found = l["placeId"] == f.placeID
		}
	}
	if !found {
		t.Errorf("the experience should be listed with its place, got %d listings", len(listings))
	}
}

func TestHostRegistration(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	f := newBookingFixture(t, "gyms", nil)
	email := "host_" + f.suffix + "@fixture.test"
	body, _ := json.Marshal(map[string]any{"email": email, "password": "testpass123!", "name": "Nimal",
		"role": "vendor", "businessName": "Nimal's lagoon tours", "category": "boat-trips", "kind": "host",
		"lat": 6.03, "lng": 80.21})
	res, err := http.Post(f.url("/auth/register"), "application/json", bytes.NewReader(body))
	must(t, err)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated && res.StatusCode != http.StatusOK {
		t.Fatalf("register: %d", res.StatusCode)
	}
	var vendorID, kind string
	must(t, store.Pool.QueryRow(f.ctx, `SELECT v.id, v.kind FROM vendors v JOIN users u ON u.vendor_id=v.id WHERE u.email=$1`, email).Scan(&vendorID, &kind))
	t.Cleanup(func() {
		store.Pool.Exec(f.ctx, `DELETE FROM users WHERE email=$1`, email)
		store.Pool.Exec(f.ctx, `DELETE FROM vendors WHERE id=$1`, vendorID)
	})
	if kind != "host" {
		t.Errorf("want kind host, got %q", kind)
	}
}

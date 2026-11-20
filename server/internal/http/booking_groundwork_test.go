package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
)

func TestBookingTransitionGuard(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 5, PartySizeMin: 1, PartySizeMax: 4})

	status, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(2), "partySize": 1})
	if status != http.StatusCreated {
		t.Fatalf("create: %d %v", status, out)
	}
	id := out["id"].(string)
	setStatus := func(st string) int {
		code, _ := doJSON(t, http.MethodPatch, f.url("/vendor/bookings/"+id+"/status"), f.vendorTk, map[string]any{"status": st})
		return code
	}
	if got := setStatus("confirmed"); got != http.StatusBadRequest {
		t.Errorf("setting confirmed by hand: want 400, got %d", got)
	}
	if got := setStatus("completed"); got != http.StatusOK {
		t.Fatalf("confirmed -> completed: want 200, got %d", got)
	}
	if got := setStatus("no_show"); got != http.StatusConflict {
		t.Errorf("completed -> no_show: want 409, got %d", got)
	}
	if code, _ := doJSON(t, http.MethodPost, f.url("/vendor/bookings/"+id+"/cancel"), f.vendorTk, nil); code == http.StatusOK {
		t.Errorf("cancelling a completed booking should fail, got 200")
	}
	if got := setStatus("completed"); got != http.StatusConflict {
		t.Errorf("completed -> completed: want 409, got %d", got)
	}
}

func TestBookingGuestCapacity(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 120, Capacity: 10,
		CapacityUnit: domain.CapacityGuests, PartySizeMin: 1, PartySizeMax: 10})
	start := slot(3)
	book := func(party int) int {
		code, _ := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
			"placeId": f.placeID, "serviceId": svc.ID, "startAt": start, "partySize": party})
		return code
	}
	if got := book(6); got != http.StatusCreated {
		t.Fatalf("6 of 10 guests: want 201, got %d", got)
	}
	if got := book(5); got != http.StatusConflict {
		t.Errorf("6+5 > 10 guests: want 409, got %d", got)
	}
	if got := book(4); got != http.StatusCreated {
		t.Errorf("6+4 = 10 guests: want 201, got %d", got)
	}
	if got := book(1); got != http.StatusConflict {
		t.Errorf("full: want 409, got %d", got)
	}
}

// A date in the availability query means a Sri Lankan calendar day: the
// first slot of an all-day service starts at 00:00 Colombo time (18:30 UTC
// the previous day), not 00:00 UTC.
func TestBookingSlotsUseColomboTime(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 1})
	colombo, _ := time.LoadLocation("Asia/Colombo")
	date := time.Now().In(colombo).Add(48 * time.Hour).Format("2006-01-02")
	status, out := doJSON(t, http.MethodGet, f.url("/places/"+f.placeID+"/services/"+svc.ID+"/availability?date="+date), f.travATk, nil)
	if status != http.StatusOK {
		t.Fatalf("availability: %d %v", status, out)
	}
	slots, _ := out["slots"].([]any)
	if len(slots) == 0 {
		t.Fatal("no slots returned")
	}
	first, _ := time.Parse(time.RFC3339, slots[0].(map[string]any)["start"].(string))
	if got := first.In(colombo).Format("2006-01-02 15:04"); got != date+" 00:00" {
		t.Errorf("first slot: want %s 00:00 Colombo, got %s", date, got)
	}
}

// A booking requested in UTC is checked against the vendor's hours in
// Colombo time: 03:00 UTC is 08:30 in Colombo, inside 08:00–17:00 hours.
func TestBookingHoursCheckedInColomboTime(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	must(t, f.store.UpdateVendorProfile(f.ctx, f.vendor.ID, []byte(`{"hours":{"monday":{"open":"08:00","close":"17:00"},"tuesday":{"open":"08:00","close":"17:00"},"wednesday":{"open":"08:00","close":"17:00"},"thursday":{"open":"08:00","close":"17:00"},"friday":{"open":"08:00","close":"17:00"},"saturday":{"open":"08:00","close":"17:00"},"sunday":{"open":"08:00","close":"17:00"}}}`)))
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 30, Capacity: 3})
	day := time.Now().UTC().Add(72 * time.Hour)
	start := time.Date(day.Year(), day.Month(), day.Day(), 3, 0, 0, 0, time.UTC).Format(time.RFC3339)
	status, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": svc.ID, "startAt": start, "partySize": 1})
	if status != http.StatusCreated {
		t.Errorf("03:00 UTC (08:30 Colombo) should be open: got %d %v", status, out)
	}
}

func TestServiceDeleteArchivesWhenBooked(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	unused := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 1})
	used := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 1})
	if code, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": used.ID, "startAt": slot(2), "partySize": 1}); code != http.StatusCreated {
		t.Fatalf("create: %d %v", code, out)
	}

	_, out := doJSON(t, http.MethodDelete, f.url("/vendor/services/"+unused.ID), f.vendorTk, nil)
	if out["status"] != "deleted" {
		t.Errorf("unbooked service: want deleted, got %v", out)
	}
	_, out = doJSON(t, http.MethodDelete, f.url("/vendor/services/"+used.ID), f.vendorTk, nil)
	if out["status"] != "archived" {
		t.Errorf("booked service: want archived, got %v", out)
	}
	bookings, err := f.store.BookingsByTraveller(f.ctx, f.travAID)
	must(t, err)
	if len(bookings) != 1 {
		t.Errorf("the booking should survive its service being removed, got %d bookings", len(bookings))
	}
	list, err := f.store.BookableServicesByVendor(f.ctx, f.vendor.ID, false)
	must(t, err)
	if len(list) != 0 {
		t.Errorf("archived service should be hidden from lists, got %d", len(list))
	}
	if code, _ := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": used.ID, "startAt": slot(4), "partySize": 1}); code == http.StatusCreated {
		t.Error("an archived service must not take new bookings")
	}
}

func TestSavedCardsNoLongerAccepted(t *testing.T) {
	f := newBookingFixture(t, "gyms", nil)
	code, _ := doJSON(t, http.MethodPost, f.url("/me/payment-methods"), f.travATk, map[string]any{
		"cardNumber": "4111111111111111", "expMonth": 12, "expYear": 2030, "holderName": "A"})
	if code != http.StatusGone {
		t.Errorf("want 410, got %d", code)
	}
}

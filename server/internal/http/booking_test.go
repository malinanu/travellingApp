package httpapi_test

// Integration tests for the booking system against the real Postgres store
// (see validate_test.go's doc comment â€” same self-skip-if-no-local-db
// convention). Covers: capacity enforcement under concurrency, closed-day
// rejection, cancel ownership, and the BOOK-/SPOTTO- prefix dispatch in
// handleValidate.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func doJSON(t *testing.T, method, url, token string, body any) (int, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, url, reader)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestBookingCapacityAndLifecycle(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	testTOTPKey := make([]byte, 32)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: testTOTPKey, VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")

	vendor := domain.Vendor{ID: "ven_booking_test", Name: "Booking Test Vendor", Category: "surf-school",
		Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, vendor))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_booking_vendor", Email: "bookingvendor@edgecase.test", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_booking_traveller_a", Email: "travellera@edgecase.test", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_booking_traveller_b", Email: "travellerb@edgecase.test", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))
	must(t, store.UpsertPlace(ctx, domain.Place{ID: "plc_booking_test", Name: "Booking Test Place", VendorID: &vendor.ID, Data: []byte(`{}`)}))

	// Capacity 1 session service, open every day all day, so the test isn't
	// sensitive to what weekday/time it happens to run.
	allDay := domain.ServiceAvailabilityDoc{}
	for _, day := range []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"} {
		allDay[day] = []domain.ServiceAvailabilityWindow{{Start: "00:00", End: "23:59"}}
	}
	svc := domain.BookableService{ID: "svc_booking_test", VendorID: vendor.ID, Type: domain.BookingTypeSession,
		Name: "Test Surf Lesson", DurationMins: 60, Capacity: 1, PartySizeMin: 1, PartySizeMax: 4,
		Availability: &allDay, Active: true}
	must(t, store.UpsertBookableService(ctx, svc))
	// Vendor must be open (Hours) for the requested slot too â€” checkVendorOpen
	// gates on this independently of the service's own Availability.
	must(t, store.UpdateVendorProfile(ctx, vendor.ID, []byte(`{"hours":{"monday":{"open":"00:00","close":"23:59"},"tuesday":{"open":"00:00","close":"23:59"},"wednesday":{"open":"00:00","close":"23:59"},"thursday":{"open":"00:00","close":"23:59"},"friday":{"open":"00:00","close":"23:59"},"saturday":{"open":"00:00","close":"23:59"},"sunday":{"open":"00:00","close":"23:59"}}}`)))

	travellerAToken := loginAs(t, srv, "travellera@edgecase.test", "testpass123")
	travellerBToken := loginAs(t, srv, "travellerb@edgecase.test", "testpass123")
	vendorToken := loginAs(t, srv, "bookingvendor@edgecase.test", "testpass123")

	slotStart := time.Now().Add(48 * time.Hour).Truncate(time.Hour).Format(time.RFC3339)
	bookingBody := map[string]any{
		"placeId": "plc_booking_test", "serviceId": svc.ID, "startAt": slotStart, "partySize": 2,
	}

	t.Run("closed day rejected", func(t *testing.T) {
		// Overwrite the vendor's hours to closed-every-day, confirm the
		// booking is rejected with 409, then restore.
		must(t, store.UpdateVendorProfile(ctx, vendor.ID, []byte(`{"hours":{}}`)))
		status, out := doJSON(t, http.MethodPost, srv.URL+"/bookings", travellerAToken, bookingBody)
		if status != http.StatusConflict {
			t.Errorf("want 409, got %d (%v)", status, out)
		}
		must(t, store.UpdateVendorProfile(ctx, vendor.ID, []byte(`{"hours":{"monday":{"open":"00:00","close":"23:59"},"tuesday":{"open":"00:00","close":"23:59"},"wednesday":{"open":"00:00","close":"23:59"},"thursday":{"open":"00:00","close":"23:59"},"friday":{"open":"00:00","close":"23:59"},"saturday":{"open":"00:00","close":"23:59"},"sunday":{"open":"00:00","close":"23:59"}}}`)))
	})

	var codeA string
	t.Run("first booking succeeds", func(t *testing.T) {
		status, out := doJSON(t, http.MethodPost, srv.URL+"/bookings", travellerAToken, bookingBody)
		if status != http.StatusCreated {
			t.Fatalf("want 201, got %d (%v)", status, out)
		}
		codeA, _ = out["code"].(string)
		if codeA == "" {
			t.Fatal("missing booking code")
		}
	})

	t.Run("second concurrent booking for the same full slot is rejected", func(t *testing.T) {
		// capacity=1, one confirmed booking already exists for this exact
		// slot â€” a second concurrent attempt must lose.
		status, out := doJSON(t, http.MethodPost, srv.URL+"/bookings", travellerBToken, bookingBody)
		if status != http.StatusConflict {
			t.Errorf("want 409 (full), got %d (%v)", status, out)
		}
	})

	t.Run("concurrent creates against a fresh capacity-1 slot: exactly one wins", func(t *testing.T) {
		slot2 := time.Now().Add(72 * time.Hour).Truncate(time.Hour).Format(time.RFC3339)
		body2 := map[string]any{"placeId": "plc_booking_test", "serviceId": svc.ID, "startAt": slot2, "partySize": 1}
		var wg sync.WaitGroup
		results := make([]int, 5)
		for i := range 5 {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				status, _ := doJSON(t, http.MethodPost, srv.URL+"/bookings", travellerAToken, body2)
				results[i] = status
			}(i)
		}
		wg.Wait()
		created := 0
		for _, s := range results {
			if s == http.StatusCreated {
				created++
			}
		}
		if created != 1 {
			t.Errorf("want exactly 1 of 5 concurrent bookings to succeed, got %d (statuses: %v)", created, results)
		}
	})

	t.Run("traveller B cannot cancel traveller A's booking", func(t *testing.T) {
		bookings, err := store.BookingsByTraveller(ctx, "usr_booking_traveller_a")
		must(t, err)
		if len(bookings) == 0 {
			t.Fatal("expected traveller A to have at least one booking")
		}
		status, _ := doJSON(t, http.MethodPost, srv.URL+"/bookings/"+bookings[0].ID+"/cancel", travellerBToken, nil)
		if status != http.StatusNotFound {
			t.Errorf("want 404 (not owner), got %d", status)
		}
	})

	t.Run("vendor check-in via /vendor/validate dispatches on BOOK- prefix", func(t *testing.T) {
		first := validate(t, srv, vendorToken, codeA)
		if first["result"] != "valid" {
			t.Fatalf("want valid, got %v", first)
		}
		if _, ok := first["booking"]; !ok {
			t.Error("expected a booking field in the response, not claim")
		}
		second := validate(t, srv, vendorToken, codeA)
		if second["result"] != "already_redeemed" {
			t.Errorf("want already_redeemed, got %v", second["result"])
		}
	})

	// cleanup so repeat runs stay idempotent
	store.Pool.Exec(ctx, `DELETE FROM bookings WHERE vendor_id='ven_booking_test'`)
	store.Pool.Exec(ctx, `DELETE FROM bookable_services WHERE vendor_id='ven_booking_test'`)
	store.Pool.Exec(ctx, `DELETE FROM places WHERE id='plc_booking_test'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id IN ('usr_booking_vendor','usr_booking_traveller_a','usr_booking_traveller_b')`)
	store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id='ven_booking_test'`)
}

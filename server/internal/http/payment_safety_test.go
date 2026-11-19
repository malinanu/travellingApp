package httpapi_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
)

// A hold past its expiry but not yet swept no longer holds the seat; if
// someone else takes it, the late payment must be refunded, not confirmed.
func TestPaymentLapsedUnsweptHoldIsNotOverbooked(t *testing.T) {
	f, svc, gw := paidFixture(t, 1)
	start := slot(4)
	a, _ := f.bookPaid(t, f.travATk, svc.ID, start, 1)
	f.store.Pool.Exec(f.ctx, `UPDATE bookings SET hold_expires_at=now()-interval '1 minute' WHERE id=$1`, a.id)
	b, code := f.bookPaid(t, f.travBTk, svc.ID, start, 1)
	if code != http.StatusCreated {
		t.Fatalf("the lapsed hold no longer holds the seat: want 201, got %d", code)
	}
	f.notify(t, gw, a.paymentID, "succeeded", "lapsed_a", a.amount, true)
	if row := f.bookingRow(t, a.id); row.Status == domain.BookingConfirmed || row.PaymentStatus != domain.PaymentRefunded {
		t.Errorf("A must be refunded, not confirmed into B's seat: %s/%s", row.Status, row.PaymentStatus)
	}
	f.notify(t, gw, b.paymentID, "succeeded", "ok_b", b.amount, true)
	if row := f.bookingRow(t, b.id); row.Status != domain.BookingConfirmed {
		t.Errorf("B keeps the seat: %s", row.Status)
	}
}

// After a full refund the vendor neither gains nor owes anything, even
// before the booking's original date.
func TestPaymentCancelledBookingSettlesNow(t *testing.T) {
	f, svc, gw := paidFixture(t, 5)
	b, _ := f.bookPaid(t, f.travATk, svc.ID, slot(10), 1)
	f.notify(t, gw, b.paymentID, "succeeded", "settle_"+b.id, b.amount, true)
	doJSON(t, http.MethodPost, f.url("/bookings/"+b.id+"/cancel"), f.travATk, nil)
	_, earn := doJSON(t, http.MethodGet, f.url("/vendor/earnings"), f.vendorTk, nil)
	if earn["payableMinor"].(float64) != 0 || earn["pendingMinor"].(float64) != 0 || earn["owedToSpottoMinor"].(float64) != 0 {
		t.Errorf("a fully refunded booking leaves nothing payable or owed: %v", earn)
	}
}

func TestPaymentHoldCapAndCheckInOfUnpaid(t *testing.T) {
	f, svc, _ := paidFixture(t, 20)
	var first paidBooking
	for i := 0; i < 3; i++ {
		b, code := f.bookPaid(t, f.travATk, svc.ID, slot(2+i), 1)
		if code != http.StatusCreated {
			t.Fatalf("hold %d: %d", i, code)
		}
		if i == 0 {
			first = b
		}
	}
	if _, code := f.bookPaid(t, f.travATk, svc.ID, slot(6), 1); code != http.StatusTooManyRequests {
		t.Errorf("a fourth unpaid hold: want 429, got %d", code)
	}
	if res := validate(t, f.srv, f.vendorTk, first.code); res["result"] != "not_paid" {
		t.Errorf("scanning an unpaid booking: want not_paid, got %v", res["result"])
	}
	doJSON(t, http.MethodPost, f.url("/bookings/"+first.id+"/cancel"), f.travATk, nil)
	if res := validate(t, f.srv, f.vendorTk, first.code); res["result"] != "cancelled" {
		t.Errorf("scanning a cancelled booking: want cancelled, got %v", res["result"])
	}
}

func TestPaymentStuckRefundIsSwept(t *testing.T) {
	f, svc, gw := paidFixture(t, 5)
	b, _ := f.bookPaid(t, f.travATk, svc.ID, slot(5), 1)
	f.notify(t, gw, b.paymentID, "succeeded", "stuck_"+b.id, b.amount, true)
	doJSON(t, http.MethodPost, f.url("/bookings/"+b.id+"/cancel"), f.travATk, nil)
	// Pretend the process died before asking the gateway.
	f.store.Pool.Exec(f.ctx, `UPDATE refunds SET status='requested', created_at=now()-interval '1 hour' WHERE booking_id=$1`, b.id)
	stuck, err := f.store.StaleRequestedRefunds(f.ctx, 10*time.Minute)
	must(t, err)
	found := false
	for _, r := range stuck {
		found = found || r.BookingID == b.id
	}
	if !found {
		t.Fatal("a refund stuck in requested should be found by the sweep")
	}
	_ = gw
}

func TestPaymentFakeGatewayIsOptIn(t *testing.T) {
	f := newBookingFixture(t, "gyms", func(c *config.Config) { c.FakeGatewaySecret = "" })
	if f.api.Payments != nil {
		t.Fatalf("without FAKE_GATEWAY_SECRET or PayHere there must be no gateway, got %T", f.api.Payments)
	}
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 5,
		PartySizeMin: 1, PartySizeMax: 4, PricingMode: domain.PricingPerPerson, PriceMinor: 100000, Currency: "LKR"})
	code, _ := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(2), "partySize": 1})
	if code != http.StatusServiceUnavailable {
		t.Errorf("paid booking with no gateway: want 503, got %d", code)
	}
	res, err := http.Post(f.url("/payments/fake/complete"), "application/x-www-form-urlencoded", nil)
	must(t, err)
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Errorf("the fake gateway's pages must not exist when it's off: got %d", res.StatusCode)
	}
}

package httpapi_test

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/payments"
	"github.com/spotto/server/internal/payments/fake"
)

// paidFixture is a fixture with a paid per-person session service at
// LKR 2,000, 24h free cancellation and 50% back later.
func paidFixture(t *testing.T, capacity int) (*bookingFixture, domain.BookableService, *fake.Gateway) {
	f := newBookingFixture(t, "gyms", nil)
	gw, ok := f.api.Payments.(*fake.Gateway)
	if !ok {
		t.Fatalf("tests expect the fake gateway, got %T", f.api.Payments)
	}
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: capacity,
		PartySizeMin: 1, PartySizeMax: 4, PricingMode: domain.PricingPerPerson, PriceMinor: 200000, Currency: "LKR",
		FreeCancelHours: 24, LateCancelRefundPct: 50})
	return f, svc, gw
}

type paidBooking struct {
	id, code, paymentID string
	amount              int64
}

func (f *bookingFixture) bookPaid(t *testing.T, token, serviceID, startAt string, party int) (paidBooking, int) {
	t.Helper()
	status, out := doJSON(t, http.MethodPost, f.url("/bookings"), token, map[string]any{
		"placeId": f.placeID, "serviceId": serviceID, "startAt": startAt, "partySize": party})
	if status != http.StatusCreated {
		return paidBooking{}, status
	}
	co, _ := out["checkout"].(map[string]any)
	if co == nil {
		t.Fatalf("a paid booking must come with checkout details: %v", out)
	}
	return paidBooking{id: out["id"].(string), code: out["code"].(string), paymentID: co["paymentId"].(string),
		amount: int64(out["amountMinor"].(float64))}, status
}

// notify posts a gateway notification, signed unless sign is false.
func (f *bookingFixture) notify(t *testing.T, gw *fake.Gateway, orderID, status, eventID string, amountMinor int64, sign bool) int {
	t.Helper()
	fields := map[string]string{"order_id": orderID, "payment_id": "gwpay_" + orderID, "status": status,
		"amount": payments.FormatAmount(amountMinor), "currency": "LKR", "event_id": eventID}
	if sign {
		fields["signature"] = gw.Sign(fields)
	} else {
		fields["signature"] = "forged"
	}
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	res, err := http.Post(f.url("/payments/fake/notify"), "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	must(t, err)
	res.Body.Close()
	return res.StatusCode
}

func (f *bookingFixture) bookingRow(t *testing.T, id string) domain.Booking {
	t.Helper()
	b, err := f.store.BookingByID(f.ctx, id)
	must(t, err)
	return b
}

func (f *bookingFixture) ledgerSum(t *testing.T, bookingID, kind string) (int64, int) {
	t.Helper()
	var sum int64
	var n int
	must(t, f.store.Pool.QueryRow(f.ctx, `SELECT COALESCE(sum(amount_minor),0), count(*) FROM ledger_entries WHERE booking_id=$1 AND kind=$2`, bookingID, kind).Scan(&sum, &n))
	return sum, n
}

func TestPaymentHappyPathAndIdempotency(t *testing.T) {
	f, svc, gw := paidFixture(t, 5)
	b, status := f.bookPaid(t, f.travATk, svc.ID, slot(3), 2)
	if status != http.StatusCreated || b.amount != 400000 {
		t.Fatalf("create: %d amount %d", status, b.amount)
	}

	// The checkout page forwards to the gateway with a signed form.
	res, err := http.Get(f.url("/payments/checkout/" + b.paymentID))
	must(t, err)
	page, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(page), `name="order_id" value="`+b.paymentID+`"`) {
		t.Errorf("checkout page: %d %s", res.StatusCode, page)
	}
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "form-action") || !strings.Contains(csp, "nonce-") {
		t.Errorf("checkout page CSP: %q", csp)
	}

	if got := f.notify(t, gw, b.paymentID, "succeeded", "ev1", b.amount, false); got != http.StatusBadRequest {
		t.Errorf("forged notification: want 400, got %d", got)
	}
	if row := f.bookingRow(t, b.id); row.Status != domain.BookingAwaitingPayment {
		t.Fatalf("a forged notification must change nothing, status %s", row.Status)
	}

	for i := 0; i < 2; i++ { // the gateway retries: same event twice
		if got := f.notify(t, gw, b.paymentID, "succeeded", "ev1", b.amount, true); got != http.StatusOK {
			t.Fatalf("notify: %d", got)
		}
	}
	row := f.bookingRow(t, b.id)
	if row.Status != domain.BookingConfirmed || row.PaymentStatus != domain.PaymentPaid || row.HoldExpiresAt != nil {
		t.Errorf("after payment: %s / %s", row.Status, row.PaymentStatus)
	}
	net, n := f.ledgerSum(t, b.id, "booking_net")
	if n != 1 || net != row.VendorNetMinor || net != 360000 {
		t.Errorf("ledger: want one booking_net of 360000 (10%% commission), got %d entries summing %d", n, net)
	}
	// A different notification for the same paid order is also harmless.
	f.notify(t, gw, b.paymentID, "succeeded", "ev2", b.amount, true)
	if _, n := f.ledgerSum(t, b.id, "booking_net"); n != 1 {
		t.Errorf("second notification must not double the ledger, got %d entries", n)
	}

	_, st := doJSON(t, http.MethodGet, f.url("/bookings/"+b.id+"/status"), f.travATk, nil)
	if st["status"] != "confirmed" || st["paymentStatus"] != "paid" || st["code"] != b.code {
		t.Errorf("status endpoint: %v", st)
	}
	if code, _ := doJSON(t, http.MethodGet, f.url("/bookings/"+b.id+"/status"), f.travBTk, nil); code != http.StatusNotFound {
		t.Errorf("someone else's booking status: want 404, got %d", code)
	}
}

func TestPaymentAmountMismatchIsNotApplied(t *testing.T) {
	f, svc, gw := paidFixture(t, 5)
	b, _ := f.bookPaid(t, f.travATk, svc.ID, slot(3), 1)
	f.notify(t, gw, b.paymentID, "succeeded", "evx", b.amount-100, true)
	if row := f.bookingRow(t, b.id); row.Status != domain.BookingAwaitingPayment {
		t.Errorf("a short payment must not confirm the booking, got %s", row.Status)
	}
}

func TestPaymentHoldExpiryAndLatePayment(t *testing.T) {
	f, svc, gw := paidFixture(t, 1)
	start := slot(4)
	a, _ := f.bookPaid(t, f.travATk, svc.ID, start, 1)
	if _, code := f.bookPaid(t, f.travBTk, svc.ID, start, 1); code != http.StatusConflict {
		t.Fatalf("a live hold takes the only seat: want 409, got %d", code)
	}
	expire := func(id string) {
		f.store.Pool.Exec(f.ctx, `UPDATE bookings SET hold_expires_at=now()-interval '1 minute' WHERE id=$1`, id)
		_, err := f.store.ExpireBookingHolds(f.ctx)
		must(t, err)
	}
	expire(a.id)
	if row := f.bookingRow(t, a.id); row.Status != domain.BookingExpired {
		t.Fatalf("hold should expire, got %s", row.Status)
	}
	b, code := f.bookPaid(t, f.travBTk, svc.ID, start, 1)
	if code != http.StatusCreated {
		t.Fatalf("seat freed by expiry: want 201, got %d", code)
	}
	// A's payment arrives late, but B holds the seat now: A is refunded.
	f.notify(t, gw, a.paymentID, "succeeded", "late_a", a.amount, true)
	if row := f.bookingRow(t, a.id); row.Status != domain.BookingExpired || row.PaymentStatus != domain.PaymentRefunded {
		t.Errorf("late payment for a taken seat: want expired+refunded, got %s/%s", row.Status, row.PaymentStatus)
	}
	if gw.RefundCount() == 0 {
		t.Error("the late payment should have been refunded through the gateway")
	}
	if _, n := f.ledgerSum(t, a.id, "booking_net"); n != 0 {
		t.Error("a refunded late payment owes the vendor nothing")
	}
	// B's hold also lapses, then B pays late while the seat is free: kept.
	expire(b.id)
	f.notify(t, gw, b.paymentID, "succeeded", "late_b", b.amount, true)
	if row := f.bookingRow(t, b.id); row.Status != domain.BookingConfirmed || row.PaymentStatus != domain.PaymentPaid {
		t.Errorf("late payment with the seat still free: want confirmed/paid, got %s/%s", row.Status, row.PaymentStatus)
	}
}

func TestPaymentCancellationRefunds(t *testing.T) {
	f, svc, gw := paidFixture(t, 10)
	pay := func(token, start string) paidBooking {
		b, code := f.bookPaid(t, token, svc.ID, start, 1)
		if code != http.StatusCreated {
			t.Fatalf("create: %d", code)
		}
		f.notify(t, gw, b.paymentID, "succeeded", "ev_"+b.id, b.amount, true)
		return b
	}

	// Well before the start: full refund, vendor's whole net reversed.
	early := pay(f.travATk, slot(5))
	_, preview := doJSON(t, http.MethodGet, f.url("/bookings/"+early.id+"/cancel-preview"), f.travATk, nil)
	if preview["refundMinor"].(float64) != 200000 || preview["fullRefund"] != true {
		t.Errorf("preview inside the free window: %v", preview)
	}
	_, out := doJSON(t, http.MethodPost, f.url("/bookings/"+early.id+"/cancel"), f.travATk, nil)
	if out["refundMinor"].(float64) != 200000 {
		t.Errorf("full refund expected: %v", out)
	}
	if rev, _ := f.ledgerSum(t, early.id, "refund_reversal"); rev != -180000 {
		t.Errorf("reversal should take back the whole net 180000, got %d", rev)
	}
	row := f.bookingRow(t, early.id)
	if row.PaymentStatus != domain.PaymentRefunded || row.CommissionMinor != 0 || row.VendorNetMinor != 0 {
		t.Errorf("after a full refund nobody keeps anything: %+v", row)
	}

	// Inside 24h: half back; the reversal and commission shrink pro rata.
	late := pay(f.travATk, time.Now().Add(5*time.Hour).Truncate(time.Hour).Format(time.RFC3339))
	_, out = doJSON(t, http.MethodPost, f.url("/bookings/"+late.id+"/cancel"), f.travATk, nil)
	if out["refundMinor"].(float64) != 100000 {
		t.Errorf("late cancel should refund 50%%: %v", out)
	}
	if rev, _ := f.ledgerSum(t, late.id, "refund_reversal"); rev != -90000 {
		t.Errorf("half refund reverses half the net: want -90000, got %d", rev)
	}
	if row := f.bookingRow(t, late.id); row.PaymentStatus != domain.PaymentPartiallyRefunded || row.CommissionMinor != 10000 || row.VendorNetMinor != 90000 {
		t.Errorf("after a half refund: commission %d net %d status %s", row.CommissionMinor, row.VendorNetMinor, row.PaymentStatus)
	}

	// The vendor cancelling always refunds everything, even late.
	vend := pay(f.travBTk, time.Now().Add(3*time.Hour).Truncate(time.Hour).Format(time.RFC3339))
	_, out = doJSON(t, http.MethodPost, f.url("/vendor/bookings/"+vend.id+"/cancel"), f.vendorTk, nil)
	if out["refundMinor"].(float64) != 200000 {
		t.Errorf("vendor cancel should refund in full: %v", out)
	}

	// A gateway that can't refund leaves it for the finance team.
	gw.RefundsUnsupported = true
	defer func() { gw.RefundsUnsupported = false }()
	manual := pay(f.travATk, slot(6))
	doJSON(t, http.MethodPost, f.url("/bookings/"+manual.id+"/cancel"), f.travATk, nil)
	var status string
	must(t, f.store.Pool.QueryRow(f.ctx, `SELECT status FROM refunds WHERE booking_id=$1`, manual.id).Scan(&status))
	if status != "manual" {
		t.Errorf("unsupported gateway refund: want manual, got %s", status)
	}

	// An unpaid hold just goes away, nothing to refund.
	held, _ := f.bookPaid(t, f.travATk, svc.ID, slot(7), 1)
	_, out = doJSON(t, http.MethodPost, f.url("/bookings/"+held.id+"/cancel"), f.travATk, nil)
	if out["status"] != "cancelled" || out["refundMinor"].(float64) != 0 {
		t.Errorf("cancel unpaid hold: %v", out)
	}
	if code, _ := doJSON(t, http.MethodPost, f.url("/bookings/"+held.id+"/cancel"), f.travATk, nil); code != http.StatusConflict {
		t.Errorf("cancelling twice: want 409, got %d", code)
	}
}

func TestPaymentPerGuestFeeOnCheckIn(t *testing.T) {
	f := newBookingFixture(t, "restaurants", nil)
	svc := f.service(domain.BookableService{Type: domain.BookingTypeTable, DurationMins: 90, Capacity: 10, PartySizeMin: 1, PartySizeMax: 8})
	book := func(party, day int) (string, string) {
		_, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
			"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(day), "partySize": party})
		return out["id"].(string), out["code"].(string)
	}
	seatedID, seatedCode := book(4, 2)
	first := validate(t, f.srv, f.vendorTk, seatedCode)
	if first["result"] != "valid" {
		t.Fatalf("check-in: %v", first)
	}
	fee, n := f.ledgerSum(t, seatedID, "per_guest_fee")
	row := f.bookingRow(t, seatedID)
	if n != 1 || fee != -4*row.FeePerGuestMinor || row.CommissionMinor != 4*row.FeePerGuestMinor {
		t.Errorf("4 seated guests: want one fee of %d, got %d entries summing %d (commission %d)", -4*row.FeePerGuestMinor, n, fee, row.CommissionMinor)
	}
	validate(t, f.srv, f.vendorTk, seatedCode)
	if _, n := f.ledgerSum(t, seatedID, "per_guest_fee"); n != 1 {
		t.Errorf("scanning twice must not charge twice, got %d entries", n)
	}

	noShowID, _ := book(2, 3)
	doJSON(t, http.MethodPatch, f.url("/vendor/bookings/"+noShowID+"/status"), f.vendorTk, map[string]any{"status": "no_show"})
	if _, n := f.ledgerSum(t, noShowID, "per_guest_fee"); n != 0 {
		t.Error("a no-show costs the vendor nothing")
	}

	manualID, _ := book(3, 4)
	doJSON(t, http.MethodPatch, f.url("/vendor/bookings/"+manualID+"/status"), f.vendorTk, map[string]any{"status": "completed"})
	if fee, n := f.ledgerSum(t, manualID, "per_guest_fee"); n != 1 || fee >= 0 {
		t.Errorf("marking seated by hand charges the fee like a scan: %d entries, %d", n, fee)
	}

	bal, err := f.store.VendorBalance(f.ctx, f.vendor.ID)
	must(t, err)
	if bal.PayableMinor != -7*row.FeePerGuestMinor {
		t.Errorf("vendor owes the fees for 7 seated guests: want %d, got %d", -7*row.FeePerGuestMinor, bal.PayableMinor)
	}
}

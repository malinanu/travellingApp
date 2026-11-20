package httpapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/payments/paypal/paypaltest"
)

// The PayPal tests charge in SGD so they never touch a USD rate someone
// set by hand in a development database.
const testPayPalCurrency = "SGD"

// payPalFixture is a paid-service fixture with PayPal (pointed at a
// stand-in for PayPal's API) running beside the fake gateway, and an SGD
// rate of 250 LKR set.
func payPalFixture(t *testing.T) (*bookingFixture, domain.BookableService, *paypaltest.Server, string) {
	t.Helper()
	pp := paypaltest.New()
	t.Cleanup(pp.Close)
	superEmail := fmt.Sprintf("super_pp_%d@fixture.test", time.Now().UnixNano())
	f := newBookingFixture(t, "gyms", func(c *config.Config) {
		c.SuperAdminEmails = []string{superEmail}
		c.PublicAPIURL = "http://api.spotto.test"
		c.PayPalClientID, c.PayPalClientSecret, c.PayPalWebhookID = "client", "secret", "WH-TEST"
		c.PayPalCurrency, c.PayPalFxMarginBps, c.PayPalAPIBaseURL = testPayPalCurrency, 300, pp.URL
	})
	pw, _ := auth.HashPassword("testpass123")
	superID := "usr_super_" + f.suffix
	must(t, f.store.CreateUser(f.ctx, domain.User{ID: superID, Email: superEmail, PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: time.Now()}))
	t.Cleanup(func() { f.store.Pool.Exec(f.ctx, `DELETE FROM users WHERE id=$1`, superID) })
	superTk := loginAs(t, f.srv, superEmail, "testpass123")
	setFxRate(t, f, "250", 0)
	t.Cleanup(func() { f.store.Pool.Exec(f.ctx, `DELETE FROM fx_rates WHERE currency=$1`, testPayPalCurrency) })
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 5,
		PartySizeMin: 1, PartySizeMax: 4, PricingMode: domain.PricingPerPerson, PriceMinor: 200000, Currency: "LKR",
		FreeCancelHours: 24, LateCancelRefundPct: 50})
	return f, svc, pp, superTk
}

// setFxRate stores an SGD rate set ageDays ago.
func setFxRate(t *testing.T, f *bookingFixture, lkrPerUnit string, ageDays int) {
	t.Helper()
	micros := int64(0)
	fmt.Sscan(lkrPerUnit, &micros)
	micros *= 1_000_000
	_, err := f.store.Pool.Exec(f.ctx, `
		INSERT INTO fx_rates (currency, lkr_per_unit_micros, set_by, set_at) VALUES ($1,$2,'test', now() - make_interval(days => $3))
		ON CONFLICT (currency) DO UPDATE SET lkr_per_unit_micros=EXCLUDED.lkr_per_unit_micros, set_at=EXCLUDED.set_at`,
		testPayPalCurrency, micros, ageDays)
	must(t, err)
}

// local rewrites a URL on the configured public API host to the test
// server.
func (f *bookingFixture) local(raw string) string {
	return strings.Replace(raw, "http://api.spotto.test", f.srv.URL, 1)
}

var noRedirects = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func getPage(t *testing.T, raw string) (int, string, string) {
	t.Helper()
	res, err := noRedirects.Get(raw)
	must(t, err)
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res.StatusCode, res.Header.Get("Location"), string(body)
}

// bookWithPayPal books 2 people (LKR 4,000) and returns the booking id,
// the checkout and PayPal's order id.
func bookWithPayPal(t *testing.T, f *bookingFixture, svc domain.BookableService, days int) (string, map[string]any, string) {
	t.Helper()
	code, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(days), "partySize": 2, "gateway": "paypal"})
	if code != http.StatusCreated {
		t.Fatalf("book with PayPal: %d %v", code, out)
	}
	co := out["checkout"].(map[string]any)
	status, loc, _ := getPage(t, f.local(co["url"].(string)))
	if status != http.StatusSeeOther || !strings.HasPrefix(loc, pp(t, loc)) {
		t.Fatalf("checkout page should redirect to PayPal, got %d %q", status, loc)
	}
	u, _ := url.Parse(loc)
	return out["id"].(string), co, u.Query().Get("token")
}

// pp returns the scheme and host of a URL, for prefix checks.
func pp(t *testing.T, raw string) string {
	u, err := url.Parse(raw)
	must(t, err)
	return u.Scheme + "://" + u.Host
}

func postWebhook(t *testing.T, f *bookingFixture, event map[string]any, signed bool) int {
	t.Helper()
	req := paypaltest.WebhookRequest(f.url("/payments/paypal/notify"), event, signed)
	body, _ := io.ReadAll(req.Body)
	out, _ := http.NewRequest(http.MethodPost, f.url("/payments/paypal/notify"), bytes.NewReader(body))
	out.Header = req.Header
	res, err := http.DefaultClient.Do(out)
	must(t, err)
	res.Body.Close()
	return res.StatusCode
}

func ledgerCount(t *testing.T, f *bookingFixture, bookingID string) int {
	t.Helper()
	var n int
	must(t, f.store.Pool.QueryRow(f.ctx, `SELECT count(*) FROM ledger_entries WHERE booking_id=$1 AND kind='booking_net'`, bookingID).Scan(&n))
	return n
}

func TestPaymentPayPalOptions(t *testing.T) {
	f, _, _, _ := payPalFixture(t)
	code, out := doJSON(t, http.MethodGet, f.url("/payments/options?amountMinor=400000"), "", nil)
	if code != http.StatusOK {
		t.Fatalf("options: %d %v", code, out)
	}
	opts := out["options"].([]any)
	byName := map[string]map[string]any{}
	for _, o := range opts {
		m := o.(map[string]any)
		byName[m["gateway"].(string)] = m
	}
	paypal := byName["paypal"]
	// LKR 4,000 at 250 = SGD 16.00, plus 3% = SGD 16.48.
	if paypal == nil || paypal["chargeCurrency"] != testPayPalCurrency || paypal["chargeAmountMinor"].(float64) != 1648 || paypal["converted"] != true {
		t.Fatalf("PayPal option = %v", paypal)
	}
	if fake := byName["fake"]; fake == nil || fake["chargeAmountMinor"].(float64) != 400000 || fake["converted"] != false {
		t.Fatalf("LKR option = %v", fake)
	}

	// Without a current rate PayPal isn't offered.
	setFxRate(t, f, "250", 8)
	_, out = doJSON(t, http.MethodGet, f.url("/payments/options?amountMinor=400000"), "", nil)
	for _, o := range out["options"].([]any) {
		if o.(map[string]any)["gateway"] == "paypal" {
			t.Fatal("PayPal must not be offered with a stale exchange rate")
		}
	}
}

func TestPaymentPayPalPayAndRefund(t *testing.T) {
	f, svc, pp, _ := payPalFixture(t)
	id, co, order := bookWithPayPal(t, f, svc, 3)
	if co["chargeCurrency"] != testPayPalCurrency || co["chargeAmountMinor"].(float64) != 1648 {
		t.Fatalf("checkout should charge SGD 16.48, got %v", co)
	}
	pays, err := f.store.PaymentsByBooking(f.ctx, id)
	must(t, err)
	if len(pays) != 1 || pays[0].Gateway != "paypal" || pays[0].AmountMinor != 1648 || pays[0].Currency != testPayPalCurrency ||
		pays[0].BookingAmountMinor != 400000 || pays[0].BookingCurrency != "LKR" || pays[0].FxRateMicros != 250_000_000 {
		t.Fatalf("payment row = %+v", pays)
	}

	// Coming back without approving captures nothing.
	ret := f.local(co["returnUrlPrefix"].(string)) + "?booking=" + id + "&gateway=paypal&token=" + order
	getPage(t, ret)
	if b := f.bookingRow(t, id); b.Status != domain.BookingAwaitingPayment {
		t.Fatalf("unapproved order must not confirm the booking: %s", b.Status)
	}
	// Asking for a fresh checkout keeps the traveller's choice of PayPal.
	code, retry := doJSON(t, http.MethodPost, f.url("/bookings/"+id+"/checkout"), f.travATk, nil)
	if code != http.StatusOK || retry["checkout"].(map[string]any)["gateway"] != "paypal" {
		t.Fatalf("retry checkout: %d %v", code, retry)
	}

	pp.Approve(order)
	if code, _, _ := getPage(t, ret+"&PayerID=BUYER1"); code != http.StatusOK {
		t.Fatalf("return page: %d", code)
	}
	b := f.bookingRow(t, id)
	if b.Status != domain.BookingConfirmed || b.PaymentStatus != domain.PaymentPaid {
		t.Fatalf("after capture: %s/%s", b.Status, b.PaymentStatus)
	}
	// PayPal's own webhook about the same capture changes nothing.
	if code := postWebhook(t, f, pp.CaptureCompletedEvent(order), true); code != http.StatusOK {
		t.Fatalf("capture webhook: %d", code)
	}
	if n := ledgerCount(t, f, id); n != 1 {
		t.Fatalf("one capture must credit the vendor once, got %d ledger rows", n)
	}
	if code := postWebhook(t, f, pp.CaptureCompletedEvent(order), false); code != http.StatusBadRequest {
		t.Fatalf("forged webhook: want 400, got %d", code)
	}

	// The traveller sees what they were charged, not just the LKR price.
	_, status := doJSON(t, http.MethodGet, f.url("/bookings/"+id+"/status"), f.travATk, nil)
	if status["chargeCurrency"] != testPayPalCurrency || status["chargeAmountMinor"].(float64) != 1648 || status["paidWith"] != "paypal" {
		t.Fatalf("status should show the SGD charge: %v", status)
	}
	_, mine := doJSONArray(t, http.MethodGet, f.url("/me/bookings"), f.travATk)
	listed := false
	for _, b := range mine {
		listed = listed || (b["id"] == id && b["chargeCurrency"] == testPayPalCurrency && b["chargeAmountMinor"].(float64) == 1648)
	}
	if !listed {
		t.Fatalf("My Bookings should show the SGD charge: %v", mine)
	}
	_, preview := doJSON(t, http.MethodGet, f.url("/bookings/"+id+"/cancel-preview"), f.travATk, nil)
	if preview["gatewayCurrency"] != testPayPalCurrency || preview["gatewayRefundMinor"].(float64) != 1648 || preview["gatewayPaidMinor"].(float64) != 1648 {
		t.Fatalf("cancel preview should be in SGD: %v", preview)
	}

	// Cancelling inside the free window refunds everything, in SGD.
	code, out := doJSON(t, http.MethodPost, f.url("/bookings/"+id+"/cancel"), f.travATk, nil)
	if code != http.StatusOK {
		t.Fatalf("cancel: %d %v", code, out)
	}
	if out["gatewayCurrency"] != testPayPalCurrency || out["gatewayRefundMinor"].(float64) != 1648 || out["refundMinor"].(float64) != 400000 {
		t.Fatalf("cancel response should give the SGD refund: %v", out)
	}
	refunds := pp.Refunds()
	if len(refunds) != 1 || refunds[0].Currency != testPayPalCurrency || refunds[0].Value != "16.48" {
		t.Fatalf("PayPal refunds = %+v", refunds)
	}
	rfs, err := f.store.RefundsForBooking(f.ctx, id)
	must(t, err)
	if len(rfs) != 1 || rfs[0].Status != "succeeded" || rfs[0].AmountMinor != 400000 || rfs[0].GatewayAmountMinor != 1648 {
		t.Fatalf("refund rows = %+v", rfs)
	}
}

func TestPaymentPayPalApprovedWebhookCaptures(t *testing.T) {
	f, svc, pp, _ := payPalFixture(t)
	id, _, order := bookWithPayPal(t, f, svc, 3)
	pp.Approve(order)
	// The traveller closed the browser: PayPal's approval webhook is
	// enough to take the money.
	if code := postWebhook(t, f, pp.ApprovedEvent(order), true); code != http.StatusOK {
		t.Fatalf("approved webhook: %d", code)
	}
	if b := f.bookingRow(t, id); b.Status != domain.BookingConfirmed {
		t.Fatalf("approved webhook should confirm the booking, got %s", b.Status)
	}
	if n := ledgerCount(t, f, id); n != 1 {
		t.Fatalf("ledger rows = %d", n)
	}
}

func TestPaymentPayPalLapsedHoldIsNotCharged(t *testing.T) {
	f, svc, pp, _ := payPalFixture(t)
	id, co, order := bookWithPayPal(t, f, svc, 3)
	_, err := f.store.Pool.Exec(f.ctx, `UPDATE bookings SET hold_expires_at = now() - interval '1 minute' WHERE id=$1`, id)
	must(t, err)
	pp.Approve(order)
	before := pp.CaptureCalls
	_, _, body := getPage(t, f.local(co["returnUrlPrefix"].(string))+"?booking="+id+"&gateway=paypal&token="+order)
	if pp.CaptureCalls != before {
		t.Fatal("a lapsed booking must not be captured")
	}
	if !strings.Contains(body, "expired") {
		t.Fatalf("return page should say the booking expired: %s", body)
	}
}

func TestPaymentPayPalRefusals(t *testing.T) {
	f, svc, _, _ := payPalFixture(t)
	book := func(gateway string) int {
		code, _ := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
			"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(4), "partySize": 1, "gateway": gateway})
		return code
	}
	if code := book("bitcoin"); code != http.StatusBadRequest {
		t.Errorf("unknown gateway: want 400, got %d", code)
	}
	setFxRate(t, f, "250", 8)
	if code := book("paypal"); code != http.StatusConflict {
		t.Errorf("PayPal with a stale rate: want 409, got %d", code)
	}
	var held int
	must(t, f.store.Pool.QueryRow(f.ctx, `SELECT count(*) FROM bookings WHERE service_id=$1`, svc.ID).Scan(&held))
	if held != 0 {
		t.Errorf("refused payments must not hold seats, %d bookings created", held)
	}
	if code := postWebhookTo(t, f, "/payments/stripe/notify"); code != http.StatusNotFound {
		t.Errorf("unknown gateway notify: want 404, got %d", code)
	}
}

func postWebhookTo(t *testing.T, f *bookingFixture, path string) int {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"id": "x"})
	res, err := http.Post(f.url(path), "application/json", bytes.NewReader(b))
	must(t, err)
	res.Body.Close()
	return res.StatusCode
}

func TestPaymentAdminFxRates(t *testing.T) {
	f, _, _, admin := payPalFixture(t)
	code, out := doJSON(t, http.MethodPut, f.url("/admin/finance/fx-rates"), admin, map[string]any{"currency": "sgd", "lkrPerUnit": "231.75"})
	if code != http.StatusOK || out["lkrPerUnitMicros"].(float64) != 231_750_000 {
		t.Fatalf("set rate: %d %v", code, out)
	}
	for _, bad := range []map[string]any{{"currency": "LKR", "lkrPerUnit": "1"}, {"currency": "SGD", "lkrPerUnit": "-3"}, {"currency": "SGD", "lkrPerUnit": "0.5"}} {
		if code, _ := doJSON(t, http.MethodPut, f.url("/admin/finance/fx-rates"), admin, bad); code != http.StatusBadRequest {
			t.Errorf("rate %v: want 400, got %d", bad, code)
		}
	}
	code, out = doJSON(t, http.MethodGet, f.url("/admin/finance/fx-rates"), admin, nil)
	if code != http.StatusOK || len(out["gateways"].([]any)) != 2 {
		t.Fatalf("list rates: %d %v", code, out)
	}
	if code, _ := doJSON(t, http.MethodPut, f.url("/admin/finance/fx-rates"), f.travATk, map[string]any{"currency": "SGD", "lkrPerUnit": "1"}); code != http.StatusForbidden && code != http.StatusUnauthorized {
		t.Errorf("a traveller must not set rates, got %d", code)
	}
}

func TestPaymentCancelRefundsThePaymentThatPaid(t *testing.T) {
	f, svc, pp, _ := payPalFixture(t)
	fakeGW := fakeGateway(t, f)
	// The traveller starts paying by card, gives up, and switches to PayPal.
	code, out := doJSON(t, http.MethodPost, f.url("/bookings"), f.travATk, map[string]any{
		"placeId": f.placeID, "serviceId": svc.ID, "startAt": slot(3), "partySize": 2, "gateway": "fake"})
	if code != http.StatusCreated {
		t.Fatalf("book: %d %v", code, out)
	}
	id := out["id"].(string)
	cardPayment := out["checkout"].(map[string]any)["paymentId"].(string)
	code, out = doJSON(t, http.MethodPost, f.url("/bookings/"+id+"/checkout"), f.travATk, map[string]any{"gateway": "paypal"})
	if code != http.StatusOK {
		t.Fatalf("switch to PayPal: %d %v", code, out)
	}
	co := out["checkout"].(map[string]any)
	_, loc, _ := getPage(t, f.local(co["url"].(string)))
	u, _ := url.Parse(loc)
	order := u.Query().Get("token")
	pp.Approve(order)
	getPage(t, f.local(co["returnUrlPrefix"].(string))+"?booking="+id+"&gateway=paypal&token="+order)
	if b := f.bookingRow(t, id); b.Status != domain.BookingConfirmed {
		t.Fatalf("PayPal should have confirmed the booking, got %s", b.Status)
	}
	// The abandoned card payment completes after all: it's refunded on its own.
	if code := f.notify(t, fakeGW, cardPayment, "succeeded", "late-card", 400000, true); code != http.StatusOK {
		t.Fatalf("late card payment: %d", code)
	}
	if code, out := doJSON(t, http.MethodPost, f.url("/bookings/"+id+"/cancel"), f.travATk, nil); code != http.StatusOK {
		t.Fatalf("cancel: %d %v", code, out)
	}
	// The cancellation refund goes back through PayPal, in SGD.
	if refunds := pp.Refunds(); len(refunds) != 1 || refunds[0].Value != "16.48" || refunds[0].Currency != testPayPalCurrency {
		t.Fatalf("PayPal refunds = %+v", refunds)
	}
	rfs, err := f.store.RefundsForBooking(f.ctx, id)
	must(t, err)
	byGateway := map[string]int{}
	for _, r := range rfs {
		byGateway[r.Gateway]++
	}
	if len(rfs) != 2 || byGateway["fake"] != 1 || byGateway["paypal"] != 1 {
		t.Fatalf("want one refund per gateway, got %+v", rfs)
	}
}

func TestPaymentPayPalChargebackIsFlagged(t *testing.T) {
	f, svc, pp, admin := payPalFixture(t)
	id, _, order := bookWithPayPal(t, f, svc, 3)
	pp.Approve(order)
	if code := postWebhook(t, f, pp.ApprovedEvent(order), true); code != http.StatusOK {
		t.Fatalf("approved webhook: %d", code)
	}
	pays, err := f.store.PaymentsByBooking(f.ctx, id)
	must(t, err)
	reversal := map[string]any{"id": "WH-REV-" + id, "event_type": "PAYMENT.CAPTURE.REVERSED",
		"resource": map[string]any{"id": "REV1", "custom_id": pays[0].ID, "invoice_id": pays[0].ID}}
	if code := postWebhook(t, f, reversal, true); code != http.StatusOK {
		t.Fatalf("reversal webhook: %d", code)
	}
	pay, err := f.store.PaymentByID(f.ctx, pays[0].ID)
	must(t, err)
	if pay.Status != "chargeback" {
		t.Fatalf("payment status = %s, want chargeback", pay.Status)
	}
	code, list := doJSONArray(t, http.MethodGet, f.url("/admin/finance/payment-issues"), admin)
	if code != http.StatusOK {
		t.Fatalf("payment issues: %d", code)
	}
	found := false
	for _, i := range list {
		found = found || (i["bookingId"] == id && strings.HasPrefix(i["outcome"].(string), "chargeback"))
	}
	if !found {
		t.Fatalf("the chargeback should be listed for finance: %v", list)
	}
	_, sum := doJSON(t, http.MethodGet, f.url("/admin/finance/summary"), admin, nil)
	if sum["chargebacks"].(float64) < 1 {
		t.Fatalf("summary should count the chargeback: %v", sum["chargebacks"])
	}
}

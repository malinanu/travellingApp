package paypal_test

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/spotto/server/internal/payments"
	"github.com/spotto/server/internal/payments/paypal"
	"github.com/spotto/server/internal/payments/paypal/paypaltest"
)

func newGateway(t *testing.T) (*paypal.Gateway, *paypaltest.Server) {
	t.Helper()
	srv := paypaltest.New()
	t.Cleanup(srv.Close)
	g := paypal.New(paypal.Config{ClientID: "id", ClientSecret: "secret", WebhookID: "WH1", Currency: "USD",
		APIBaseURL: srv.URL, RedirectHosts: []string{"127.0.0.1"}})
	return g, srv
}

func checkout(t *testing.T, g *paypal.Gateway, orderID string, amount int64) payments.CheckoutSession {
	t.Helper()
	s, err := g.CreateCheckout(context.Background(), payments.CheckoutRequest{OrderID: orderID, AmountMinor: amount, Currency: "USD",
		Description: "Surf lesson (BOOK-1234)", ReturnURL: "https://api.example/payments/return?booking=b1", CancelURL: "https://api.example/payments/return?cancelled=1"})
	if err != nil {
		t.Fatalf("CreateCheckout: %v", err)
	}
	return s
}

func TestCreateCheckoutAndCapture(t *testing.T) {
	g, srv := newGateway(t)
	s := checkout(t, g, "pay_1", 1533)
	if s.Method != payments.CheckoutGet || s.GatewayRef == "" {
		t.Fatalf("want a GET redirect with an order ref, got %+v", s)
	}
	o, _ := srv.Order(s.GatewayRef)
	if o.CustomID != "pay_1" || o.Value != "15.33" || o.Currency != "USD" || o.ReturnURL == "" {
		t.Fatalf("order sent to PayPal = %+v", o)
	}
	if _, err := g.Capture(context.Background(), s.GatewayRef); err == nil {
		t.Fatal("capturing before the buyer approves must fail")
	}
	srv.Approve(s.GatewayRef)
	ev, err := g.Capture(context.Background(), s.GatewayRef)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if ev.Status != payments.EventSucceeded || ev.OrderID != "pay_1" || ev.AmountMinor != 1533 || ev.Currency != "USD" || ev.GatewayPaymentID == "" {
		t.Fatalf("capture event = %+v", ev)
	}
	// Capturing again reports the same capture under the same event key.
	again, err := g.Capture(context.Background(), s.GatewayRef)
	if err != nil || again.EventKey != ev.EventKey {
		t.Fatalf("second capture = %+v, %v; want key %s", again, err, ev.EventKey)
	}
}

func TestCheckoutRejectsWrongCurrency(t *testing.T) {
	g, _ := newGateway(t)
	_, err := g.CreateCheckout(context.Background(), payments.CheckoutRequest{OrderID: "p", AmountMinor: 100, Currency: "LKR"})
	if err == nil {
		t.Fatal("PayPal must refuse to charge a currency it isn't configured for")
	}
}

func TestRedirectAllowed(t *testing.T) {
	live := paypal.New(paypal.Config{ClientID: "x", ClientSecret: "y"})
	for raw, want := range map[string]bool{
		"https://www.paypal.com/checkoutnow?token=1":         true,
		"https://www.sandbox.paypal.com/checkoutnow?token=1": false,
		"http://www.paypal.com/checkoutnow":                  false,
		"https://www.paypal.com.evil.example/":               false,
		"https://user@www.paypal.com/":                       false,
	} {
		u, _ := url.Parse(raw)
		if got := live.RedirectAllowed(u); got != want {
			t.Errorf("RedirectAllowed(%s) = %v, want %v", raw, got, want)
		}
	}
}

func TestVerifyNotify(t *testing.T) {
	g, srv := newGateway(t)
	s := checkout(t, g, "pay_2", 2000)
	srv.Approve(s.GatewayRef)
	if _, err := g.Capture(context.Background(), s.GatewayRef); err != nil {
		t.Fatal(err)
	}
	ev, err := g.VerifyNotify(paypaltest.WebhookRequest("/payments/paypal/notify", srv.CaptureCompletedEvent(s.GatewayRef), true))
	if err != nil {
		t.Fatalf("VerifyNotify: %v", err)
	}
	if ev.Status != payments.EventSucceeded || ev.OrderID != "pay_2" || ev.AmountMinor != 2000 {
		t.Fatalf("event = %+v", ev)
	}
	if _, err := g.VerifyNotify(paypaltest.WebhookRequest("/payments/paypal/notify", srv.CaptureCompletedEvent(s.GatewayRef), false)); !errors.Is(err, payments.ErrBadSignature) {
		t.Fatalf("forged webhook: err = %v, want ErrBadSignature", err)
	}
	req := paypaltest.WebhookRequest("/payments/paypal/notify", srv.CaptureCompletedEvent(s.GatewayRef), true)
	req.Header.Del("PAYPAL-TRANSMISSION-SIG")
	if _, err := g.VerifyNotify(req); !errors.Is(err, payments.ErrBadSignature) {
		t.Fatalf("missing headers: err = %v, want ErrBadSignature", err)
	}
	approved, err := g.VerifyNotify(paypaltest.WebhookRequest("/payments/paypal/notify", srv.ApprovedEvent(s.GatewayRef), true))
	if err != nil || approved.Status != payments.EventApproved || approved.CheckoutRef != s.GatewayRef {
		t.Fatalf("approved event = %+v, %v", approved, err)
	}
	other := map[string]any{"id": "WH-X", "event_type": "BILLING.PLAN.CREATED", "resource": map[string]any{}}
	if _, err := g.VerifyNotify(paypaltest.WebhookRequest("/payments/paypal/notify", other, true)); !errors.Is(err, payments.ErrIgnoredEvent) {
		t.Fatalf("unrelated event: err = %v, want ErrIgnoredEvent", err)
	}
}

func TestRefund(t *testing.T) {
	g, srv := newGateway(t)
	s := checkout(t, g, "pay_3", 3000)
	srv.Approve(s.GatewayRef)
	ev, _ := g.Capture(context.Background(), s.GatewayRef)
	req := payments.RefundRequest{IdempotencyKey: "rfd_1", GatewayPaymentID: ev.GatewayPaymentID, AmountMinor: 1500, Currency: "USD", Reason: "cancelled"}
	res, err := g.Refund(context.Background(), req)
	if err != nil || res.GatewayRefundID == "" {
		t.Fatalf("Refund = %+v, %v", res, err)
	}
	// Retrying the same refund doesn't pay twice.
	if _, err := g.Refund(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	refunds := srv.Refunds()
	if len(refunds) != 1 || refunds[0].Value != "15.00" || refunds[0].Currency != "USD" || refunds[0].CaptureID != ev.GatewayPaymentID {
		t.Fatalf("refunds at PayPal = %+v", refunds)
	}
}

func TestVerifyNotifyChargebacks(t *testing.T) {
	g, _ := newGateway(t)
	reversed := map[string]any{"id": "WH-REV", "event_type": "PAYMENT.CAPTURE.REVERSED",
		"resource": map[string]any{"id": "REFUND-REV", "status": "COMPLETED", "custom_id": "pay_9", "invoice_id": "pay_9",
			"amount": map[string]string{"currency_code": "USD", "value": "-20.00"}}}
	ev, err := g.VerifyNotify(paypaltest.WebhookRequest("/payments/paypal/notify", reversed, true))
	if err != nil || ev.Status != payments.EventChargeback || ev.OrderID != "pay_9" || ev.EventKey != "paypal:event:WH-REV" {
		t.Fatalf("reversal = %+v, %v", ev, err)
	}
	dispute := map[string]any{"id": "WH-DSP", "event_type": "CUSTOMER.DISPUTE.CREATED",
		"resource": map[string]any{"dispute_id": "PP-D-1", "disputed_transactions": []map[string]any{
			{"seller_transaction_id": "CAP1", "invoice_number": "pay_9"}}}}
	ev, err = g.VerifyNotify(paypaltest.WebhookRequest("/payments/paypal/notify", dispute, true))
	if err != nil || ev.Status != payments.EventChargeback || ev.OrderID != "pay_9" || ev.GatewayPaymentID != "CAP1" {
		t.Fatalf("dispute = %+v, %v", ev, err)
	}
	// A dispute that names no transaction of ours is acknowledged and dropped.
	empty := map[string]any{"id": "WH-DSP2", "event_type": "CUSTOMER.DISPUTE.CREATED", "resource": map[string]any{"dispute_id": "PP-D-2"}}
	if _, err := g.VerifyNotify(paypaltest.WebhookRequest("/payments/paypal/notify", empty, true)); !errors.Is(err, payments.ErrIgnoredEvent) {
		t.Fatalf("dispute without a transaction: err = %v", err)
	}
}

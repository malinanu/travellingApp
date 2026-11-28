package payhere

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/spotto/server/internal/payments"
)

func TestCheckoutAndNotifySignatures(t *testing.T) {
	g := New(Config{MerchantID: "1211149", MerchantSecret: "secret", Sandbox: true})
	session, err := g.CreateCheckout(context.Background(), payments.CheckoutRequest{
		OrderID: "pay_1", AmountMinor: 150000, Currency: "LKR", Description: "Surf lesson"})
	if err != nil {
		t.Fatal(err)
	}
	if session.URL != sandboxCheckoutURL || session.Fields["amount"] != "1500.00" ||
		session.Fields["hash"] != md5Upper("1211149pay_11500.00LKR"+md5Upper("secret")) {
		t.Errorf("checkout: %+v", session)
	}

	notify := func(code, sig string) (payments.NotifyEvent, error) {
		form := url.Values{"merchant_id": {"1211149"}, "order_id": {"pay_1"}, "payment_id": {"320025"},
			"payhere_amount": {"1500.00"}, "payhere_currency": {"LKR"}, "status_code": {code}, "md5sig": {sig},
			"card_no": {"************1292"}}
		req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return g.VerifyNotify(req)
	}
	good := md5Upper("1211149pay_11500.00LKR2" + md5Upper("secret"))
	ev, err := notify("2", good)
	if err != nil || ev.Status != payments.EventSucceeded || ev.AmountMinor != 150000 || ev.GatewayPaymentID != "320025" {
		t.Errorf("valid notify: %+v %v", ev, err)
	}
	if _, ok := ev.Raw["card_no"]; ok {
		t.Error("card details must not be kept")
	}
	if _, err := notify("2", strings.ToLower(good)); err != nil {
		t.Error("signature comparison should ignore case")
	}
	if _, err := notify("2", md5Upper("forged")); err != payments.ErrBadSignature {
		t.Errorf("forged signature: want ErrBadSignature, got %v", err)
	}
	// Changing the status without re-signing must fail.
	if _, err := notify("-2", good); err != payments.ErrBadSignature {
		t.Errorf("tampered status: want ErrBadSignature, got %v", err)
	}
}

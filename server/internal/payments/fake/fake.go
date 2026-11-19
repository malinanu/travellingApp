// Package fake is a stand-in payment gateway for development and tests.
// Its checkout page is served by Spotto itself, and its notifications are
// signed with HMAC-SHA256 so the real verification path is exercised.
package fake

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/spotto/server/internal/payments"
)

type Gateway struct {
	secret []byte
	// CheckoutBaseURL is where the fake checkout page lives (this API).
	CheckoutBaseURL string

	mu sync.Mutex
	// RefundsUnsupported makes Refund behave like a gateway with no refund
	// API, so the manual path can be tested.
	RefundsUnsupported bool
	Refunds            []payments.RefundRequest
}

func New(secret, checkoutBaseURL string) *Gateway {
	return &Gateway{secret: []byte(secret), CheckoutBaseURL: strings.TrimRight(checkoutBaseURL, "/")}
}

func (g *Gateway) Name() string { return "fake" }

func (g *Gateway) CreateCheckout(_ context.Context, req payments.CheckoutRequest) (payments.CheckoutSession, error) {
	return payments.CheckoutSession{
		URL: g.CheckoutBaseURL + "/payments/fake/pay",
		Fields: map[string]string{
			"order_id":   req.OrderID,
			"amount":     payments.FormatAmount(req.AmountMinor),
			"currency":   req.Currency,
			"return_url": req.ReturnURL,
			"cancel_url": req.CancelURL,
			"notify_url": req.NotifyURL,
		},
	}, nil
}

// Sign returns the signature a genuine notification with these fields
// carries.
func (g *Gateway) Sign(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if k != "signature" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	mac := hmac.New(sha256.New, g.secret)
	for _, k := range keys {
		mac.Write([]byte(k + "=" + fields[k] + "\n"))
	}
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyNotify accepts a form post of order_id, payment_id, status
// (succeeded|failed|cancelled|chargeback), amount, currency, event_id and
// signature.
func (g *Gateway) VerifyNotify(r *http.Request) (payments.NotifyEvent, error) {
	if err := r.ParseForm(); err != nil {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	fields := map[string]string{}
	for k := range r.PostForm {
		fields[k] = r.PostForm.Get(k)
	}
	if !hmac.Equal([]byte(g.Sign(fields)), []byte(fields["signature"])) {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	amount, err := payments.ParseAmount(fields["amount"])
	if err != nil {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	return payments.NotifyEvent{
		EventKey:         "fake:" + fields["event_id"],
		OrderID:          fields["order_id"],
		GatewayPaymentID: fields["payment_id"],
		Status:           payments.EventStatus(fields["status"]),
		AmountMinor:      amount,
		Currency:         fields["currency"],
		Raw:              fields,
	}, nil
}

func (g *Gateway) Refund(_ context.Context, req payments.RefundRequest) (payments.RefundResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.RefundsUnsupported {
		return payments.RefundResult{}, payments.ErrRefundUnsupported
	}
	g.Refunds = append(g.Refunds, req)
	return payments.RefundResult{GatewayRefundID: "fake_refund_" + req.OrderID}, nil
}

// RefundCount is how many refunds the gateway has issued.
func (g *Gateway) RefundCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.Refunds)
}

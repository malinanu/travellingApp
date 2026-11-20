// Package payhere connects Spotto to PayHere (payhere.lk), the Sri Lankan
// card and wallet gateway, through its hosted Checkout page and its
// notify_url callback.
//
// UNVERIFIED: the field names, hash formulas and status codes below follow
// PayHere's published Checkout API as known when this was written, but
// could not be checked against PayHere's current documentation from the
// build environment. Confirm them in the PayHere sandbox before going live.
// Automatic refunds are not implemented: refunds go to the finance team's
// manual queue (payments.ErrRefundUnsupported) until PayHere's refund API
// is confirmed for this merchant account.
package payhere

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/spotto/server/internal/payments"
)

const (
	sandboxCheckoutURL = "https://sandbox.payhere.lk/pay/checkout"
	liveCheckoutURL    = "https://www.payhere.lk/pay/checkout"
)

type Config struct {
	MerchantID     string
	MerchantSecret string
	Sandbox        bool
}

type Gateway struct{ cfg Config }

func New(cfg Config) *Gateway { return &Gateway{cfg: cfg} }

func (g *Gateway) Name() string { return "payhere" }

func (g *Gateway) DisplayName() string { return "PayHere" }

// ChargeCurrency: it charges in LKR, the bookings' own currency.
func (g *Gateway) ChargeCurrency() string { return "LKR" }

func md5Upper(s string) string {
	sum := md5.Sum([]byte(s))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// checkoutHash is MD5(merchant_id + order_id + amount + currency +
// MD5(merchant_secret)), uppercased.
func (g *Gateway) checkoutHash(orderID, amount, currency string) string {
	return md5Upper(g.cfg.MerchantID + orderID + amount + currency + md5Upper(g.cfg.MerchantSecret))
}

func (g *Gateway) CreateCheckout(_ context.Context, req payments.CheckoutRequest) (payments.CheckoutSession, error) {
	amount := payments.FormatAmount(req.AmountMinor)
	url := liveCheckoutURL
	if g.cfg.Sandbox {
		url = sandboxCheckoutURL
	}
	first, last := req.Customer.FirstName, req.Customer.LastName
	if first == "" {
		first = "Spotto"
	}
	if last == "" {
		last = "Traveller"
	}
	return payments.CheckoutSession{URL: url, Fields: map[string]string{
		"merchant_id": g.cfg.MerchantID,
		"return_url":  req.ReturnURL,
		"cancel_url":  req.CancelURL,
		"notify_url":  req.NotifyURL,
		"order_id":    req.OrderID,
		"items":       req.Description,
		"currency":    req.Currency,
		"amount":      amount,
		"first_name":  first,
		"last_name":   last,
		"email":       req.Customer.Email,
		"phone":       req.Customer.Phone,
		"address":     "",
		"city":        "",
		"country":     "Sri Lanka",
		"hash":        g.checkoutHash(req.OrderID, amount, req.Currency),
	}}, nil
}

// VerifyNotify checks md5sig = MD5(merchant_id + order_id + payhere_amount
// + payhere_currency + status_code + MD5(merchant_secret)), uppercased.
// status_code: 2 success, 0 pending, -1 cancelled, -2 failed,
// -3 chargeback.
func (g *Gateway) VerifyNotify(r *http.Request) (payments.NotifyEvent, error) {
	if err := r.ParseForm(); err != nil {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	f := r.PostForm
	merchantID, orderID := f.Get("merchant_id"), f.Get("order_id")
	amount, currency, code := f.Get("payhere_amount"), f.Get("payhere_currency"), f.Get("status_code")
	want := md5Upper(merchantID + orderID + amount + currency + code + md5Upper(g.cfg.MerchantSecret))
	if merchantID != g.cfg.MerchantID || subtle.ConstantTimeCompare([]byte(want), []byte(strings.ToUpper(f.Get("md5sig")))) != 1 {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	minor, err := payments.ParseAmount(amount)
	if err != nil {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	status := payments.EventFailed
	switch code {
	case "2":
		status = payments.EventSucceeded
	case "0":
		status = payments.EventPending
	case "-1":
		status = payments.EventCancelled
	case "-3":
		status = payments.EventChargeback
	}
	raw := map[string]string{}
	for k := range f {
		if k != "card_no" && k != "card_holder_name" && k != "card_expiry" {
			raw[k] = f.Get(k)
		}
	}
	return payments.NotifyEvent{
		EventKey:         "payhere:" + orderID + ":" + f.Get("payment_id") + ":" + code,
		OrderID:          orderID,
		GatewayPaymentID: f.Get("payment_id"),
		Status:           status,
		AmountMinor:      minor,
		Currency:         currency,
		Raw:              raw,
	}, nil
}

func (g *Gateway) Refund(context.Context, payments.RefundRequest) (payments.RefundResult, error) {
	return payments.RefundResult{}, payments.ErrRefundUnsupported
}

// Package payments is how Spotto takes money for bookings. Card details
// never touch Spotto: the traveller pays on the gateway's own hosted
// checkout page, and the gateway tells Spotto the result through a signed
// server-to-server notification.
package payments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// Customer is who is paying, as the gateway's checkout wants it.
type Customer struct {
	FirstName string
	LastName  string
	Email     string
	Phone     string
}

// CheckoutRequest asks the gateway to take one payment.
type CheckoutRequest struct {
	OrderID     string // unique per payment attempt
	AmountMinor int64
	Currency    string
	Description string
	Customer    Customer
	ReturnURL   string // where the traveller lands after paying
	CancelURL   string // where they land after giving up
	NotifyURL   string // where the gateway posts the result
}

// Checkout methods: how the traveller is sent to the gateway.
const (
	CheckoutPost = "POST" // an HTML form posted to URL with Fields
	CheckoutGet  = "GET"  // a plain redirect to URL
)

// CheckoutSession is how to send the traveller to the gateway.
type CheckoutSession struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
	// Method is CheckoutPost (the default when empty) or CheckoutGet.
	Method string `json:"method,omitempty"`
	// GatewayRef is the gateway's own id for this checkout, for gateways
	// that hand one back (PayPal's order id). It comes back on the
	// traveller's return and in some notifications.
	GatewayRef string `json:"-"`
}

// EventStatus is the outcome a gateway notification reports.
type EventStatus string

const (
	EventSucceeded  EventStatus = "succeeded"
	EventPending    EventStatus = "pending"
	EventFailed     EventStatus = "failed"
	EventCancelled  EventStatus = "cancelled"
	EventChargeback EventStatus = "chargeback"
	// EventApproved means the traveller approved the payment but the money
	// hasn't moved yet: the server must capture it (Capturer).
	EventApproved EventStatus = "approved"
)

// NotifyEvent is a verified gateway notification.
type NotifyEvent struct {
	// EventKey identifies this notification so a repeat is recognised.
	EventKey         string
	OrderID          string
	GatewayPaymentID string
	Status           EventStatus
	AmountMinor      int64
	Currency         string
	// CheckoutRef is the gateway's checkout id (CheckoutSession.GatewayRef),
	// set on EventApproved.
	CheckoutRef string
	Raw         map[string]string
}

// RefundRequest gives money back for a succeeded payment.
type RefundRequest struct {
	// IdempotencyKey is the same on every attempt for one refund, so a
	// retried request can't pay twice.
	IdempotencyKey   string
	OrderID          string
	GatewayPaymentID string
	AmountMinor      int64
	Currency         string
	Reason           string
}

type RefundResult struct {
	GatewayRefundID string
}

// ErrBadSignature means a notification didn't come from the gateway (or
// was altered); it must be ignored.
var ErrBadSignature = errors.New("payment notification signature invalid")

// ErrRefundUnsupported means the gateway can't refund automatically; the
// refund goes to the finance team to pay back by hand.
var ErrRefundUnsupported = errors.New("gateway can't refund automatically")

// ErrIgnoredEvent means a genuine notification Spotto has no use for; it
// is acknowledged and dropped.
var ErrIgnoredEvent = errors.New("payment notification not relevant")

// Gateway is one payment provider.
type Gateway interface {
	// Name is the gateway's id in URLs and stored payments ("payhere").
	Name() string
	// DisplayName is what travellers see ("PayHere").
	DisplayName() string
	// ChargeCurrency is the currency the gateway charges in. When it
	// isn't the booking's currency, the amount is converted first.
	ChargeCurrency() string
	CreateCheckout(ctx context.Context, req CheckoutRequest) (CheckoutSession, error)
	VerifyNotify(r *http.Request) (NotifyEvent, error)
	Refund(ctx context.Context, req RefundRequest) (RefundResult, error)
}

// Capturer is a gateway where an approved payment only moves money once
// Spotto captures it, server to server.
type Capturer interface {
	// Capture takes the money for an approved checkout and reports the
	// result as a notification would.
	Capture(ctx context.Context, checkoutRef string) (NotifyEvent, error)
}

// Redirector is a gateway that sends travellers to its checkout by
// redirect; RedirectAllowed vets each target before Spotto redirects.
type Redirector interface {
	RedirectAllowed(u *url.URL) bool
}

// FormatAmount renders minor units as a decimal string with two places
// ("1500.00"), the way gateways expect amounts.
func FormatAmount(minor int64) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	return fmt.Sprintf("%s%d.%02d", sign, minor/100, minor%100)
}

// ParseAmount reverses FormatAmount, accepting "1500", "1500.5" or
// "1500.00".
func ParseAmount(s string) (int64, error) {
	var whole, frac int64
	var fracDigits int
	seenDot := false
	if s == "" {
		return 0, errors.New("empty amount")
	}
	for _, r := range s {
		switch {
		case r == '.' && !seenDot:
			seenDot = true
		case r >= '0' && r <= '9':
			if seenDot {
				if fracDigits == 2 {
					return 0, fmt.Errorf("amount %q has more than two decimals", s)
				}
				frac = frac*10 + int64(r-'0')
				fracDigits++
			} else {
				whole = whole*10 + int64(r-'0')
			}
		default:
			return 0, fmt.Errorf("invalid amount %q", s)
		}
	}
	if fracDigits == 1 {
		frac *= 10
	}
	return whole*100 + frac, nil
}

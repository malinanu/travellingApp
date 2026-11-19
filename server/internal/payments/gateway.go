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

// CheckoutSession is how to send the traveller to the gateway: an HTML
// form posted to URL with Fields.
type CheckoutSession struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

// EventStatus is the outcome a gateway notification reports.
type EventStatus string

const (
	EventSucceeded  EventStatus = "succeeded"
	EventPending    EventStatus = "pending"
	EventFailed     EventStatus = "failed"
	EventCancelled  EventStatus = "cancelled"
	EventChargeback EventStatus = "chargeback"
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
	Raw              map[string]string
}

// RefundRequest gives money back for a succeeded payment.
type RefundRequest struct {
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

// Gateway is one payment provider.
type Gateway interface {
	Name() string
	CreateCheckout(ctx context.Context, req CheckoutRequest) (CheckoutSession, error)
	VerifyNotify(r *http.Request) (NotifyEvent, error)
	Refund(ctx context.Context, req RefundRequest) (RefundResult, error)
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

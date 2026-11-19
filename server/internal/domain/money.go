package domain

import (
	"math"
	"time"
)

// All money in Spotto is an integer number of minor units (cents) of a
// currency, never a float.

const CurrencyLKR = "LKR"

// PricingMode says how a BookableService's PriceMinor turns into what a
// booking costs.
type PricingMode string

const (
	PricingFree       PricingMode = "free"        // nothing to pay (e.g. most table reservations)
	PricingPerPerson  PricingMode = "per_person"  // price × party size
	PricingPerBooking PricingMode = "per_booking" // price once, whatever the party size
	PricingPerNight   PricingMode = "per_night"   // price × nights (one booking is one room)
)

// ValidPricingMode reports whether m is a known pricing mode.
func ValidPricingMode(m PricingMode) bool {
	switch m {
	case PricingFree, PricingPerPerson, PricingPerBooking, PricingPerNight:
		return true
	}
	return false
}

// CommissionMode is how Spotto earns on a booking.
type CommissionMode string

const (
	CommissionNone CommissionMode = "none"
	// CommissionPercent takes a share of what the traveller paid.
	CommissionPercent CommissionMode = "percent"
	// CommissionPerGuestFee charges the vendor a fixed fee for each guest of
	// a free booking, only once the booking is checked in.
	CommissionPerGuestFee CommissionMode = "per_guest_fee"
)

// PaymentStatus tracks the money side of a booking.
type PaymentStatus string

const (
	PaymentNotRequired       PaymentStatus = "not_required"
	PaymentPending           PaymentStatus = "pending"
	PaymentPaid              PaymentStatus = "paid"
	PaymentRefunded          PaymentStatus = "refunded"
	PaymentPartiallyRefunded PaymentStatus = "partially_refunded"
	PaymentFailed            PaymentStatus = "failed"
)

// CommissionScope is what a CommissionRule applies to. More specific
// scopes win: vendor, then category, then booking type, then default.
type CommissionScope string

const (
	ScopeDefault     CommissionScope = "default"
	ScopeBookingType CommissionScope = "booking_type"
	ScopeCategory    CommissionScope = "category"
	ScopeVendor      CommissionScope = "vendor"
)

// CommissionRule sets Spotto's cut for one scope from ActiveFrom onward.
// A percent rule (RateBps, basis points: 1000 = 10%) applies to paid
// bookings; a per_guest_fee rule applies to free ones.
type CommissionRule struct {
	ID               string          `json:"id"`
	Scope            CommissionScope `json:"scope"`
	ScopeKey         string          `json:"scopeKey"`
	Mode             CommissionMode  `json:"mode"`
	RateBps          int             `json:"rateBps"`
	FeePerGuestMinor int64           `json:"feePerGuestMinor"`
	ActiveFrom       time.Time       `json:"activeFrom"`
	Note             string          `json:"note"`
	CreatedBy        string          `json:"createdBy"`
	CreatedAt        time.Time       `json:"createdAt"`
}

// Nights is the number of nights between two dates (a stay's check-in and
// check-out), at least 1.
func Nights(startAt, endAt time.Time) int {
	n := int(math.Round(endAt.Sub(startAt).Hours() / 24))
	if n < 1 {
		return 1
	}
	return n
}

// QuoteBooking is what a booking of svc for partySize between startAt and
// endAt costs the traveller.
func QuoteBooking(svc BookableService, partySize int, startAt, endAt time.Time) int64 {
	if partySize < 1 {
		partySize = 1
	}
	switch svc.PricingMode {
	case PricingPerPerson:
		return svc.PriceMinor * int64(partySize)
	case PricingPerBooking:
		return svc.PriceMinor
	case PricingPerNight:
		return svc.PriceMinor * int64(Nights(startAt, endAt))
	}
	return 0
}

// CommissionModeFor is which kind of rule a booking of this amount needs:
// a percent rule when the traveller pays, a per-guest fee rule when free.
func CommissionModeFor(amountMinor int64) CommissionMode {
	if amountMinor > 0 {
		return CommissionPercent
	}
	return CommissionPerGuestFee
}

// shareOf is amount × num / den rounded half up, for non-negative inputs.
func shareOf(amount, num, den int64) int64 {
	if den == 0 {
		return 0
	}
	return (amount*num + den/2) / den
}

// ApplyCommission snapshots rule onto b (which must already carry its
// AmountMinor): the commission mode and rate, Spotto's commission and the
// vendor's net. With no rule, or a rule of the wrong kind for the amount,
// Spotto earns nothing on the booking. A per-guest fee is only charged at
// check-in, so it adds no commission yet.
func ApplyCommission(b *Booking, rule *CommissionRule) {
	b.CommissionMode, b.CommissionRateBps, b.FeePerGuestMinor = CommissionNone, 0, 0
	b.CommissionMinor, b.VendorNetMinor = 0, b.AmountMinor
	if rule == nil || rule.Mode != CommissionModeFor(b.AmountMinor) {
		return
	}
	switch rule.Mode {
	case CommissionPercent:
		b.CommissionMode, b.CommissionRateBps = CommissionPercent, rule.RateBps
		b.CommissionMinor = shareOf(b.AmountMinor, int64(rule.RateBps), 10000)
		b.VendorNetMinor = b.AmountMinor - b.CommissionMinor
	case CommissionPerGuestFee:
		if rule.FeePerGuestMinor > 0 {
			b.CommissionMode, b.FeePerGuestMinor = CommissionPerGuestFee, rule.FeePerGuestMinor
		}
	}
}

// PerGuestFee is what the vendor owes for a checked-in free booking.
func PerGuestFee(b Booking) int64 {
	if b.CommissionMode != CommissionPerGuestFee {
		return 0
	}
	party := b.PartySize
	if party < 1 {
		party = 1
	}
	return b.FeePerGuestMinor * int64(party)
}

// RefundFor is how much of a paid booking goes back to the traveller if it
// is cancelled at now. A vendor cancellation always refunds everything.
// A traveller cancelling at least FreeCancelHours before the start gets
// everything back; later, LateCancelRefundPct of the price. Anything
// already refunded is deducted.
func RefundFor(b Booking, now time.Time, byVendor bool) int64 {
	if b.PaymentStatus != PaymentPaid && b.PaymentStatus != PaymentPartiallyRefunded {
		return 0
	}
	remaining := b.AmountMinor - b.RefundedMinor
	if remaining <= 0 {
		return 0
	}
	if byVendor || !now.After(b.StartAt.Add(-time.Duration(b.FreeCancelHours)*time.Hour)) {
		return remaining
	}
	due := shareOf(b.AmountMinor, int64(b.LateCancelRefundPct), 100) - b.RefundedMinor
	if due < 0 {
		return 0
	}
	if due > remaining {
		return remaining
	}
	return due
}

// RefundReversal is how much of the vendor's net a refund takes back:
// the refund is split between vendor and Spotto in the same proportion as
// the original payment, so Spotto returns its commission on the refunded
// part too.
func RefundReversal(b Booking, refundMinor int64) int64 {
	if b.AmountMinor <= 0 || refundMinor <= 0 {
		return 0
	}
	return shareOf(b.VendorNetMinor, refundMinor, b.AmountMinor)
}

package domain

import (
	"testing"
	"time"
)

func TestQuoteBooking(t *testing.T) {
	start := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name  string
		mode  PricingMode
		party int
		end   time.Time
		want  int64
	}{
		{"free", PricingFree, 4, start.Add(time.Hour), 0},
		{"per person", PricingPerPerson, 3, start.Add(time.Hour), 3 * 450000},
		{"per booking", PricingPerBooking, 6, start.Add(time.Hour), 450000},
		{"per night, 3 nights", PricingPerNight, 2, start.AddDate(0, 0, 3), 3 * 450000},
		{"per night, same day counts 1", PricingPerNight, 2, start, 450000},
		{"party below 1 counts as 1", PricingPerPerson, 0, start.Add(time.Hour), 450000},
	}
	for _, c := range cases {
		svc := BookableService{PriceMinor: 450000, PricingMode: c.mode}
		if got := QuoteBooking(svc, c.party, start, c.end); got != c.want {
			t.Errorf("%s: want %d, got %d", c.name, c.want, got)
		}
	}
}

func TestApplyCommission(t *testing.T) {
	percent := &CommissionRule{Mode: CommissionPercent, RateBps: 1250}
	fee := &CommissionRule{Mode: CommissionPerGuestFee, FeePerGuestMinor: 10000}

	b := Booking{AmountMinor: 100001}
	ApplyCommission(&b, percent)
	// 12.5% of 1000.01 = 125.00125 → 12500 minor, rounded half up.
	if b.CommissionMode != CommissionPercent || b.CommissionMinor != 12500 || b.VendorNetMinor != 87501 {
		t.Errorf("percent: got %+v", b)
	}
	if b.CommissionMinor+b.VendorNetMinor != b.AmountMinor {
		t.Error("commission and net must add up to the amount")
	}

	b = Booking{AmountMinor: 3}
	ApplyCommission(&b, &CommissionRule{Mode: CommissionPercent, RateBps: 5000})
	if b.CommissionMinor != 2 { // 1.5 rounds half up to 2
		t.Errorf("half-up rounding: want 2, got %d", b.CommissionMinor)
	}

	free := Booking{AmountMinor: 0, PartySize: 4}
	ApplyCommission(&free, fee)
	if free.CommissionMode != CommissionPerGuestFee || free.CommissionMinor != 0 || free.FeePerGuestMinor != 10000 {
		t.Errorf("free booking with fee rule: got %+v", free)
	}
	if got := PerGuestFee(free); got != 40000 {
		t.Errorf("per guest fee for 4: want 40000, got %d", got)
	}

	// The wrong kind of rule for the amount means no commission.
	paidWithFeeRule := Booking{AmountMinor: 5000}
	ApplyCommission(&paidWithFeeRule, fee)
	if paidWithFeeRule.CommissionMode != CommissionNone || paidWithFeeRule.VendorNetMinor != 5000 {
		t.Errorf("paid booking with only a fee rule: got %+v", paidWithFeeRule)
	}
	none := Booking{AmountMinor: 5000}
	ApplyCommission(&none, nil)
	if none.CommissionMode != CommissionNone || none.VendorNetMinor != 5000 || PerGuestFee(none) != 0 {
		t.Errorf("no rule: got %+v", none)
	}
}

func TestRefundFor(t *testing.T) {
	start := time.Date(2026, 12, 10, 10, 0, 0, 0, time.UTC)
	paid := Booking{AmountMinor: 200000, PaymentStatus: PaymentPaid, StartAt: start,
		FreeCancelHours: 24, LateCancelRefundPct: 50}

	if got := RefundFor(paid, start.Add(-25*time.Hour), false); got != 200000 {
		t.Errorf("inside free window: want full, got %d", got)
	}
	if got := RefundFor(paid, start.Add(-24*time.Hour), false); got != 200000 {
		t.Errorf("exactly at the window edge: want full, got %d", got)
	}
	if got := RefundFor(paid, start.Add(-23*time.Hour), false); got != 100000 {
		t.Errorf("late: want 50%%, got %d", got)
	}
	if got := RefundFor(paid, start.Add(-time.Hour), true); got != 200000 {
		t.Errorf("vendor cancel: want full, got %d", got)
	}
	partly := paid
	partly.PaymentStatus, partly.RefundedMinor = PaymentPartiallyRefunded, 150000
	if got := RefundFor(partly, start.Add(-time.Hour), false); got != 0 {
		t.Errorf("late after a larger partial refund: want 0, got %d", got)
	}
	if got := RefundFor(partly, start.Add(-48*time.Hour), false); got != 50000 {
		t.Errorf("free window after partial refund: want the rest, got %d", got)
	}
	unpaid := paid
	unpaid.PaymentStatus = PaymentPending
	if got := RefundFor(unpaid, start.Add(-48*time.Hour), false); got != 0 {
		t.Errorf("unpaid booking: want 0, got %d", got)
	}
}

func TestRefundReversal(t *testing.T) {
	b := Booking{AmountMinor: 200000, VendorNetMinor: 180000}
	if got := RefundReversal(b, 200000); got != 180000 {
		t.Errorf("full refund takes back the whole net: got %d", got)
	}
	if got := RefundReversal(b, 100000); got != 90000 {
		t.Errorf("half refund takes back half the net: got %d", got)
	}
	if got := RefundReversal(Booking{}, 100); got != 0 {
		t.Errorf("free booking: want 0, got %d", got)
	}
}

func TestCanTransition(t *testing.T) {
	allowed := [][2]BookingStatus{
		{BookingAwaitingPayment, BookingConfirmed},
		{BookingAwaitingPayment, BookingExpired},
		{BookingAwaitingPayment, BookingCancelled},
		{BookingExpired, BookingConfirmed},
		{BookingConfirmed, BookingCompleted},
		{BookingConfirmed, BookingNoShow},
		{BookingConfirmed, BookingCancelled},
	}
	for _, p := range allowed {
		if !CanTransition(p[0], p[1]) {
			t.Errorf("%s -> %s should be allowed", p[0], p[1])
		}
	}
	denied := [][2]BookingStatus{
		{BookingCancelled, BookingConfirmed},
		{BookingCompleted, BookingNoShow},
		{BookingNoShow, BookingCompleted},
		{BookingExpired, BookingCancelled},
		{BookingConfirmed, BookingAwaitingPayment},
		{BookingCompleted, BookingCancelled},
	}
	for _, p := range denied {
		if CanTransition(p[0], p[1]) {
			t.Errorf("%s -> %s should be denied", p[0], p[1])
		}
	}
}

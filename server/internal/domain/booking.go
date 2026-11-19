package domain

import "time"

// BookingType drives which fields/validation apply to a BookableService and
// its Bookings — set once per VendorCategory (see VendorCategory.BookingType)
// and copied onto every BookableService/Booking created under that vendor.
type BookingType string

const (
	BookingTypeSession BookingType = "session" // fixed start time + duration + capacity
	BookingTypeTable   BookingType = "table"   // party size + short fixed duration
	BookingTypeStay    BookingType = "stay"    // date-range check-in/out + guest count + room capacity
	BookingTypeRental  BookingType = "rental"  // fixed start time + duration + item capacity
)

// BookableService is one bookable thing a vendor offers (e.g. "Beginner surf
// lesson", "Deluxe Room"). A vendor may define several; most will have one.
type BookableService struct {
	ID          string      `json:"id"`
	VendorID    string      `json:"vendorId"`
	Type        BookingType `json:"type"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	// DurationMins applies to session/table only (how long one booking
	// occupies capacity). Unused (0) for stay — its span is the
	// traveller's own check-in/check-out range.
	DurationMins int `json:"durationMins,omitempty"`
	// Capacity is how much of this service may be held *concurrently*:
	// concurrent bookings (tables, rooms of this type, cars) or, when
	// CapacityUnit is guests, the summed party size across them.
	Capacity int `json:"capacity"`
	// CapacityUnit says whether Capacity counts bookings or guests;
	// empty is treated as bookings.
	CapacityUnit  CapacityUnit `json:"capacityUnit"`
	PartySizeMin  int          `json:"partySizeMin,omitempty"`  // session/table only
	PartySizeMax  int          `json:"partySizeMax,omitempty"`  // session/table only
	GuestsPerRoom int          `json:"guestsPerRoom,omitempty"` // stay only
	// PriceLabel is optional free text shown next to the price
	// ("includes breakfast").
	PriceLabel string `json:"priceLabel"`
	// PriceMinor and PricingMode are what a booking costs (see QuoteBooking
	// in money.go); PriceLabel is now optional display text.
	PriceMinor  int64       `json:"priceMinor"`
	Currency    string      `json:"currency"`
	PricingMode PricingMode `json:"pricingMode"`
	// Cancellation terms: a traveller cancelling at least FreeCancelHours
	// before the start gets a full refund, later LateCancelRefundPct.
	FreeCancelHours     int `json:"freeCancelHours"`
	LateCancelRefundPct int `json:"lateCancelRefundPct"`
	// Availability is a weekly day-of-week rule for session/table types —
	// nil for stay (stay availability is capacity-only). Shape mirrors
	// vendorHoursDoc's spirit (see internal/http/vendor.go).
	Availability *ServiceAvailabilityDoc `json:"availability,omitempty"`
	Active       bool                    `json:"active"`
	// ArchivedAt is set when a vendor deletes a service that already has
	// bookings: the row stays so booking and payment history keep their
	// reference, but it is hidden from every list.
	ArchivedAt *time.Time `json:"archivedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type ServiceAvailabilityWindow struct {
	Start string `json:"start"` // "HH:MM"
	End   string `json:"end"`   // "HH:MM"
}

// ServiceAvailabilityDoc is keyed by lowercase weekday name, same keys as
// vendorHoursDoc (see weekdayKeys in internal/http/vendor.go). An absent day
// or empty Windows means the service isn't bookable that day — always
// further constrained by the vendor's own Hours/HolidayOverrides on top.
type ServiceAvailabilityDoc map[string][]ServiceAvailabilityWindow

type BookingStatus string

const (
	// BookingAwaitingPayment holds capacity for a priced booking until the
	// gateway confirms payment or HoldExpiresAt passes.
	BookingAwaitingPayment BookingStatus = "awaiting_payment"
	// BookingConfirmed is the starting state for a free booking, and the
	// state a priced booking reaches once its payment succeeds.
	BookingConfirmed BookingStatus = "confirmed"
	BookingCancelled BookingStatus = "cancelled"
	BookingCompleted BookingStatus = "completed"
	BookingNoShow    BookingStatus = "no_show"
	// BookingExpired is a priced booking whose payment hold ran out.
	BookingExpired BookingStatus = "expired"
)

// bookingTransitions lists, for each target status, the statuses a
// booking may move to it from. Nothing moves back to an earlier state: a
// cancelled, completed or no-show booking is final, and an expired one can
// only be revived by its own late payment.
var bookingTransitions = map[BookingStatus][]BookingStatus{
	// expired → confirmed: a payment the gateway confirms after the hold
	// lapsed is still honoured while the slot has room (else refunded).
	BookingConfirmed: {BookingAwaitingPayment, BookingExpired},
	BookingCancelled: {BookingAwaitingPayment, BookingConfirmed},
	BookingCompleted: {BookingConfirmed},
	BookingNoShow:    {BookingConfirmed},
	BookingExpired:   {BookingAwaitingPayment},
}

// AllowedFromStatuses returns the statuses a booking may be in to move to
// target (empty for an unknown or initial-only target).
func AllowedFromStatuses(target BookingStatus) []BookingStatus {
	return bookingTransitions[target]
}

// CanTransition reports whether a booking may move from one status to
// another.
func CanTransition(from, to BookingStatus) bool {
	for _, s := range bookingTransitions[to] {
		if s == from {
			return true
		}
	}
	return false
}

// CapacityUnit says what a BookableService's Capacity counts.
type CapacityUnit string

const (
	// CapacityBookings counts concurrent bookings (tables, rooms, cars).
	CapacityBookings CapacityUnit = "bookings"
	// CapacityGuests counts people across concurrent bookings (a tour of
	// 12, a class of 20).
	CapacityGuests CapacityUnit = "guests"
)

// Booking is a traveller's real commitment against a vendor's
// BookableService, free or paid. StartAt/EndAt are always both
// set (session/table: start + start+DurationMins; stay: check-in +
// check-out), so every BookingType shares one overlap query.
type Booking struct {
	ID          string        `json:"id"`
	Code        string        `json:"code"` // "BOOK-XXXX" — distinct namespace from claims' "SPOTTO-XXXX"
	VendorID    string        `json:"vendorId"`
	ServiceID   string        `json:"serviceId"`
	Type        BookingType   `json:"type"`
	PlaceID     string        `json:"placeId"`
	TravellerID string        `json:"travellerId"`
	StartAt     time.Time     `json:"startAt"`
	EndAt       time.Time     `json:"endAt"`
	PartySize   int           `json:"partySize"` // party size (session/table) or guest count (stay)
	Status      BookingStatus `json:"status"`
	Note        string        `json:"note,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	CancelledAt *time.Time    `json:"cancelledAt,omitempty"`
	CancelledBy string        `json:"cancelledBy,omitempty"` // "traveller" | "vendor"
	CheckedInAt *time.Time    `json:"checkedInAt,omitempty"`
	// HoldExpiresAt is when an awaiting_payment booking gives its seats up.
	HoldExpiresAt *time.Time `json:"holdExpiresAt,omitempty"`

	// Money, fixed when the booking is made (see money.go).
	AmountMinor       int64          `json:"amountMinor"`
	Currency          string         `json:"currency"`
	PaymentStatus     PaymentStatus  `json:"paymentStatus"`
	RefundedMinor     int64          `json:"refundedMinor"`
	CommissionMode    CommissionMode `json:"commissionMode"`
	CommissionRateBps int            `json:"commissionRateBps"`
	FeePerGuestMinor  int64          `json:"feePerGuestMinor"`
	CommissionMinor   int64          `json:"commissionMinor"`
	VendorNetMinor    int64          `json:"vendorNetMinor"`
	// Cancellation terms copied from the service at booking time.
	FreeCancelHours     int `json:"freeCancelHours"`
	LateCancelRefundPct int `json:"lateCancelRefundPct"`
	// ReminderSentAt gates the reminder scheduler's idempotency (see
	// booking_reminder_scheduler.go) — not client-facing.
	ReminderSentAt *time.Time `json:"-"`
	// Joined display fields (not columns) — same convention as Claim's
	// TravellerName/TravellerCountry.
	ServiceName      string `json:"serviceName,omitempty"`
	TravellerName    string `json:"travellerName,omitempty"`
	TravellerCountry string `json:"travellerCountry,omitempty"`
	TravellerEmail   string `json:"-"` // joined only for the reminder scheduler's mailer
	VendorName       string `json:"vendorName,omitempty"`
}

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
	// Capacity is how many bookings may hold this service *concurrently* —
	// concurrent session/table bookings, or total rooms of this type for
	// stay. Deliberately a booking count, not a summed party-size/guest
	// count (keeps the overlap check a single count(*)) — PartySizeMin/Max
	// and GuestsPerRoom are informational for the traveller, not enforced
	// against Capacity.
	Capacity      int `json:"capacity"`
	PartySizeMin  int `json:"partySizeMin,omitempty"`  // session/table only
	PartySizeMax  int `json:"partySizeMax,omitempty"`  // session/table only
	GuestsPerRoom int `json:"guestsPerRoom,omitempty"` // stay only
	// PriceLabel is free-text display only ("LKR 4,500pp", "From $80/night")
	// — no payment processing exists on this platform, vendor collects
	// payment in person, same trust model as offer redemption.
	PriceLabel string `json:"priceLabel"`
	// Availability is a weekly day-of-week rule for session/table types —
	// nil for stay (stay availability is capacity-only). Shape mirrors
	// vendorHoursDoc's spirit (see internal/http/vendor.go).
	Availability *ServiceAvailabilityDoc `json:"availability,omitempty"`
	Active       bool                    `json:"active"`
	CreatedAt    time.Time               `json:"createdAt"`
	UpdatedAt    time.Time               `json:"updatedAt"`
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
	BookingConfirmed BookingStatus = "confirmed" // auto-confirmed on creation, no pending/approval step
	BookingCancelled BookingStatus = "cancelled"
	BookingCompleted BookingStatus = "completed"
	BookingNoShow    BookingStatus = "no_show"
)

// Booking is a traveller's real commitment against a vendor's
// BookableService. No payment collected. StartAt/EndAt are always both
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

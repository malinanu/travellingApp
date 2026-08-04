package httpapi

import (
	"time"

	"github.com/spotto/server/internal/domain"
)

// bookingAvailabilityError is a user-facing rejection reason ("closed that
// day", "outside business hours") — mapped to 409 by its callers, distinct
// from postgres.ErrNotFound (404) and store.ErrBookingFull (also 409, but a
// capacity conflict rather than a scheduling one).
type bookingAvailabilityError struct{ reason string }

func (e *bookingAvailabilityError) Error() string { return e.reason }

// withinTimeWindow mirrors isOpenNow's overnight-range logic (vendor.go),
// parameterized on an arbitrary "HH:MM" check time instead of hardcoding
// time.Now() — isOpenNow itself is left untouched since it's only ever
// used for "right now" checks elsewhere.
func withinTimeWindow(check, open, close string) bool {
	if close > open {
		return check >= open && check < close
	}
	return check >= open || check < close
}

// checkVendorOpen mirrors todayHoursAndOpenNow's Closed-override-wins logic
// (vendor.go) for an arbitrary future instant rather than "now" — rejects
// session/table bookings on a day the vendor's Hours mark closed, or an
// explicit HolidayOverrides closure for that date. Not applied to stay
// bookings (a hotel's front-desk hours don't gate reserving a room) —
// callers only invoke this for session/table.
func checkVendorOpen(profile vendorProfileDoc, at time.Time) error {
	dateStr := at.Format("2006-01-02")
	for _, ov := range profile.HolidayOverrides {
		if ov.Date == dateStr && ov.Closed {
			name := ov.Name
			if name == "" {
				name = "a public holiday"
			}
			return &bookingAvailabilityError{"vendor is closed that day — " + name}
		}
	}
	day, ok := profile.Hours[weekdayKey(at)]
	if !ok || day.Closed || day.Open == "" || day.Close == "" {
		return &bookingAvailabilityError{"vendor is closed that day"}
	}
	if !withinTimeWindow(at.Format("15:04"), day.Open, day.Close) {
		return &bookingAvailabilityError{"outside business hours"}
	}
	return nil
}

// checkServiceAvailability validates a session/table booking's start/end
// against the service's own weekly windows — nil Availability means
// unrestricted (the vendor's own Hours, checked separately via
// checkVendorOpen, is the only gate). Assumes startAt/endAt fall on the
// same calendar day, true for every session/table booking (short,
// same-day by construction).
func checkServiceAvailability(availability *domain.ServiceAvailabilityDoc, startAt, endAt time.Time) error {
	if availability == nil {
		return nil
	}
	windows := (*availability)[weekdayKey(startAt)]
	if len(windows) == 0 {
		return &bookingAvailabilityError{"not available that day"}
	}
	startStr, endStr := startAt.Format("15:04"), endAt.Format("15:04")
	for _, win := range windows {
		if startStr >= win.Start && endStr <= win.End {
			return nil
		}
	}
	return &bookingAvailabilityError{"outside this service's available hours"}
}

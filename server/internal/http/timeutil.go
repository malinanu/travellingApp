package httpapi

import "time"

// Bookings are made for places in Sri Lanka, so calendar dates and clock
// times in booking requests always mean Sri Lankan time (Asia/Colombo,
// UTC+5:30), wherever the traveller's phone or the server happens to be.

// parseLocalDate parses a "2006-01-02" date as midnight in Sri Lanka.
func parseLocalDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, sriLankaLocation)
}

// inSriLanka converts an instant to Sri Lankan wall-clock time, so weekday
// and "15:04" checks against a vendor's hours use the vendor's own clock.
func inSriLanka(t time.Time) time.Time {
	return t.In(sriLankaLocation)
}

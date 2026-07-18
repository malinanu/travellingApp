package holidays

import "time"

// StaticSriLankaHolidays is the fallback used when GOOGLE_CALENDAR_API_KEY
// is unset. It is intentionally minimal: only fixed-date public holidays
// that don't require external data to get right.
//
// KNOWN GAP: this deliberately omits every lunar/lunisolar holiday — Poya
// (monthly full-moon) days, Vesak, Deepavali, Eid — since their dates shift
// year to year and aren't reliably known without a real calendar source.
// Many Sri Lankan businesses do close for these, so this static list
// under-covers real closures; it exists only as a same-day-works fallback,
// not a substitute for the live Google Calendar integration. Needs a
// yearly review/extension regardless.
var StaticSriLankaHolidays = []Holiday{
	{Date: mustDate("2026-01-01"), Name: "New Year's Day"},
	{Date: mustDate("2026-01-14"), Name: "Tamil Thai Pongal Day"},
	{Date: mustDate("2026-02-04"), Name: "Independence Day"},
	{Date: mustDate("2026-04-13"), Name: "Sinhala & Tamil New Year Eve"},
	{Date: mustDate("2026-04-14"), Name: "Sinhala & Tamil New Year Day"},
	{Date: mustDate("2026-05-01"), Name: "International Workers' Day"},
	{Date: mustDate("2026-12-25"), Name: "Christmas Day"},

	{Date: mustDate("2027-01-01"), Name: "New Year's Day"},
	{Date: mustDate("2027-01-14"), Name: "Tamil Thai Pongal Day"},
	{Date: mustDate("2027-02-04"), Name: "Independence Day"},
	{Date: mustDate("2027-04-13"), Name: "Sinhala & Tamil New Year Eve"},
	{Date: mustDate("2027-04-14"), Name: "Sinhala & Tamil New Year Day"},
	{Date: mustDate("2027-05-01"), Name: "International Workers' Day"},
	{Date: mustDate("2027-12-25"), Name: "Christmas Day"},
}

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

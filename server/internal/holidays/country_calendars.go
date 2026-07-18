package holidays

// CountryHolidayCalendars maps an ISO 3166-1 alpha-2 country code to
// Google's public holiday calendar ID for that country. This is NOT a
// derivable formula — Google set these calendars up organically over ~15
// years and the IDs mix ISO codes (en.lk#) with demonym words (en.usa#,
// en.indian#, en.australian#), so every integration that supports more
// than one country maintains a table like this one. There's no API to
// discover them dynamically.
//
// Only "LK" is verified against the live API (confirmed working in this
// codebase). Every other entry is a best-effort guess from general
// knowledge of Google's well-known calendar list — this is a safe
// failure mode, not a risky one: if an ID is wrong, the live fetch just
// 404s/errors and Upcoming falls back to an empty list (see holidays.go),
// never to the wrong country's holidays. Fix or add entries as they're
// verified.
var CountryHolidayCalendars = map[string]string{
	"LK": "en.lk#holiday@group.v.calendar.google.com", // verified live

	// Best-effort — unverified, safe to correct:
	"US": "en.usa#holiday@group.v.calendar.google.com",
	"GB": "en.uk#holiday@group.v.calendar.google.com",
	"IN": "en.indian#holiday@group.v.calendar.google.com",
	"AU": "en.australian#holiday@group.v.calendar.google.com",
	"CA": "en.canadian#holiday@group.v.calendar.google.com",
	"DE": "en.german#holiday@group.v.calendar.google.com",
	"FR": "en.french#holiday@group.v.calendar.google.com",
	"JP": "en.japanese#holiday@group.v.calendar.google.com",
	"SG": "en.singapore#holiday@group.v.calendar.google.com",
	"NZ": "en.new_zealand#holiday@group.v.calendar.google.com",
	"ZA": "en.sa#holiday@group.v.calendar.google.com",
	"IE": "en.irish#holiday@group.v.calendar.google.com",
	"NL": "en.dutch#holiday@group.v.calendar.google.com",
	"BR": "en.brazilian#holiday@group.v.calendar.google.com",
	"MY": "en.malaysia#holiday@group.v.calendar.google.com",
	"TH": "en.thai#holiday@group.v.calendar.google.com",
	"ID": "en.indonesian#holiday@group.v.calendar.google.com",
	"PH": "en.philippines#holiday@group.v.calendar.google.com",
	"AE": "en.uae#holiday@group.v.calendar.google.com",
}

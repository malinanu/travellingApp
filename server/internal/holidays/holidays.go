// Package holidays looks up each vendor's upcoming public holidays for the
// Business Profile's automatic closure-prompt feature, keyed by the
// vendor's country (see country_calendars.go for the country→Google
// calendar ID mapping and its caveats) — see server/internal/http/
// holiday_scheduler.go for how this is used.
package holidays

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Holiday struct {
	Date time.Time
	Name string
}

// Upcoming returns countryCode's public holidays in [now, now+days] (an
// ISO 3166-1 alpha-2 code, e.g. "LK", "US" — see domain.Vendor.Country).
//
// If countryCode has no entry in CountryHolidayCalendars, this returns an
// empty list immediately without attempting any API call — no country's
// holidays are ever substituted for another's.
//
// Otherwise: if apiKey is empty (GOOGLE_CALENDAR_API_KEY unset) — or the
// live call fails for any reason (network error, API not enabled, a
// referrer-restricted key being used from this server-side call, which
// sends no Referer header at all — referrer restrictions are meant for
// the browser-side Maps key, not this one) — Sri Lanka ("LK") falls back
// to a bundled static list rather than erroring, since a stale-but-present
// holiday list is much better than breaking the Business Profile page
// entirely; see static_lk_holidays.go for that list's known gaps (notably:
// lunar Poya days aren't included, since they don't fall on fixed dates
// and aren't reliably known without a real data source). Every other
// country just gets an empty list on the same failure, since there's no
// curated static list for it yet.
func Upcoming(ctx context.Context, apiKey, countryCode string, days int) ([]Holiday, error) {
	calendarID, ok := CountryHolidayCalendars[countryCode]
	if !ok {
		return nil, nil
	}
	if apiKey == "" {
		if countryCode == "LK" {
			return filterWindow(StaticSriLankaHolidays, days), nil
		}
		return nil, nil
	}
	got, err := fetchFromGoogle(ctx, apiKey, calendarID, days)
	if err != nil {
		log.Printf("holidays: live Google Calendar fetch failed for country=%s: %v", countryCode, err)
		if countryCode == "LK" {
			return filterWindow(StaticSriLankaHolidays, days), nil
		}
		return nil, nil
	}
	return got, nil
}

func filterWindow(all []Holiday, days int) []Holiday {
	now := time.Now()
	cutoff := now.AddDate(0, 0, days)
	out := make([]Holiday, 0, len(all))
	for _, h := range all {
		if !h.Date.Before(truncateToDay(now)) && h.Date.Before(cutoff) {
			out = append(out, h)
		}
	}
	return out
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

type calendarEventsResponse struct {
	Items []struct {
		Summary string `json:"summary"`
		Start   struct {
			Date string `json:"date"`
		} `json:"start"`
	} `json:"items"`
}

func fetchFromGoogle(ctx context.Context, apiKey, calendarID string, days int) ([]Holiday, error) {
	now := time.Now()
	q := url.Values{
		"key":          {apiKey},
		"timeMin":      {now.Format(time.RFC3339)},
		"timeMax":      {now.AddDate(0, 0, days).Format(time.RFC3339)},
		"singleEvents": {"true"},
		"orderBy":      {"startTime"},
	}
	calendarPath := url.PathEscape(calendarID)
	reqURL := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events?%s", calendarPath, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	referer := os.Getenv("NEXT_PUBLIC_API_URL")
	if referer == "" {
		referer = "https://spottoapi.threelogiq.com/"
	}
	req.Header.Set("Referer", referer)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google calendar api: unexpected status %d", res.StatusCode)
	}

	var parsed calendarEventsResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parseCalendarEvents(parsed), nil
}

func parseCalendarEvents(parsed calendarEventsResponse) []Holiday {
	out := make([]Holiday, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		if item.Start.Date == "" {
			continue // skip any non-all-day entries (shouldn't occur for a holiday calendar)
		}
		d, err := time.Parse("2006-01-02", item.Start.Date)
		if err != nil {
			continue
		}
		out = append(out, Holiday{Date: d, Name: item.Summary})
	}
	return out
}

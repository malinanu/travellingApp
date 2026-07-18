// Package holidays looks up upcoming Sri Lankan public holidays for the
// vendor Business Profile's automatic closure-prompt feature. There is no
// vendor→country field anywhere in this app yet (every vendor/region in
// the current dataset is Sri Lankan by convention), so the calendar is
// hardcoded rather than looked up per vendor — see server/internal/http/
// holiday_scheduler.go for how this is used.
package holidays

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Holiday struct {
	Date time.Time
	Name string
}

// sriLankaHolidayCalendarID is Google's public holiday calendar for Sri
// Lanka — readable via a plain API-key GET, no user OAuth consent needed.
const sriLankaHolidayCalendarID = "en.lk#holiday@group.v.calendar.google.com"

// Upcoming returns Sri Lankan public holidays in [now, now+days]. If apiKey
// is empty (GOOGLE_CALENDAR_API_KEY unset), it falls back to a bundled
// static list — see static_lk_holidays.go for that list's known gaps
// (notably: lunar Poya days aren't included, since they don't fall on
// fixed dates and aren't reliably known without a real data source).
func Upcoming(ctx context.Context, apiKey string, days int) ([]Holiday, error) {
	if apiKey == "" {
		return filterWindow(StaticSriLankaHolidays, days), nil
	}
	return fetchFromGoogle(ctx, apiKey, days)
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

func fetchFromGoogle(ctx context.Context, apiKey string, days int) ([]Holiday, error) {
	now := time.Now()
	q := url.Values{
		"key":          {apiKey},
		"timeMin":      {now.Format(time.RFC3339)},
		"timeMax":      {now.AddDate(0, 0, days).Format(time.RFC3339)},
		"singleEvents": {"true"},
		"orderBy":      {"startTime"},
	}
	calendarPath := url.PathEscape(sriLankaHolidayCalendarID)
	reqURL := fmt.Sprintf("https://www.googleapis.com/calendar/v3/calendars/%s/events?%s", calendarPath, q.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
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

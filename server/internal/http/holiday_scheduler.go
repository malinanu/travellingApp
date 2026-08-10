package httpapi

// The holiday scheduler is the first *scheduled* background job in this
// codebase â€” internal/jobs.Queue is fire-and-forget work triggered by a
// request (e.g. sending an email after an invite), not a periodic check.
// This runs independently of any HTTP request, so it lives here rather
// than in internal/jobs.

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/holidays"
)

const holidaySchedulerInterval = 12 * time.Hour

// StartHolidayScheduler periodically checks for public holidays landing
// within holidayNoticeDays and creates a VendorNotification for every
// approved vendor that hasn't yet recorded an open/closed decision for
// that date (see handleVendorHolidayDecision) â€” deterministic notification
// IDs (see runHolidayCheck) make re-running this idempotent rather than
// spamming duplicates. Blocks until ctx is cancelled; call with `go`.
func (s *Server) StartHolidayScheduler(ctx context.Context) {
	s.runHolidayCheck(ctx)
	ticker := time.NewTicker(holidaySchedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runHolidayCheck(ctx)
		}
	}
}

func (s *Server) runHolidayCheck(ctx context.Context) {
	vendors, err := s.Store.Vendors(ctx, string(domain.VendorApproved))
	if err != nil {
		log.Printf("holiday scheduler: list vendors: %v", err)
		return
	}

	// Fetch each represented country's holiday list once per tick, not
	// once per vendor â€” country here always means Vendor.Country ("LK" for
	// any vendor predating that field).
	byCountry := make(map[string][]holidays.Holiday)
	for _, v := range vendors {
		if _, ok := byCountry[v.Country]; ok {
			continue
		}
		upcoming, err := holidays.Upcoming(ctx, s.GoogleCalendarAPIKey, v.Country, holidayNoticeDays)
		if err != nil {
			log.Printf("holiday scheduler: fetch upcoming holidays for country=%s: %v", v.Country, err)
			continue
		}
		byCountry[v.Country] = upcoming
	}

	for _, v := range vendors {
		upcoming := byCountry[v.Country]
		if len(upcoming) == 0 {
			continue
		}
		var profile vendorProfileDoc
		_ = json.Unmarshal(orEmpty(v.Profile), &profile)
		decided := make(map[string]bool, len(profile.HolidayOverrides))
		for _, ov := range profile.HolidayOverrides {
			decided[ov.Date] = true
		}
		for _, h := range upcoming {
			date := h.Date.Format("2006-01-02")
			if decided[date] {
				continue
			}
			notif := domain.VendorNotification{
				// Deterministic per vendor+date so re-running the ticker
				// upserts the same row rather than creating a duplicate.
				ID:     "notif_holiday_" + v.ID + "_" + date,
				Icon:   "Clock",
				Tone:   "sun",
				Title:  h.Name + " is coming up",
				Sub:    fmt.Sprintf("%s â€” will you be open? Answer it on your Business Profile.", h.Date.Format("Jan 2")),
				Time:   "just now",
				Unread: true,
			}
			if err := s.Store.UpsertVendorNotification(ctx, v.ID, notif); err != nil {
				log.Printf("holiday scheduler: notify vendor=%s date=%s: %v", v.ID, date, err)
			}
		}
	}
}

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// handlePlaceServices lists a place's active bookable services â€” empty for
// a place with no vendor, or a vendor whose category isn't bookable (no
// services will ever have been created for it).
func (s *Server) handlePlaceServices(w http.ResponseWriter, r *http.Request) {
	place, err := s.Store.PlaceByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if place.VendorID == nil {
		writeJSON(w, http.StatusOK, []domain.BookableService{})
		return
	}
	services, err := s.Store.BookableServicesByVendor(r.Context(), *place.VendorID, true)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, services)
}

type availabilitySlot struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Remaining int       `json:"remaining"`
}

// availabilityWindowsForDay returns the windows a service is bookable
// within on a given date: the service's own weekly rule if it has one,
// else the vendor's own Hours entry for that day as a single window
// (checkVendorOpen is still consulted per-slot below regardless, so a
// later HolidayOverrides closure still wins even for a service that has
// its own explicit rule).
func availabilityWindowsForDay(svc domain.BookableService, profile vendorProfileDoc, date time.Time) []domain.ServiceAvailabilityWindow {
	if svc.Availability != nil {
		return (*svc.Availability)[weekdayKey(date)]
	}
	if !usesVendorHours(svc) {
		return nil
	}
	day, ok := profile.Hours[weekdayKey(date)]
	if !ok || day.Closed || day.Open == "" || day.Close == "" {
		return nil
	}
	return []domain.ServiceAvailabilityWindow{{Start: day.Open, End: day.Close}}
}

// usesVendorHours reports whether a service is also limited by its
// vendor's opening hours. Experiences and events run on their own
// schedule (a host has no opening hours), as do stays.
func usesVendorHours(svc domain.BookableService) bool {
	switch svc.Type {
	case domain.BookingTypeExperience, domain.BookingTypeEvent, domain.BookingTypeStay:
		return false
	}
	return true
}

// isBlackoutDate reports whether the vendor closed this service on the
// Sri Lankan calendar day of t.
func isBlackoutDate(svc domain.BookableService, t time.Time) bool {
	day := inSriLanka(t).Format("2006-01-02")
	for _, d := range svc.BlackoutDates {
		if d == day {
			return true
		}
	}
	return false
}

func parseClockOnDate(date time.Time, clock string) (time.Time, error) {
	t, err := time.Parse("15:04", clock)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(date.Year(), date.Month(), date.Day(), t.Hour(), t.Minute(), 0, 0, date.Location()), nil
}

// handleServiceAvailability computes real bookable windows for a
// session/table service on a given ?date=, or remaining room capacity for
// a stay service given ?checkIn=&checkOut= â€” checked live against the
// vendor's Hours/HolidayOverrides and existing overlapping confirmed
// bookings, never a pre-materialized slot table.
func (s *Server) handleServiceAvailability(w http.ResponseWriter, r *http.Request) {
	place, err := s.Store.PlaceByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	svc, err := s.Store.BookableServiceByID(r.Context(), chi.URLParam(r, "serviceId"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if place.VendorID == nil || svc.VendorID != *place.VendorID {
		writeErr(w, http.StatusNotFound, "The requested item could not be found.")
		return
	}

	if svc.Scheduling == domain.SchedulingOccurrences {
		from := time.Now()
		if d, err := parseLocalDate(r.URL.Query().Get("date")); err == nil && d.After(from) {
			from = d
		}
		occurrences, err := s.Store.OccurrencesWithRemaining(r.Context(), svc, from, false, 60)
		if err != nil {
			storeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"occurrences": occurrences})
		return
	}

	if svc.Type == domain.BookingTypeStay {
		checkIn, err1 := parseLocalDate(r.URL.Query().Get("checkIn"))
		checkOut, err2 := parseLocalDate(r.URL.Query().Get("checkOut"))
		if err1 != nil || err2 != nil || !checkOut.After(checkIn) {
			writeErr(w, http.StatusBadRequest, "valid checkIn/checkOut required")
			return
		}
		existing, err := s.Store.HeldCapacity(r.Context(), svc, checkIn, checkOut)
		if err != nil {
			storeErr(w, err)
			return
		}
		remaining := svc.Capacity - existing
		if remaining < 0 {
			remaining = 0
		}
		writeJSON(w, http.StatusOK, map[string]any{"remaining": remaining})
		return
	}

	date, err := parseLocalDate(r.URL.Query().Get("date"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "valid date required")
		return
	}
	if isBlackoutDate(svc, date) {
		writeJSON(w, http.StatusOK, map[string]any{"slots": []availabilitySlot{}})
		return
	}
	vendor, err := s.Store.VendorByID(r.Context(), svc.VendorID)
	if err != nil {
		storeErr(w, err)
		return
	}
	var profile vendorProfileDoc
	json.Unmarshal(vendor.Profile, &profile)

	duration := time.Duration(svc.DurationMins) * time.Minute
	if duration <= 0 {
		duration = time.Hour
	}
	var slots []availabilitySlot
	for _, win := range availabilityWindowsForDay(svc, profile, date) {
		start, err1 := parseClockOnDate(date, win.Start)
		end, err2 := parseClockOnDate(date, win.End)
		if err1 != nil || err2 != nil {
			continue
		}
		for t := start; !t.Add(duration).After(end); t = t.Add(duration) {
			slotEnd := t.Add(duration)
			if !t.After(time.Now()) {
				continue
			}
			if usesVendorHours(svc) && checkVendorOpen(profile, t) != nil {
				continue
			}
			existing, err := s.Store.HeldCapacity(r.Context(), svc, t, slotEnd)
			if err != nil {
				storeErr(w, err)
				return
			}
			remaining := svc.Capacity - existing
			if remaining <= 0 {
				continue
			}
			slots = append(slots, availabilitySlot{Start: t, End: slotEnd, Remaining: remaining})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"slots": slots})
}

// handleCreateBooking creates a real booking â€” always auto-confirmed, no
// payment collected. The server always computes EndAt itself for
// session/table (StartAt + service.DurationMins), never trusting a
// client-supplied duration; for stay, StartAt/EndAt come straight from the
// requested check-in/check-out dates.
func (s *Server) handleCreateBooking(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		PlaceID      string `json:"placeId"`
		ServiceID    string `json:"serviceId"`
		OccurrenceID string `json:"occurrenceId,omitempty"` // dated services
		StartAt      string `json:"startAt,omitempty"`      // RFC3339, session/table
		CheckIn      string `json:"checkIn,omitempty"`      // "2006-01-02", stay
		CheckOut     string `json:"checkOut,omitempty"`
		PartySize    int    `json:"partySize"`
		Note         string `json:"note"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.ServiceID == "" {
		writeErr(w, http.StatusBadRequest, "serviceId required")
		return
	}
	if req.PartySize < 1 {
		req.PartySize = 1
	}

	svc, err := s.Store.BookableServiceByID(r.Context(), req.ServiceID)
	if err != nil {
		storeErr(w, err)
		return
	}
	// placeId is optional: an experience opened from a listing or a shared
	// link is booked against its host's own place.
	var place domain.Place
	if req.PlaceID != "" {
		place, err = s.Store.PlaceByID(r.Context(), req.PlaceID)
	} else {
		place, err = s.Store.PlaceByVendorID(r.Context(), svc.VendorID)
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	if place.VendorID == nil {
		writeErr(w, http.StatusBadRequest, "this place doesn't take bookings")
		return
	}
	if svc.VendorID != *place.VendorID {
		writeErr(w, http.StatusBadRequest, "service does not belong to this place")
		return
	}
	if !svc.Active {
		writeErr(w, http.StatusConflict, "this service is no longer bookable")
		return
	}

	if svc.PartySizeMin > 0 && req.PartySize < svc.PartySizeMin && svc.Type != domain.BookingTypeStay {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("minimum party size is %d", svc.PartySizeMin))
		return
	}
	if svc.PartySizeMax > 0 && req.PartySize > svc.PartySizeMax && svc.Type != domain.BookingTypeStay {
		writeErr(w, http.StatusBadRequest, fmt.Sprintf("maximum party size is %d", svc.PartySizeMax))
		return
	}

	var startAt, endAt time.Time
	var occurrenceID *string
	if svc.Scheduling == domain.SchedulingOccurrences {
		if req.OccurrenceID == "" {
			writeErr(w, http.StatusBadRequest, "occurrenceId required: pick a date")
			return
		}
		occ, err := s.Store.OccurrenceByID(r.Context(), req.OccurrenceID)
		if err != nil || occ.ServiceID != svc.ID {
			writeErr(w, http.StatusNotFound, "that date isn't available")
			return
		}
		if occ.Status != "scheduled" || !occ.StartAt.After(time.Now()) {
			writeErr(w, http.StatusConflict, "that date is no longer bookable")
			return
		}
		startAt, endAt, occurrenceID = occ.StartAt, occ.EndAt, &occ.ID
	} else if svc.Type == domain.BookingTypeStay {
		checkIn, err1 := parseLocalDate(req.CheckIn)
		checkOut, err2 := parseLocalDate(req.CheckOut)
		if err1 != nil || err2 != nil || !checkOut.After(checkIn) {
			writeErr(w, http.StatusBadRequest, "valid checkIn/checkOut dates required")
			return
		}
		if today, _ := parseLocalDate(inSriLanka(time.Now()).Format("2006-01-02")); checkIn.Before(today) {
			writeErr(w, http.StatusBadRequest, "check-in can't be in the past")
			return
		}
		startAt, endAt = checkIn, checkOut
	} else {
		start, err := time.Parse(time.RFC3339, req.StartAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "valid startAt required")
			return
		}
		// Hours and service windows are Sri Lankan wall-clock times,
		// whatever offset the client sent the instant in.
		startAt = inSriLanka(start)
		duration := time.Duration(svc.DurationMins) * time.Minute
		if duration <= 0 {
			duration = time.Hour
		}
		endAt = start.Add(duration)

		vendor, err := s.Store.VendorByID(r.Context(), svc.VendorID)
		if err != nil {
			storeErr(w, err)
			return
		}
		var profile vendorProfileDoc
		json.Unmarshal(vendor.Profile, &profile)
		if !startAt.After(time.Now()) {
			writeErr(w, http.StatusBadRequest, "that time has already passed")
			return
		}
		if isBlackoutDate(svc, startAt) {
			writeErr(w, http.StatusConflict, "not available that day")
			return
		}
		if usesVendorHours(svc) {
			if err := checkVendorOpen(profile, startAt); err != nil {
				writeErr(w, http.StatusConflict, err.Error())
				return
			}
		}
		if err := checkServiceAvailability(svc.Availability, startAt, endAt); err != nil {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
	}

	booking := domain.Booking{
		ID: newID("bkg"), VendorID: svc.VendorID, ServiceID: svc.ID, Type: svc.Type,
		PlaceID: place.ID, TravellerID: c.Subject, StartAt: startAt, EndAt: endAt,
		PartySize: req.PartySize, Note: req.Note, OccurrenceID: occurrenceID,
		AmountMinor: domain.QuoteBooking(svc, req.PartySize, startAt, endAt), Currency: svc.Currency,
		FreeCancelHours: svc.FreeCancelHours, LateCancelRefundPct: svc.LateCancelRefundPct,
	}
	if err := s.applyCommission(r.Context(), &booking); err != nil {
		storeErr(w, err)
		return
	}
	if booking.AmountMinor > 0 && s.Payments == nil {
		writeErr(w, http.StatusServiceUnavailable, "Online payment isn't available right now, so paid bookings are paused.")
		return
	}
	if booking.AmountMinor > 0 {
		// Unpaid holds block seats for others, so one traveller can only
		// hold a few at a time.
		if n, err := s.Store.LiveHoldCount(r.Context(), c.Subject); err != nil {
			storeErr(w, err)
			return
		} else if n >= maxLiveHolds {
			writeErr(w, http.StatusTooManyRequests, "Finish paying for your other bookings first, or let them lapse.")
			return
		}
	}
	var created domain.Booking
	for range 5 {
		booking.Code = newBookingCode()
		created, err = s.Store.CreateBooking(r.Context(), booking)
		if err == nil {
			break
		}
		// Real failures (capacity/inactive/not-found) won't change on
		// retry â€” only a code unique-collision should be retried, same
		// distinction handleCreateClaim doesn't need to make since
		// CreateClaim has no other failure mode.
		if errors.Is(err, postgres.ErrNotFound) || errors.Is(err, postgres.ErrServiceInactive) || errors.Is(err, postgres.ErrBookingFull) {
			break
		}
	}
	if errors.Is(err, postgres.ErrBookingFull) {
		writeErr(w, http.StatusConflict, "no availability left for that time")
		return
	}
	if errors.Is(err, postgres.ErrServiceInactive) {
		writeErr(w, http.StatusConflict, "this service is no longer bookable")
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	created.ServiceName = svc.Name
	created.VendorName = vendorNameOf(r.Context(), s, svc.VendorID)

	if created.Status == domain.BookingConfirmed {
		// Free: confirmed now. A paid booking is confirmed by its payment.
		s.notifyBookingConfirmed(r.Context(), created)
		writeJSON(w, http.StatusCreated, bookingWithCheckout{Booking: created})
		return
	}
	co, err := s.startCheckout(r, created)
	if err != nil {
		// Give the seats back rather than hold them for a payment that
		// can't start.
		s.Store.CancelBookingWithRefund(r.Context(), postgres.CancelRequest{BookingID: created.ID, By: "system", Actor: "system", Now: time.Now()})
		writePaymentStartErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, bookingWithCheckout{Booking: created, Checkout: co})
}

// maxLiveHolds is how many unpaid bookings one traveller may hold at once.
const maxLiveHolds = 3

func vendorNameOf(ctx context.Context, s *Server, vendorID string) string {
	if v, err := s.Store.VendorByID(ctx, vendorID); err == nil {
		return v.Name
	}
	return ""
}

// applyCommission snapshots the commission rule that applies to b now.
func (s *Server) applyCommission(ctx context.Context, b *domain.Booking) error {
	vendor, err := s.Store.VendorByID(ctx, b.VendorID)
	if err != nil {
		return err
	}
	rule, err := s.Store.ResolveCommissionRule(ctx, domain.CommissionModeFor(b.AmountMinor),
		b.VendorID, vendor.Category, b.Type, time.Now())
	if err != nil {
		return err
	}
	domain.ApplyCommission(b, rule)
	return nil
}

func (s *Server) handleMyBookings(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	bookings, err := s.Store.BookingsByTraveller(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, travellerBookings(bookings))
}

func (s *Server) handleCancelMyBooking(w http.ResponseWriter, r *http.Request) {
	booking, ok := s.ownBooking(w, r)
	if !ok {
		return
	}
	refunded, err := s.cancelBooking(r.Context(), booking, cancelByTraveller, "traveller:"+booking.TravellerID)
	if err != nil {
		writeCancelErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled", "refundMinor": refunded, "currency": booking.Currency})
}

func writeCancelErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, postgres.ErrInvalidTransition):
		writeErr(w, http.StatusConflict, "This booking can no longer be cancelled.")
	case errors.Is(err, postgres.ErrBookingStarted):
		writeErr(w, http.StatusConflict, "This booking has already started, so it can't be cancelled in the app.")
	default:
		storeErr(w, err)
	}
}

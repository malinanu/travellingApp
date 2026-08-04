package httpapi

import (
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

// handlePlaceServices lists a place's active bookable services — empty for
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
	day, ok := profile.Hours[weekdayKey(date)]
	if !ok || day.Closed || day.Open == "" || day.Close == "" {
		return nil
	}
	return []domain.ServiceAvailabilityWindow{{Start: day.Open, End: day.Close}}
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
// a stay service given ?checkIn=&checkOut= — checked live against the
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
		writeErr(w, http.StatusNotFound, "not found")
		return
	}

	if svc.Type == domain.BookingTypeStay {
		checkIn, err1 := time.Parse("2006-01-02", r.URL.Query().Get("checkIn"))
		checkOut, err2 := time.Parse("2006-01-02", r.URL.Query().Get("checkOut"))
		if err1 != nil || err2 != nil || !checkOut.After(checkIn) {
			writeErr(w, http.StatusBadRequest, "valid checkIn/checkOut required")
			return
		}
		existing, err := s.Store.OverlappingBookingsCount(r.Context(), svc.ID, checkIn, checkOut)
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

	date, err := time.Parse("2006-01-02", r.URL.Query().Get("date"))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "valid date required")
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
			if checkVendorOpen(profile, t) != nil {
				continue
			}
			existing, err := s.Store.OverlappingBookingsCount(r.Context(), svc.ID, t, slotEnd)
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

// handleCreateBooking creates a real booking — always auto-confirmed, no
// payment collected. The server always computes EndAt itself for
// session/table (StartAt + service.DurationMins), never trusting a
// client-supplied duration; for stay, StartAt/EndAt come straight from the
// requested check-in/check-out dates.
func (s *Server) handleCreateBooking(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		PlaceID   string `json:"placeId"`
		ServiceID string `json:"serviceId"`
		StartAt   string `json:"startAt,omitempty"` // RFC3339, session/table
		CheckIn   string `json:"checkIn,omitempty"` // "2006-01-02", stay
		CheckOut  string `json:"checkOut,omitempty"`
		PartySize int    `json:"partySize"`
		Note      string `json:"note"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.PlaceID == "" || req.ServiceID == "" {
		writeErr(w, http.StatusBadRequest, "placeId and serviceId required")
		return
	}
	if req.PartySize < 1 {
		req.PartySize = 1
	}

	place, err := s.Store.PlaceByID(r.Context(), req.PlaceID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if place.VendorID == nil {
		writeErr(w, http.StatusBadRequest, "this place doesn't take bookings")
		return
	}
	svc, err := s.Store.BookableServiceByID(r.Context(), req.ServiceID)
	if err != nil {
		storeErr(w, err)
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

	var startAt, endAt time.Time
	if svc.Type == domain.BookingTypeStay {
		checkIn, err1 := time.Parse("2006-01-02", req.CheckIn)
		checkOut, err2 := time.Parse("2006-01-02", req.CheckOut)
		if err1 != nil || err2 != nil || !checkOut.After(checkIn) {
			writeErr(w, http.StatusBadRequest, "valid checkIn/checkOut dates required")
			return
		}
		startAt, endAt = checkIn, checkOut
	} else {
		start, err := time.Parse(time.RFC3339, req.StartAt)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "valid startAt required")
			return
		}
		if svc.PartySizeMin > 0 && req.PartySize < svc.PartySizeMin {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("minimum party size is %d", svc.PartySizeMin))
			return
		}
		if svc.PartySizeMax > 0 && req.PartySize > svc.PartySizeMax {
			writeErr(w, http.StatusBadRequest, fmt.Sprintf("maximum party size is %d", svc.PartySizeMax))
			return
		}
		startAt = start
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
		if err := checkVendorOpen(profile, startAt); err != nil {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
		if err := checkServiceAvailability(svc.Availability, startAt, endAt); err != nil {
			writeErr(w, http.StatusConflict, err.Error())
			return
		}
	}

	booking := domain.Booking{
		ID: newID("bkg"), VendorID: svc.VendorID, ServiceID: svc.ID, Type: svc.Type,
		PlaceID: place.ID, TravellerID: c.Subject, StartAt: startAt, EndAt: endAt,
		PartySize: req.PartySize, Note: req.Note,
	}
	var created domain.Booking
	for range 5 {
		booking.Code = newBookingCode()
		created, err = s.Store.CreateBooking(r.Context(), booking)
		if err == nil {
			break
		}
		// Real failures (capacity/inactive/not-found) won't change on
		// retry — only a code unique-collision should be retried, same
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

	// Best-effort: never fail the booking over a notification/email hiccup.
	me, meErr := s.Store.UserByID(r.Context(), c.Subject)
	if meErr == nil && me.Email != "" {
		s.Jobs.Enqueue(func() {
			s.Mail.Send(me.Email, "Your Spotto booking is confirmed",
				fmt.Sprintf("<p>Your booking for <strong>%s</strong> is confirmed.</p><p>Code: <strong>%s</strong></p>", svc.Name, created.Code))
		})
	}
	s.Store.UpsertVendorNotification(r.Context(), svc.VendorID, domain.VendorNotification{
		ID:     "notif_booking_" + created.ID,
		Icon:   "Calendar",
		Tone:   "teal",
		Title:  "New booking: " + svc.Name,
		Sub:    fmt.Sprintf("%s, party of %d", created.StartAt.Format("Jan 2, 3:04 PM"), created.PartySize),
		Time:   "just now",
		Unread: true,
	})

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleMyBookings(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	bookings, err := s.Store.BookingsByTraveller(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookings)
}

func (s *Server) handleCancelMyBooking(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	booking, err := s.Store.BookingByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	if booking.TravellerID != c.Subject {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	if err := s.Store.CancelBookingAsTraveller(r.Context(), id, c.Subject); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

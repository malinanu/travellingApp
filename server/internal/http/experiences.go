package httpapi

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/store/postgres"
)

// handleExperiences lists bookable experiences and events: tours, classes,
// trips run by local hosts and venues, and ticketed event nights.
//
//	?category=surf            one category
//	?type=event               only events (or experience)
//	?lat=..&lng=..&radiusKm=  nearest first, within the radius (default 50)
func (s *Server) handleExperiences(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := postgres.ExperienceFilter{Category: strings.TrimSpace(q.Get("category")), Limit: 200}
	if t := domain.BookingType(q.Get("type")); t == domain.BookingTypeExperience || t == domain.BookingTypeEvent {
		f.Types = []domain.BookingType{t}
	}
	listings, err := s.Store.ExperienceListings(r.Context(), f)
	if err != nil {
		storeErr(w, err)
		return
	}
	lat, errLat := strconv.ParseFloat(q.Get("lat"), 64)
	lng, errLng := strconv.ParseFloat(q.Get("lng"), 64)
	if errLat == nil && errLng == nil {
		radius := 50.0
		if v, err := strconv.ParseFloat(q.Get("radiusKm"), 64); err == nil && v > 0 && v <= 500 {
			radius = v
		}
		within := listings[:0]
		for _, l := range listings {
			d := geo.HaversineKm(lat, lng, l.Lat, l.Lng)
			if d <= radius {
				l.DistanceKm = &d
				within = append(within, l)
			}
		}
		listings = within
		sort.SliceStable(listings, func(i, j int) bool { return *listings[i].DistanceKm < *listings[j].DistanceKm })
	}
	writeJSON(w, http.StatusOK, listings)
}

// vendorServiceForOccurrences loads one of the calling vendor's own
// services that is scheduled by dated occurrences.
func (s *Server) vendorServiceForOccurrences(w http.ResponseWriter, r *http.Request, v domain.Vendor, serviceID string) (domain.BookableService, bool) {
	svc, err := s.Store.BookableServiceByID(r.Context(), serviceID)
	if err != nil || svc.VendorID != v.ID || svc.ArchivedAt != nil {
		writeErr(w, http.StatusNotFound, "service not found")
		return svc, false
	}
	if svc.Scheduling != domain.SchedulingOccurrences {
		writeErr(w, http.StatusBadRequest, "this service is booked by weekly times, not dates")
		return svc, false
	}
	return svc, true
}

// handleVendorOccurrences lists a service's dates from today, cancelled
// ones included so the vendor sees their history.
func (s *Server) handleVendorOccurrences(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	svc, ok := s.vendorServiceForOccurrences(w, r, v, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	from := time.Now().Add(-24 * time.Hour)
	list, err := s.Store.OccurrencesWithRemaining(r.Context(), svc, from, true, 500)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

type occurrenceRequest struct {
	StartAt  string `json:"startAt"` // RFC3339
	EndAt    string `json:"endAt"`   // optional: start + the service's duration
	Capacity *int   `json:"capacity,omitempty"`
}

func (req occurrenceRequest) parse(svc domain.BookableService) (domain.ServiceOccurrence, string) {
	var o domain.ServiceOccurrence
	start, err := time.Parse(time.RFC3339, req.StartAt)
	if err != nil {
		return o, "valid startAt required"
	}
	end := start.Add(time.Duration(svc.DurationMins) * time.Minute)
	if req.EndAt != "" {
		if end, err = time.Parse(time.RFC3339, req.EndAt); err != nil {
			return o, "endAt must be a valid time"
		}
	}
	if !end.After(start) {
		return o, "endAt must be after startAt"
	}
	if end.Sub(start) > 31*24*time.Hour {
		return o, "an occurrence can last at most 31 days"
	}
	if !start.After(time.Now()) {
		return o, "pick a date in the future"
	}
	if req.Capacity != nil && *req.Capacity < 1 {
		return o, "capacity must be at least 1"
	}
	o.StartAt, o.EndAt, o.Capacity = start, end, req.Capacity
	return o, ""
}

func (s *Server) handleCreateOccurrence(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	svc, ok := s.vendorServiceForOccurrences(w, r, v, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req occurrenceRequest
	if !readJSON(w, r, &req) {
		return
	}
	o, msg := req.parse(svc)
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	o.ID, o.ServiceID, o.Status = newID("occ"), svc.ID, "scheduled"
	if err := s.Store.CreateOccurrence(r.Context(), o); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

// vendorOccurrence loads an occurrence of one of the calling vendor's
// services.
func (s *Server) vendorOccurrence(w http.ResponseWriter, r *http.Request, v domain.Vendor) (domain.ServiceOccurrence, domain.BookableService, bool) {
	o, err := s.Store.OccurrenceByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusNotFound, "date not found")
		return o, domain.BookableService{}, false
	}
	svc, err := s.Store.BookableServiceByID(r.Context(), o.ServiceID)
	if err != nil || svc.VendorID != v.ID {
		writeErr(w, http.StatusNotFound, "date not found")
		return o, svc, false
	}
	return o, svc, true
}

func (s *Server) handleUpdateOccurrence(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	existing, svc, ok := s.vendorOccurrence(w, r, v)
	if !ok {
		return
	}
	var req occurrenceRequest
	if !readJSON(w, r, &req) {
		return
	}
	o, msg := req.parse(svc)
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	o.ID, o.ServiceID, o.Status = existing.ID, svc.ID, existing.Status
	if err := s.Store.UpdateOccurrence(r.Context(), o); err != nil {
		if err == postgres.ErrOccurrenceLocked {
			writeErr(w, http.StatusConflict, "People have booked this date, so its time can't change. Cancel it instead, which refunds them.")
			return
		}
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// handleCancelOccurrence calls off a date: it takes no more bookings and
// everyone booked on it is cancelled with a full refund.
func (s *Server) handleCancelOccurrence(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	o, _, ok := s.vendorOccurrence(w, r, v)
	if !ok {
		return
	}
	bookings, err := s.Store.CancelOccurrence(r.Context(), o.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	cancelled := 0
	for _, b := range bookings {
		if _, err := s.cancelBooking(r.Context(), b, cancelByVendor, "vendor:"+v.ID); err == nil {
			cancelled++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled", "bookingsCancelled": cancelled})
}

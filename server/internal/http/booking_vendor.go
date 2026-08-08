package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func (s *Server) handleVendorServices(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	services, err := s.Store.BookableServicesByVendor(r.Context(), v.ID, false)
	if err != nil {
		storeErr(w, err)
		return
	}
	if services == nil {
		services = []domain.BookableService{}
	}
	writeJSON(w, http.StatusOK, services)
}

// validateServiceType rejects a booking type that isn't one of the known
// enum values, or that doesn't match what the vendor's category is allowed
// to take bookings for (e.g. a bakery, or a table-only restaurant trying to
// offer a "session" service).
func validateServiceType(category string, t domain.BookingType) error {
	switch t {
	case domain.BookingTypeSession, domain.BookingTypeTable, domain.BookingTypeStay, domain.BookingTypeRental:
	default:
		return errors.New("type must be session, table, stay, or rental")
	}
	if allowed := domain.BookingTypeByCategory(category); allowed == "" {
		return errors.New("vendor category does not support bookings")
	} else if t != allowed {
		return fmt.Errorf("vendor category only supports %s bookings", allowed)
	}
	return nil
}

func (s *Server) handleCreateService(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var svc domain.BookableService
	if !readJSON(w, r, &svc) {
		return
	}
	if svc.Name == "" {
		writeErr(w, http.StatusBadRequest, "name required")
		return
	}
	if err := validateServiceType(v.Category, svc.Type); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if svc.Capacity < 1 {
		svc.Capacity = 1
	}
	svc.ID = newID("svc")
	svc.VendorID = v.ID
	svc.Active = true
	if err := s.Store.UpsertBookableService(r.Context(), svc); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, svc)
}

func (s *Server) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	existing, err := s.Store.BookableServiceByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || existing.VendorID != v.ID {
		writeErr(w, http.StatusNotFound, "service not found")
		return
	}
	var svc domain.BookableService
	if !readJSON(w, r, &svc) {
		return
	}
	if err := validateServiceType(v.Category, svc.Type); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	svc.ID = existing.ID
	svc.VendorID = v.ID
	if svc.Capacity < 1 {
		svc.Capacity = 1
	}
	if err := s.Store.UpsertBookableService(r.Context(), svc); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

// DeleteBookableService cascades to that service's booking history (ON
// DELETE CASCADE) — see the store method's own doc comment.
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteBookableService(r.Context(), chi.URLParam(r, "id"), v.ID); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleVendorBookings(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	bookings, err := s.Store.BookingsByVendor(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if bookings == nil {
		bookings = []domain.Booking{}
	}
	writeJSON(w, http.StatusOK, bookings)
}

func (s *Server) handleVendorCancelBooking(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	if err := s.Store.CancelBookingAsVendor(r.Context(), chi.URLParam(r, "id"), v.ID); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// handleSetBookingStatus lets a vendor mark a booking no_show/completed/
// cancelled by hand — covers a phone booking or a no-show whose code was
// never scanned.
func (s *Server) handleSetBookingStatus(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Status domain.BookingStatus `json:"status"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	switch req.Status {
	case domain.BookingConfirmed, domain.BookingCancelled, domain.BookingCompleted, domain.BookingNoShow:
	default:
		writeErr(w, http.StatusBadRequest, "invalid status")
		return
	}
	if err := s.Store.SetBookingStatus(r.Context(), chi.URLParam(r, "id"), v.ID, req.Status); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleBookingCheckIn is dispatched from handleValidate when a scanned/
// entered code has the "BOOK-" prefix — mirrors the claim-redemption
// result shape (valid/already_redeemed/wrong_vendor/unknown) so the vendor
// portal's existing Validate page needs no structural change, just new
// copy for r.booking vs r.claim.
func (s *Server) handleBookingCheckIn(w http.ResponseWriter, r *http.Request, v domain.Vendor, code string) {
	booking, err := s.Store.BookingByCode(r.Context(), code)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"result": "unknown"})
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	if booking.VendorID != v.ID {
		writeJSON(w, http.StatusOK, map[string]any{"result": "wrong_vendor"})
		return
	}
	if booking.Status != domain.BookingConfirmed {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":     "already_redeemed",
			"redeemedAt": booking.CheckedInAt,
			"booking":    booking,
		})
		return
	}
	checkedIn, err := s.Store.CheckInBooking(r.Context(), booking.ID)
	if errors.Is(err, postgres.ErrNotFound) {
		// Lost the race with a concurrent check-in.
		booking, _ = s.Store.BookingByCode(r.Context(), code)
		writeJSON(w, http.StatusOK, map[string]any{
			"result":     "already_redeemed",
			"redeemedAt": booking.CheckedInAt,
			"booking":    booking,
		})
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "valid", "booking": checkedIn})
}

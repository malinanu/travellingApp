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

// normalizeService applies defaults and rejects invalid settings shared by
// service create and update.
func normalizeService(svc *domain.BookableService) error {
	if svc.Capacity < 1 {
		svc.Capacity = 1
	}
	switch svc.CapacityUnit {
	case "":
		svc.CapacityUnit = domain.CapacityBookings
	case domain.CapacityBookings, domain.CapacityGuests:
	default:
		return errors.New("capacityUnit must be bookings or guests")
	}
	if svc.PricingMode == "" {
		svc.PricingMode = domain.PricingFree
	}
	if !domain.ValidPricingMode(svc.PricingMode) {
		return errors.New("pricingMode must be free, per_person, per_booking or per_night")
	}
	if svc.Currency == "" {
		svc.Currency = domain.CurrencyLKR
	}
	if svc.Currency != domain.CurrencyLKR {
		return errors.New("prices are in LKR")
	}
	switch {
	case svc.PriceMinor < 0:
		return errors.New("price can't be negative")
	case svc.PricingMode == domain.PricingFree:
		svc.PriceMinor = 0
	case svc.PriceMinor == 0:
		return errors.New("set a price, or make the service free")
	case svc.PriceMinor > maxServicePriceMinor:
		return errors.New("price is too high")
	}
	if svc.PricingMode == domain.PricingPerNight && svc.Type != domain.BookingTypeStay {
		return errors.New("per-night pricing is only for stays")
	}
	if svc.FreeCancelHours < 0 || svc.FreeCancelHours > 24*60 {
		return errors.New("freeCancelHours must be between 0 and 1440")
	}
	if svc.LateCancelRefundPct < 0 || svc.LateCancelRefundPct > 100 {
		return errors.New("lateCancelRefundPct must be between 0 and 100")
	}
	return nil
}

// maxServicePriceMinor caps a service price at LKR 10 million, well above
// any real booking, to catch typos (an extra zero or two).
const maxServicePriceMinor = 10_000_000_00

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
	if err := normalizeService(&svc); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	svc.ID = newID("svc")
	svc.VendorID = v.ID
	svc.Active = true
	if err := s.Store.UpsertBookableService(r.Context(), svc); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
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
	if err := normalizeService(&svc); err != nil {
		writeErr(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.Store.UpsertBookableService(r.Context(), svc); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, svc)
}

// handleDeleteService deletes a service, or archives it when it already has
// bookings so their history keeps its reference (see the store method).
func (s *Server) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	archived, err := s.Store.DeleteBookableService(r.Context(), chi.URLParam(r, "id"), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	status := "deleted"
	if archived {
		status = "archived"
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
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
// cancelled by hand â€” covers a phone booking or a no-show whose code was
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
	// Cancelling goes through POST /vendor/bookings/{id}/cancel, which
	// handles refunds; this endpoint only records how a booking ended.
	switch req.Status {
	case domain.BookingCompleted, domain.BookingNoShow:
	default:
		writeErr(w, http.StatusBadRequest, "status must be completed or no_show")
		return
	}
	if err := s.Store.SetBookingStatus(r.Context(), chi.URLParam(r, "id"), v.ID, req.Status); err != nil {
		if errors.Is(err, postgres.ErrInvalidTransition) {
			writeErr(w, http.StatusConflict, "this booking can no longer be changed")
			return
		}
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// handleBookingCheckIn is dispatched from handleValidate when a scanned/
// entered code has the "BOOK-" prefix â€” mirrors the claim-redemption
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

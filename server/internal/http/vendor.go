package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func (s *Server) handleVendorBusiness(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleUpdateVendorBusiness(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var profile json.RawMessage
	if !readJSON(w, r, &profile) {
		return
	}
	if err := s.Store.UpdateVendorProfile(r.Context(), v.ID, profile); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	v.Profile = profile
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleVendorMenu(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, json.RawMessage(orEmpty(v.Menu)))
}

func orEmpty(m json.RawMessage) []byte {
	if len(m) == 0 {
		return []byte(`{}`)
	}
	return m
}

func (s *Server) handleUpdateVendorMenu(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var menu json.RawMessage
	if !readJSON(w, r, &menu) {
		return
	}
	if err := s.Store.UpdateVendorMenu(r.Context(), v.ID, menu); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, menu)
}

func (s *Server) handleVendorOffers(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	offers, err := s.Store.OffersByVendor(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if offers == nil {
		offers = []domain.Offer{}
	}
	writeJSON(w, http.StatusOK, offers)
}

func (s *Server) handleCreateOffer(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var o domain.Offer
	if !readJSON(w, r, &o) {
		return
	}
	if o.Title == "" {
		writeErr(w, http.StatusBadRequest, "title required")
		return
	}
	o.ID = newID("off")
	o.VendorID = v.ID
	if o.Status == "" {
		o.Status = domain.OfferDraft
	}
	o.Claimed, o.Redeemed = 0, 0
	if err := s.Store.UpsertOffer(r.Context(), o); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) handleUpdateOffer(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	existing, err := s.Store.OfferByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || existing.VendorID != v.ID {
		writeErr(w, http.StatusNotFound, "offer not found")
		return
	}
	var o domain.Offer
	if !readJSON(w, r, &o) {
		return
	}
	o.ID = existing.ID
	o.VendorID = v.ID
	if err := s.Store.UpsertOffer(r.Context(), o); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) handleOfferStatus(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Status domain.OfferStatus `json:"status"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Store.SetOfferStatus(r.Context(), chi.URLParam(r, "id"), v.ID, req.Status); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(req.Status)})
}

func (s *Server) handleDeleteOffer(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteOffer(r.Context(), chi.URLParam(r, "id"), v.ID); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleValidate marks a claim redeemed. Responses mirror the prototype's
// validator states: valid / already_redeemed / wrong_vendor / unknown.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		writeErr(w, http.StatusBadRequest, "code required")
		return
	}

	claim, err := s.Store.ClaimByCode(r.Context(), code)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"result": "unknown"})
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	if claim.VendorID == nil || *claim.VendorID != v.ID {
		writeJSON(w, http.StatusOK, map[string]any{"result": "wrong_vendor"})
		return
	}
	if claim.Status == domain.ClaimRedeemed {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":     "already_redeemed",
			"redeemedAt": claim.RedeemedAt,
			"claim":      claim,
		})
		return
	}

	redeemed, err := s.Store.RedeemClaim(r.Context(), claim.ID)
	if errors.Is(err, postgres.ErrNotFound) {
		// Lost the race with a concurrent redeem.
		claim, _ = s.Store.ClaimByCode(r.Context(), code)
		writeJSON(w, http.StatusOK, map[string]any{
			"result":     "already_redeemed",
			"redeemedAt": claim.RedeemedAt,
			"claim":      claim,
		})
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "valid", "claim": redeemed})
}

// blobHandler serves a seeded JSONB document (dashboard/growth/reports).
func (s *Server) blobHandler(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.requireVendor(w, r)
		if !ok {
			return
		}
		data, err := s.Store.VendorBlob(r.Context(), v.ID, kind)
		if errors.Is(err, postgres.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
		if err != nil {
			storeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, json.RawMessage(data))
	}
}

func (s *Server) handleVendorDashboard(w http.ResponseWriter, r *http.Request) {
	s.blobHandler("dashboard")(w, r)
}

func (s *Server) handleVendorGrowth(w http.ResponseWriter, r *http.Request) {
	s.blobHandler("growth")(w, r)
}

func (s *Server) handleVendorReports(w http.ResponseWriter, r *http.Request) {
	s.blobHandler("reports")(w, r)
}

// handleVendorFlashSale records a real flash-sale send against the growth
// blob's `flash.lastSentMins`, enforcing the same 1-per-3h limit the UI
// displays (server-side, not just a disabled button) — a client can't just
// re-enable the button to bypass it.
func (s *Server) handleVendorFlashSale(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	raw, err := s.Store.VendorBlob(r.Context(), v.ID, "growth")
	if err != nil && !errors.Is(err, postgres.ErrNotFound) {
		storeErr(w, err)
		return
	}
	var blob map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &blob); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	if blob == nil {
		blob = map[string]json.RawMessage{}
	}
	var flash struct {
		RadiusKm     int `json:"radiusKm"`
		InRange      int `json:"inRange"`
		Nomads       int `json:"nomads"`
		Surfers      int `json:"surfers"`
		LastSentMins int `json:"lastSentMins"`
	}
	if f, ok := blob["flash"]; ok {
		json.Unmarshal(f, &flash)
	}
	if flash.LastSentMins < 180 {
		writeErr(w, http.StatusTooManyRequests, "a flash sale was already sent within the last 3 hours")
		return
	}
	flash.LastSentMins = 0
	updated, err := json.Marshal(flash)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	blob["flash"] = updated
	newData, err := json.Marshal(blob)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.Store.UpsertVendorBlob(r.Context(), v.ID, "growth", newData); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

func (s *Server) handleVendorRedemptions(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	after := r.URL.Query().Get("after")
	claims, err := s.Store.RedemptionsByVendor(r.Context(), v.ID, &after)
	if err != nil {
		storeErr(w, err)
		return
	}
	if claims == nil {
		claims = []domain.Claim{}
	}
	writeJSON(w, http.StatusOK, claims)
}

func (s *Server) handleVendorNotifications(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	notifs, err := s.Store.VendorNotifications(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if notifs == nil {
		notifs = []domain.VendorNotification{}
	}
	writeJSON(w, http.StatusOK, notifs)
}

func (s *Server) handleVendorNotificationsRead(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	if err := s.Store.MarkVendorNotificationsRead(r.Context(), v.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

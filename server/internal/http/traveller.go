package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// placeJSON flattens Place.Data with id/name/vendorId for the Flutter model.
func placeJSON(p domain.Place) map[string]any {
	m := map[string]any{}
	json.Unmarshal(p.Data, &m)
	m["id"] = p.ID
	m["name"] = p.Name
	if p.VendorID != nil {
		m["vendorId"] = *p.VendorID
	}
	return m
}

func (s *Server) handlePlaces(w http.ResponseWriter, r *http.Request) {
	places, err := s.Store.Places(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(places))
	for _, p := range places {
		out = append(out, placeJSON(p))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePlace(w http.ResponseWriter, r *http.Request) {
	p, err := s.Store.PlaceByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, placeJSON(p))
}

func (s *Server) handleVibeVote(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	placeID := chi.URLParam(r, "id")
	if err := s.Store.AddVibeVote(r.Context(), placeID, chi.URLParam(r, "vibeId"), c.Subject); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	counts, err := s.Store.VibeCounts(r.Context(), placeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"placeId": placeID, "vibes": counts})
}

func (s *Server) handlePlaceReviews(w http.ResponseWriter, r *http.Request) {
	reviews, err := s.Store.ReviewsByPlace(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if reviews == nil {
		reviews = []domain.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

// handleCreateReview is idempotent per traveller+place (submitting again
// edits your existing review). Ratings of 1-2 count against the vendor for
// the admin map's red-flag threshold.
func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	placeID := chi.URLParam(r, "id")
	var req struct {
		Rating int    `json:"rating"`
		Text   string `json:"text"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Rating < 1 || req.Rating > 5 {
		writeErr(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}
	place, err := s.Store.PlaceByID(r.Context(), placeID)
	if err != nil {
		storeErr(w, err)
		return
	}
	review := domain.Review{
		ID:          newID("rev"),
		PlaceID:     place.ID,
		VendorID:    place.VendorID,
		TravellerID: c.Subject,
		Rating:      req.Rating,
		Text:        req.Text,
		CreatedAt:   time.Now(),
	}
	if err := s.Store.UpsertReview(r.Context(), review); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

func (s *Server) handleCheckin(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		PlaceID string `json:"placeId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.PlaceID == "" {
		writeErr(w, http.StatusBadRequest, "placeId required")
		return
	}
	if err := s.Store.AddCheckin(r.Context(), req.PlaceID, c.Subject); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

// handleCreateClaim issues a unique single-use SPOTTO code, idempotent per
// traveller+place (re-claiming returns the existing unredeemed code).
func (s *Server) handleCreateClaim(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		PlaceID    string `json:"placeId"`
		OfferLabel string `json:"offerLabel"`
		OfferValue string `json:"offerValue"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.PlaceID == "" {
		writeErr(w, http.StatusBadRequest, "placeId required")
		return
	}

	if existing, err := s.Store.ClaimForTravellerPlace(r.Context(), c.Subject, req.PlaceID); err == nil &&
		existing.Status == domain.ClaimClaimed {
		writeJSON(w, http.StatusOK, existing)
		return
	}

	place, err := s.Store.PlaceByID(r.Context(), req.PlaceID)
	if err != nil {
		storeErr(w, err)
		return
	}

	claim := domain.Claim{
		ID:          newID("clm"),
		PlaceID:     place.ID,
		OfferLabel:  req.OfferLabel,
		OfferValue:  req.OfferValue,
		VendorID:    place.VendorID,
		TravellerID: c.Subject,
		Status:      domain.ClaimClaimed,
		ClaimedAt:   time.Now(),
	}
	// Retry on the rare code collision (unique index on code).
	for range 5 {
		claim.Code = newClaimCode()
		err = s.Store.CreateClaim(r.Context(), claim)
		if err == nil {
			writeJSON(w, http.StatusCreated, claim)
			return
		}
	}
	writeErr(w, http.StatusInternalServerError, "could not allocate code")
}

func (s *Server) handleMyClaims(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	claims, err := s.Store.ClaimsByTraveller(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	if claims == nil {
		claims = []domain.Claim{}
	}
	writeJSON(w, http.StatusOK, claims)
}

func (s *Server) handleGetTrip(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	t, err := s.Store.TripByOwner(r.Context(), c.Subject)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// handlePutTrip stores the whole trip document (same shape as Trip.toJson()).
func (s *Server) handlePutTrip(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var t domain.Trip
	if !readJSON(w, r, &t) {
		return
	}
	t.OwnerID = c.Subject
	if err := s.Store.UpsertTrip(r.Context(), t); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleGetGroup(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	g, err := s.Store.GroupByOwner(r.Context(), c.Subject)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, g)
}

func (s *Server) handlePutGroup(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var g domain.Group
	if !readJSON(w, r, &g) {
		return
	}
	g.OwnerID = c.Subject
	if err := s.Store.UpsertGroup(r.Context(), g); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

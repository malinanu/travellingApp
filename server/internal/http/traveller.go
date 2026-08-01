package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/placedetails"
	"github.com/spotto/server/internal/store/postgres"
)

// placeJSON flattens Place.Data with id/name/vendorId for the Flutter model,
// plus real vibe vote counts — so the app never needs a hardcoded vibe seed
// even on a place's first load, before anyone in this session has voted.
func (s *Server) placeJSON(ctx context.Context, p domain.Place) map[string]any {
	m := map[string]any{}
	json.Unmarshal(p.Data, &m)
	m["id"] = p.ID
	m["name"] = p.Name
	if p.VendorID != nil {
		m["vendorId"] = *p.VendorID
	}
	if counts, err := s.Store.VibeCounts(ctx, p.ID); err == nil {
		m["vibes"] = counts
	}
	s.signMediaRefs(m)
	return m
}

// signMediaRefs walks an already-unmarshaled JSON value in place (maps and
// slices from encoding/json use shared backing storage, so mutating a
// nested map/slice found inside m is visible through m itself) and signs
// every string that looks like a stored "/media/raw/<key>" path — cover,
// gallery, promoVideo, nested menu item photos, and anything else future
// fields add, without needing to enumerate each field by name.
func (s *Server) signMediaRefs(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if str, ok := child.(string); ok {
				t[k] = s.signMediaPath(str)
			} else {
				s.signMediaRefs(child)
			}
		}
	case []any:
		for i, child := range t {
			if str, ok := child.(string); ok {
				t[i] = s.signMediaPath(str)
			} else {
				s.signMediaRefs(child)
			}
		}
	}
}

func (s *Server) handlePlaces(w http.ResponseWriter, r *http.Request) {
	places, err := s.Store.Places(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	out := make([]map[string]any, 0, len(places))
	for _, p := range places {
		out = append(out, s.placeJSON(r.Context(), p))
	}

	// Optional traveller position — when present, sort nearest-first and
	// inject a human-readable distance instead of leaving Place.distance
	// blank (real imported places never set one).
	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, lngErr := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if latErr == nil && lngErr == nil {
		type withDist struct {
			m    map[string]any
			dist float64
			has  bool
		}
		withDists := make([]withDist, len(out))
		for i, m := range out {
			pLat, latOk := m["lat"].(float64)
			pLng, lngOk := m["lng"].(float64)
			wd := withDist{m: m}
			if latOk && lngOk && (pLat != 0 || pLng != 0) {
				wd.dist = geo.HaversineKm(lat, lng, pLat, pLng)
				wd.has = true
				m["distance"] = fmt.Sprintf("%.1f km away", wd.dist)
				m["distanceKm"] = wd.dist
			}
			withDists[i] = wd
		}
		sort.SliceStable(withDists, func(i, j int) bool {
			if withDists[i].has != withDists[j].has {
				return withDists[i].has // known-distance places sort before unknown
			}
			return withDists[i].dist < withDists[j].dist
		})
		for i, wd := range withDists {
			out[i] = wd.m
		}
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePlace(w http.ResponseWriter, r *http.Request) {
	p, err := s.Store.PlaceByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	m := s.placeJSON(r.Context(), p)
	// Only the single-place fetch enriches with live Places details, not
	// the bulk handlePlaces list above — Google bills per Details call, so
	// this only runs for the one place a traveller actually opened.
	s.enrichWithPlaceDetails(r.Context(), m)
	writeJSON(w, http.StatusOK, m)
}

// enrichWithPlaceDetails backfills phone/website/wheelchairAccessible/
// priceLevel from a live (cached) Google Places Details lookup — only for
// places actually imported from Google (data.source == "google_places"
// with a data.sourceId), and only into fields that are still empty/zero,
// so a curated business's real vendor-entered contact info can never be
// clobbered. Best-effort: any failure (no API key configured, Google
// error, etc.) just leaves the place as-is instead of failing the request.
func (s *Server) enrichWithPlaceDetails(ctx context.Context, m map[string]any) {
	if s.GooglePlacesAPIKey == "" {
		return
	}
	if source, _ := m["source"].(string); source != "google_places" {
		return
	}
	sourceID, _ := m["sourceId"].(string)
	if sourceID == "" {
		return
	}

	details, err := s.getOrFetchPlaceDetails(ctx, sourceID)
	if err != nil {
		log.Printf("placedetails: fetch for %s: %v", sourceID, err)
		return
	}

	contact, _ := m["contact"].(map[string]any)
	if contact == nil {
		contact = map[string]any{}
	}
	if phone, _ := contact["phone"].(string); phone == "" && details.Phone != "" {
		contact["phone"] = details.Phone
	}
	if website, _ := contact["website"].(string); website == "" && details.Website != "" {
		contact["website"] = details.Website
	}
	if len(contact) > 0 {
		m["contact"] = contact
	}

	if details.WheelchairAccessible != nil {
		m["wheelchairAccessible"] = *details.WheelchairAccessible
	}
	if priceLevel, _ := m["priceLevel"].(string); priceLevel == "" && details.PriceLevel != "" {
		m["priceLevel"] = details.PriceLevel
	}
}

// getOrFetchPlaceDetails is the three-tier cache path for Places Details:
// in-memory cache, then the Postgres-backed cache (survives a process
// restart, e.g. every deploy), and only then a real paid Google call —
// mirrors getOrFetchWeather in weather.go.
func (s *Server) getOrFetchPlaceDetails(ctx context.Context, googlePlaceID string) (*placedetails.Details, error) {
	if cached, ok := s.PlaceDetailsCache.Get(googlePlaceID); ok {
		return cached, nil
	}

	if raw, ok, err := s.Store.GetPlaceDetailsCache(ctx, googlePlaceID); err != nil {
		log.Printf("placedetails: read persisted cache for %s: %v", googlePlaceID, err)
	} else if ok {
		var cached placedetails.Details
		if err := json.Unmarshal(raw, &cached); err == nil {
			s.PlaceDetailsCache.Set(googlePlaceID, &cached)
			return &cached, nil
		}
	}

	fresh, err := placedetails.Fetch(ctx, s.GooglePlacesAPIKey, googlePlaceID)
	if err != nil {
		return nil, err
	}
	s.PlaceDetailsCache.Set(googlePlaceID, fresh)
	if raw, err := json.Marshal(fresh); err != nil {
		log.Printf("placedetails: marshal cache for %s: %v", googlePlaceID, err)
	} else if err := s.Store.SetPlaceDetailsCache(ctx, googlePlaceID, raw, time.Now().Add(placedetails.TTL)); err != nil {
		log.Printf("placedetails: persist cache for %s: %v", googlePlaceID, err)
	}
	return fresh, nil
}

// handlePlacePhotos returns the traveller-contributed photos for a place
// (media owned by the place with role "contrib") — kept off the main /places
// payload to avoid an N+1, fetched lazily by the place detail sheet.
func (s *Server) handlePlacePhotos(w http.ResponseWriter, r *http.Request) {
	all, err := s.Store.MediaByOwner(r.Context(), "place", chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	out := []domain.Media{}
	for _, m := range all {
		if m.Role == "contrib" {
			out = append(out, m)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleOfferView records one real traveller impression of an offer's deal
// card — called once per render (not per rebuild) from the Flutter app.
func (s *Server) handleOfferView(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.IncrementOfferViews(r.Context(), chi.URLParam(r, "id")); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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
	// Attach each review's photos in one batch query.
	ids := make([]string, len(reviews))
	for i, rv := range reviews {
		ids[i] = rv.ID
	}
	if photos, err := s.Store.ReviewPhotos(r.Context(), ids); err == nil {
		for i := range reviews {
			urls := photos[reviews[i].ID]
			signed := make([]string, len(urls))
			for j, u := range urls {
				signed[j] = s.signMediaPath(u)
			}
			reviews[i].Photos = signed
		}
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
	// On edit, ON CONFLICT kept the original row's id — return the real
	// stored id so the client attaches photos to the right review.
	if id, err := s.Store.ReviewIDForTravellerPlace(r.Context(), c.Subject, placeID); err == nil {
		review.ID = id
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
		OfferID    string `json:"offerId,omitempty"`
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
	// Resolve offerId defensively — never fail the claim over a bad/stale
	// id; offerLabel/offerValue keep working exactly as before this field
	// existed either way.
	if req.OfferID != "" {
		if o, err := s.Store.OfferByID(r.Context(), req.OfferID); err == nil &&
			place.VendorID != nil && o.VendorID == *place.VendorID {
			claim.OfferID = &o.ID
		}
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

// groupMember is the resolved-profile shape the trip-invite sheet and
// group map markers need — never expose the full domain.User (password
// hash, TOTP fields). Lat/Lng are only set when the member has opted into
// share_location AND reported a position in the last 15 minutes — group
// membership is a mutual-trust circle the member explicitly joined, unlike
// the broader NearbyTravellers lookup which never exposes exact position.
type groupMember struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Email string   `json:"email"`
	Lat   *float64 `json:"lat,omitempty"`
	Lng   *float64 `json:"lng,omitempty"`
}

// groupResp adds resolved member profiles alongside the raw memberIds, so
// the app doesn't need a separate round trip (or fake local contacts) to
// show who's actually in the group.
type groupResp struct {
	domain.Group
	Members []groupMember `json:"members"`
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
	users, err := s.Store.UsersByIDs(r.Context(), g.MemberIDs)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	members := make([]groupMember, 0, len(users))
	for _, u := range users {
		m := groupMember{ID: u.ID, Name: u.Name, Email: u.Email}
		fresh := u.LocationUpdatedAt != nil && time.Since(*u.LocationUpdatedAt) < 15*time.Minute
		if u.ShareLocation && fresh {
			m.Lat, m.Lng = u.Lat, u.Lng
		}
		members = append(members, m)
	}
	writeJSON(w, http.StatusOK, groupResp{Group: g, Members: members})
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

// handleUpdateMyLocation records the caller's current position for the
// nearby-travellers feature. Requires share_location to already be on
// (via PUT /me/privacy) — reporting a position you haven't consented to
// share would defeat the point of the toggle.
func (s *Server) handleUpdateMyLocation(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	if !u.ShareLocation {
		writeErr(w, http.StatusForbidden, "enable location sharing first (PUT /me/privacy)")
		return
	}
	var req struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Lat < -90 || req.Lat > 90 || req.Lng < -180 || req.Lng > 180 {
		writeErr(w, http.StatusBadRequest, "lat/lng out of range")
		return
	}
	if err := s.Store.UpdateUserLocation(r.Context(), u.ID, req.Lat, req.Lng); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleNearbyTravellers requires the caller to be opted in AND have a
// recent position of their own — seeing others without being visible
// yourself would be a one-way surveillance pattern this deliberately avoids.
func (s *Server) handleNearbyTravellers(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	u, err := s.Store.UserByID(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	if !u.ShareLocation || u.Lat == nil || u.Lng == nil {
		writeErr(w, http.StatusForbidden, "enable location sharing and report your position first")
		return
	}
	radiusKm := 5.0
	if v := r.URL.Query().Get("radiusKm"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil && parsed > 0 {
			radiusKm = parsed
		}
	}
	if radiusKm > 50 {
		radiusKm = 50 // don't let this become a whole-country query
	}
	nearby, err := s.Store.NearbyTravellers(r.Context(), u.ID, *u.Lat, *u.Lng, radiusKm)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, nearby)
}

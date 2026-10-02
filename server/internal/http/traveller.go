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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/placedetails"
	"github.com/spotto/server/internal/recommend"
	"github.com/spotto/server/internal/store/postgres"
)

// normalizePlaceCategory fills in categoryId and the pin color for places
// stored before those fields were written (imported spots never carried
// them), so the app can show the right category icon on the map without a
// destructive re-import. Rows that already have a known categoryId keep it.
func normalizePlaceCategory(m map[string]any) {
	id, _ := m["categoryId"].(string)
	cat, ok := domain.CategoryByID(id)
	if !ok {
		cat, ok = domain.CategoryByID(domain.CategoryForPlaceData(m))
		if !ok {
			return
		}
		m["categoryId"] = cat.ID
	}
	if color, _ := m["color"].(string); color == "" || color == "teal" {
		m["color"] = domain.PinColorForSegment(cat.Segment)
	}
}

// placeJSON flattens Place.Data with id/name/vendorId for the Flutter model,
// plus the place's real current vibe aggregate â€” so the app never needs a
// hardcoded vibe seed even on a place's first load, before anyone in this
// session has checked in.
func (s *Server) placeJSON(ctx context.Context, p domain.Place) map[string]any {
	m := map[string]any{}
	json.Unmarshal(p.Data, &m)
	delete(m, "apiKey")
	delete(m, "googleApiKey")
	delete(m, "key")
	delete(m, "secret")
	delete(m, "token")
	m["id"] = p.ID
	m["name"] = p.Name
	if p.VendorID != nil {
		m["vendorId"] = *p.VendorID
	}
	normalizePlaceCategory(m)
	if vibe, err := s.Store.VibeAggregate(ctx, p.ID); err == nil {
		m["vibe"] = vibe
	}
	s.signMediaRefs(m)
	return m
}

// signMediaRefs walks an already-unmarshaled JSON value in place (maps and
// slices from encoding/json use shared backing storage, so mutating a
// nested map/slice found inside m is visible through m itself) and signs
// every string that looks like a stored "/media/raw/<key>" path â€” cover,
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

	categoryFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("category")))
	searchFilter := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("search")))
	if searchFilter == "" {
		searchFilter = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	}

	out := make([]map[string]any, 0, len(places))
	for _, p := range places {
		m := s.placeJSON(r.Context(), p)

		if categoryFilter != "" {
			pCat, _ := m["category"].(string)
			pSub, _ := m["subcategory"].(string)
			if !strings.EqualFold(pCat, categoryFilter) && !strings.EqualFold(pSub, categoryFilter) {
				continue
			}
		}

		if searchFilter != "" {
			pName, _ := m["name"].(string)
			pCat, _ := m["category"].(string)
			pDesc, _ := m["description"].(string)
			pTown, _ := m["town"].(string)
			match := strings.Contains(strings.ToLower(pName), searchFilter) ||
				strings.Contains(strings.ToLower(pCat), searchFilter) ||
				strings.Contains(strings.ToLower(pDesc), searchFilter) ||
				strings.Contains(strings.ToLower(pTown), searchFilter)
			if !match {
				continue
			}
		}

		out = append(out, m)
	}

	lat, latErr := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, lngErr := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	maxDistStr := r.URL.Query().Get("maxDistanceKm")
	if maxDistStr == "" {
		maxDistStr = r.URL.Query().Get("radiusKm")
	}
	maxDist, maxDistErr := strconv.ParseFloat(maxDistStr, 64)

	if latErr == nil && lngErr == nil {
		type withDist struct {
			m    map[string]any
			dist float64
			has  bool
		}
		withDists := make([]withDist, 0, len(out))
		for _, m := range out {
			pLat, latOk := m["lat"].(float64)
			pLng, lngOk := m["lng"].(float64)
			wd := withDist{m: m}
			if latOk && lngOk && (pLat != 0 || pLng != 0) {
				wd.dist = geo.HaversineKm(lat, lng, pLat, pLng)
				wd.has = true
				if maxDistErr == nil && maxDist > 0 && wd.dist > maxDist {
					continue
				}
				m["distance"] = fmt.Sprintf("%.1f km away", wd.dist)
				m["distanceKm"] = wd.dist
			}
			withDists = append(withDists, wd)
		}
		sort.SliceStable(withDists, func(i, j int) bool {
			if withDists[i].has != withDists[j].has {
				return withDists[i].has
			}
			return withDists[i].dist < withDists[j].dist
		})
		out = make([]map[string]any, len(withDists))
		for i, wd := range withDists {
			out[i] = wd.m
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 && limit < len(out) {
			out = out[:limit]
		}
	}

	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSearchPlaces(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeErr(w, http.StatusBadRequest, "query 'q' is required")
		return
	}

	if s.GeminiAPIKey == "" {
		writeErr(w, http.StatusServiceUnavailable, "semantic search is not configured")
		return
	}

	emb, err := gemini.Embed(r.Context(), s.GeminiAPIKey, []string{q})
	if err != nil || len(emb) == 0 {
		writeErr(w, http.StatusInternalServerError, "failed to embed query")
		return
	}

	places, err := s.Store.Places(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}

	prefs := recommend.Preferences{
		QueryEmbedding: emb[0],
	}

	scored := recommend.Rank(places, prefs, "")
	
	limit := 10
	if len(scored) < limit {
		limit = len(scored)
	}
	
	out := make([]map[string]any, 0, limit)
	for i := 0; i < limit; i++ {
		// Only include places with a positive semantic match (> 0.5)
		if scored[i].Score <= 0.5 {
			continue
		}
		m := s.placeJSON(r.Context(), scored[i].Place)
		m["semanticScore"] = scored[i].Score
		m["matchReasons"] = scored[i].Reasons
		out = append(out, m)
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
	// Only the single-place fetch looks up the caller's own latest vibe
	// check-in, not the bulk handlePlaces list above â€” same "don't N+1 the
	// list" reasoning as enrichWithPlaceDetails below.
	if mine, err := s.Store.MyLatestVibeCheckin(r.Context(), p.ID, auth.FromContext(r.Context()).Subject); err == nil {
		m["myVibe"] = mine
	}
	// Only the single-place fetch enriches with live Places details, not
	// the bulk handlePlaces list above â€” Google bills per Details call, so
	// this only runs for the one place a traveller actually opened.
	s.enrichWithPlaceDetails(r.Context(), m)
	writeJSON(w, http.StatusOK, m)
}

// enrichWithPlaceDetails backfills phone/website/wheelchairAccessible/
// priceLevel from a live (cached) Google Places Details lookup â€” only for
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
// restart, e.g. every deploy), and only then a real paid Google call â€”
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
// (media owned by the place with role "contrib") â€” kept off the main /places
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
	writeJSON(w, http.StatusOK, s.signMediaList(out))
}

// handleOfferView records one real traveller impression of an offer's deal
// card â€” called once per render (not per rebuild) from the Flutter app.
func (s *Server) handleOfferView(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.IncrementOfferViews(r.Context(), chi.URLParam(r, "id")); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// vibeCheckinGeofenceMeters bounds how close a traveller must be to a place
// to rate its vibe. This is collected for future vibe-based recommendation
// analytics â€” a rating submitted from anywhere would poison that, so it's
// enforced here server-side, not just hidden client-side.
const vibeCheckinGeofenceMeters = 150.0

// vibeCheckinCooldown is the minimum gap between one traveller's check-ins
// for the same place â€” check-ins are otherwise unlimited (see
// handleVibeCheckin's doc comment), so without this a traveller standing at
// a place could resubmit every request and effectively overwrite the
// aggregate at will.
const vibeCheckinCooldown = 15 * time.Minute

// handleVibeCheckin records one traveller's full vibe snapshot for a place â€”
// always a new row (see Store.AddVibeCheckin), never an edit of a previous
// check-in, so a traveller can re-rate a place on a later visit. Gated by
// vibeCheckinCooldown so the same traveller can't resubmit indefinitely.
func (s *Server) handleVibeCheckin(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	placeID := chi.URLParam(r, "id")
	var req struct {
		Productive int     `json:"productive"`
		Loud       int     `json:"loud"`
		Crowded    int     `json:"crowded"`
		Lat        float64 `json:"lat"`
		Lng        float64 `json:"lng"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	// 0 stays valid â€” the "not yet rated" sentinel for whichever dimension
	// wasn't part of this particular tap (see VibeCheckInsNotifier.rate());
	// 1-5 are the real ratings.
	if req.Productive < 0 || req.Productive > 5 || req.Loud < 0 || req.Loud > 5 || req.Crowded < 0 || req.Crowded > 5 {
		writeErr(w, http.StatusBadRequest, "ratings must be between 0 and 5")
		return
	}
	place, err := s.Store.PlaceByID(r.Context(), placeID)
	if err != nil {
		storeErr(w, err)
		return
	}
	var loc struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	json.Unmarshal(place.Data, &loc)
	if geo.HaversineKm(req.Lat, req.Lng, loc.Lat, loc.Lng)*1000 > vibeCheckinGeofenceMeters {
		writeErr(w, http.StatusForbidden, "you're too far from this place to rate its vibe")
		return
	}
	if last, err := s.Store.MyLatestVibeCheckin(r.Context(), placeID, c.Subject); err == nil {
		if wait := vibeCheckinCooldown - time.Since(last.CreatedAt); wait > 0 {
			writeErr(w, http.StatusTooManyRequests, fmt.Sprintf("you can rate this place again in %d minutes", int(wait.Minutes())+1))
			return
		}
	} else if !errors.Is(err, postgres.ErrNotFound) {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if err := s.Store.AddVibeCheckin(r.Context(), placeID, c.Subject, req.Productive, req.Loud, req.Crowded); err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	vibe, err := s.Store.VibeAggregate(r.Context(), placeID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"placeId": placeID,
		"vibe":    vibe,
		"myVibe": domain.VibeCheckin{
			PlaceID: placeID, TravellerID: c.Subject,
			Productive: req.Productive, Loud: req.Loud, Crowded: req.Crowded, CreatedAt: time.Now(),
		},
	})
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
			list := photos[reviews[i].ID]
			for j := range list {
				list[j].URL = s.signMediaPath(list[j].URL)
				if list[j].ThumbKey != "" && s.Media != nil {
					list[j].ThumbURL = s.signMediaPath(s.Media.PublicURL(list[j].ThumbKey))
				}
			}
			reviews[i].Photos = list
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
	req.Text = sanitizeText(req.Text, 2000)
	if !validateTextLength(w, "Review text", req.Text, 2000) {
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if place.VendorID != nil {
		go s.TriggerReviewNotification(context.Background(), *place.VendorID, "Traveller", req.Rating)
	}
	// On edit, ON CONFLICT kept the original row's id â€” return the real
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) handleGetQuest(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	
	// Check progress today
	progress, err := s.Store.GetDailyQuestProgress(r.Context(), c.Subject)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get quest progress")
		return
	}

	// Fetch the schedule for today
	today := time.Now().UTC().Format("2006-01-02")
	schedule, err := s.Store.GetQuestSchedule(r.Context(), today)
	if err != nil && err != postgres.ErrNotFound {
		writeErr(w, http.StatusInternalServerError, "failed to get quest schedule")
		return
	}

	goal := 5
	rewardDesc := "Mystery Reward"
	sponsorName := "Spotto"
	sponsorID := "system"

	if err == nil {
		goal = schedule.Goal
		rewardDesc = schedule.RewardDescription
		sponsorName = schedule.SponsorName
		sponsorID = schedule.VendorID
	}

	quest := domain.DailyQuest{
		Goal:              goal,
		Progress:          progress,
		RewardDescription: rewardDesc,
		IsComplete:        progress >= goal,
		SponsorName:       sponsorName,
		SponsorID:         sponsorID,
	}

	writeJSON(w, http.StatusOK, quest)
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

	if existing, err := s.Store.ClaimForTravellerPlace(r.Context(), c.Subject, req.PlaceID); err == nil {
		if existing.Status == domain.ClaimClaimed {
			writeJSON(w, http.StatusOK, existing)
			return
		}
		if existing.Status == domain.ClaimRedeemed {
			writeErr(w, http.StatusBadRequest, "you have already redeemed an offer at this spot")
			return
		}
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
	// Resolve offerId defensively â€” never fail the claim over a bad/stale
	// id; offerLabel/offerValue keep working exactly as before this field
	// existed either way.
	if req.OfferID != "" {
		if o, err := s.Store.OfferByID(r.Context(), req.OfferID); err == nil &&
			place.VendorID != nil && o.VendorID == *place.VendorID {
			if o.Limit > 0 && o.Claimed >= o.Limit {
				writeErr(w, http.StatusBadRequest, "this offer has reached its claim limit")
				return
			}
			claim.OfferID = &o.ID
		}
	}
	// Retry on the rare code collision (unique index on code).
	for range 5 {
		claim.Code = newClaimCode()
		err = s.Store.CreateClaim(r.Context(), claim)
		if err == nil {
			if place.VendorID != nil {
				go s.TriggerClaimNotification(context.Background(), *place.VendorID, "Traveller", req.OfferLabel)
			}
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// groupMember is the resolved-profile shape the trip-invite sheet and
// group map markers need â€” never expose the full domain.User (password
// hash, TOTP fields). Lat/Lng are only set when the member has opted into
// share_location AND reported a position in the last 15 minutes â€” group
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, g)
}

// handleUpdateMyLocation records the caller's current position for the
// nearby-travellers feature. Requires share_location to already be on
// (via PUT /me/privacy) â€” reporting a position you haven't consented to
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// handleNearbyTravellers requires the caller to be opted in AND have a
// recent position of their own â€” seeing others without being visible
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusOK, nearby)
}

// handleListTrips returns all saved trips for the authenticated traveler.
func (s *Server) handleListTrips(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	trips, err := s.Store.TripsByOwner(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trips)
}

// handleSaveTrip saves or updates a traveler's trip itinerary.
func (s *Server) handleSaveTrip(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var trip domain.Trip
	if !readJSON(w, r, &trip) {
		return
	}
	trip.OwnerID = c.Subject
	trip.UpdatedAt = time.Now().UTC()
	if err := s.Store.UpsertTrip(r.Context(), trip); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, trip)
}

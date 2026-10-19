package httpapi

// Inspiration Board API: a traveller's saved links with the places found in
// each (inspiration_worker.go does the finding), their "save for later"
// places, and Google place photos.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/reelextract"
	"github.com/spotto/server/internal/store/postgres"
)

type extractedPlaceView struct {
	domain.ExtractedPlace
	GeoPlace *geoPlaceView `json:"geoPlace,omitempty"`
}

type geoPlaceView struct {
	domain.GeoPlace
	// PhotoURL is relative to the API (like /media/raw paths); "" when Google has no photo.
	PhotoURL string `json:"photoUrl,omitempty"`
}

func newGeoPlaceView(p domain.GeoPlace) *geoPlaceView {
	v := &geoPlaceView{GeoPlace: p}
	if p.HasPhoto {
		v.PhotoURL = "/geo-places/" + p.ID + "/photo"
	}
	return v
}

type inspirationView struct {
	ID           string               `json:"id"`
	URL          string               `json:"url"`
	CanonicalURL string               `json:"canonicalUrl"`
	Note         string               `json:"note"`
	CreatedAt    time.Time            `json:"createdAt"`
	Platform     string               `json:"platform"`
	Status       string               `json:"status"`
	Error        string               `json:"error,omitempty"`
	Caption      string               `json:"caption,omitempty"`
	Author       string               `json:"author,omitempty"`
	ThumbnailURL string               `json:"thumbnailUrl,omitempty"`
	Places       []extractedPlaceView `json:"places"`
}

const maxCaptionChars = 600

// inspirationViews joins links with their shared extraction and resolved places.
func (s *Server) inspirationViews(ctx context.Context, links []domain.InspirationLink) ([]inspirationView, error) {
	urls := make([]string, 0, len(links))
	for _, l := range links {
		urls = append(urls, l.CanonicalURL)
	}
	exts, err := s.Store.LinkExtractionsByURLs(ctx, urls)
	if err != nil {
		return nil, err
	}
	var geoIDs []string
	for _, e := range exts {
		for _, p := range e.Results {
			if p.GeoPlaceID != "" {
				geoIDs = append(geoIDs, p.GeoPlaceID)
			}
		}
	}
	geos, err := s.Store.GeoPlacesByIDs(ctx, geoIDs)
	if err != nil {
		return nil, err
	}

	out := make([]inspirationView, 0, len(links))
	for _, l := range links {
		e := exts[l.CanonicalURL]
		v := inspirationView{
			ID: l.ID, URL: l.URL, CanonicalURL: l.CanonicalURL, Note: l.Note, CreatedAt: l.CreatedAt,
			Platform: e.Platform, Status: e.Status, Error: e.Error, Author: e.Author,
			ThumbnailURL: e.ThumbnailURL, Places: []extractedPlaceView{},
		}
		if v.Status == domain.ExtractionProcessing {
			v.Error = "" // a retry's previous error isn't news while it's running
		}
		v.Caption = truncateRunes(e.Caption, maxCaptionChars)
		for _, p := range e.Results {
			pv := extractedPlaceView{ExtractedPlace: p}
			if g, ok := geos[p.GeoPlaceID]; ok {
				pv.GeoPlace = newGeoPlaceView(g)
			}
			v.Places = append(v.Places, pv)
		}
		out = append(out, v)
	}
	return out, nil
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func (s *Server) inspirationView(ctx context.Context, l domain.InspirationLink) (inspirationView, error) {
	views, err := s.inspirationViews(ctx, []domain.InspirationLink{l})
	if err != nil {
		return inspirationView{}, err
	}
	return views[0], nil
}

func (s *Server) handleListInspirations(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	links, err := s.Store.InspirationLinks(r.Context(), c.Subject)
	if err != nil {
		log.Printf("list inspirations user=%s: %v", c.Subject, err)
		storeErr(w, err)
		return
	}
	views, err := s.inspirationViews(r.Context(), links)
	if err != nil {
		log.Printf("list inspirations views user=%s: %v", c.Subject, err)
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"links": views})
}

func (s *Server) handleAddInspiration(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		URL  string `json:"url"`
		Note string `json:"note"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	canonical, platform, err := reelextract.Canonical(reelextract.ResolveShortLink(r.Context(), req.URL))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "Paste a link to an Instagram reel or post, a TikTok video, or a YouTube video.")
		return
	}

	since := time.Now().Add(-24 * time.Hour)
	n, err := s.Store.CountInspirationLinksSince(r.Context(), c.Subject, since)
	if err != nil {
		log.Printf("add inspiration count user=%s: %v", c.Subject, err)
		storeErr(w, err)
		return
	}
	if n >= s.InspirationDailyLimit {
		writeErr(w, http.StatusTooManyRequests,
			fmt.Sprintf("You've added %d links today. You can add more tomorrow.", s.InspirationDailyLimit))
		return
	}

	if err := s.Store.EnsureLinkExtraction(r.Context(), canonical, platform); err != nil {
		log.Printf("add inspiration extraction %s: %v", canonical, err)
		storeErr(w, err)
		return
	}
	link, created, err := s.Store.CreateInspirationLink(r.Context(), domain.InspirationLink{
		ID: newID("insp"), UserID: c.Subject, URL: truncateRunes(req.URL, 2000),
		CanonicalURL: canonical, Note: sanitizeText(req.Note, 500),
	})
	if err != nil {
		log.Printf("add inspiration user=%s: %v", c.Subject, err)
		storeErr(w, err)
		return
	}
	view, err := s.inspirationView(r.Context(), link)
	if err != nil {
		storeErr(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusAccepted // extraction runs in the background
	}
	writeJSON(w, status, view)
}

func (s *Server) handleGetInspiration(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	link, err := s.Store.InspirationLink(r.Context(), c.Subject, chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	view, err := s.inspirationView(r.Context(), link)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) handleDeleteInspiration(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	if err := s.Store.DeleteInspirationLink(r.Context(), c.Subject, chi.URLParam(r, "id")); err != nil {
		storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRetryInspiration(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	link, err := s.Store.InspirationLink(r.Context(), c.Subject, chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if err := s.Store.RequeueExtraction(r.Context(), link.CanonicalURL); err != nil {
		log.Printf("retry inspiration %s: %v", link.CanonicalURL, err)
		storeErr(w, err)
		return
	}
	view, err := s.inspirationView(r.Context(), link)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, view)
}

// ---- saved places ----

type catalogPlaceSummary struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
	CategoryID string  `json:"categoryId,omitempty"`
}

type savedPlaceView struct {
	domain.SavedPlace
	GeoPlace *geoPlaceView        `json:"geoPlace,omitempty"`
	Place    *catalogPlaceSummary `json:"place,omitempty"`
}

func (s *Server) catalogPlaceSummary(ctx context.Context, id string) (*catalogPlaceSummary, error) {
	p, err := s.Store.PlaceByID(ctx, id)
	if err != nil {
		return nil, err
	}
	var facts struct {
		Lat        float64 `json:"lat"`
		Lng        float64 `json:"lng"`
		CategoryID string  `json:"categoryId"`
	}
	_ = json.Unmarshal(p.Data, &facts)
	return &catalogPlaceSummary{ID: p.ID, Name: p.Name, Lat: facts.Lat, Lng: facts.Lng, CategoryID: facts.CategoryID}, nil
}

func (s *Server) savedPlaceViews(ctx context.Context, saved []domain.SavedPlace) ([]savedPlaceView, error) {
	var geoIDs []string
	for _, sp := range saved {
		if sp.GeoPlaceID != "" {
			geoIDs = append(geoIDs, sp.GeoPlaceID)
		}
	}
	geos, err := s.Store.GeoPlacesByIDs(ctx, geoIDs)
	if err != nil {
		return nil, err
	}
	out := make([]savedPlaceView, 0, len(saved))
	for _, sp := range saved {
		v := savedPlaceView{SavedPlace: sp}
		if g, ok := geos[sp.GeoPlaceID]; ok {
			v.GeoPlace = newGeoPlaceView(g)
		}
		if sp.PlaceID != "" {
			p, err := s.catalogPlaceSummary(ctx, sp.PlaceID)
			if errors.Is(err, postgres.ErrNotFound) {
				continue // the catalog dropped it on reimport; nothing to show
			}
			if err != nil {
				return nil, err
			}
			v.Place = p
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Server) handleListSavedPlaces(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	saved, err := s.Store.SavedPlaces(r.Context(), c.Subject)
	if err != nil {
		log.Printf("list saved places user=%s: %v", c.Subject, err)
		storeErr(w, err)
		return
	}
	views, err := s.savedPlaceViews(r.Context(), saved)
	if err != nil {
		log.Printf("list saved places views user=%s: %v", c.Subject, err)
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"places": views})
}

func (s *Server) handleAddSavedPlace(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	var req struct {
		PlaceID      string `json:"placeId"`
		GeoPlaceID   string `json:"geoPlaceId"`
		SourceLinkID string `json:"sourceLinkId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if (req.PlaceID == "") == (req.GeoPlaceID == "") {
		writeErr(w, http.StatusBadRequest, "send exactly one of placeId or geoPlaceId")
		return
	}
	if req.PlaceID != "" {
		if _, err := s.Store.PlaceByID(r.Context(), req.PlaceID); err != nil {
			storeErr(w, err)
			return
		}
	} else if _, err := s.Store.GeoPlaceByID(r.Context(), req.GeoPlaceID); err != nil {
		storeErr(w, err)
		return
	}
	if req.SourceLinkID != "" {
		if _, err := s.Store.InspirationLink(r.Context(), c.Subject, req.SourceLinkID); err != nil {
			req.SourceLinkID = "" // not this traveller's link; don't record it
		}
	}
	sp, err := s.Store.AddSavedPlace(r.Context(), domain.SavedPlace{
		ID: newID("sav"), UserID: c.Subject, PlaceID: req.PlaceID,
		GeoPlaceID: req.GeoPlaceID, SourceLinkID: req.SourceLinkID,
	})
	if err != nil {
		log.Printf("add saved place user=%s: %v", c.Subject, err)
		storeErr(w, err)
		return
	}
	views, err := s.savedPlaceViews(r.Context(), []domain.SavedPlace{sp})
	if err != nil || len(views) == 0 {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, views[0])
}

func (s *Server) handleDeleteSavedPlace(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	if err := s.Store.DeleteSavedPlace(r.Context(), c.Subject, chi.URLParam(r, "id")); err != nil {
		storeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- photos ----

// geoPhotoCache keeps recent photos in memory: each Google photo fetch is
// billed, and a board shows the same few dozen photos over and over.
var geoPhotoCache = newPhotoCache(300)

type cachedPhoto struct {
	data        []byte
	contentType string
}

type photoCache struct {
	mu    sync.Mutex
	max   int
	order []string
	items map[string]cachedPhoto
}

func newPhotoCache(max int) *photoCache {
	return &photoCache{max: max, items: map[string]cachedPhoto{}}
}

func (c *photoCache) get(id string) (cachedPhoto, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, ok := c.items[id]
	return p, ok
}

func (c *photoCache) put(id string, p cachedPhoto) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.items[id]; ok {
		return
	}
	if len(c.order) >= c.max {
		delete(c.items, c.order[0])
		c.order = c.order[1:]
	}
	c.items[id] = p
	c.order = append(c.order, id)
}

func (s *Server) handleGeoPlacePhoto(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	photo, ok := geoPhotoCache.get(id)
	if !ok {
		if s.Places == nil {
			writeErr(w, http.StatusNotFound, "no photo")
			return
		}
		gp, err := s.Store.GeoPlaceByID(r.Context(), id)
		if err != nil {
			storeErr(w, err)
			return
		}
		if gp.PhotoName == "" {
			writeErr(w, http.StatusNotFound, "no photo")
			return
		}
		data, ct, err := s.Places.Photo(r.Context(), gp.PhotoName, 800)
		if err != nil {
			log.Printf("geo place photo %s: %v", id, err)
			writeErr(w, http.StatusBadGateway, "couldn't load this photo right now")
			return
		}
		if ct == "" {
			ct = "image/jpeg"
		}
		photo = cachedPhoto{data: data, contentType: ct}
		geoPhotoCache.put(id, photo)
	}
	w.Header().Set("Content-Type", photo.contentType)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	_, _ = w.Write(photo.data)
}

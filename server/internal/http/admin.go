package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/domain"
)

// handleAdminKPIs mixes live counts with prototype-style topline numbers.
func (s *Server) handleAdminKPIs(w http.ResponseWriter, r *http.Request) {
	pending, feedCount, err := s.Store.CountAdminKPIBase(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	subs, err := s.Store.Subscriptions(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	mrr := 0
	for _, sub := range subs {
		mrr += sub.MRR
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pendingVendors": pending,
		"feedItems":      feedCount,
		"mrr":            mrr,
		"activeSubs":     len(subs),
	})
}

func (s *Server) handleAdminVendors(w http.ResponseWriter, r *http.Request) {
	vendors, err := s.Store.Vendors(r.Context(), r.URL.Query().Get("status"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if vendors == nil {
		vendors = []domain.Vendor{}
	}
	writeJSON(w, http.StatusOK, vendors)
}

func (s *Server) vendorStatusHandler(status domain.VendorStatus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := s.Store.SetVendorStatus(r.Context(), id, status); err != nil {
			storeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": string(status)})
	}
}

func (s *Server) handleAdminFeed(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	items, err := s.Store.FeedItems(r.Context(), after, r.URL.Query().Get("status"), 100)
	if err != nil {
		storeErr(w, err)
		return
	}
	if items == nil {
		items = []domain.FeedItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) feedResolveHandler(resolved string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "invalid id")
			return
		}
		if err := s.Store.ResolveFeedItem(r.Context(), id, resolved); err != nil {
			storeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": id, "resolved": resolved})
	}
}

func (s *Server) handleAdminRegions(w http.ResponseWriter, r *http.Request) {
	regions, err := s.Store.Regions(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	if regions == nil {
		regions = []domain.Region{}
	}
	writeJSON(w, http.StatusOK, regions)
}

func (s *Server) handleAdminSubscriptions(w http.ResponseWriter, r *http.Request) {
	subs, err := s.Store.Subscriptions(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	if subs == nil {
		subs = []domain.Subscription{}
	}
	writeJSON(w, http.StatusOK, subs)
}

func (s *Server) handleUpgradeSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MRR int `json:"mrr"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.MRR <= 0 {
		req.MRR = 49 // Premium price in USD, matching the seeded subscriptions
	}
	if err := s.Store.UpgradeSubscription(r.Context(), chi.URLParam(r, "id"), req.MRR); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "upgraded"})
}

func (s *Server) handleAudioGuides(w http.ResponseWriter, r *http.Request) {
	guides, err := s.Store.AudioGuides(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	if guides == nil {
		guides = []domain.AudioGuide{}
	}
	writeJSON(w, http.StatusOK, guides)
}

func (s *Server) handleCreateAudioGuide(w http.ResponseWriter, r *http.Request) {
	var g domain.AudioGuide
	if !readJSON(w, r, &g) {
		return
	}
	if g.Site == "" {
		writeErr(w, http.StatusBadRequest, "site required")
		return
	}
	g.ID = newID("aud")
	if err := s.Store.UpsertAudioGuide(r.Context(), g); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handlePublishAudioGuide(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Published bool `json:"published"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Store.SetAudioGuidePublished(r.Context(), chi.URLParam(r, "id"), req.Published); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"published": req.Published})
}

package httpapi

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spotto/server/internal/domain"
)

func (s *Server) handleAdminListPendingSponsorships(w http.ResponseWriter, r *http.Request) {
	spons, err := s.Store.ListVendorSponsorshipsByStatus(r.Context(), "pending")
	if err != nil {
		storeErr(w, err)
		return
	}
	if spons == nil {
		spons = []domain.VendorSponsorship{}
	}
	writeJSON(w, http.StatusOK, spons)
}

func (s *Server) handleAdminListApprovedSponsorships(w http.ResponseWriter, r *http.Request) {
	spons, err := s.Store.ListVendorSponsorshipsByStatus(r.Context(), "approved")
	if err != nil {
		storeErr(w, err)
		return
	}
	if spons == nil {
		spons = []domain.VendorSponsorship{}
	}
	writeJSON(w, http.StatusOK, spons)
}

func (s *Server) handleAdminUpdateSponsorshipStatus(w http.ResponseWriter, r *http.Request) {
	vendorID := chi.URLParam(r, "vendor_id")
	var req struct {
		Status string `json:"status"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Status != "approved" && req.Status != "rejected" && req.Status != "pending" {
		writeErr(w, http.StatusBadRequest, "invalid status")
		return
	}
	
	if err := s.Store.UpdateVendorSponsorshipStatus(r.Context(), vendorID, req.Status); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAdminListQuestSchedules(w http.ResponseWriter, r *http.Request) {
	scheds, err := s.Store.ListQuestSchedules(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	if scheds == nil {
		scheds = []domain.QuestSchedule{}
	}
	writeJSON(w, http.StatusOK, scheds)
}

func (s *Server) handleAdminSetQuestSchedule(w http.ResponseWriter, r *http.Request) {
	var req domain.QuestSchedule
	if !readJSON(w, r, &req) {
		return
	}
	if req.Date == "" || req.VendorID == "" {
		writeErr(w, http.StatusBadRequest, "date and vendorId required")
		return
	}
	if req.Goal <= 0 {
		req.Goal = 5
	}
	if err := s.Store.SetQuestSchedule(r.Context(), req); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleAdminQuestCompletions lists recorded quest completions, newest
// first. Filters: ?date=YYYY-MM-DD, ?vendorId=, ?travellerId=, and
// ?before=<RFC3339 completedAt of the last row> to page.
func (s *Server) handleAdminQuestCompletions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.QuestCompletionFilter{
		Date:        q.Get("date"),
		VendorID:    q.Get("vendorId"),
		TravellerID: q.Get("travellerId"),
		Limit:       100,
	}
	if b := q.Get("before"); b != "" {
		t, err := time.Parse(time.RFC3339Nano, b)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "before must be an RFC3339 timestamp")
			return
		}
		f.Before = &t
	}
	list, err := s.Store.ListQuestCompletions(r.Context(), f)
	if err != nil {
		log.Printf("admin quest completions: %v", err)
		writeErr(w, http.StatusInternalServerError, "Couldn't load quest completions. Please try again.")
		return
	}
	writeJSON(w, http.StatusOK, list)
}

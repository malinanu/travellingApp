package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/spotto/server/internal/domain"
)

func (s *Server) handleAdminCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.Store.GetVendorCategories(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	if cats == nil {
		cats = []domain.VendorCategory{}
	}
	writeJSON(w, http.StatusOK, cats)
}

func (s *Server) handleAdminCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req domain.VendorCategory
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.ID = strings.TrimSpace(req.ID)
	if req.ID == "" || req.Label == "" || req.Segment == "" || req.Icon == "" {
		writeErr(w, http.StatusBadRequest, "missing required fields (id, label, segment, icon)")
		return
	}

	if err := s.Store.CreateVendorCategory(r.Context(), req); err != nil {
		storeErr(w, err)
		return
	}

	// Update in-memory cache
	domain.AddVendorCategory(req)

	writeJSON(w, http.StatusCreated, req)
}

func (s *Server) handleAdminUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req domain.VendorCategory
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.ID = id // ensure ID matches URL
	if req.Label == "" || req.Segment == "" || req.Icon == "" {
		writeErr(w, http.StatusBadRequest, "missing required fields (label, segment, icon)")
		return
	}

	if err := s.Store.UpdateVendorCategoryRecord(r.Context(), req); err != nil {
		storeErr(w, err)
		return
	}

	// Update in-memory cache
	domain.AddVendorCategory(req)

	writeJSON(w, http.StatusOK, req)
}

func (s *Server) handleAdminDeleteCategory(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.Store.DeleteVendorCategory(r.Context(), id); err != nil {
		// If it fails due to FK constraint (vendors depend on it), storeErr will return a generic error or 500,
		// but ideally we could check for pq.Error with code 23503. For simplicity, storeErr is fine.
		storeErr(w, err)
		return
	}

	// Update in-memory cache
	domain.RemoveVendorCategory(id)

	w.WriteHeader(http.StatusNoContent)
}

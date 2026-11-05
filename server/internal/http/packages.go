package httpapi

import (
	"net/http"

	"github.com/spotto/server/internal/domain"
)

// handlePublicPackages returns the active vendor packages for marketing website pricing page.
func (s *Server) handlePublicPackages(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Store.GetVendorPackages(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handleAdminGetPackages returns the full package configuration and feature pool for the Command Center.
func (s *Server) handleAdminGetPackages(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.Store.GetVendorPackages(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, cfg)
}

// handleAdminSavePackages updates packages, feature allocations, and pricing.
func (s *Server) handleAdminSavePackages(w http.ResponseWriter, r *http.Request) {
	var req domain.PackagesConfig
	if !readJSON(w, r, &req) {
		return
	}

	if len(req.Packages) == 0 {
		writeErr(w, http.StatusBadRequest, "at least one package is required")
		return
	}

	// Validate and sanitize each package
	seenSlugs := make(map[string]bool, len(req.Packages))
	for i := range req.Packages {
		pkg := &req.Packages[i]
		if pkg.Slug == "" || pkg.Name == "" {
			writeErr(w, http.StatusBadRequest, "package slug and name are required")
			return
		}
		if seenSlugs[pkg.Slug] {
			writeErr(w, http.StatusBadRequest, "duplicate package slug: "+pkg.Slug)
			return
		}
		seenSlugs[pkg.Slug] = true

		if pkg.MonthlyPriceUSD < 0 || pkg.YearlyPriceUSD < 0 {
			writeErr(w, http.StatusBadRequest, "package prices cannot be negative")
			return
		}
		if pkg.ActiveOffers < -1 {
			pkg.ActiveOffers = -1
		}
		if pkg.FlashSalesPerMonth < -1 {
			pkg.FlashSalesPerMonth = -1
		}
		if pkg.TeamSeats < -1 {
			pkg.TeamSeats = -1
		}
		if pkg.ID == "" {
			pkg.ID = "pkg_" + pkg.Slug
		}
		// Always synchronize sortOrder to array sequence
		pkg.SortOrder = i + 1

		// Sanitize ExcludedFeatures: remove any feature that is included in Features
		if len(pkg.ExcludedFeatures) > 0 {
			included := make(map[string]bool, len(pkg.Features))
			for _, f := range pkg.Features {
				included[f.ID] = true
			}
			cleanExcluded := make([]string, 0, len(pkg.ExcludedFeatures))
			for _, exID := range pkg.ExcludedFeatures {
				if !included[exID] {
					cleanExcluded = append(cleanExcluded, exID)
				}
			}
			pkg.ExcludedFeatures = cleanExcluded
		}
	}

	if len(req.FeaturePool) == 0 {
		req.FeaturePool = domain.DefaultMasterFeatures()
	}

	if err := s.Store.SaveVendorPackages(r.Context(), req); err != nil {
		storeErr(w, err)
		return
	}

	s.logAdminAction(r.Context(), "packages.update", "system", "packages", map[string]any{
		"packageCount":     len(req.Packages),
		"featurePoolCount": len(req.FeaturePool),
	})

	writeJSON(w, http.StatusOK, req)
}

// HandleAdminSavePackagesForTest exposes the save handler for unit tests without network overhead.
func (s *Server) HandleAdminSavePackagesForTest(w http.ResponseWriter, r *http.Request) {
	s.handleAdminSavePackages(w, r)
}


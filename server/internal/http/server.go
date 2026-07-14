// Package httpapi wires the chi router, middleware and handlers.
package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

type Server struct {
	Store  *postgres.Store
	Tokens *auth.Tokens
}

func New(store *postgres.Store, tokens *auth.Tokens) *Server {
	return &Server{Store: store, Tokens: tokens}
}

func (s *Server) Router(cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Post("/auth/register", s.handleRegister)
	r.Post("/auth/login", s.handleLogin)
	r.Post("/auth/2fa/login-verify", s.handleTwoFactorLoginVerify)

	// Authenticated (any role).
	r.Group(func(r chi.Router) {
		r.Use(s.Tokens.Authenticated)
		r.Get("/me", s.handleMe)
		r.Put("/me", s.handleUpdateMe)
		r.Post("/auth/2fa/setup", s.handleTwoFactorSetup)
		r.Post("/auth/2fa/enable", s.handleTwoFactorEnable)
		r.Post("/auth/2fa/disable", s.handleTwoFactorDisable)

		r.Get("/places", s.handlePlaces)
		r.Get("/places/{id}", s.handlePlace)
		r.Get("/places/{id}/reviews", s.handlePlaceReviews)
	})

	// Traveller.
	r.Group(func(r chi.Router) {
		r.Use(s.Tokens.Authenticated, auth.RequireRole(domain.RoleTraveller))
		r.Post("/places/{id}/vibes/{vibeId}", s.handleVibeVote)
		r.Post("/places/{id}/reviews", s.handleCreateReview)
		r.Post("/checkins", s.handleCheckin)
		r.Post("/claims", s.handleCreateClaim)
		r.Get("/me/claims", s.handleMyClaims)
		r.Get("/me/trip", s.handleGetTrip)
		r.Put("/me/trip", s.handlePutTrip)
		r.Get("/me/group", s.handleGetGroup)
		r.Put("/me/group", s.handlePutGroup)
	})

	// Vendor (approved vendors only, checked in handlers where relevant).
	r.Group(func(r chi.Router) {
		r.Use(s.Tokens.Authenticated, auth.RequireRole(domain.RoleVendor))
		r.Get("/vendor/business", s.handleVendorBusiness)
		r.Put("/vendor/business", s.handleUpdateVendorBusiness)
		r.Get("/vendor/menu", s.handleVendorMenu)
		r.Put("/vendor/menu", s.handleUpdateVendorMenu)
		r.Get("/vendor/offers", s.handleVendorOffers)
		r.Post("/vendor/offers", s.handleCreateOffer)
		r.Put("/vendor/offers/{id}", s.handleUpdateOffer)
		r.Patch("/vendor/offers/{id}/status", s.handleOfferStatus)
		r.Delete("/vendor/offers/{id}", s.handleDeleteOffer)
		r.Post("/vendor/validate", s.handleValidate)
		r.Get("/vendor/dashboard", s.handleVendorDashboard)
		r.Get("/vendor/redemptions", s.handleVendorRedemptions)
		r.Get("/vendor/notifications", s.handleVendorNotifications)
		r.Post("/vendor/notifications/read", s.handleVendorNotificationsRead)
		r.Get("/vendor/growth", s.handleVendorGrowth)
		r.Get("/vendor/reports", s.handleVendorReports)
	})

	// Admin.
	r.Group(func(r chi.Router) {
		r.Use(s.Tokens.Authenticated, auth.RequireRole(domain.RoleAdmin))
		r.Get("/admin/kpis", s.handleAdminKPIs)
		r.Get("/admin/vendors", s.handleAdminVendors)
		r.Post("/admin/vendors/{id}/approve", s.vendorStatusHandler(domain.VendorApproved))
		r.Post("/admin/vendors/{id}/reject", s.vendorStatusHandler(domain.VendorRejected))
		r.Post("/admin/vendors/{id}/request-edits", s.vendorStatusHandler(domain.VendorEdits))
		r.Post("/admin/vendors/{id}/suspend", s.vendorStatusHandler(domain.VendorSuspended))
		r.Post("/admin/vendors/{id}/reinstate", s.vendorStatusHandler(domain.VendorApproved))
		r.Get("/admin/vendors/map", s.handleAdminVendorsMap)
		r.Get("/admin/vendors/near", s.handleAdminVendorsNear)
		r.Get("/admin/vendors/{id}/reviews", s.handleAdminVendorReviews)
		r.Get("/admin/feed", s.handleAdminFeed)
		r.Post("/admin/feed/{id}/approve", s.feedResolveHandler("approved"))
		r.Post("/admin/feed/{id}/remove", s.feedResolveHandler("removed"))
		r.Post("/admin/feed/{id}/warn", s.feedResolveHandler("warned"))
		r.Get("/admin/regions", s.handleAdminRegions)
		r.Get("/admin/subscriptions", s.handleAdminSubscriptions)
		r.Post("/admin/subscriptions/{id}/upgrade", s.handleUpgradeSubscription)
		r.Get("/admin/audio-guides", s.handleAudioGuides)
		r.Post("/admin/audio-guides", s.handleCreateAudioGuide)
		r.Patch("/admin/audio-guides/{id}/publish", s.handlePublishAudioGuide)
	})

	return r
}

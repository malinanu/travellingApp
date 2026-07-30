// Package httpapi wires the chi router, middleware and handlers.
package httpapi

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/jobs"
	"github.com/spotto/server/internal/mail"
	"github.com/spotto/server/internal/media"
	"github.com/spotto/server/internal/store/postgres"
	"github.com/spotto/server/internal/weather"
)

type Server struct {
	Store   *postgres.Store
	Tokens  *auth.Tokens
	TOTPKey []byte // AES-256 key for encrypting TOTP secrets at rest

	Mail *mail.Mailer
	// Jobs runs slow side effects (email sends) off the request goroutine —
	// see internal/jobs for why this is best-effort, not durable.
	Jobs         *jobs.Queue
	VendorAppURL string
	AdminAppURL  string
	// SuperAdminEmails holds lowercased, trimmed emails from
	// config.Config.SuperAdminEmails for O(1) membership checks — see
	// isSuperAdmin in superadmin.go.
	SuperAdminEmails map[string]bool
	// GoogleCalendarAPIKey enables live public-holiday lookups (see
	// internal/holidays and holiday_scheduler.go) — empty means the
	// scheduler falls back to a bundled static list instead.
	GoogleCalendarAPIKey string
	// TomorrowIOAPIKey enables live weather (see internal/weather) — empty
	// means GET /weather reports itself unconfigured (501) instead.
	TomorrowIOAPIKey string
	// WeatherCache holds recent Tomorrow.io responses keyed by a coarse
	// lat/lng cell (see internal/weather/cache.go) to stay within the free
	// tier's tight rate limit.
	WeatherCache *weather.Cache
	// GoogleOAuth is nil when GOOGLE_CLIENT_ID/SECRET aren't configured —
	// handleGoogleLogin/handleGoogleCallback report 501 in that case rather
	// than panicking on a nil config.
	GoogleOAuth *oauth2.Config
	// Media uploads to Cloudflare R2; nil when R2 isn't configured, in which
	// case the /media endpoints report 501 instead of failing startup.
	Media media.Uploader

	loginLimit    *loginLimiter // guards /auth/login (key: ip+email)
	totpLimit     *loginLimiter // guards /auth/2fa/login-verify (key: ip+userID)
	resetLimit    *loginLimiter // guards /auth/forgot-password (key: ip+email)
	grantPinLimit *loginLimiter // guards /vendor/profile-grants/{id}/redeem (key: ip+userID) — a 6-digit PIN only has 10^6 possibilities
	oauthCodes    *authCodeStore
}

func New(store *postgres.Store, tokens *auth.Tokens, cfg config.Config) *Server {
	s := &Server{
		Store:   store,
		Tokens:  tokens,
		TOTPKey: cfg.TOTPEncryptionKey,
		Mail: mail.New(mail.Config{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort, User: cfg.SMTPUser, Pass: cfg.SMTPPass, From: cfg.EmailFrom,
		}),
		Jobs:                 jobs.NewQueue(4, 256),
		VendorAppURL:         cfg.VendorAppURL,
		AdminAppURL:          cfg.AdminAppURL,
		SuperAdminEmails:     superAdminSet(cfg.SuperAdminEmails),
		GoogleCalendarAPIKey: cfg.GoogleCalendarAPIKey,
		TomorrowIOAPIKey:     cfg.TomorrowIOAPIKey,
		WeatherCache:         weather.NewCache(),
		loginLimit:           newLoginLimiter(5, 15*time.Minute),
		totpLimit:            newLoginLimiter(5, 15*time.Minute),
		resetLimit:           newLoginLimiter(5, 15*time.Minute),
		grantPinLimit:        newLoginLimiter(5, 15*time.Minute),
		oauthCodes:           newAuthCodeStore(),
	}
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		s.GoogleOAuth = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
			},
			Endpoint: google.Endpoint,
		}
	}
	if cfg.R2Configured() {
		s.Media = media.NewR2(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicBaseURL)
	}
	return s
}

// Authenticated requires a valid Bearer session token AND that its embedded
// token_version still matches the user's current one in the database — a
// signature-valid token whose version has been bumped (e.g. by a 2FA
// disable) is rejected even though Tokens.Verify alone would accept it.
func (s *Server) Authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := s.Tokens.ParseBearer(r.Header.Get("Authorization"))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		version, err := s.Store.UserTokenVersion(r.Context(), c.Subject)
		if err != nil || version != c.TokenVersion {
			writeErr(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), c)))
	})
}

func (s *Server) Router(cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(realIP)
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
	r.Post("/auth/forgot-password", s.handleForgotPassword)
	r.Post("/auth/reset-password", s.handleResetPassword)
	r.Get("/auth/google/login", s.handleGoogleLogin)
	r.Get("/auth/google/callback", s.handleGoogleCallback)
	r.Post("/auth/exchange", s.handleAuthExchange)

	// Authenticated (any role).
	r.Group(func(r chi.Router) {
		r.Use(s.Authenticated)
		r.Get("/me", s.handleMe)
		r.Put("/me", s.handleUpdateMe)
		r.Put("/me/privacy", s.handleUpdatePrivacy)
		r.Put("/me/contact", s.handleUpdateMyContact)
		r.Put("/me/notification-prefs", s.handleUpdateNotificationPrefs)
		r.Put("/auth/password", s.handleChangePassword)
		r.Post("/auth/2fa/setup", s.handleTwoFactorSetup)
		r.Post("/auth/2fa/enable", s.handleTwoFactorEnable)
		r.Post("/auth/2fa/disable", s.handleTwoFactorDisable)

		r.Get("/weather", s.handleWeather)
		r.Get("/places", s.handlePlaces)
		r.Get("/places/{id}", s.handlePlace)
		r.Get("/places/{id}/reviews", s.handlePlaceReviews)
		r.Get("/places/{id}/photos", s.handlePlacePhotos)
		r.Post("/offers/{id}/view", s.handleOfferView)

		// Media uploads (R2) — any authenticated role; per-owner authorisation
		// is enforced inside the handlers (see canUploadFor).
		r.Post("/media/presign", s.handleMediaPresign)
		r.Post("/media/confirm", s.handleMediaConfirm)
		r.Delete("/media/{id}", s.handleMediaDelete)
	})

	// Traveller.
	r.Group(func(r chi.Router) {
		r.Use(s.Authenticated, auth.RequireRole(domain.RoleTraveller))
		r.Post("/places/{id}/vibes/{vibeId}", s.handleVibeVote)
		r.Post("/places/{id}/reviews", s.handleCreateReview)
		r.Post("/checkins", s.handleCheckin)
		r.Post("/claims", s.handleCreateClaim)
		r.Get("/me/claims", s.handleMyClaims)
		r.Get("/me/trip", s.handleGetTrip)
		r.Put("/me/trip", s.handlePutTrip)
		r.Get("/me/group", s.handleGetGroup)
		r.Put("/me/group", s.handlePutGroup)
		r.Put("/me/location", s.handleUpdateMyLocation)
		r.Get("/travellers/nearby", s.handleNearbyTravellers)
	})

	// Vendor (approved vendors only, checked in handlers where relevant).
	r.Group(func(r chi.Router) {
		r.Use(s.Authenticated, auth.RequireRole(domain.RoleVendor))
		// Business Profile writes are gated by requireBusinessProfileAccess
		// inside the handlers themselves (owner, or a manager with an active
		// temporary grant) rather than by role-tier middleware — see
		// vendor_grants.go.
		r.Get("/vendor/business", s.handleVendorBusiness)
		r.Put("/vendor/business", s.handleUpdateVendorBusiness)
		r.Patch("/vendor/business/category", s.handleUpdateVendorCategory)
		r.Patch("/vendor/business/identity", s.handleUpdateVendorIdentity)
		r.Get("/vendor/menu", s.handleVendorMenu)
		r.Get("/vendor/offers", s.handleVendorOffers)
		r.Get("/vendor/offers/{id}/claims", s.handleOfferClaims)
		r.Post("/vendor/validate", s.handleValidate)
		r.Get("/vendor/dashboard", s.handleVendorDashboard)
		r.Get("/vendor/redemptions", s.handleVendorRedemptions)
		r.Get("/vendor/notifications", s.handleVendorNotifications)
		r.Post("/vendor/notifications/read", s.handleVendorNotificationsRead)
		r.Get("/vendor/team", s.handleVendorTeam)
		r.Get("/vendor/media", s.handleVendorOwnMedia)
		r.Get("/vendor/profile-grants/mine", s.handleMyVendorProfileGrant)
		r.Post("/vendor/profile-grants/{id}/redeem", s.handleRedeemVendorProfileGrant)

		// Offers/menu mutations, growth, and reports — staff has zero
		// access (scan/validate only per the account page's own copy).
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireVendorManagerOrAbove)
			r.Put("/vendor/menu", s.handleUpdateVendorMenu)
			r.Post("/vendor/offers", s.handleCreateOffer)
			r.Put("/vendor/offers/{id}", s.handleUpdateOffer)
			r.Patch("/vendor/offers/{id}/status", s.handleOfferStatus)
			r.Delete("/vendor/offers/{id}", s.handleDeleteOffer)
			r.Get("/vendor/growth", s.handleVendorGrowth)
			r.Post("/vendor/growth/flash-sale", s.handleVendorFlashSale)
			r.Get("/vendor/reports", s.handleVendorReports)
			// Answering a holiday-closure prompt is a lighter action than a
			// full Business Profile edit, so it's gated here (owner+manager,
			// no PIN grant needed) rather than by requireBusinessProfileAccess.
			r.Get("/vendor/holidays/upcoming", s.handleUpcomingVendorHolidays)
			r.Post("/vendor/holidays/decision", s.handleVendorHolidayDecision)
		})

		// Team management — the account's own owner only, not invited staff
		// or managers.
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireVendorOwner)
			r.Post("/vendor/team/invite", s.handleCreateVendorInvite)
			r.Delete("/vendor/team/members/{userId}", s.handleRemoveVendorTeamMember)
			r.Delete("/vendor/team/invites/{id}", s.handleRevokeVendorInvite)
			r.Post("/vendor/profile-grants", s.handleCreateVendorProfileGrant)
			r.Get("/vendor/profile-grants", s.handleListVendorProfileGrants)
			r.Delete("/vendor/profile-grants/{id}", s.handleRevokeVendorProfileGrant)
		})
	})

	// Admin.
	r.Group(func(r chi.Router) {
		r.Use(s.Authenticated, auth.RequireRole(domain.RoleAdmin))
		r.Get("/admin/kpis", s.handleAdminKPIs)
		r.Get("/admin/search", s.handleAdminSearch)
		r.Get("/admin/vendors", s.handleAdminVendors)
		r.Post("/admin/vendors/{id}/approve", s.vendorStatusHandler(domain.VendorApproved, "vendor.approve"))
		r.Post("/admin/vendors/{id}/reject", s.vendorStatusHandler(domain.VendorRejected, "vendor.reject"))
		r.Post("/admin/vendors/{id}/request-edits", s.vendorStatusHandler(domain.VendorEdits, "vendor.request_edits"))
		r.Post("/admin/vendors/{id}/suspend", s.vendorStatusHandler(domain.VendorSuspended, "vendor.suspend"))
		r.Post("/admin/vendors/{id}/reinstate", s.vendorStatusHandler(domain.VendorApproved, "vendor.reinstate"))
		r.Get("/admin/vendors/summary", s.handleAdminVendorSummary)
		r.Get("/admin/vendors/map", s.handleAdminVendorsMap)
		r.Get("/admin/vendors/near", s.handleAdminVendorsNear)
		r.Get("/admin/vendors/{id}/reviews", s.handleAdminVendorReviews)
		r.Get("/admin/vendors/{id}/stats", s.handleAdminVendorStats)
		r.Get("/admin/vendors/{id}/offers", s.handleAdminVendorOffers)
		r.Get("/admin/vendors/{id}/redemptions", s.handleAdminVendorRedemptions)
		r.Get("/admin/vendors/{id}/team", s.handleAdminVendorTeam)
		r.Get("/admin/vendors/{id}/audit", s.handleAdminVendorAudit)
		r.Get("/admin/vendors/{id}/media", s.handleVendorMedia)
		r.Patch("/admin/vendors/{id}", s.handleAdminVendorEdit)
		r.Post("/admin/vendors/{id}/takedown", s.handleAdminVendorTakedown)
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
		r.Get("/admin/notifications", s.handleAdminNotifications)
		r.Post("/admin/notifications/read", s.handleAdminNotificationsRead)
		r.Get("/admin/vendor-grants", s.handleAdminVendorGrants)
		r.Post("/admin/vendor-grants/{id}/extend", s.handleExtendVendorGrant)
		r.Post("/admin/vendor-grants/{id}/revoke", s.handleRevokeVendorGrantAdmin)

		// Cross-admin history of destructive actions — restricted further
		// to super-admins, since it names which specific admin did what.
		r.Group(func(r chi.Router) {
			r.Use(s.RequireSuperAdmin)
			r.Get("/admin/audit-log", s.handleAdminAuditLog)
		})
	})

	return r
}

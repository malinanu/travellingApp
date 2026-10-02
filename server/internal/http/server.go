// Package httpapi wires the chi router, middleware and handlers.
package httpapi

import (
	"context"
	"net/http"
	"time"

	firebase "firebase.google.com/go/v4"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/jobs"
	"github.com/spotto/server/internal/mail"
	"github.com/spotto/server/internal/media"
	"github.com/spotto/server/internal/placedetails"
	"github.com/spotto/server/internal/store/postgres"
	"github.com/spotto/server/internal/weather"
)

type Server struct {
	Store   *postgres.Store
	Tokens  *auth.Tokens
	TOTPKey []byte // AES-256 key for encrypting TOTP secrets at rest

	Mail *mail.Mailer
	// Jobs runs slow side effects (email sends) off the request goroutine â€”
	// see internal/jobs for why this is best-effort, not durable.
	Jobs         *jobs.Queue
	VendorAppURL string
	AdminAppURL  string
	// SuperAdminEmails holds lowercased, trimmed emails from
	// config.Config.SuperAdminEmails for O(1) membership checks â€” see
	// isSuperAdmin in superadmin.go.
	SuperAdminEmails map[string]bool
	// GoogleCalendarAPIKey enables live public-holiday lookups (see
	// internal/holidays and holiday_scheduler.go) â€” empty means the
	// scheduler falls back to a bundled static list instead.
	GoogleCalendarAPIKey string
	// TomorrowIOAPIKey enables live weather (see internal/weather) â€” empty
	// means GET /weather reports itself unconfigured (501) instead.
	TomorrowIOAPIKey string
	// GeminiAPIKey enables BR-certificate OCR pre-fill (see
	// internal/gemini) â€” empty means the br-cert scheduler's OCR sweep is
	// a no-op. It also powers the Local Guide chat/embeddings; empty means
	// POST /guide/chat reports itself unconfigured (501).
	GeminiAPIKey string
	// WeatherCache holds recent Tomorrow.io responses keyed by a coarse
	// lat/lng cell (see internal/weather/cache.go) to stay within the free
	// tier's tight rate limit.
	WeatherCache *weather.Cache
	// GooglePlacesAPIKey enables live Places Details enrichment (see
	// internal/placedetails) for GET /places/{id} â€” sourced from the same
	// shared Google Maps/Places-enabled key as GoogleCalendarAPIKey (see
	// its doc comment in config.Config), not a separate credential. Empty
	// means that enrichment is silently skipped.
	GooglePlacesAPIKey string
	// PlaceDetailsCache holds recent Places Details lookups keyed by Google
	// place ID (see internal/placedetails/cache.go) to keep paid API spend
	// down â€” these attributes rarely change, so the TTL is long.
	PlaceDetailsCache *placedetails.Cache
	// GoogleOAuth is nil when GOOGLE_CLIENT_ID/SECRET aren't configured â€”
	// handleGoogleLogin/handleGoogleCallback report 501 in that case rather
	// than panicking on a nil config.
	GoogleOAuth *oauth2.Config
	// AppleAuth verifies Sign in with Apple identity tokens for /auth/apple.
	AppleAuth *auth.AppleVerifier
	// Media uploads to Cloudflare R2; nil when R2 isn't configured, in which
	// case the /media endpoints report 501 instead of failing startup.
	Media media.Uploader
	
	// FirebaseApp is used to dispatch push notifications to Flutter clients.
	// Nil if FIREBASE_SERVICE_ACCOUNT_JSON is not configured.
	FirebaseApp *firebase.App

	loginLimit    *loginLimiter // guards /auth/login (key: ip+email)
	totpLimit     *loginLimiter // guards /auth/2fa/login-verify (key: ip+userID)
	resetLimit    *loginLimiter // guards /auth/forgot-password (key: ip+email)
	grantPinLimit *loginLimiter // guards /vendor/profile-grants/{id}/redeem (key: ip+userID) â€” a 6-digit PIN only has 10^6 possibilities
	guideLimit    *dynamicRateLimiter // guards /guide/chat (key: userID or clientIP)
	globalLimit   *dynamicRateLimiter
	oauthCodes    *authCodeStore
	GroupHub      *GroupHub
}

func New(store *postgres.Store, tokens *auth.Tokens, cfg config.Config) *Server {
	initialLimit, err := store.GetRateLimitRPM(context.Background())
	if err != nil {
		initialLimit = 60
	}

	initialGuideLimit, err := store.GetGuideRateLimitRPM(context.Background())
	if err != nil {
		initialGuideLimit = 15
	}

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
		GeminiAPIKey:         cfg.GeminiAPIKey,
		WeatherCache:         weather.NewCache(),
		// Same key as GoogleCalendarAPIKey â€” see its doc comment.
		GooglePlacesAPIKey: cfg.GoogleCalendarAPIKey,
		PlaceDetailsCache:  placedetails.NewCache(),
		loginLimit:         newLoginLimiter(5, 15*time.Minute),
		totpLimit:          newLoginLimiter(5, 15*time.Minute),
		resetLimit:         newLoginLimiter(5, 15*time.Minute),
		grantPinLimit:      newLoginLimiter(5, 15*time.Minute),
		guideLimit:         newDynamicRateLimiter(initialGuideLimit, 1*time.Minute),
		globalLimit:        newDynamicRateLimiter(initialLimit, 1*time.Minute),
		oauthCodes:         newAuthCodeStore(),
		GroupHub:           NewGroupHub(),
	}
	appleAudience := cfg.AppleBundleID
	if appleAudience == "" {
		appleAudience = "com.spotto.spotto"
	}
	s.AppleAuth = auth.NewAppleVerifier(appleAudience)
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
		s.Media = media.NewR2(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket)
	}
	if cfg.FirebaseServiceAccount != "" {
		opt := option.WithCredentialsJSON([]byte(cfg.FirebaseServiceAccount))
		app, err := firebase.NewApp(context.Background(), nil, opt)
		if err == nil {
			s.FirebaseApp = app
		}
	}
	return s
}

// Authenticated requires a valid Bearer session token AND that its embedded
// token_version still matches the user's current one in the database â€” a
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

func maxBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) Router(cfg config.Config) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(realIP)
	r.Use(s.GlobalRateLimitMiddleware)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(maxBytesMiddleware(10 * 1024 * 1024))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("API-Version", "v1")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "0")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(self)")
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "version": "v1"})
	})

	// Proxies a private R2 object read â€” deliberately outside the
	// Authenticated group below: it's loaded via plain <img>/<video>/<a>
	// tags (app and web), which can't attach an Authorization header, so it
	// authorises via a signed token query param instead (see
	// handleMediaRaw). The trailing "*" is chi's wildcard match, needed
	// because object keys contain slashes (e.g. vendor/ven_x/m_y.jpg).
	r.Get("/media/raw/*", s.handleMediaRaw)

	r.Get("/healthz", s.handleHealthz)
	r.Get("/categories", s.handleAdminCategories)
	r.Get("/readyz", s.handleReadyz)
	r.Get("/ws/group", s.handleGroupWS)

	r.Post("/auth/register", s.handleRegister)
	r.Post("/auth/login", s.handleLogin)
	r.Post("/auth/2fa/login-verify", s.handleTwoFactorLoginVerify)
	r.Post("/auth/forgot-password", s.handleForgotPassword)
	r.Post("/auth/reset-password", s.handleResetPassword)
	r.Get("/auth/google/login", s.handleGoogleLogin)
	r.Get("/auth/google/callback", s.handleGoogleCallback)
	r.Post("/auth/apple", s.handleAppleAuth)
	r.Post("/auth/exchange", s.handleAuthExchange)

	// Authenticated (any role).
	r.Group(func(r chi.Router) {
		r.Use(s.Authenticated)
		r.Get("/me", s.handleMe)
		r.Delete("/me", s.handleDeleteAccount)
		r.Get("/me/export", s.handleExportData)
		r.Put("/me", s.handleUpdateMe)
		r.Put("/me/privacy", s.handleUpdatePrivacy)
		r.Put("/me/contact", s.handleUpdateMyContact)
		r.Put("/me/notification-prefs", s.handleUpdateNotificationPrefs)
		r.Post("/me/push-token", s.handleRegisterPushToken)
		r.Delete("/me/push-token", s.handleUnregisterPushToken)
		r.Get("/me/payment-methods", s.handleListPaymentMethods)
		r.Post("/me/payment-methods", s.handleCreatePaymentMethod)
		r.Delete("/me/payment-methods/{id}", s.handleDeletePaymentMethod)
		r.Post("/me/payment-methods/{id}/default", s.handleSetDefaultPaymentMethod)
		r.Post("/vendor/subscription/checkout", s.handleVendorSubscriptionCheckout)
		r.Put("/auth/password", s.handleChangePassword)
		r.Post("/auth/2fa/setup", s.handleTwoFactorSetup)
		r.Post("/auth/2fa/enable", s.handleTwoFactorEnable)
		r.Post("/auth/2fa/disable", s.handleTwoFactorDisable)

		r.Get("/weather", s.handleWeather)
		r.Post("/weather/batch", s.handleWeatherBatch)
		r.Get("/trips", s.handleListTrips)
		r.Post("/trips", s.handleSaveTrip)
		r.Get("/places", s.handlePlaces)
		r.Get("/places/search", s.handleSearchPlaces)
		r.Get("/places/{id}", s.handlePlace)
		r.Get("/places/{id}/reviews", s.handlePlaceReviews)
		r.Get("/places/{id}/photos", s.handlePlacePhotos)
		r.Get("/places/{id}/services", s.handlePlaceServices)
		r.Get("/places/{id}/services/{serviceId}/availability", s.handleServiceAvailability)
		r.Post("/offers/{id}/view", s.handleOfferView)

		// Media uploads (R2) â€” any authenticated role; per-owner authorisation
		// is enforced inside the handlers (see canUploadFor).
		r.Post("/media/presign", s.handleMediaPresign)
		r.Post("/media/confirm", s.handleMediaConfirm)
		r.Delete("/media/{id}", s.handleMediaDelete)
	})

	// Traveller.
	r.Group(func(r chi.Router) {
		r.Use(s.Authenticated, auth.RequireRole(domain.RoleTraveller))
		r.Post("/places/{id}/vibe-checkin", s.handleVibeCheckin)
		r.Post("/places/{id}/reviews", s.handleCreateReview)
		r.Post("/checkins", s.handleCheckin)
		r.Get("/me/quest", s.handleGetQuest)
		r.Post("/claims", s.handleCreateClaim)
		r.Get("/me/claims", s.handleMyClaims)
		r.Post("/bookings", s.handleCreateBooking)
		r.Post("/bookings/{id}/cancel", s.handleCancelMyBooking)
		r.Get("/me/bookings", s.handleMyBookings)
		r.Get("/me/trip", s.handleGetTrip)
		r.Put("/me/trip", s.handlePutTrip)
		r.Get("/me/group", s.handleGetGroup)
		r.Put("/me/group", s.handlePutGroup)
		r.Put("/me/location", s.handleUpdateMyLocation)
		r.Get("/travellers/nearby", s.handleNearbyTravellers)
		r.Post("/guide/chat", s.handleGuideChat)
		r.Get("/guide/conversations/latest", s.handleGuideLatestConversation)
		r.Post("/trip/inspiration/extract-places", s.handleExtractPlacesFromLink)
	})

	// Vendor (approved vendors only, checked in handlers where relevant).
	r.Group(func(r chi.Router) {
		r.Use(s.Authenticated, auth.RequireRole(domain.RoleVendor))
		// Business Profile writes are gated by requireBusinessProfileAccess
		// inside the handlers themselves (owner, or a manager with an active
		// temporary grant) rather than by role-tier middleware â€” see
		// vendor_grants.go.
		r.Get("/vendor/business", s.handleVendorBusiness)
		r.Put("/vendor/business", s.handleUpdateVendorBusiness)
		r.Patch("/vendor/business/category", s.handleUpdateVendorCategory)
		r.Patch("/vendor/business/identity", s.handleUpdateVendorIdentity)
		r.Patch("/vendor/business/br-cert-number", s.handleUpdateVendorBRCertNumber)
		r.Get("/vendor/sponsorship", s.handleGetSponsorship)
		r.Put("/vendor/sponsorship", s.handlePutSponsorship)
		r.Get("/vendor/menu", s.handleVendorMenu)
		r.Get("/vendor/offers", s.handleVendorOffers)
		r.Get("/vendor/offers/{id}/claims", s.handleOfferClaims)
		r.Post("/vendor/validate", s.handleValidate)
		r.Get("/vendor/services", s.handleVendorServices)
		r.Get("/vendor/bookings", s.handleVendorBookings)
		r.Get("/vendor/dashboard", s.handleVendorDashboard)
		r.Get("/vendor/redemptions", s.handleVendorRedemptions)
		r.Get("/vendor/notifications", s.handleVendorNotifications)
		r.Post("/vendor/notifications/read", s.handleVendorNotificationsRead)
		r.Get("/vendor/team", s.handleVendorTeam)
		r.Get("/vendor/media", s.handleVendorOwnMedia)
		r.Get("/vendor/profile-grants/mine", s.handleMyVendorProfileGrant)
		r.Post("/vendor/profile-grants/{id}/redeem", s.handleRedeemVendorProfileGrant)

		// Offers/menu mutations, growth, and reports â€” staff has zero
		// access (scan/validate only per the account page's own copy).
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireVendorManagerOrAbove)
			r.Put("/vendor/menu", s.handleUpdateVendorMenu)
			r.Post("/vendor/offers", s.handleCreateOffer)
			r.Put("/vendor/offers/{id}", s.handleUpdateOffer)
			r.Patch("/vendor/offers/{id}/status", s.handleOfferStatus)
			r.Delete("/vendor/offers/{id}", s.handleDeleteOffer)
			r.Post("/vendor/services", s.handleCreateService)
			r.Put("/vendor/services/{id}", s.handleUpdateService)
			r.Delete("/vendor/services/{id}", s.handleDeleteService)
			r.Post("/vendor/bookings/{id}/cancel", s.handleVendorCancelBooking)
			r.Patch("/vendor/bookings/{id}/status", s.handleSetBookingStatus)
			r.Get("/vendor/growth", s.handleVendorGrowth)
			r.Post("/vendor/growth/flash-sale", s.handleVendorFlashSale)
			r.Get("/vendor/reports", s.handleVendorReports)
			// Answering a holiday-closure prompt is a lighter action than a
			// full Business Profile edit, so it's gated here (owner+manager,
			// no PIN grant needed) rather than by requireBusinessProfileAccess.
			r.Get("/vendor/holidays/upcoming", s.handleUpcomingVendorHolidays)
			r.Post("/vendor/holidays/decision", s.handleVendorHolidayDecision)
		})

		// Team management â€” the account's own owner only, not invited staff
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
		r.Get("/admin/analytics", s.handleAdminAnalytics)
		r.Get("/admin/settings", s.handleAdminGetSettings)
		r.Put("/admin/settings", s.handleAdminPutSettings)
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
		r.Delete("/admin/reviews/{id}", s.handleAdminDeleteReview)
		r.Get("/admin/vendors/{id}/stats", s.handleAdminVendorStats)
		r.Get("/admin/vendors/{id}/offers", s.handleAdminVendorOffers)
		r.Get("/admin/vendors/{id}/redemptions", s.handleAdminVendorRedemptions)
		r.Post("/admin/claims/{id}/void", s.handleAdminVoidClaim)
		r.Get("/admin/vendors/{id}/team", s.handleAdminVendorTeam)
		r.Get("/admin/vendors/{id}/audit", s.handleAdminVendorAudit)
		r.Get("/admin/vendors/{id}/vibe", s.handleAdminVendorVibe)
		r.Get("/admin/vendors/{id}/media", s.handleVendorMedia)
		r.Patch("/admin/vendors/{id}", s.handleAdminVendorEdit)
		r.Post("/admin/vendors/{id}/takedown", s.handleAdminVendorTakedown)
		r.Get("/admin/feed", s.handleAdminFeed)
		r.Get("/admin/feed/sse", s.handleAdminFeedSSE)
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
		r.Get("/admin/local-guide/stats", s.handleAdminGuideStats)
		r.Put("/admin/local-guide/rate-limit", s.handleAdminPutGuideRateLimit)
		r.Get("/admin/local-guide/users", s.handleAdminGuideUsers)
		r.Get("/admin/local-guide/users/{id}/messages", s.handleAdminGuideUserMessages)
		r.Get("/admin/local-guide/conversations/{id}", s.handleAdminGuideConversationDetail)
		r.Get("/admin/local-guide/errors", s.handleAdminGuideErrors)

		// Quests
		r.Get("/admin/quests/sponsorships/pending", s.handleAdminListPendingSponsorships)
		r.Get("/admin/quests/sponsorships/approved", s.handleAdminListApprovedSponsorships)
		r.Post("/admin/quests/sponsorships/{vendor_id}/status", s.handleAdminUpdateSponsorshipStatus)
		r.Get("/admin/quests/schedules", s.handleAdminListQuestSchedules)
		r.Post("/admin/quests/schedules", s.handleAdminSetQuestSchedule)

		// Categories
		r.Get("/admin/categories", s.handleAdminCategories)
		r.Post("/admin/categories", s.handleAdminCreateCategory)
		r.Put("/admin/categories/{id}", s.handleAdminUpdateCategory)
		r.Delete("/admin/categories/{id}", s.handleAdminDeleteCategory)

		// Cross-admin history of destructive actions â€” restricted further
		// to super-admins, since it names which specific admin did what.
		r.Group(func(r chi.Router) {
			r.Use(s.RequireSuperAdmin)
			r.Get("/admin/audit-log", s.handleAdminAuditLog)
		})
	})

	return r
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.Pool.Ping(r.Context()); err != nil {
		writeErr(w, http.StatusServiceUnavailable, "database connection failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}


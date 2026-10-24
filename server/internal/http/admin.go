package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/store/postgres"
)

// handleAdminKPIs mixes live counts with prototype-style topline numbers.
func (s *Server) handleAdminKPIs(w http.ResponseWriter, r *http.Request) {
	pending, err := s.Store.CountPendingVendors(r.Context())
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
		"mrr":            mrr,
		"activeSubs":     len(subs),
	})
}

func (s *Server) handleAdminAnalytics(w http.ResponseWriter, r *http.Request) {
	analytics, err := s.Store.PlatformAnalytics(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, analytics)
}

func (s *Server) handleAdminGetSettings(w http.ResponseWriter, r *http.Request) {
	limit, err := s.Store.GetRateLimitRPM(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"rateLimitRPM": limit})
}

func (s *Server) handleAdminPutSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RateLimitRPM int `json:"rateLimitRPM"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid json")
		return
	}
	if body.RateLimitRPM < 1 {
		writeErr(w, http.StatusBadRequest, "rate limit must be >= 1")
		return
	}
	if err := s.Store.SetRateLimitRPM(r.Context(), body.RateLimitRPM); err != nil {
		storeErr(w, err)
		return
	}
	s.globalLimit.setMax(body.RateLimitRPM)
	writeJSON(w, http.StatusOK, map[string]int{"rateLimitRPM": body.RateLimitRPM})
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

// handleAdminVendorsMap powers the geo page's vendor pins: every vendor with
// real coordinates, plus a computed badReviewCount/flagged.
func (s *Server) handleAdminVendorsMap(w http.ResponseWriter, r *http.Request) {
	vendors, err := s.Store.VendorsForMap(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	if vendors == nil {
		vendors = []domain.Vendor{}
	}
	writeJSON(w, http.StatusOK, vendors)
}

// handleAdminVendorsNear powers the investigation-pin feature: vendors
// within radiusKm (default 5) of an arbitrary point a super-admin clicks.
func (s *Server) handleAdminVendorsNear(w http.ResponseWriter, r *http.Request) {
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 != nil || err2 != nil {
		writeErr(w, http.StatusBadRequest, "lat and lng are required")
		return
	}
	radiusKm := 5.0
	if v, err := strconv.ParseFloat(r.URL.Query().Get("radiusKm"), 64); err == nil && v > 0 {
		radiusKm = v
	}
	vendors, err := s.Store.VendorsNear(r.Context(), lat, lng, radiusKm)
	if err != nil {
		storeErr(w, err)
		return
	}
	if vendors == nil {
		vendors = []domain.Vendor{}
	}
	writeJSON(w, http.StatusOK, vendors)
}

// handleAdminVendorReviews lists a vendor's reviews. Default is the bad
// (<=2 star) reviews behind the geo map's flag; ?all=1 returns every review
// for the vendor-detail Reviews tab.
func (s *Server) handleAdminVendorReviews(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var reviews []domain.Review
	var err error
	if r.URL.Query().Get("all") == "1" {
		reviews, err = s.Store.ReviewsByVendor(r.Context(), id)
	} else {
		reviews, err = s.Store.ReviewsByVendorBad(r.Context(), id)
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	if reviews == nil {
		reviews = []domain.Review{}
	}
	writeJSON(w, http.StatusOK, reviews)
}

// handleAdminVendorStats powers the Vendors page monitor pane â€” one
// vendor's live activity roll-up (redemptions, check-ins, reviews, offers,
// content, subscription, team).
func (s *Server) handleAdminVendorStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.Store.VendorStats(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// handleAdminVendorVibe returns the crowdsourced Current Vibe reading for a
// vendor's synced place â€” lazy per-vendor detail, same convention as
// handleAdminVendorStats/Offers/Redemptions, not attached to the bulk
// GET /admin/vendors list to avoid adding an N+1 query there. Best-effort:
// a vendor with no synced place yet (never approved, or syncVendorPlace
// hasn't run) gets a zero-value vibe rather than an error, same as the
// vendor's own GET /vendor/business view.
func (s *Server) handleAdminVendorVibe(w http.ResponseWriter, r *http.Request) {
	place, err := s.Store.PlaceByVendorID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusOK, domain.VibeAggregate{})
		return
	}
	vibe, err := s.Store.VibeAggregate(r.Context(), place.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, vibe)
}

// handleAdminVendorSummary powers the Vendors page's aggregate band â€” the
// network-wide roll-up across every vendor (status breakdown, redemptions,
// check-ins, rating, offers, subscriptions).
func (s *Server) handleAdminVendorSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.Store.VendorSummary(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// handleAdminVendorOffers lists a vendor's offers with live claimed/
// redeemed/views counts, for the vendor-detail Offers tab.
func (s *Server) handleAdminVendorOffers(w http.ResponseWriter, r *http.Request) {
	offers, err := s.Store.OffersByVendor(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if offers == nil {
		offers = []domain.Offer{}
	}
	writeJSON(w, http.StatusOK, offers)
}

// handleAdminVendorRedemptions lists a vendor's recent redeemed claims
// (traveller, offer, timestamp) for the vendor-detail Activity tab.
func (s *Server) handleAdminVendorRedemptions(w http.ResponseWriter, r *http.Request) {
	claims, err := s.Store.RedemptionsByVendor(r.Context(), chi.URLParam(r, "id"), nil)
	if err != nil {
		storeErr(w, err)
		return
	}
	if claims == nil {
		claims = []domain.Claim{}
	}
	writeJSON(w, http.StatusOK, claims)
}

// handleAdminDeleteReview allows an admin to delete an abusive or fake review.
func (s *Server) handleAdminDeleteReview(w http.ResponseWriter, r *http.Request) {
	reviewID := chi.URLParam(r, "id")
	if err := s.Store.DeleteReview(r.Context(), reviewID); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// adminVendorTeamResponse is the vendor-detail Team tab payload: staff roster,
// still-pending invites, and any active/pending profile-access grants.
type adminVendorTeamResponse struct {
	Members []domain.User                   `json:"members"`
	Invites []domain.VendorInvite           `json:"invites"`
	Grants  []postgres.AdminVendorGrantView `json:"grants"`
}

func (s *Server) handleAdminVendorTeam(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	members, err := s.Store.VendorTeamMembers(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	invites, err := s.Store.VendorInvites(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	allGrants, err := s.Store.AdminListVendorProfileGrants(r.Context(), "")
	if err != nil {
		storeErr(w, err)
		return
	}
	resp := adminVendorTeamResponse{
		Members: members,
		Invites: invites,
		Grants:  []postgres.AdminVendorGrantView{},
	}
	if resp.Members == nil {
		resp.Members = []domain.User{}
	}
	if resp.Invites == nil {
		resp.Invites = []domain.VendorInvite{}
	}
	for _, g := range allGrants {
		if g.VendorID == id {
			resp.Grants = append(resp.Grants, g)
		}
	}
	for i := range resp.Members {
		resp.Members[i].AvatarURL = s.signMediaPath(resp.Members[i].AvatarURL)
	}
	writeJSON(w, http.StatusOK, resp)
}

// handleAdminVendorAudit returns the admin-action history for one vendor â€”
// the vendor-detail History tab.
func (s *Server) handleAdminVendorAudit(w http.ResponseWriter, r *http.Request) {
	entries, err := s.Store.AuditLogsByTarget(r.Context(), "vendor", chi.URLParam(r, "id"), 100)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// handleAdminVendorEdit is the admin override-edit path: it patches any
// provided vendor field (identity, internal note/flags/priority, the profile
// blob, and the menu) regardless of the vendor's status, then re-syncs the
// traveller-facing place so corrections show immediately.
func (s *Server) handleAdminVendorEdit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Name     *string         `json:"name"`
		Category *string         `json:"category"`
		Town     *string         `json:"town"`
		Country  *string         `json:"country"`
		Priority *string         `json:"priority"`
		Note     *string         `json:"note"`
		Flags    *[]string       `json:"flags"`
		Profile  json.RawMessage `json:"profile"`
		Menu     json.RawMessage `json:"menu"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	v, err := s.Store.VendorByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	changed := []string{}
	if req.Name != nil {
		v.Name, changed = *req.Name, append(changed, "name")
	}
	if req.Category != nil {
		if !domain.ValidCategoryID(*req.Category) {
			writeErr(w, http.StatusBadRequest, "unknown category")
			return
		}
		v.Category, changed = *req.Category, append(changed, "category")
	}
	if req.Town != nil {
		v.Town, changed = *req.Town, append(changed, "town")
	}
	if req.Country != nil {
		v.Country, changed = *req.Country, append(changed, "country")
	}
	if req.Priority != nil {
		v.Priority, changed = *req.Priority, append(changed, "priority")
	}
	if req.Note != nil {
		v.Note, changed = *req.Note, append(changed, "note")
	}
	if req.Flags != nil {
		v.Flags, changed = *req.Flags, append(changed, "flags")
	}
	if len(req.Profile) > 0 {
		v.Profile, changed = req.Profile, append(changed, "profile")
	}
	if len(req.Menu) > 0 {
		v.Menu, changed = req.Menu, append(changed, "menu")
	}
	if len(changed) == 0 {
		writeErr(w, http.StatusBadRequest, "no editable fields provided")
		return
	}
	if err := s.Store.UpsertVendor(r.Context(), v); err != nil {
		storeErr(w, err)
		return
	}
	s.syncVendorPlaceBestEffort(r.Context(), v)
	s.logAdminAction(r.Context(), "vendor.admin_edit", "vendor", id, map[string]any{"changed": changed})
	writeJSON(w, http.StatusOK, v)
}

// handleAdminVendorTakedown is the emergency control: it suspends the vendor
// (which pulls its place from the traveller feed via placeVisibleWhere) and
// records the mandatory reason in the audit log.
func (s *Server) handleAdminVendorTakedown(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Reason string `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Reason) == "" {
		writeErr(w, http.StatusBadRequest, "a takedown reason is required")
		return
	}
	if err := s.Store.SetVendorStatus(r.Context(), id, domain.VendorSuspended); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "vendor.takedown", "vendor", id, map[string]any{"reason": req.Reason})
	writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": string(domain.VendorSuspended)})
}

// vendorStatusHandler is shared by approve/reject/request-edits/suspend/
// reinstate â€” action is a distinct audit-log label per call site even
// though approve and reinstate both resolve to VendorApproved (server.go).
func (s *Server) vendorStatusHandler(status domain.VendorStatus, action string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if err := s.Store.SetVendorStatus(r.Context(), id, status); err != nil {
			storeErr(w, err)
			return
		}
		s.logAdminAction(r.Context(), action, "vendor", id, nil)
		writeJSON(w, http.StatusOK, map[string]string{"id": id, "status": string(status)})
	}
}

const (
	// regionActivityWindow is how far back traveller activity counts as
	// demand.
	regionActivityWindow = 30 * 24 * time.Hour
	// regionMaxKm: points further than this from every region centre
	// (abroad, or test data) belong to no region. Every spot in Sri Lanka
	// is well within it of one of the seeded towns.
	regionMaxKm = 100
)

// handleAdminRegions powers the geo page's town circles and Opportunity
// leaderboard with live numbers. Each region's Vendors is the approved
// businesses nearest to it, and Travelers is the traveller actions
// (check-ins, vibe check-ins, reviews, saved places) on places nearest to
// it in the last 30 days — the stored regions rows only supply names and
// centres. Anything over regionMaxKm from every centre isn't counted.
func (s *Server) handleAdminRegions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	regions, err := s.Store.Regions(ctx)
	if err != nil {
		storeErr(w, err)
		return
	}
	if regions == nil {
		regions = []domain.Region{}
	}
	vendors, err := s.Store.LiveVendorPoints(ctx)
	if err != nil {
		storeErr(w, err)
		return
	}
	activity, err := s.Store.PlaceActivitySince(ctx, time.Now().Add(-regionActivityWindow))
	if err != nil {
		storeErr(w, err)
		return
	}
	for i := range regions {
		regions[i].Travelers, regions[i].Vendors = 0, 0
	}
	for _, p := range vendors {
		if i := nearestRegion(regions, p.Lat, p.Lng); i >= 0 {
			regions[i].Vendors += p.Weight
		}
	}
	for _, p := range activity {
		if i := nearestRegion(regions, p.Lat, p.Lng); i >= 0 {
			regions[i].Travelers += p.Weight
		}
	}
	writeJSON(w, http.StatusOK, regions)
}

// nearestRegion is the index of the region whose centre is closest to
// lat/lng, or -1 when none is within regionMaxKm.
func nearestRegion(regions []domain.Region, lat, lng float64) int {
	best, bestKm := -1, float64(regionMaxKm)
	for i, rg := range regions {
		if km := geo.HaversineKm(lat, lng, rg.Lat, rg.Lng); km <= bestKm {
			best, bestKm = i, km
		}
	}
	return best
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
	id := chi.URLParam(r, "id")
	if err := s.Store.UpgradeSubscription(r.Context(), id, req.MRR); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "subscription.upgrade", "subscription", id, map[string]any{"mrr": req.MRR})
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
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

// handleAdminSearch powers the command bar's âŒ˜K search â€” see
// Store.AdminSearch for the query shape.
func (s *Server) handleAdminSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, postgres.AdminSearchResults{
			Vendors: []domain.Vendor{}, Subscriptions: []domain.Subscription{},
		})
		return
	}
	results, err := s.Store.AdminSearch(r.Context(), q, 8)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}

// handleAdminNotifications powers the command-center bell â€” see
// Store.AdminNotifications for how buckets are derived and marked unread.
func (s *Server) handleAdminNotifications(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	notifs, err := s.Store.AdminNotifications(r.Context(), c.Subject)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, notifs)
}

func (s *Server) handleAdminNotificationsRead(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	if err := s.Store.MarkAdminNotificationsRead(r.Context(), c.Subject); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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

// handleAdminVoidClaim allows admins to void or reverse an offer claim/redemption.
func (s *Server) handleAdminVoidClaim(w http.ResponseWriter, r *http.Request) {
	claimID := chi.URLParam(r, "id")
	if err := s.Store.VoidClaim(r.Context(), claimID); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

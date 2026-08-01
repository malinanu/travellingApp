package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
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

// handleAdminVendorStats powers the Vendors page monitor pane — one
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

// handleAdminVendorSummary powers the Vendors page's aggregate band — the
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

// handleAdminVendorAudit returns the admin-action history for one vendor —
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
// reinstate — action is a distinct audit-log label per call site even
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
		s.logAdminAction(r.Context(), "feed."+resolved, "feed_item", strconv.FormatInt(id, 10), nil)
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
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

// handleAdminSearch powers the command bar's ⌘K search — see
// Store.AdminSearch for the query shape.
func (s *Server) handleAdminSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, postgres.AdminSearchResults{
			Vendors: []domain.Vendor{}, FeedItems: []domain.FeedItem{}, Subscriptions: []domain.Subscription{},
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

// handleAdminNotifications powers the command-center bell — see
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

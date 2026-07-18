package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func (s *Server) handleVendorBusiness(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleUpdateVendorBusiness(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var profile json.RawMessage
	if !readJSON(w, r, &profile) {
		return
	}
	if err := s.Store.UpdateVendorProfile(r.Context(), v.ID, profile); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	v.Profile = profile
	writeJSON(w, http.StatusOK, v)
}

// handleUpdateVendorCategory changes the business type — separate from
// handleUpdateVendorBusiness since category is a top-level vendors column,
// not part of the profile JSON blob that endpoint replaces wholesale.
func (s *Server) handleUpdateVendorCategory(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Category string `json:"category"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Category == "" {
		writeErr(w, http.StatusBadRequest, "category is required")
		return
	}
	if err := s.Store.UpdateVendorCategory(r.Context(), v.ID, req.Category); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	v.Category = req.Category
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) handleVendorMenu(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, json.RawMessage(orEmpty(v.Menu)))
}

func orEmpty(m json.RawMessage) []byte {
	if len(m) == 0 {
		return []byte(`{}`)
	}
	return m
}

func (s *Server) handleUpdateVendorMenu(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var menu json.RawMessage
	if !readJSON(w, r, &menu) {
		return
	}
	if err := s.Store.UpdateVendorMenu(r.Context(), v.ID, menu); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, menu)
}

func (s *Server) handleVendorOffers(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	offers, err := s.Store.OffersByVendor(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if offers == nil {
		offers = []domain.Offer{}
	}
	writeJSON(w, http.StatusOK, offers)
}

func (s *Server) handleCreateOffer(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var o domain.Offer
	if !readJSON(w, r, &o) {
		return
	}
	if o.Title == "" {
		writeErr(w, http.StatusBadRequest, "title required")
		return
	}
	o.ID = newID("off")
	o.VendorID = v.ID
	if o.Status == "" {
		o.Status = domain.OfferDraft
	}
	if err := s.Store.UpsertOffer(r.Context(), o); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) handleUpdateOffer(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	existing, err := s.Store.OfferByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || existing.VendorID != v.ID {
		writeErr(w, http.StatusNotFound, "offer not found")
		return
	}
	var o domain.Offer
	if !readJSON(w, r, &o) {
		return
	}
	o.ID = existing.ID
	o.VendorID = v.ID
	if err := s.Store.UpsertOffer(r.Context(), o); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) handleOfferStatus(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Status domain.OfferStatus `json:"status"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Store.SetOfferStatus(r.Context(), chi.URLParam(r, "id"), v.ID, req.Status); err != nil {
		storeErr(w, err)
		return
	}
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusOK, map[string]string{"status": string(req.Status)})
}

func (s *Server) handleDeleteOffer(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	if err := s.Store.DeleteOffer(r.Context(), chi.URLParam(r, "id"), v.ID); err != nil {
		storeErr(w, err)
		return
	}
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// handleOfferClaims lists the claim/redemption activity for one of the
// caller's own offers — drives the vendor portal's offer detail drawer.
func (s *Server) handleOfferClaims(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	offer, err := s.Store.OfferByID(r.Context(), id)
	if err != nil || offer.VendorID != v.ID {
		writeErr(w, http.StatusNotFound, "offer not found")
		return
	}
	claims, err := s.Store.ClaimsByOfferID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	if claims == nil {
		claims = []domain.Claim{}
	}
	writeJSON(w, http.StatusOK, claims)
}

// syncVendorPlaceBestEffort keeps the traveller-facing Place row's headline
// "deal" in sync with vendor v's most relevant live/ending offer, logging
// (not failing the request) on error — the primary offer CRUD operation
// already succeeded by the time this runs, and a sync hiccup shouldn't
// undo it from the caller's perspective.
func (s *Server) syncVendorPlaceBestEffort(ctx context.Context, v domain.Vendor) {
	if err := s.syncVendorPlace(ctx, v); err != nil {
		log.Printf("syncVendorPlace vendor=%s: %v", v.ID, err)
	}
}

// syncVendorPlace mirrors vendor v's current headline offer into the
// traveller-facing Place.Data.deal (with a real offer id, so a resulting
// claim can be linked back to the offer), lazily creating a minimal Place
// if v doesn't have one yet — real (non-seeded) vendors otherwise have no
// map presence at all. Deliberately does not sync the rest of Vendor.Profile
// (wifi/amenities/photos/hours) into Place.Data — a separate, pre-existing
// gap out of scope here.
func (s *Server) syncVendorPlace(ctx context.Context, v domain.Vendor) error {
	offers, err := s.Store.OffersByVendor(ctx, v.ID)
	if err != nil {
		return err
	}
	headline := pickHeadlineOffer(offers)

	place, err := s.Store.PlaceByVendorID(ctx, v.ID)
	if errors.Is(err, postgres.ErrNotFound) {
		place = domain.Place{ID: newID("plc"), Name: v.Name, VendorID: &v.ID, Data: json.RawMessage(`{}`)}
	} else if err != nil {
		return err
	}

	var data map[string]any
	_ = json.Unmarshal(orEmpty(place.Data), &data)
	if data == nil {
		data = map[string]any{}
	}
	// Minimal base fields for a lazily-created Place only — never overwrite
	// hand-curated data on an existing Place.
	if _, ok := data["lat"]; !ok {
		data["lat"] = v.Lat
	}
	if _, ok := data["lng"]; !ok {
		data["lng"] = v.Lng
	}
	if _, ok := data["type"]; !ok {
		data["type"] = v.Category
	}
	if headline != nil {
		data["deal"] = map[string]any{
			"id": headline.ID, "active": true,
			"label": headline.Title, "tag": headline.Value,
		}
	} else {
		data["deal"] = map[string]any{"id": nil, "active": false, "label": "", "tag": ""}
	}
	place.Data, err = json.Marshal(data)
	if err != nil {
		return err
	}
	return s.Store.UpsertPlace(ctx, place)
}

// pickHeadlineOffer picks the single offer synced into Place.Data.deal —
// live beats ending, ties broken by most-recently-updated (offer IDs are
// random hex, not chronological). Returns nil if the vendor has no
// live/ending offer.
func pickHeadlineOffer(offers []domain.Offer) *domain.Offer {
	var best *domain.Offer
	for i := range offers {
		o := &offers[i]
		if o.Status != domain.OfferLive && o.Status != domain.OfferEnding {
			continue
		}
		switch {
		case best == nil:
			best = o
		case o.Status == domain.OfferLive && best.Status == domain.OfferEnding:
			best = o
		case o.Status == best.Status && o.UpdatedAt.After(best.UpdatedAt):
			best = o
		}
	}
	return best
}

// handleValidate marks a claim redeemed. Responses mirror the prototype's
// validator states: valid / already_redeemed / wrong_vendor / unknown.
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		writeErr(w, http.StatusBadRequest, "code required")
		return
	}

	claim, err := s.Store.ClaimByCode(r.Context(), code)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"result": "unknown"})
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	if claim.VendorID == nil || *claim.VendorID != v.ID {
		writeJSON(w, http.StatusOK, map[string]any{"result": "wrong_vendor"})
		return
	}
	if claim.Status == domain.ClaimRedeemed {
		writeJSON(w, http.StatusOK, map[string]any{
			"result":     "already_redeemed",
			"redeemedAt": claim.RedeemedAt,
			"claim":      claim,
		})
		return
	}

	redeemed, err := s.Store.RedeemClaim(r.Context(), claim.ID)
	if errors.Is(err, postgres.ErrNotFound) {
		// Lost the race with a concurrent redeem.
		claim, _ = s.Store.ClaimByCode(r.Context(), code)
		writeJSON(w, http.StatusOK, map[string]any{
			"result":     "already_redeemed",
			"redeemedAt": claim.RedeemedAt,
			"claim":      claim,
		})
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"result": "valid", "claim": redeemed})
}

// blobHandler serves a seeded JSONB document (dashboard/growth/reports).
func (s *Server) blobHandler(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, ok := s.requireVendor(w, r)
		if !ok {
			return
		}
		data, err := s.Store.VendorBlob(r.Context(), v.ID, kind)
		if errors.Is(err, postgres.ErrNotFound) {
			writeJSON(w, http.StatusOK, map[string]any{})
			return
		}
		if err != nil {
			storeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, json.RawMessage(data))
	}
}

func (s *Server) handleVendorDashboard(w http.ResponseWriter, r *http.Request) {
	s.blobHandler("dashboard")(w, r)
}

func (s *Server) handleVendorGrowth(w http.ResponseWriter, r *http.Request) {
	s.blobHandler("growth")(w, r)
}

func (s *Server) handleVendorReports(w http.ResponseWriter, r *http.Request) {
	s.blobHandler("reports")(w, r)
}

// handleVendorFlashSale records a real flash-sale send against the growth
// blob's `flash.lastSentMins`, enforcing the same 1-per-3h limit the UI
// displays (server-side, not just a disabled button) — a client can't just
// re-enable the button to bypass it.
func (s *Server) handleVendorFlashSale(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		DiscountPct int `json:"discountPct"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	raw, err := s.Store.VendorBlob(r.Context(), v.ID, "growth")
	if err != nil && !errors.Is(err, postgres.ErrNotFound) {
		storeErr(w, err)
		return
	}
	var blob map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &blob); err != nil {
			writeErr(w, http.StatusInternalServerError, "internal error")
			return
		}
	}
	if blob == nil {
		blob = map[string]json.RawMessage{}
	}
	var flash struct {
		RadiusKm     int `json:"radiusKm"`
		InRange      int `json:"inRange"`
		Nomads       int `json:"nomads"`
		Surfers      int `json:"surfers"`
		LastSentMins int `json:"lastSentMins"`
		DiscountPct  int `json:"discountPct"`
	}
	if f, ok := blob["flash"]; ok {
		json.Unmarshal(f, &flash)
	}
	if flash.LastSentMins < 180 {
		writeErr(w, http.StatusTooManyRequests, "a flash sale was already sent within the last 3 hours")
		return
	}
	flash.LastSentMins = 0
	if req.DiscountPct > 0 {
		flash.DiscountPct = req.DiscountPct
	} else if flash.DiscountPct == 0 {
		flash.DiscountPct = 20
	}
	updated, err := json.Marshal(flash)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	blob["flash"] = updated
	newData, err := json.Marshal(blob)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.Store.UpsertVendorBlob(r.Context(), v.ID, "growth", newData); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

func (s *Server) handleVendorRedemptions(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	after := r.URL.Query().Get("after")
	claims, err := s.Store.RedemptionsByVendor(r.Context(), v.ID, &after)
	if err != nil {
		storeErr(w, err)
		return
	}
	if claims == nil {
		claims = []domain.Claim{}
	}
	writeJSON(w, http.StatusOK, claims)
}

func (s *Server) handleVendorNotifications(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	notifs, err := s.Store.VendorNotifications(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	if notifs == nil {
		notifs = []domain.VendorNotification{}
	}
	writeJSON(w, http.StatusOK, notifs)
}

func (s *Server) handleVendorNotificationsRead(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	if err := s.Store.MarkVendorNotificationsRead(r.Context(), v.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// teamMember is the safe subset of domain.User exposed on the team list —
// never the full record (password hash, TOTP fields).
type teamMember struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"vendorRole"`
	CreatedAt time.Time `json:"createdAt"`
}

type teamInvite struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// handleVendorTeam lists current members (owner + staff) and still-pending
// invites — viewable by anyone on the team, not just the owner.
func (s *Server) handleVendorTeam(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	users, err := s.Store.VendorTeamMembers(r.Context(), v.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	members := make([]teamMember, 0, len(users))
	for _, u := range users {
		members = append(members, teamMember{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.VendorRole, CreatedAt: u.CreatedAt})
	}
	invites, err := s.Store.VendorInvites(r.Context(), v.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	pending := make([]teamInvite, 0, len(invites))
	for _, inv := range invites {
		pending = append(pending, teamInvite{ID: inv.ID, Code: inv.Code, Email: inv.Email, Role: inv.Role, CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt})
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": members, "invites": pending})
}

// inviteCodeAlphabet avoids visually-ambiguous characters (no 0/O/1/I),
// matching newClaimCode's convention for codes a human might retype.
const inviteCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func newInviteCode() string {
	b := make([]byte, 8)
	rand.Read(b)
	out := make([]byte, 8)
	for i, v := range b {
		out[i] = inviteCodeAlphabet[int(v)%len(inviteCodeAlphabet)]
	}
	return string(out)
}

// handleCreateVendorInvite generates a 7-day invite code for a new staff
// or manager member to join this vendor account. Owner-only (see
// server.go's RequireVendorOwner). If an email is given and the mailer is
// configured, the invite link is emailed to it — but the code/link is
// always returned to the caller too, so the invite still works as a
// copy-link if email isn't configured or delivery fails.
func (s *Server) handleCreateVendorInvite(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	role := strings.TrimSpace(req.Role)
	if role == "" {
		role = domain.VendorRoleStaff
	}
	if role != domain.VendorRoleStaff && role != domain.VendorRoleManager {
		writeErr(w, http.StatusBadRequest, "role must be staff or manager")
		return
	}
	inv := domain.VendorInvite{
		ID:        newID("inv"),
		VendorID:  v.ID,
		Code:      newInviteCode(),
		Email:     strings.TrimSpace(req.Email),
		Role:      role,
		CreatedBy: c.Subject,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}
	if err := s.Store.CreateVendorInvite(r.Context(), inv); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if inv.Email != "" {
		link := s.VendorAppURL + "/join/" + inv.Code
		email, subject, body := inv.Email, "You've been invited to join "+v.Name+" on Spotto", inviteEmailHTML(v.Name, role, link)
		s.Jobs.Enqueue(func() { s.Mail.Send(email, subject, body) })
	}
	writeJSON(w, http.StatusCreated, teamInvite{ID: inv.ID, Code: inv.Code, Email: inv.Email, Role: inv.Role, CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt})
}

func inviteEmailHTML(vendorName, role, link string) string {
	return fmt.Sprintf(
		`<p>You've been invited to join <strong>%s</strong> on Spotto as a <strong>%s</strong>.</p>
		<p><a href="%s">Accept the invite</a> (expires in 7 days).</p>
		<p>If the link doesn't work, use invite code from the vendor who invited you at %s.</p>`,
		vendorName, role, link, link)
}

// handleRemoveVendorTeamMember detaches a staff or manager member from
// the vendor account. Owner-only; RemoveVendorTeamMember itself also
// refuses to touch the owner's own row, so an owner account can't be
// removed this way.
func (s *Server) handleRemoveVendorTeamMember(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	userID := chi.URLParam(r, "userId")
	if err := s.Store.RemoveVendorTeamMember(r.Context(), v.ID, userID); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"removed": true})
}

// handleRevokeVendorInvite cancels a still-pending invite. Owner-only.
func (s *Server) handleRevokeVendorInvite(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.Store.RevokeVendorInvite(r.Context(), id, v.ID); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"revoked": true})
}

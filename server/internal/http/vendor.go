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
	"github.com/spotto/server/internal/holidays"
	"github.com/spotto/server/internal/store/postgres"
)

func (s *Server) handleVendorBusiness(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, v)
}

// requireBusinessProfileAccess additionally gates Business Profile writes:
// owners (and legacy empty-VendorRole rows) always pass; managers need an
// active (redeemed, unrevoked, unexpired) profile grant; staff are always
// rejected. Call after requireEditableVendor/requireApprovedVendor, before
// any Business Profile mutation. GET /vendor/business intentionally stays
// open to all roles (see server.go) since the portal shell reads it on
// every page — only writes are gated here.
func (s *Server) requireBusinessProfileAccess(w http.ResponseWriter, r *http.Request, v domain.Vendor) bool {
	c := auth.FromContext(r.Context())
	if c.VendorRole == domain.VendorRoleOwner || c.VendorRole == "" {
		return true
	}
	if c.VendorRole == domain.VendorRoleManager {
		if active, err := s.Store.ActiveVendorProfileGrant(r.Context(), v.ID, c.Subject); err == nil && active {
			return true
		}
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "profile_access_locked"})
	return false
}

func (s *Server) handleUpdateVendorBusiness(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireEditableVendor(w, r)
	if !ok {
		return
	}
	if !s.requireBusinessProfileAccess(w, r, v) {
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
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusOK, v)
}

// handleUpdateVendorIdentity corrects business name/town/map-pin location —
// these are otherwise set once at registration and never touched again.
func (s *Server) handleUpdateVendorIdentity(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireEditableVendor(w, r)
	if !ok {
		return
	}
	if !s.requireBusinessProfileAccess(w, r, v) {
		return
	}
	var req struct {
		Name string  `json:"name"`
		Town string  `json:"town"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name == "" || req.Town == "" {
		writeErr(w, http.StatusBadRequest, "name and town are required")
		return
	}
	if err := s.Store.UpdateVendorIdentity(r.Context(), v.ID, req.Name, req.Town, req.Lat, req.Lng); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	v.Name, v.Town, v.Lat, v.Lng = req.Name, req.Town, req.Lat, req.Lng
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusOK, v)
}

// handleUpdateVendorBRCertNumber sets/edits the vendor's Business
// Registration certificate number — either a Gemini pre-fill the vendor
// confirmed as-is, or one they typed/corrected themselves. Any edit
// re-queues eROC verification (see Store.SetVendorBRCertNumber).
func (s *Server) handleUpdateVendorBRCertNumber(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireEditableVendor(w, r)
	if !ok {
		return
	}
	if !s.requireBusinessProfileAccess(w, r, v) {
		return
	}
	var req struct {
		CertNumber string `json:"certNumber"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	certNo := strings.ToUpper(strings.TrimSpace(req.CertNumber))
	if len(certNo) > 20 {
		writeErr(w, http.StatusBadRequest, "certificate number is too long")
		return
	}
	if err := s.Store.SetVendorBRCertNumber(r.Context(), v.ID, certNo); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	v.BRCertNumber, v.BRCertCompanyName, v.BRCertMessage, v.BRCertAttempts = certNo, "", "", 0
	v.BRCertVerifiedAt = nil
	if certNo == "" {
		v.BRCertVerifyStatus = domain.BRCertVerifyNone
	} else {
		v.BRCertVerifyStatus = domain.BRCertVerifyPending
	}
	writeJSON(w, http.StatusOK, v)
}

// handleUpdateVendorCategory changes the business type — separate from
// handleUpdateVendorBusiness since category is a top-level vendors column,
// not part of the profile JSON blob that endpoint replaces wholesale.
func (s *Server) handleUpdateVendorCategory(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireEditableVendor(w, r)
	if !ok {
		return
	}
	if !s.requireBusinessProfileAccess(w, r, v) {
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
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusOK, v)
}

const holidayNoticeDays = 7

type upcomingHoliday struct {
	Date string `json:"date"` // "2006-01-02"
	Name string `json:"name"`
}

// handleUpcomingVendorHolidays lists public holidays within the notice
// window that this vendor hasn't yet recorded an open/closed decision for
// — gated to owner+manager (RequireVendorManagerOrAbove in server.go), not
// staff, and deliberately not behind requireBusinessProfileAccess since
// answering it is a narrow, lighter action than full profile edits.
func (s *Server) handleUpcomingVendorHolidays(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	all, err := holidays.Upcoming(r.Context(), s.GoogleCalendarAPIKey, v.Country, holidayNoticeDays)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	var profile vendorProfileDoc
	_ = json.Unmarshal(orEmpty(v.Profile), &profile)
	decided := make(map[string]bool, len(profile.HolidayOverrides))
	for _, ov := range profile.HolidayOverrides {
		decided[ov.Date] = true
	}

	out := make([]upcomingHoliday, 0, len(all))
	for _, h := range all {
		date := h.Date.Format("2006-01-02")
		if !decided[date] {
			out = append(out, upcomingHoliday{Date: date, Name: h.Name})
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// handleVendorHolidayDecision records an owner/manager's open/closed
// decision for a specific date — a narrow read-modify-write on just the
// holidayOverrides field, reusing the existing full-profile store method
// rather than needing a dedicated one.
func (s *Server) handleVendorHolidayDecision(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	var req struct {
		Date   string `json:"date"`
		Name   string `json:"name"`
		Closed bool   `json:"closed"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		writeErr(w, http.StatusBadRequest, "date must be YYYY-MM-DD")
		return
	}

	var data map[string]any
	_ = json.Unmarshal(orEmpty(v.Profile), &data)
	if data == nil {
		data = map[string]any{}
	}
	var profile vendorProfileDoc
	_ = json.Unmarshal(orEmpty(v.Profile), &profile)

	overrides := make([]holidayOverrideDoc, 0, len(profile.HolidayOverrides)+1)
	for _, ov := range profile.HolidayOverrides {
		if ov.Date != req.Date {
			overrides = append(overrides, ov)
		}
	}
	overrides = append(overrides, holidayOverrideDoc{Date: req.Date, Name: req.Name, Closed: req.Closed})
	data["holidayOverrides"] = overrides

	updated, err := json.Marshal(data)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	if err := s.Store.UpdateVendorProfile(r.Context(), v.ID, updated); err != nil {
		writeErr(w, http.StatusInternalServerError, "internal error")
		return
	}
	v.Profile = updated
	s.syncVendorPlaceBestEffort(r.Context(), v)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	v, ok := s.requireEditableVendor(w, r)
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
	v.Menu = menu
	s.syncVendorPlaceBestEffort(r.Context(), v)
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

// vendorProfileDoc mirrors the vendor-facing VendorProfile shape
// (web/packages/shared/src/types.ts) stored as opaque JSON on Vendor.Profile.
type vendorProfileDoc struct {
	Wifi *struct {
		Enabled bool    `json:"enabled"`
		Score   float64 `json:"score"`
	} `json:"wifi"`
	BestTimes []string `json:"bestTimes"`
	SeaView   bool     `json:"seaView"`
	// Hours is keyed by lowercase weekday name ("monday".."sunday") — see
	// weekdayKey. A missing entry or one with Closed=true means the vendor
	// is closed that day. vendorHoursDoc's UnmarshalJSON transparently
	// migrates the pre-per-day {open,close} shape on read.
	Hours     vendorHoursDoc `json:"hours"`
	Amenities []string       `json:"amenities"`
	Blurb     string         `json:"blurb"`
	Contact   *struct {
		Phone     string `json:"phone"`
		Website   string `json:"website"`
		Instagram string `json:"instagram"`
		Facebook  string `json:"facebook"`
	} `json:"contact"`
	// HolidayOverrides records an owner/manager's explicit open/closed
	// decision for a specific date (see handleVendorHolidayDecision) — a
	// Closed:true entry for today always wins over the regular weekly
	// Hours schedule in syncVendorPlace.
	HolidayOverrides []holidayOverrideDoc `json:"holidayOverrides"`
}

type dayHoursDoc struct {
	Open   string `json:"open"`
	Close  string `json:"close"`
	Closed bool   `json:"closed"`
}

// vendorHoursDoc is keyed by lowercase weekday name. Its UnmarshalJSON
// transparently migrates the single {open, close} pair every vendor's
// hours were stored as before per-day hours existed: on read, if the
// per-day shape doesn't parse, it falls back to the legacy shape and
// expands it across all 7 days, so existing vendors don't silently show
// blank hours to travellers until they happen to resave via the new UI.
// Saves always write the new per-day shape (the vendor web app no longer
// produces the legacy one).
type vendorHoursDoc map[string]dayHoursDoc

func (h *vendorHoursDoc) UnmarshalJSON(b []byte) error {
	var perDay map[string]dayHoursDoc
	if err := json.Unmarshal(b, &perDay); err == nil {
		*h = perDay
		return nil
	}
	var legacy struct {
		Open  string `json:"open"`
		Close string `json:"close"`
	}
	if err := json.Unmarshal(b, &legacy); err != nil {
		return err
	}
	expanded := make(vendorHoursDoc, len(weekdayKeys))
	for _, k := range weekdayKeys {
		expanded[k] = dayHoursDoc{Open: legacy.Open, Close: legacy.Close}
	}
	*h = expanded
	return nil
}

type holidayOverrideDoc struct {
	Date   string `json:"date"` // "2006-01-02"
	Name   string `json:"name"`
	Closed bool   `json:"closed"`
}

// vendorMenuPresetDoc mirrors one MenuPreset from
// web/packages/shared/src/types.ts — a whole named menu the vendor can
// toggle live via vendorMenuDoc.ActivePresetID.
type vendorMenuPresetDoc struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Sections []map[string]any `json:"sections"`
}

// vendorMenuDoc mirrors StoredMenuDoc — either the legacy bare
// {sections: [...]} shape any vendor who hasn't resaved since presets
// shipped may still have, or {presets: [...], activePresetId} once they
// have. Presets wins whenever non-empty; see effectiveMenuSections. The
// traveller-facing Place.Data.menu is always a bare array of the
// resolved/filtered sections, never this wrapped document.
type vendorMenuDoc struct {
	Sections       []map[string]any      `json:"sections"` // legacy-only, ignored once Presets is non-empty
	Presets        []vendorMenuPresetDoc `json:"presets"`
	ActivePresetID string                `json:"activePresetId"`
}

// availableToday implements MenuItem/MenuSection.availableDays' whitelist
// semantics: an absent or nil "availableDays" means unrestricted (always
// available); once present, only an explicit `true` for today's weekday
// key counts — an absent or `false` entry excludes it.
func availableToday(m map[string]any, today string) bool {
	raw, ok := m["availableDays"]
	if !ok || raw == nil {
		return true
	}
	days, ok := raw.(map[string]any)
	if !ok {
		return true
	}
	v, _ := days[today].(bool)
	return v
}

// effectiveMenuSections picks the preset travellers should currently see
// (the one matching ActivePresetID, falling back to the first preset if
// it matches none, or to the legacy bare Sections wrapped as an implicit
// single preset if there are no Presets at all), then filters out any
// section/item not available today, dropping a section left with zero
// items. Returns the same []map[string]any shape data["menu"] has always
// held in syncVendorPlace.
func effectiveMenuSections(doc vendorMenuDoc, now time.Time) []map[string]any {
	sections := doc.Sections
	if len(doc.Presets) > 0 {
		active := doc.Presets[0]
		for _, p := range doc.Presets {
			if p.ID == doc.ActivePresetID {
				active = p
				break
			}
		}
		sections = active.Sections
	}

	today := weekdayKey(now)
	out := make([]map[string]any, 0, len(sections))
	for _, sec := range sections {
		if !availableToday(sec, today) {
			continue
		}
		rawItems, _ := sec["items"].([]any)
		kept := make([]any, 0, len(rawItems))
		for _, ri := range rawItems {
			if item, ok := ri.(map[string]any); ok && availableToday(item, today) {
				kept = append(kept, item)
			}
		}
		if len(kept) == 0 {
			continue
		}
		sec["items"] = kept
		out = append(out, sec)
	}
	return out
}

// wifiSpeedLabel derives the two display strings the traveller app expects
// (speed/label) from the single 0-10 score a vendor sets — the vendor
// portal only collects a score, not free text, so this keeps both sides
// consistent with the score slider's own "Patchy ↔ Blazing fast" labels.
func wifiSpeedLabel(score float64) string {
	switch {
	case score >= 8:
		return "Blazing fast"
	case score >= 5:
		return "Good"
	case score > 0:
		return "Patchy"
	default:
		return ""
	}
}

// categoryOriginalLabels maps the original 6 lowercase category ids
// (used by the vendor portal's onboarding wizard) to a traveller-friendly
// display label. Every other category — the richer free-text set real/
// seeded vendor applications carry (e.g. "Spa & Wellness") — already
// stores its own display label as the id, so it passes through unchanged.
// Mirrors web/vendor/lib/business-categories.ts's BUSINESS_CATEGORIES.
var categoryOriginalLabels = map[string]string{
	"restaurant": "Restaurant",
	"cafe":       "Café",
	"bar":        "Bar & Sunset",
	"surf":       "Surf & Water",
	"craft":      "Craft & Studio",
	"shop":       "Shop",
}

func categoryLabel(category string) string {
	if label, ok := categoryOriginalLabels[category]; ok {
		return label
	}
	return category
}

// formatHourLabel renders a "HH:MM" 24h time (as produced by an
// &lt;input type="time"&gt;) as a traveller-friendly "3:04 PM" string.
// Returns the input unchanged if it doesn't parse.
func formatHourLabel(hhmm string) string {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return hhmm
	}
	return t.Format("3:04 PM")
}

// isOpenNow does a naive same-day/overnight comparison of the current
// local time against open/close — there's no vendor timezone tracked
// anywhere else in this app either, so this matches existing precision.
func isOpenNow(open, close string) bool {
	now := time.Now().Format("15:04")
	if close > open {
		return now >= open && now < close
	}
	// Overnight range (e.g. open 18:00, close 02:00).
	return now >= open || now < close
}

// weekdayKeys maps time.Weekday's Sunday=0 ordering to the lowercase
// keys vendorProfileDoc.Hours and the vendor web app both use.
var weekdayKeys = [...]string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}

func weekdayKey(t time.Time) string { return weekdayKeys[t.Weekday()] }

// todayHoursAndOpenNow computes the traveller-facing "hours" display string
// and "openNow" bool for `now`'s date: a Closed:true holiday override for
// today always wins over the regular weekly schedule (see
// vendorProfileDoc.HolidayOverrides), otherwise today's weekday entry in
// Hours is used (or "closed today" if absent/marked closed).
func todayHoursAndOpenNow(profile vendorProfileDoc, now time.Time) (string, bool) {
	todayDate := now.Format("2006-01-02")
	for _, ov := range profile.HolidayOverrides {
		if ov.Date == todayDate && ov.Closed {
			name := ov.Name
			if name == "" {
				name = "public holiday"
			}
			return "Closed today — " + name, false
		}
	}
	day, ok := profile.Hours[weekdayKey(now)]
	if !ok || day.Closed || day.Open == "" || day.Close == "" {
		if ok && day.Closed {
			return "Closed today", false
		}
		return "", false
	}
	return formatHourLabel(day.Open) + " – " + formatHourLabel(day.Close), isOpenNow(day.Open, day.Close)
}

// syncVendorPlace mirrors vendor v's current profile — headline offer,
// Wi-Fi/sea-view/best-times/amenities/blurb/hours/contact, and menu — into
// the traveller-facing Place.Data, lazily creating a minimal Place if v
// doesn't have one yet (real, non-seeded vendors otherwise have no map
// presence at all). Vendor is the sole authoritative source for
// name/lat/lng/category now that all of them have a real update path, so
// this always overwrites those rather than only seeding them once.
func (s *Server) syncVendorPlace(ctx context.Context, v domain.Vendor) error {
	offers, err := s.Store.OffersByVendor(ctx, v.ID)
	if err != nil {
		return err
	}
	headline := pickHeadlineOffer(offers)

	place, err := s.Store.PlaceByVendorID(ctx, v.ID)
	if errors.Is(err, postgres.ErrNotFound) {
		place = domain.Place{ID: newID("plc"), VendorID: &v.ID, Data: json.RawMessage(`{}`)}
	} else if err != nil {
		return err
	}
	place.Name = v.Name

	var data map[string]any
	_ = json.Unmarshal(orEmpty(place.Data), &data)
	if data == nil {
		data = map[string]any{}
	}
	data["lat"] = v.Lat
	data["lng"] = v.Lng
	data["type"] = categoryLabel(v.Category)

	if headline != nil {
		data["deal"] = map[string]any{
			"id": headline.ID, "active": true,
			"label": headline.Title, "tag": headline.Value,
		}
	} else {
		data["deal"] = map[string]any{"id": nil, "active": false, "label": "", "tag": ""}
	}

	var profile vendorProfileDoc
	_ = json.Unmarshal(orEmpty(v.Profile), &profile)

	if profile.Wifi != nil && profile.Wifi.Enabled {
		data["wifi"] = map[string]any{
			"score": profile.Wifi.Score,
			"speed": wifiSpeedLabel(profile.Wifi.Score),
			"label": wifiSpeedLabel(profile.Wifi.Score),
		}
	} else {
		delete(data, "wifi")
	}
	data["seaView"] = profile.SeaView
	bestTimes := profile.BestTimes
	if bestTimes == nil {
		bestTimes = []string{}
	}
	data["bestTimes"] = bestTimes
	tags := profile.Amenities
	if tags == nil {
		tags = []string{}
	}
	data["tags"] = tags
	data["blurb"] = profile.Blurb

	data["hours"], data["openNow"] = todayHoursAndOpenNow(profile, time.Now())

	contact := map[string]any{"phone": "", "website": "", "instagram": "", "facebook": ""}
	if profile.Contact != nil {
		contact = map[string]any{
			"phone": profile.Contact.Phone, "website": profile.Contact.Website,
			"instagram": profile.Contact.Instagram, "facebook": profile.Contact.Facebook,
		}
	}
	data["contact"] = contact

	var menuDoc vendorMenuDoc
	_ = json.Unmarshal(orEmpty(v.Menu), &menuDoc)
	data["menu"] = effectiveMenuSections(menuDoc, time.Now())

	// Mirror the vendor's uploaded media into the traveller listing: a cover
	// image, the gallery, and a promo video (all public R2 URLs) — plus
	// each one's thumbnail/blur-placeholder pair alongside it (additive,
	// "cover"/"gallery" themselves are left exactly as-is for backward
	// compatibility). No signing needed here for the new fields:
	// signMediaRefs (traveller.go) already walks the whole unmarshaled
	// Place.Data JSON recursively and signs every "/media/raw/<key>" path
	// by structure, not by field name; coverMicro is a data: URI, not a
	// stored path, so signMediaPath leaves it untouched by its own
	// existing early-return.
	if mediaRows, mErr := s.Store.MediaByOwner(ctx, "vendor", v.ID); mErr == nil {
		gallery, galleryThumbs := []string{}, []string{}
		cover, coverThumb, coverMicro, promo := "", "", "", ""
		for _, m := range mediaRows {
			thumbPath := ""
			if m.ThumbKey != "" && s.Media != nil {
				thumbPath = s.Media.PublicURL(m.ThumbKey)
			}
			switch m.Role {
			case "cover":
				cover, coverThumb, coverMicro = m.URL, thumbPath, m.MicroThumb
			case "promo":
				promo = m.URL
			case "gallery":
				gallery = append(gallery, m.URL)
				galleryThumbs = append(galleryThumbs, thumbPath)
			}
		}
		if cover == "" && len(gallery) > 0 {
			cover, coverThumb = gallery[0], galleryThumbs[0]
		}
		data["cover"] = cover
		data["coverThumb"] = coverThumb
		data["coverMicro"] = coverMicro
		data["gallery"] = gallery
		data["galleryThumbs"] = galleryThumbs
		data["promoVideo"] = promo
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

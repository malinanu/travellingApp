package httpapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/media"
)

// singleValueRoles hold at most one object per owner — a new upload replaces
// (and deletes from R2) the previous one. br_cert is single-value (unlike
// the generic multi-value "doc") because a re-upload should replace the
// vendor's one Business Registration certificate, not add another.
var singleValueRoles = map[string]bool{"cover": true, "promo": true, "avatar": true, "br_cert": true}

// roleAllowsKind gates which file kind a role accepts: promo is video-only,
// docs (and the dedicated br_cert role) may be a PDF or a scanned image,
// everything else is an image.
func roleAllowsKind(role, kind string) bool {
	switch role {
	case "promo":
		return kind == media.KindVideo
	case "doc", "br_cert":
		return kind == media.KindDoc || kind == media.KindImage
	default:
		return kind == media.KindImage
	}
}

// canUploadFor authorises a caller to attach media to an owner. Admins may
// touch anything; otherwise a vendor may only upload to its own vendor, a
// user to its own avatar, any authenticated user may contribute a place
// photo, and a review's photos only by its author.
func (s *Server) canUploadFor(ctx context.Context, c *auth.Claims, ownerType, ownerID string) bool {
	if ownerType == "" || ownerID == "" {
		return false
	}
	if c.Role == domain.RoleAdmin {
		return true
	}
	switch ownerType {
	case "vendor":
		return c.VendorID != "" && c.VendorID == ownerID
	case "user":
		return c.Subject == ownerID
	case "place":
		return true
	case "review":
		rv, err := s.Store.ReviewByID(ctx, ownerID)
		return err == nil && rv.TravellerID == c.Subject
	}
	return false
}

// handleMediaPresign validates the request and hands back a short-lived
// presigned PUT URL the client uploads the file straight to R2 with.
func (s *Server) handleMediaPresign(w http.ResponseWriter, r *http.Request) {
	if s.Media == nil {
		writeErr(w, http.StatusNotImplemented, "uploads are not configured")
		return
	}
	var req struct {
		OwnerType   string `json:"ownerType"`
		OwnerID     string `json:"ownerId"`
		Role        string `json:"role"`
		ContentType string `json:"contentType"`
		Size        int64  `json:"size"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	c := auth.FromContext(r.Context())
	if !s.canUploadFor(r.Context(), c, req.OwnerType, req.OwnerID) {
		writeErr(w, http.StatusForbidden, "not allowed to upload here")
		return
	}
	ext, kind, maxSize, ok := media.Resolve(req.ContentType)
	if !ok {
		writeErr(w, http.StatusBadRequest, "unsupported file type")
		return
	}
	if !roleAllowsKind(req.Role, kind) {
		writeErr(w, http.StatusBadRequest, "wrong kind of file for this upload")
		return
	}
	if req.Size > maxSize {
		writeErr(w, http.StatusBadRequest, "file is too large")
		return
	}
	key := fmt.Sprintf("%s/%s/%s%s", req.OwnerType, req.OwnerID, newID("m"), ext)
	url, err := s.Media.PresignPut(r.Context(), key, req.ContentType, 10*time.Minute)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not start upload")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"uploadUrl": url, "key": key, "publicUrl": s.Media.PublicURL(key), "kind": kind,
	})
}

// handleMediaConfirm records an uploaded object and associates it with its
// owner after verifying it actually landed in R2.
func (s *Server) handleMediaConfirm(w http.ResponseWriter, r *http.Request) {
	if s.Media == nil {
		writeErr(w, http.StatusNotImplemented, "uploads are not configured")
		return
	}
	var req struct {
		Key       string `json:"key"`
		OwnerType string `json:"ownerType"`
		OwnerID   string `json:"ownerId"`
		Role      string `json:"role"`
		Width     int    `json:"width"`
		Height    int    `json:"height"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	c := auth.FromContext(r.Context())
	if !s.canUploadFor(r.Context(), c, req.OwnerType, req.OwnerID) {
		writeErr(w, http.StatusForbidden, "not allowed")
		return
	}
	size, contentType, err := s.Media.Head(r.Context(), req.Key)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "upload not found in storage")
		return
	}
	_, kind, maxSize, ok := media.Resolve(contentType)
	if !ok || size > maxSize || !roleAllowsKind(req.Role, kind) {
		_ = s.Media.Delete(r.Context(), req.Key)
		writeErr(w, http.StatusBadRequest, "uploaded file was rejected")
		return
	}

	// Single-value roles (cover/promo/avatar) replace the previous object.
	if singleValueRoles[req.Role] {
		old, _ := s.Store.ClearOwnerRole(r.Context(), req.OwnerType, req.OwnerID, req.Role)
		for _, o := range old {
			_ = s.Media.Delete(r.Context(), o.ObjectKey)
			if o.ThumbKey != "" {
				_ = s.Media.Delete(r.Context(), o.ThumbKey)
			}
		}
	}

	// Thumbnail generation happens off the request path — see
	// thumbnail_worker.go. Marking it pending here just queues it; the
	// confirm response goes out immediately with no thumb/micro yet
	// (clients already fall back to the full-size url in that case).
	var thumbStatus string
	if kind == media.KindImage {
		thumbStatus = "pending"
	}

	m := domain.Media{
		ID: newID("med"), OwnerType: req.OwnerType, OwnerID: req.OwnerID, Kind: kind, Role: req.Role,
		ObjectKey: req.Key, URL: s.Media.PublicURL(req.Key), ThumbStatus: thumbStatus,
		ContentType: contentType, Size: size,
		Width: req.Width, Height: req.Height, UploadedBy: c.Subject, Status: "ready", CreatedAt: time.Now(),
	}
	if err := s.Store.InsertMedia(r.Context(), m); err != nil {
		storeErr(w, err)
		return
	}
	if req.Role == "avatar" {
		_ = s.Store.SetUserAvatar(r.Context(), req.OwnerID, m.URL)
	}
	if req.OwnerType == "vendor" && req.Role == "doc" {
		_ = s.Store.SetVendorDocs(r.Context(), req.OwnerID, true)
	}
	if req.OwnerType == "vendor" && req.Role == "br_cert" {
		_ = s.Store.SetVendorDocs(r.Context(), req.OwnerID, true)
		// Queues the br-cert scheduler's OCR sweep; leaves any
		// already-confirmed number/verify state untouched.
		_ = s.Store.StartVendorBRCertOCR(r.Context(), req.OwnerID)
	}
	s.resyncVendorMediaBestEffort(r.Context(), req.OwnerType, req.OwnerID)
	// ThumbURL/MicroThumb are always empty at this point — thumb_status is
	// only just now "pending" (see thumbnail_worker.go) — so no signing is
	// needed here, unlike m.URL, which is always present.
	m.URL = s.signMediaPath(m.URL)
	writeJSON(w, http.StatusOK, m)
}

// handleMediaRaw proxies a read of one R2 object through our own API — the
// bucket is private, so this is the only way any client (app or web) ever
// sees the bytes. Authorisation is a signed, single-key token in the query
// string (see auth.Tokens.IssueMediaView) rather than the usual bearer
// session header: plain <img>/<video>/<a> tags in the vendor/admin web apps
// can't attach an Authorization header, so this route intentionally sits
// outside the Authenticated middleware group (see server.go).
func (s *Server) handleMediaRaw(w http.ResponseWriter, r *http.Request) {
	if s.Media == nil {
		writeErr(w, http.StatusNotImplemented, "uploads are not configured")
		return
	}
	key := chi.URLParam(r, "*")
	tokenKey, err := s.Tokens.VerifyMediaView(r.URL.Query().Get("token"))
	if err != nil || tokenKey != key {
		writeErr(w, http.StatusForbidden, "invalid or expired token")
		return
	}
	body, contentType, err := s.Media.Get(r.Context(), key)
	if err != nil {
		log.Printf("R2 Get error for key %s: %v", key, err)
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	defer body.Close()
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	io.Copy(w, body)
}

// signMediaPath appends a fresh, short-lived view token to a stored
// "/media/raw/<key>" path — called at HTTP-response time, never when a
// media row is written, so a signed URL is never stale by the time a client
// sees it. Paths that don't start with the proxy prefix (empty strings,
// or anything else) pass through unchanged. Idempotent: if path already
// carries a "?token=..." (e.g. a client persisted an already-signed URL
// it was handed, like the vendor menu editor does with a photo's confirm
// response), any existing query string is discarded first rather than
// appended to, so this never produces a doubly-signed URL.
func (s *Server) signMediaPath(path string) string {
	// Auto-heal any database records that were rewritten to the absolute CDN
	// URL by the importer script, bringing them back to the proxy architecture.
	path = strings.ReplaceAll(path, "https://spotto-images.threelogiq.com/", "/media/raw/")

	const prefix = "/media/raw/"
	base, _, _ := strings.Cut(path, "?")
	if !strings.HasPrefix(base, prefix) {
		return path
	}
	key := strings.TrimPrefix(base, prefix)
	token, err := s.Tokens.IssueMediaView(key)
	if err != nil {
		return base
	}
	return base + "?token=" + token
}

// handleMediaDelete removes an object (R2 + row) after an ownership check.
func (s *Server) handleMediaDelete(w http.ResponseWriter, r *http.Request) {
	if s.Media == nil {
		writeErr(w, http.StatusNotImplemented, "uploads are not configured")
		return
	}
	m, err := s.Store.MediaByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	c := auth.FromContext(r.Context())
	if !s.canUploadFor(r.Context(), c, m.OwnerType, m.OwnerID) {
		writeErr(w, http.StatusForbidden, "not allowed")
		return
	}
	// canUploadFor's blanket true for ownerType "place" is correct for
	// uploads (any traveller can contribute a photo to a public place) but
	// far too permissive for deletion — without this, any signed-in
	// traveller could delete any other traveller's place photo, not just
	// their own.
	if m.OwnerType == "place" && c.Role != domain.RoleAdmin && m.UploadedBy != c.Subject {
		writeErr(w, http.StatusForbidden, "not allowed")
		return
	}
	_ = s.Media.Delete(r.Context(), m.ObjectKey)
	if m.ThumbKey != "" {
		_ = s.Media.Delete(r.Context(), m.ThumbKey)
	}
	if err := s.Store.DeleteMedia(r.Context(), m.ID); err != nil {
		storeErr(w, err)
		return
	}
	if m.Role == "avatar" {
		_ = s.Store.SetUserAvatar(r.Context(), m.OwnerID, "")
	}
	if m.OwnerType == "vendor" && (m.Role == "doc" || m.Role == "br_cert") {
		// Recompute the docs flag from whatever remains.
		remaining, _ := s.Store.MediaByOwner(r.Context(), "vendor", m.OwnerID)
		hasDoc := false
		for _, x := range remaining {
			if x.Role == "doc" || x.Role == "br_cert" {
				hasDoc = true
				break
			}
		}
		_ = s.Store.SetVendorDocs(r.Context(), m.OwnerID, hasDoc)
		if m.Role == "br_cert" {
			// Nothing left to OCR — but a previously confirmed/verified
			// certificate number is preserved, not reset.
			_ = s.Store.ResetVendorBRCertOCR(r.Context(), m.OwnerID)
		}
	}
	s.resyncVendorMediaBestEffort(r.Context(), m.OwnerType, m.OwnerID)
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// signMediaList signs every item's URL in place before it goes out in a
// response — the stored value is always the unsigned "/media/raw/<key>"
// path (see signMediaPath).
func (s *Server) signMediaList(list []domain.Media) []domain.Media {
	for i := range list {
		list[i].URL = s.signMediaPath(list[i].URL)
		// ThumbURL isn't a stored column (only ThumbKey is, via scanMedia) —
		// derive it here the same way handleMediaConfirm does right after
		// generating it, so it round-trips correctly through the DB too.
		if list[i].ThumbKey != "" {
			list[i].ThumbURL = s.signMediaPath(s.Media.PublicURL(list[i].ThumbKey))
		}
	}
	return list
}

// handleVendorMedia lists a vendor's media (cover/gallery/promo/doc) for the
// vendor portal and admin detail.
func (s *Server) handleVendorMedia(w http.ResponseWriter, r *http.Request) {
	media, err := s.Store.MediaByOwner(r.Context(), "vendor", chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.signMediaList(media))
}

// handleVendorOwnMedia lists the calling vendor's own media (vendor portal).
func (s *Server) handleVendorOwnMedia(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	media, err := s.Store.MediaByOwner(r.Context(), "vendor", v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.signMediaList(media))
}

// resyncVendorMediaBestEffort re-mirrors a vendor's media into its traveller
// place whenever that vendor's media changes.
func (s *Server) resyncVendorMediaBestEffort(ctx context.Context, ownerType, ownerID string) {
	if ownerType != "vendor" {
		return
	}
	if v, err := s.Store.VendorByID(ctx, ownerID); err == nil {
		s.syncVendorPlaceBestEffort(ctx, v)
	}
}

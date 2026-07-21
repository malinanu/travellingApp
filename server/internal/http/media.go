package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/media"
)

// singleValueRoles hold at most one object per owner — a new upload replaces
// (and deletes from R2) the previous one.
var singleValueRoles = map[string]bool{"cover": true, "promo": true, "avatar": true}

// roleAllowsKind gates which file kind a role accepts: promo is video-only,
// docs may be a PDF or a scanned image, everything else is an image.
func roleAllowsKind(role, kind string) bool {
	switch role {
	case "promo":
		return kind == media.KindVideo
	case "doc":
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
		}
	}

	m := domain.Media{
		ID: newID("med"), OwnerType: req.OwnerType, OwnerID: req.OwnerID, Kind: kind, Role: req.Role,
		ObjectKey: req.Key, URL: s.Media.PublicURL(req.Key), ContentType: contentType, Size: size,
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
	s.resyncVendorMediaBestEffort(r.Context(), req.OwnerType, req.OwnerID)
	writeJSON(w, http.StatusOK, m)
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
	_ = s.Media.Delete(r.Context(), m.ObjectKey)
	if err := s.Store.DeleteMedia(r.Context(), m.ID); err != nil {
		storeErr(w, err)
		return
	}
	if m.Role == "avatar" {
		_ = s.Store.SetUserAvatar(r.Context(), m.OwnerID, "")
	}
	if m.OwnerType == "vendor" && m.Role == "doc" {
		// Recompute the docs flag from whatever remains.
		remaining, _ := s.Store.MediaByOwner(r.Context(), "vendor", m.OwnerID)
		hasDoc := false
		for _, x := range remaining {
			if x.Role == "doc" {
				hasDoc = true
				break
			}
		}
		_ = s.Store.SetVendorDocs(r.Context(), m.OwnerID, hasDoc)
	}
	s.resyncVendorMediaBestEffort(r.Context(), m.OwnerType, m.OwnerID)
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// handleVendorMedia lists a vendor's media (cover/gallery/promo/doc) for the
// vendor portal and admin detail.
func (s *Server) handleVendorMedia(w http.ResponseWriter, r *http.Request) {
	media, err := s.Store.MediaByOwner(r.Context(), "vendor", chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, media)
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
	writeJSON(w, http.StatusOK, media)
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

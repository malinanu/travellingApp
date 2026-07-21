package httpapi_test

// Integration tests for the R2 media-upload endpoints, using a stub Uploader
// so no real Cloudflare bucket is needed: they exercise the auth checks,
// validation, DB association, avatar wiring, the vendor-place media
// passthrough, and delete. Same real-Postgres pattern as the other tests.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/store/postgres"
)

// stubUploader stands in for R2. Head infers the content type from the key's
// extension (which presign derived from the declared content type), so the
// confirm-side validation runs against realistic values.
type stubUploader struct{ deleted []string }

func (u *stubUploader) PresignPut(_ context.Context, key, _ string, _ time.Duration) (string, error) {
	return "https://r2.example.test/" + key + "?sig=stub", nil
}
func (u *stubUploader) Head(_ context.Context, key string) (int64, string, error) {
	ct := "image/jpeg"
	switch {
	case strings.HasSuffix(key, ".mp4"):
		ct = "video/mp4"
	case strings.HasSuffix(key, ".pdf"):
		ct = "application/pdf"
	}
	return 2048, ct, nil
}
func (u *stubUploader) Delete(_ context.Context, key string) error {
	u.deleted = append(u.deleted, key)
	return nil
}
func (u *stubUploader) PublicURL(key string) string { return "https://cdn.example.test/" + key }

func newMediaTestServer(t *testing.T, store *postgres.Store, withUploader bool) (*httptest.Server, *stubUploader) {
	t.Helper()
	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	s := httpapi.New(store, tokens, cfg)
	var up *stubUploader
	if withUploader {
		up = &stubUploader{}
		s.Media = up
	}
	return httptest.NewServer(s.Router(cfg)), up
}

func TestMediaUploads(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()

	purgeMediaTestData(t, store, ctx)
	t.Cleanup(func() { purgeMediaTestData(t, store, ctx) })

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")
	v1 := domain.Vendor{ID: "ven_media_1", Name: "Media One", Category: "cafe", Town: "Ella", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now, Lat: 6.87, Lng: 81.05}
	v2 := domain.Vendor{ID: "ven_media_2", Name: "Media Two", Category: "cafe", Town: "Galle", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, v1))
	must(t, store.UpsertVendor(ctx, v2))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_media_owner1", Email: "owner1@mediatest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &v1.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_media_traveller", Email: "traveller@mediatest.dev", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))

	srv, up := newMediaTestServer(t, store, true)
	defer srv.Close()
	ownerToken := loginAs(t, srv, "owner1@mediatest.dev", "testpass123")
	travellerToken := loginAs(t, srv, "traveller@mediatest.dev", "testpass123")

	presign := func(token string, body map[string]any) (int, map[string]any) {
		var out map[string]any
		code := doReqJSON(t, srv, http.MethodPost, "/media/presign", token, body, &out)
		return code, out
	}

	t.Run("presign rejects a non-owner and bad types", func(t *testing.T) {
		if code, _ := presign(ownerToken, map[string]any{"ownerType": "vendor", "ownerId": v2.ID, "role": "cover", "contentType": "image/jpeg", "size": 1000}); code != http.StatusForbidden {
			t.Errorf("upload to another vendor: want 403, got %d", code)
		}
		if code, _ := presign(ownerToken, map[string]any{"ownerType": "vendor", "ownerId": v1.ID, "role": "cover", "contentType": "application/zip", "size": 1000}); code != http.StatusBadRequest {
			t.Errorf("bad content type: want 400, got %d", code)
		}
		if code, _ := presign(ownerToken, map[string]any{"ownerType": "vendor", "ownerId": v1.ID, "role": "promo", "contentType": "image/jpeg", "size": 1000}); code != http.StatusBadRequest {
			t.Errorf("image for a video role: want 400, got %d", code)
		}
	})

	t.Run("cover upload persists and reaches the traveller feed", func(t *testing.T) {
		code, out := presign(ownerToken, map[string]any{"ownerType": "vendor", "ownerId": v1.ID, "role": "cover", "contentType": "image/jpeg", "size": 1000})
		if code != http.StatusOK {
			t.Fatalf("presign: want 200, got %d", code)
		}
		key, _ := out["key"].(string)
		if key == "" {
			t.Fatal("presign returned no key")
		}
		if got := doReq(t, srv, http.MethodPost, "/media/confirm", ownerToken, map[string]any{"key": key, "ownerType": "vendor", "ownerId": v1.ID, "role": "cover"}); got != http.StatusOK {
			t.Fatalf("confirm: want 200, got %d", got)
		}
		// Listed on the vendor's media.
		var vm []domain.Media
		doReqJSON(t, srv, http.MethodGet, "/vendor/media", ownerToken, nil, &vm)
		if len(vm) != 1 || vm[0].Role != "cover" {
			t.Fatalf("vendor media: want 1 cover, got %+v", vm)
		}
		// Mirrored into the traveller place feed.
		var places []map[string]any
		doReqJSON(t, srv, http.MethodGet, "/places", travellerToken, nil, &places)
		found := false
		for _, p := range places {
			if p["vendorId"] == v1.ID {
				found = p["cover"] != nil && p["cover"] != ""
			}
		}
		if !found {
			t.Error("vendor cover did not appear on its place in /places")
		}
	})

	t.Run("avatar upload sets users.avatar_url", func(t *testing.T) {
		code, out := presign(travellerToken, map[string]any{"ownerType": "user", "ownerId": "usr_media_traveller", "role": "avatar", "contentType": "image/png", "size": 500})
		if code != http.StatusOK {
			t.Fatalf("avatar presign: want 200, got %d", code)
		}
		key, _ := out["key"].(string)
		if got := doReq(t, srv, http.MethodPost, "/media/confirm", travellerToken, map[string]any{"key": key, "ownerType": "user", "ownerId": "usr_media_traveller", "role": "avatar"}); got != http.StatusOK {
			t.Fatalf("avatar confirm: want 200, got %d", got)
		}
		var me map[string]any
		doReqJSON(t, srv, http.MethodGet, "/me", travellerToken, nil, &me)
		if av, _ := me["avatarUrl"].(string); av == "" {
			t.Error("avatarUrl not set on /me after upload")
		}
	})

	t.Run("delete removes the row and the R2 object", func(t *testing.T) {
		var vm []domain.Media
		doReqJSON(t, srv, http.MethodGet, "/vendor/media", ownerToken, nil, &vm)
		if len(vm) == 0 {
			t.Fatal("no vendor media to delete")
		}
		id, key := vm[0].ID, vm[0].ObjectKey
		if got := doReq(t, srv, http.MethodDelete, "/media/"+id, ownerToken, nil); got != http.StatusOK {
			t.Fatalf("delete: want 200, got %d", got)
		}
		var after []domain.Media
		doReqJSON(t, srv, http.MethodGet, "/vendor/media", ownerToken, nil, &after)
		if len(after) != 0 {
			t.Errorf("media not removed: %+v", after)
		}
		deleted := false
		for _, k := range up.deleted {
			if k == key {
				deleted = true
			}
		}
		if !deleted {
			t.Error("R2 object was not deleted")
		}
	})

	t.Run("endpoints report 501 when R2 is unconfigured", func(t *testing.T) {
		noR2, _ := newMediaTestServer(t, store, false)
		defer noR2.Close()
		tok := loginAs(t, noR2, "owner1@mediatest.dev", "testpass123")
		if got := doReq(t, noR2, http.MethodPost, "/media/presign", tok, map[string]any{"ownerType": "vendor", "ownerId": v1.ID, "role": "cover", "contentType": "image/jpeg", "size": 10}); got != http.StatusNotImplemented {
			t.Errorf("want 501 without R2, got %d", got)
		}
	})
}

func purgeMediaTestData(t *testing.T, store *postgres.Store, ctx context.Context) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM media WHERE owner_id LIKE 'ven_media_%' OR owner_id LIKE 'usr_media_%'`,
		`DELETE FROM places WHERE vendor_id LIKE 'ven_media_%'`,
		`DELETE FROM users WHERE id LIKE 'usr_media_%'`,
		`DELETE FROM vendors WHERE id LIKE 'ven_media_%'`,
	} {
		if _, err := store.Pool.Exec(ctx, q); err != nil {
			t.Fatalf("purge (%s): %v", q, err)
		}
	}
}

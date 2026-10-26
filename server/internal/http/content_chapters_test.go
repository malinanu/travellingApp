package httpapi_test

// Photo chapters on audio guides: a chapter's photo is one of the post's own
// uploads, stored as its bare /media/raw path and served signed, so a
// listener sees which spot is being described. Same real-Postgres pattern
// as content_test.go.

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func purgeChapterTestData(store *postgres.Store) {
	ctx := context.Background()
	store.Pool.Exec(ctx, `DELETE FROM content_items WHERE author_id LIKE 'usr_chp_%'`)
	store.Pool.Exec(ctx, `DELETE FROM media WHERE owner_type='content' AND uploaded_by LIKE 'usr_chp_%'`)
	store.Pool.Exec(ctx, `DELETE FROM admin_audit_log WHERE actor_id LIKE 'usr_chp_%'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_chp_%'`)
}

func TestContentChapterPhotos(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()
	purgeChapterTestData(store)
	t.Cleanup(func() { purgeChapterTestData(store) })

	pw, _ := auth.HashPassword("testpass123")
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_chp_admin", Email: "admin@chptest.dev", Role: domain.RoleAdmin, PasswordHash: pw, CreatedAt: time.Now()}))
	srv, _ := newMediaTestServer(t, store, true)
	t.Cleanup(srv.Close)
	admin := loginAs(t, srv, "admin@chptest.dev", "testpass123")

	call := func(method, path string, body any) (int, map[string]any) {
		t.Helper()
		var out map[string]any
		code := doReqJSON(t, srv, method, path, admin, body, &out)
		return code, out
	}
	want := func(code, wantCode int, what string, out map[string]any) {
		t.Helper()
		if code != wantCode {
			t.Fatalf("%s: got %d, want %d (%v)", what, code, wantCode, out)
		}
	}
	upload := func(postID, role, contentType string) string {
		t.Helper()
		code, out := call(http.MethodPost, "/media/presign", map[string]any{
			"ownerType": "content", "ownerId": postID, "role": role, "contentType": contentType, "size": 1000,
		})
		want(code, http.StatusOK, role+" presign", out)
		code, conf := call(http.MethodPost, "/media/confirm", map[string]any{
			"key": out["key"], "ownerType": "content", "ownerId": postID, "role": role,
		})
		want(code, http.StatusOK, role+" confirm", conf)
		path, _, _ := strings.Cut(conf["url"].(string), "?") // served signed
		return path
	}
	chapterImage := func(item map[string]any, i int) string {
		t.Helper()
		chapters, _ := item["chapters"].([]any)
		if len(chapters) <= i {
			t.Fatalf("chapters = %v", item["chapters"])
		}
		s, _ := chapters[i].(map[string]any)["imageUrl"].(string)
		return s
	}

	code, post := call(http.MethodPost, "/studio/content", map[string]any{"type": "audio", "title": "Sigiriya rock"})
	want(code, http.StatusCreated, "create", post)
	id := post["id"].(string)
	upload(id, "audio", "audio/mpeg")

	// A chapter photo is an image; audio can't be one.
	code, out := call(http.MethodPost, "/media/presign", map[string]any{
		"ownerType": "content", "ownerId": id, "role": "chapter", "contentType": "audio/mpeg", "size": 1000,
	})
	want(code, http.StatusBadRequest, "audio as a chapter photo", out)

	photo := upload(id, "chapter", "image/jpeg")
	if !strings.HasPrefix(photo, "/media/raw/content/"+id+"/") {
		t.Fatalf("chapter photo url = %q", photo)
	}
	save := func(chapters []map[string]any) (int, map[string]any) {
		return call(http.MethodPut, "/studio/content/"+id, map[string]any{
			"type": "audio", "title": "Sigiriya rock", "audioSeconds": 600, "chapters": chapters,
		})
	}

	// The editor sends back the signed link it was given.
	code, out = save([]map[string]any{
		{"title": "Mirror wall", "startSec": 120},
		{"title": "Lion's Paw", "startSec": 0, "imageUrl": photo + "?token=stale"},
	})
	want(code, http.StatusOK, "save chapters with a photo", out)
	if got := chapterImage(out, 0); !strings.HasPrefix(got, photo+"?token=") || strings.Contains(got, "stale") {
		t.Errorf("served chapter photo = %q, want %s signed afresh", got, photo)
	}
	if got := chapterImage(out, 1); got != "" {
		t.Errorf("chapter without a photo = %q", got)
	}
	var stored string
	must(t, store.Pool.QueryRow(ctx, `SELECT chapters::text FROM content_items WHERE id=$1`, id).Scan(&stored))
	if strings.Contains(stored, "token") || !strings.Contains(stored, photo) {
		t.Errorf("stored chapters = %s", stored)
	}

	// Only this post's own uploads.
	for _, bad := range []string{"/media/raw/content/cnt_someone_else/m_x.jpg", "https://example.com/x.jpg", "/media/raw/vendor/v1/m_x.jpg"} {
		code, out = save([]map[string]any{{"title": "Entrance", "startSec": 0, "imageUrl": bad}})
		want(code, http.StatusBadRequest, "foreign photo "+bad, out)
	}

	// Travellers get it signed on the live post.
	code, out = call(http.MethodPost, "/admin/content/"+id+"/publish", nil)
	want(code, http.StatusOK, "publish", out)
	var live map[string]any
	code = doReqJSON(t, srv, http.MethodGet, "/content/"+id, "", nil, &live)
	want(code, http.StatusOK, "public detail", live)
	if got := chapterImage(live, 0); !strings.HasPrefix(got, photo+"?token=") {
		t.Errorf("public chapter photo = %q", got)
	}
}

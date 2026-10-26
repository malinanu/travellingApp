package httpapi_test

// Travel content: who can write, the review flow from draft to live, what
// travellers can read without signing in, post uploads (audio included) and
// ranged reads of media. Same real-Postgres pattern as validate_test.go
// (skips without a database), with a stub R2 from media_test.go.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

func purgeContentTestData(store *postgres.Store) {
	ctx := context.Background()
	store.Pool.Exec(ctx, `DELETE FROM content_items WHERE author_id LIKE 'usr_cnt_%'`)
	store.Pool.Exec(ctx, `DELETE FROM media WHERE owner_type='content' AND uploaded_by LIKE 'usr_cnt_%'`)
	store.Pool.Exec(ctx, `DELETE FROM admin_audit_log WHERE actor_id LIKE 'usr_cnt_%'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_cnt_%'`)
	store.Pool.Exec(ctx, `DELETE FROM regions WHERE id='cnttest'`)
	store.Pool.Exec(ctx, `DELETE FROM places WHERE id='plc_cnt_1'`)
}

func TestContentPublishing(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()
	purgeContentTestData(store)
	t.Cleanup(func() { purgeContentTestData(store) })

	pw, _ := auth.HashPassword("testpass123")
	for _, u := range []domain.User{
		{ID: "usr_cnt_admin", Email: "admin@cnttest.dev", Role: domain.RoleAdmin},
		{ID: "usr_cnt_pub", Email: "pub@cnttest.dev", Role: domain.RoleTraveller},
		{ID: "usr_cnt_pub2", Email: "pub2@cnttest.dev", Role: domain.RoleTraveller},
	} {
		u.PasswordHash, u.CreatedAt = pw, time.Now()
		must(t, store.CreateUser(ctx, u))
	}
	must(t, store.UpsertRegion(ctx, domain.Region{ID: "cnttest", Name: "Testville", Lat: -50, Lng: -100}))
	must(t, store.UpsertPlace(ctx, domain.Place{ID: "plc_cnt_1", Name: "Cnt Test Cafe", Data: []byte(`{"lat":-50.1,"lng":-100.1,"type":"Coffee"}`)}))

	srv, up := newMediaTestServer(t, store, true)
	t.Cleanup(srv.Close)
	admin := loginAs(t, srv, "admin@cnttest.dev", "testpass123")
	pub := loginAs(t, srv, "pub@cnttest.dev", "testpass123")
	pub2 := loginAs(t, srv, "pub2@cnttest.dev", "testpass123")

	call := func(tok, method, path string, body any) (int, map[string]any) {
		t.Helper()
		var out map[string]any
		code := doReqJSON(t, srv, method, path, tok, body, &out)
		return code, out
	}
	want := func(code, wantCode int, what string, out map[string]any) {
		t.Helper()
		if code != wantCode {
			t.Fatalf("%s: got %d, want %d (%v)", what, code, wantCode, out)
		}
	}
	grant := func(id string, on bool) {
		t.Helper()
		code, out := call(admin, http.MethodPut, "/admin/users/"+id+"/publisher", map[string]any{"publisher": on, "reason": "writes for us"})
		want(code, http.StatusOK, "grant publisher", out)
	}

	// Without the flag the studio is closed, and only admins can grant it.
	code, out := call(pub, http.MethodGet, "/studio/content", nil)
	want(code, http.StatusForbidden, "studio without the flag", out)
	code, out = call(pub, http.MethodPut, "/admin/users/usr_cnt_pub/publisher", map[string]any{"publisher": true, "reason": "me"})
	want(code, http.StatusForbidden, "self-grant", out)
	code, out = call(admin, http.MethodPut, "/admin/users/usr_cnt_pub/publisher", map[string]any{"publisher": true})
	want(code, http.StatusBadRequest, "grant without a reason", out)
	grant("usr_cnt_pub", true)
	grant("usr_cnt_pub2", true)
	var audited int
	store.Pool.QueryRow(ctx, `SELECT count(*) FROM admin_audit_log WHERE action='user.publisher' AND target_id='usr_cnt_pub'`).Scan(&audited)
	if audited != 1 {
		t.Errorf("publisher grant audited %d times, want 1", audited)
	}
	var me map[string]any
	doReqJSON(t, srv, http.MethodGet, "/me", pub, nil, &me)
	if me["publisher"] != true {
		t.Errorf("/me publisher = %v, want true", me["publisher"])
	}

	// A publisher writes a blog post.
	code, out = call(pub, http.MethodPost, "/studio/content", map[string]any{"type": "podcast", "title": "x"})
	want(code, http.StatusBadRequest, "unknown type", out)
	code, post := call(pub, http.MethodPost, "/studio/content", map[string]any{
		"type": "blog", "title": "Ten Things to Eat in Testville!", "summary": "Street food.",
		"tags": []string{"Food", "#food", "street"}, "regionId": "cnttest",
	})
	want(code, http.StatusCreated, "create", post)
	id := post["id"].(string)
	if post["status"] != "draft" || post["authorId"] != "usr_cnt_pub" || !strings.HasPrefix(post["slug"].(string), "ten-things-to-eat-in-testville-") {
		t.Fatalf("new post: %v", post)
	}
	if tags := fmt.Sprint(post["tags"]); tags != "[food street]" {
		t.Errorf("tags = %s, want [food street]", tags)
	}

	// Other publishers can't see or touch it.
	code, out = call(pub2, http.MethodGet, "/studio/content/"+id, nil)
	want(code, http.StatusNotFound, "other publisher reads", out)
	code, out = call(pub2, http.MethodGet, "/studio/content", nil)
	want(code, http.StatusOK, "other publisher lists", out)
	if items := out["items"].([]any); len(items) != 0 {
		t.Errorf("other publisher sees %d posts", len(items))
	}
	code, out = call(pub, http.MethodPost, "/admin/content/"+id+"/publish", nil)
	want(code, http.StatusForbidden, "publisher publishes", out)

	// It can't be sent for review empty; once sent, it's locked.
	code, out = call(pub, http.MethodPost, "/studio/content/"+id+"/submit", nil)
	want(code, http.StatusBadRequest, "submit without text", out)
	body := map[string]any{"type": "blog", "title": "Ten Things to Eat in Testville", "summary": "Street food.",
		"body": "## Kottu\nChopped roti, ![stall](/media/raw/content/x/m_1.jpg?token=old).", "tags": []string{"food"}, "regionId": "cnttest"}
	code, out = call(pub, http.MethodPut, "/studio/content/"+id, body)
	want(code, http.StatusOK, "save", out)
	var storedBody string
	store.Pool.QueryRow(ctx, `SELECT body FROM content_items WHERE id=$1`, id).Scan(&storedBody)
	if strings.Contains(storedBody, "token=old") {
		t.Errorf("stored body kept a view token: %q", storedBody)
	}
	code, out = call(pub, http.MethodPost, "/studio/content/"+id+"/submit", nil)
	want(code, http.StatusOK, "submit", out)
	if out["status"] != "in_review" {
		t.Fatalf("after submit: %v", out["status"])
	}
	code, out = call(pub, http.MethodPut, "/studio/content/"+id, body)
	want(code, http.StatusConflict, "edit while in review", out)
	var count map[string]any
	doReqJSON(t, srv, http.MethodGet, "/admin/content/review-count", admin, nil, &count)
	if count["count"].(float64) < 1 {
		t.Errorf("review count = %v", count["count"])
	}

	// The admin sends it back with a note, the author fixes it and resends.
	code, out = call(admin, http.MethodPost, "/admin/content/"+id+"/reject", map[string]any{"note": ""})
	want(code, http.StatusBadRequest, "reject without a note", out)
	code, out = call(admin, http.MethodPost, "/admin/content/"+id+"/reject", map[string]any{"note": "Add opening hours."})
	want(code, http.StatusOK, "reject", out)
	code, out = call(pub, http.MethodGet, "/studio/content/"+id, nil)
	if out["status"] != "rejected" || out["reviewNote"] != "Add opening hours." {
		t.Fatalf("author sees: %v / %v", out["status"], out["reviewNote"])
	}
	code, out = call(pub, http.MethodPut, "/studio/content/"+id, body)
	want(code, http.StatusOK, "edit after rejection", out)
	code, out = call(pub, http.MethodPost, "/studio/content/"+id+"/submit", nil)
	want(code, http.StatusOK, "resubmit", out)
	if out["reviewNote"] != "" {
		t.Errorf("note kept after resubmit: %v", out["reviewNote"])
	}

	// Published, it's public by slug or id, with fresh media tokens.
	code, out = call(admin, http.MethodPost, "/admin/content/"+id+"/publish", nil)
	want(code, http.StatusOK, "publish", out)
	slug := out["slug"].(string)
	code, out = call("", http.MethodGet, "/content/"+slug, nil)
	want(code, http.StatusOK, "public detail by slug", out)
	if out["reviewNote"] != "" || out["authorId"] != nil || !strings.Contains(out["body"].(string), "m_1.jpg?token=") {
		t.Errorf("public detail: note=%v author=%v body=%q", out["reviewNote"], out["authorId"], out["body"])
	}
	call("", http.MethodGet, "/content/"+id, nil)
	var views int
	store.Pool.QueryRow(ctx, `SELECT views FROM content_items WHERE id=$1`, id).Scan(&views)
	if views != 2 {
		t.Errorf("views = %d, want 2", views)
	}

	// Admins can publish their own posts without a review, but not another
	// author's unsent draft.
	code, draft := call(pub2, http.MethodPost, "/studio/content", map[string]any{"type": "tip", "title": "Carry cash", "summary": "Small shops rarely take cards."})
	want(code, http.StatusCreated, "pub2 draft", draft)
	code, out = call(admin, http.MethodPost, "/admin/content/"+draft["id"].(string)+"/publish", nil)
	want(code, http.StatusConflict, "publish someone's draft", out)
	code, own := call(admin, http.MethodPost, "/studio/content", map[string]any{
		"type": "itinerary", "title": "Two days in Testville", "regionId": "cnttest",
		"days": []map[string]any{{"title": "Day 1", "stops": []map[string]any{
			{"placeId": "plc_cnt_1", "name": "Cnt Test Cafe", "time": "09:00"},
			{"name": "Sunset point", "lat": -50.2, "lng": -100.2, "time": "18:30"},
		}}},
	})
	want(code, http.StatusCreated, "admin itinerary", own)
	code, out = call(admin, http.MethodPost, "/admin/content/"+own["id"].(string)+"/publish", nil)
	want(code, http.StatusOK, "admin publishes own draft", out)
	code, out = call("", http.MethodGet, "/content/"+own["id"].(string), nil)
	if places := out["places"].([]any); len(places) != 1 || places[0].(map[string]any)["name"] != "Cnt Test Cafe" {
		t.Errorf("itinerary places = %v", out["places"])
	}

	// The public list shows live posts only, and filters.
	list := func(q string) []string {
		t.Helper()
		var res struct {
			Items []struct {
				ID string `json:"id"`
			} `json:"items"`
		}
		if code := doReqJSON(t, srv, http.MethodGet, "/content?region=cnttest"+q, "", nil, &res); code != http.StatusOK {
			t.Fatalf("list %s: %d", q, code)
		}
		ids := []string{}
		for _, it := range res.Items {
			ids = append(ids, it.ID)
		}
		return ids
	}
	if got := list(""); len(got) != 2 {
		t.Errorf("live posts = %v, want 2", got)
	}
	if got := list("&type=blog"); len(got) != 1 || got[0] != id {
		t.Errorf("blogs = %v", got)
	}
	if got := list("&tag=food"); len(got) != 1 || got[0] != id {
		t.Errorf("tag food = %v", got)
	}
	if got := list("&q=two+days"); len(got) != 1 || got[0] != own["id"] {
		t.Errorf("search = %v", got)
	}

	// Unpublishing takes it down; the author can then edit again.
	code, out = call(admin, http.MethodPost, "/admin/content/"+id+"/unpublish", nil)
	want(code, http.StatusOK, "unpublish", out)
	code, out = call("", http.MethodGet, "/content/"+slug, nil)
	want(code, http.StatusNotFound, "public after unpublish", out)
	code, out = call(pub, http.MethodPut, "/studio/content/"+id, body)
	want(code, http.StatusOK, "edit after unpublish", out)

	// Uploads: only the author (or an admin) on an editable post, audio
	// within its limit.
	presign := func(tok, role, ct string, size int64) (int, map[string]any) {
		return call(tok, http.MethodPost, "/media/presign", map[string]any{
			"ownerType": "content", "ownerId": draft["id"], "role": role, "contentType": ct, "size": size,
		})
	}
	code, out = call(pub2, http.MethodPut, "/studio/content/"+draft["id"].(string), map[string]any{"type": "audio", "title": "Carry cash"})
	want(code, http.StatusOK, "make it audio", out)
	code, out = presign(pub, "audio", "audio/mpeg", 1000)
	want(code, http.StatusForbidden, "upload to someone else's post", out)
	code, out = presign(pub2, "audio", "image/jpeg", 1000)
	want(code, http.StatusBadRequest, "image as audio", out)
	code, out = presign(pub2, "audio", "audio/mpeg", 250<<20)
	want(code, http.StatusBadRequest, "audio over 200 MB", out)
	code, out = call(pub2, http.MethodPost, "/studio/content/"+draft["id"].(string)+"/submit", nil)
	want(code, http.StatusBadRequest, "submit audio without a file", out)
	code, out = presign(pub2, "audio", "audio/mpeg", 150<<20)
	want(code, http.StatusOK, "audio presign", out)
	code, conf := call(pub2, http.MethodPost, "/media/confirm", map[string]any{
		"key": out["key"], "ownerType": "content", "ownerId": draft["id"], "role": "audio",
	})
	want(code, http.StatusOK, "audio confirm", conf)
	code, out = call(pub2, http.MethodPost, "/studio/content/"+draft["id"].(string)+"/submit", nil)
	want(code, http.StatusOK, "submit audio", out)
	if !strings.Contains(out["audioUrl"].(string), ".mp3?token=") {
		t.Errorf("audioUrl = %v", out["audioUrl"])
	}
	code, out = presign(pub2, "cover", "image/jpeg", 1000)
	want(code, http.StatusForbidden, "upload while in review", out)

	// A revoked publisher is shut out at once.
	grant("usr_cnt_pub2", false)
	code, out = call(pub2, http.MethodGet, "/studio/content", nil)
	want(code, http.StatusForbidden, "studio after revoke", out)

	// Deleting a post removes its files from storage.
	code, out = call(admin, http.MethodDelete, "/studio/content/"+draft["id"].(string), nil)
	want(code, http.StatusOK, "admin deletes", out)
	if len(up.deleted) == 0 || !strings.HasSuffix(up.deleted[len(up.deleted)-1], ".mp3") {
		t.Errorf("deleted from storage: %v", up.deleted)
	}

	// Ranged reads of media come back as 206 with just those bytes.
	rangeGet := func(rng string) (int, string, string) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+conf["url"].(string), nil)
		if rng != "" {
			req.Header.Set("Range", rng)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, res.Header.Get("Content-Range"), string(b)
	}
	if code, cr, b := rangeGet("bytes=0-4"); code != http.StatusPartialContent || b != "stub-" || !strings.HasPrefix(cr, "bytes 0-4/") {
		t.Errorf("range 0-4: %d %q %q", code, cr, b)
	}
	if code, cr, b := rangeGet(""); code != http.StatusOK || cr != "" || !strings.HasPrefix(b, "stub-bytes-content/") {
		t.Errorf("no range: %d %q %q", code, cr, b)
	}
	if code, _, _ := rangeGet("bytes=99999-"); code != http.StatusRequestedRangeNotSatisfiable {
		t.Errorf("range past the end: %d", code)
	}
	if code, _, _ := rangeGet("bytes=0-1,4-5"); code != http.StatusOK {
		t.Errorf("several ranges should send it all: %d", code)
	}
}

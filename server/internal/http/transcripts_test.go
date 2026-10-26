package httpapi_test

// Audio guide transcripts: uploading audio queues one, the transcriber
// service claims it with its token, posts it back (or reports a failure),
// and publishers/admins review and correct it; travellers get the finished
// transcript, and search matches its words. Real Postgres like
// content_test.go, with the stub R2 from media_test.go.

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

const testTranscriberToken = "transcriber-test-token-0123456789"

func purgeTranscriptTestData(store *postgres.Store) {
	ctx := context.Background()
	store.Pool.Exec(ctx, `DELETE FROM content_items WHERE author_id LIKE 'usr_trn_%'`)
	store.Pool.Exec(ctx, `DELETE FROM media WHERE owner_type='content' AND uploaded_by LIKE 'usr_trn_%'`)
	store.Pool.Exec(ctx, `DELETE FROM admin_audit_log WHERE actor_id LIKE 'usr_trn_%'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_trn_%'`)
}

func transcriptTestServer(t *testing.T, store *postgres.Store, token string) *httptest.Server {
	t.Helper()
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), TranscriberToken: token}
	s := httpapi.New(store, auth.NewTokens("test-secret-"+t.Name()), cfg)
	s.Media = &stubUploader{}
	srv := httptest.NewServer(s.Router(cfg))
	t.Cleanup(srv.Close)
	return srv
}

func TestContentTranscripts(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()
	purgeTranscriptTestData(store)
	t.Cleanup(func() { purgeTranscriptTestData(store) })

	pw, _ := auth.HashPassword("testpass123")
	for _, u := range []domain.User{
		{ID: "usr_trn_admin", Email: "admin@trntest.dev", Role: domain.RoleAdmin},
		{ID: "usr_trn_pub", Email: "pub@trntest.dev", Role: domain.RoleTraveller},
	} {
		u.PasswordHash, u.CreatedAt = pw, time.Now()
		must(t, store.CreateUser(ctx, u))
	}
	execSQL(t, store, `UPDATE users SET publisher=true WHERE id='usr_trn_pub'`)
	// Leftover jobs from other runs would be claimed first.
	execSQL(t, store, `DELETE FROM content_transcripts`)

	srv := transcriptTestServer(t, store, testTranscriberToken)
	admin := loginAs(t, srv, "admin@trntest.dev", "testpass123")
	pub := loginAs(t, srv, "pub@trntest.dev", "testpass123")

	call := func(tok, method, path string, body any) (int, map[string]any) {
		t.Helper()
		var out map[string]any
		return doReqJSON(t, srv, method, path, tok, body, &out), out
	}
	want := func(code, wantCode int, what string, out map[string]any) {
		t.Helper()
		if code != wantCode {
			t.Fatalf("%s: got %d, want %d (%v)", what, code, wantCode, out)
		}
	}
	transcript := func(id string) map[string]any {
		t.Helper()
		code, out := call(pub, http.MethodGet, "/studio/content/"+id, nil)
		want(code, http.StatusOK, "studio get", out)
		tr, _ := out["transcript"].(map[string]any)
		return tr
	}
	uploadAudio := func(id string) string {
		t.Helper()
		code, out := call(pub, http.MethodPost, "/media/presign", map[string]any{
			"ownerType": "content", "ownerId": id, "role": "audio", "contentType": "audio/mpeg", "size": 1000,
		})
		want(code, http.StatusOK, "presign", out)
		code, conf := call(pub, http.MethodPost, "/media/confirm", map[string]any{
			"key": out["key"], "ownerType": "content", "ownerId": id, "role": "audio",
		})
		want(code, http.StatusOK, "confirm", conf)
		return conf["id"].(string)
	}

	code, post := call(pub, http.MethodPost, "/studio/content", map[string]any{
		"type": "audio", "title": "Galle Fort walking tour", "summary": "Ramparts at dusk.",
	})
	want(code, http.StatusCreated, "create", post)
	id := post["id"].(string)
	if tr := transcript(id); tr != nil {
		t.Fatalf("transcript before any audio: %v", tr)
	}
	uploadAudio(id)
	if tr := transcript(id); tr == nil || tr["status"] != "queued" {
		t.Fatalf("after upload: %v", tr)
	}

	// The internal endpoints want the service's token.
	worker := func(tok, method, path string, body any) (int, map[string]any) {
		t.Helper()
		var out map[string]any
		return doReqJSON(t, srv, method, path, tok, body, &out), out
	}
	code, out := worker("", http.MethodPost, "/internal/transcripts/claim", nil)
	want(code, http.StatusUnauthorized, "claim without token", out)
	code, out = worker(admin, http.MethodPost, "/internal/transcripts/claim", nil)
	want(code, http.StatusUnauthorized, "claim with a user token", out)

	claim := func() (int, map[string]any) {
		t.Helper()
		return worker(testTranscriberToken, http.MethodPost, "/internal/transcripts/claim", nil)
	}
	code, job := claim()
	want(code, http.StatusOK, "claim", job)
	audioPath := job["audioPath"].(string)
	if job["id"] != id || !strings.HasPrefix(audioPath, "/media/raw/content/"+id+"/") ||
		!strings.Contains(job["audioUrl"].(string), "?token=") || job["attempt"] != float64(1) {
		t.Fatalf("job: %v", job)
	}
	if code, out = claim(); code != http.StatusNoContent {
		t.Fatalf("second claim: %d %v", code, out)
	}
	if tr := transcript(id); tr["status"] != "working" {
		t.Fatalf("while working: %v", tr["status"])
	}

	// A transient failure is retried after a delay.
	code, out = worker(testTranscriberToken, http.MethodPost, "/internal/transcripts/"+id+"/fail",
		map[string]any{"audioPath": audioPath, "error": "connection reset", "retryable": true})
	want(code, http.StatusOK, "fail", out)
	if tr := transcript(id); tr["status"] != "queued" {
		t.Fatalf("after a retryable failure: %v", tr)
	}
	if code, _ = claim(); code != http.StatusNoContent {
		t.Fatalf("claimed during the retry delay: %d", code)
	}
	execSQL(t, store, `UPDATE content_transcripts SET lease_until = now() - interval '1 second' WHERE content_id=$1`, id)
	code, job = claim()
	want(code, http.StatusOK, "claim after the delay", job)
	if job["attempt"] != float64(2) {
		t.Errorf("attempt = %v, want 2", job["attempt"])
	}

	// Results with impossible timings are refused; a good one is stored once.
	code, out = worker(testTranscriberToken, http.MethodPost, "/internal/transcripts/"+id+"/result", map[string]any{
		"audioPath": audioPath, "segments": []map[string]any{{"start": 9, "end": 12, "text": "b"}, {"start": 1, "end": 3, "text": "a"}},
	})
	want(code, http.StatusBadRequest, "out-of-order segments", out)
	result := map[string]any{
		"audioPath": audioPath, "language": "en", "durationSec": 200.5,
		"segments": []map[string]any{
			{"start": 0, "end": 4.2, "text": " Welcome to Galle Fort. "},
			{"start": 4.2, "end": 9.8, "text": "Walk towards the old lighthouse."},
			{"start": 9.8, "end": 10, "text": "   "},
		},
	}
	code, out = worker(testTranscriberToken, http.MethodPost, "/internal/transcripts/"+id+"/result", result)
	want(code, http.StatusOK, "result", out)
	if out["stored"] != true {
		t.Fatalf("result not stored: %v", out)
	}
	tr := transcript(id)
	segs := tr["segments"].([]any)
	if tr["status"] != "done" || tr["language"] != "en" || len(segs) != 2 ||
		segs[0].(map[string]any)["text"] != "Welcome to Galle Fort." {
		t.Fatalf("done transcript: %v", tr)
	}
	code, out = worker(testTranscriberToken, http.MethodPost, "/internal/transcripts/"+id+"/result", result)
	if code != http.StatusOK || out["stored"] != false {
		t.Errorf("repeated result: %d %v", code, out)
	}

	// Live: travellers get the transcript (without review details) and
	// search finds the spoken words.
	code, out = call(pub, http.MethodPost, "/studio/content/"+id+"/submit", nil)
	want(code, http.StatusOK, "submit", out)
	code, out = call(admin, http.MethodPost, "/admin/content/"+id+"/publish", nil)
	want(code, http.StatusOK, "publish", out)
	code, out = call("", http.MethodGet, "/content/"+id, nil)
	want(code, http.StatusOK, "public detail", out)
	ptr, _ := out["transcript"].(map[string]any)
	if ptr == nil || ptr["status"] != nil || len(ptr["segments"].([]any)) != 2 {
		t.Fatalf("public transcript: %v", out["transcript"])
	}
	var found struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	doReqJSON(t, srv, http.MethodGet, "/content?q=old+lighthouse", "", nil, &found)
	if len(found.Items) != 1 || found.Items[0].ID != id {
		t.Errorf("search by spoken words: %v", found.Items)
	}

	// Corrections: locked for the publisher while live, open to admins.
	code, out = call(pub, http.MethodPut, "/studio/content/"+id+"/transcript", map[string]any{"texts": []string{"x", "y"}})
	want(code, http.StatusConflict, "publisher edits a live transcript", out)
	code, out = call(admin, http.MethodPut, "/studio/content/"+id+"/transcript", map[string]any{"texts": []string{"Welcome to Galle Fort!"}})
	want(code, http.StatusConflict, "wrong number of lines", out)
	code, out = call(admin, http.MethodPut, "/studio/content/"+id+"/transcript",
		map[string]any{"texts": []string{"Welcome to Galle Fort!", "Walk towards the lighthouse."}})
	want(code, http.StatusOK, "admin edits", out)
	if etr := out["transcript"].(map[string]any); etr["edited"] != true {
		t.Errorf("edited flag: %v", etr)
	}
	doReqJSON(t, srv, http.MethodGet, "/content?q=old+lighthouse", "", nil, &found)
	if len(found.Items) != 0 {
		t.Errorf("search still matches the old words: %v", found.Items)
	}

	// Run again: back in the queue, corrections gone.
	code, out = call(admin, http.MethodPost, "/studio/content/"+id+"/transcript/retry", nil)
	want(code, http.StatusOK, "retry", out)
	if rtr := out["transcript"].(map[string]any); rtr["status"] != "queued" || len(rtr["segments"].([]any)) != 0 {
		t.Fatalf("after retry: %v", rtr)
	}

	// New audio while a job runs: the old job's result is ignored.
	code, job = claim()
	want(code, http.StatusOK, "claim for the retry", job)
	oldPath := job["audioPath"].(string)
	code, out = call(admin, http.MethodPost, "/admin/content/"+id+"/unpublish", nil)
	want(code, http.StatusOK, "unpublish", out)
	newMedia := uploadAudio(id)
	code, out = worker(testTranscriberToken, http.MethodPost, "/internal/transcripts/"+id+"/result",
		map[string]any{"audioPath": oldPath, "segments": []map[string]any{{"start": 0, "end": 1, "text": "stale"}}})
	if code != http.StatusOK || out["stored"] != false {
		t.Errorf("stale result: %d %v", code, out)
	}
	if tr := transcript(id); tr["status"] != "queued" {
		t.Errorf("after new audio: %v", tr)
	}

	// A recording that keeps killing the transcriber ends up failed.
	code, job = claim()
	want(code, http.StatusOK, "claim new audio", job)
	execSQL(t, store, `UPDATE content_transcripts SET attempts=3, lease_until = now() - interval '1 second' WHERE content_id=$1`, id)
	if code, _ = claim(); code != http.StatusNoContent {
		t.Errorf("claimed a recording past its attempts: %d", code)
	}
	if tr := transcript(id); tr["status"] != "failed" || tr["error"] == "" {
		t.Errorf("after too many crashes: %v", tr)
	}

	// Removing the audio removes its transcript.
	code, out = call(pub, http.MethodDelete, "/media/"+newMedia, nil)
	want(code, http.StatusOK, "delete audio", out)
	if tr := transcript(id); tr != nil {
		t.Errorf("transcript after the audio was deleted: %v", tr)
	}

	// Without a token, transcripts are off: nothing is queued and the
	// internal endpoints don't exist.
	off := transcriptTestServer(t, store, "")
	pubOff := loginAs(t, off, "pub@trntest.dev", "testpass123")
	var offPost map[string]any
	doReqJSON(t, off, http.MethodPost, "/studio/content", pubOff, map[string]any{"type": "audio", "title": "Off"}, &offPost)
	var pres, conf map[string]any
	doReqJSON(t, off, http.MethodPost, "/media/presign", pubOff, map[string]any{
		"ownerType": "content", "ownerId": offPost["id"], "role": "audio", "contentType": "audio/mpeg", "size": 1000,
	}, &pres)
	doReqJSON(t, off, http.MethodPost, "/media/confirm", pubOff, map[string]any{
		"key": pres["key"], "ownerType": "content", "ownerId": offPost["id"], "role": "audio",
	}, &conf)
	var n int
	store.Pool.QueryRow(ctx, `SELECT count(*) FROM content_transcripts WHERE content_id=$1`, offPost["id"]).Scan(&n)
	if n != 0 {
		t.Errorf("queued with transcripts off: %d", n)
	}
	if code := doReqJSON(t, off, http.MethodPost, "/internal/transcripts/claim", testTranscriberToken, nil, nil); code != http.StatusNotFound {
		t.Errorf("claim with transcripts off: %d", code)
	}
	if code := doReqJSON(t, off, http.MethodPost, "/studio/content/"+offPost["id"].(string)+"/transcript/retry", pubOff, nil, nil); code != http.StatusServiceUnavailable {
		t.Errorf("retry with transcripts off: %d", code)
	}
}

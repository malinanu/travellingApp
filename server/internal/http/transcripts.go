package httpapi

// Audio guide transcripts. Uploading a post's audio queues one; the
// transcriber service (transcriber/, faster-whisper) claims it through the
// token-protected /internal/transcripts endpoints, downloads the audio
// through /media/raw, and posts the transcript back. Publishers and admins
// review and correct it in the Command Center; travellers see a finished
// one with the post.

import (
	"context"
	"crypto/subtle"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

const (
	// transcriptLease is how long a claimed job may take before another
	// transcriber may pick it up (the first must have crashed). Generous:
	// a long audiobook on a small CPU takes a while.
	transcriptLease = 3 * time.Hour
	// transcriptMaxAttempts caps tries per recording.
	transcriptMaxAttempts = 3
	// transcriptRetryDelay spaces out retries after a transient failure.
	transcriptRetryDelay = 10 * time.Minute

	maxTranscriptSegments = 20_000
	maxSegmentChars       = 2_000
)

// requireTranscriber lets only the transcriber service through. With no
// token configured the endpoints don't exist (transcripts are off).
func (s *Server) requireTranscriber(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.TranscriberToken == "" {
			writeErr(w, http.StatusNotFound, "The requested item could not be found.")
			return
		}
		got, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || subtle.ConstantTimeCompare([]byte(got), []byte(s.TranscriberToken)) != 1 {
			writeErr(w, http.StatusUnauthorized, "invalid transcriber token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// queueTranscript asks for a transcript of a post's new audio upload, when
// transcripts are on (best effort: the upload itself already succeeded).
func (s *Server) queueTranscript(ctx context.Context, contentID, audioPath string) {
	if s.TranscriberToken == "" || audioPath == "" {
		return
	}
	if err := s.Store.QueueTranscript(ctx, contentID, audioPath); err != nil {
		log.Printf("queue transcript content=%s: %v", contentID, err)
	}
}

// handleTranscriptClaim gives the transcriber the next recording, or 204.
func (s *Server) handleTranscriptClaim(w http.ResponseWriter, r *http.Request) {
	job, ok, err := s.Store.ClaimTranscript(r.Context(), transcriptLease, transcriptMaxAttempts)
	if err != nil {
		log.Printf("transcript claim: %v", err)
		storeErr(w, err)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	job.AudioURL = s.signMediaPath(job.AudioPath)
	writeJSON(w, http.StatusOK, job)
}

// handleTranscriptResult stores a finished transcript.
func (s *Server) handleTranscriptResult(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AudioPath   string                     `json:"audioPath"`
		Language    string                     `json:"language"`
		DurationSec float64                    `json:"durationSec"`
		Segments    []domain.TranscriptSegment `json:"segments"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	segments, msg := cleanSegments(req.Segments)
	if msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	lang := strings.ToLower(strings.TrimSpace(req.Language))
	if len(lang) > 8 {
		lang = ""
	}
	if req.DurationSec < 0 || math.IsNaN(req.DurationSec) || math.IsInf(req.DurationSec, 0) {
		req.DurationSec = 0
	}
	id := chi.URLParam(r, "id")
	stored, err := s.Store.CompleteTranscript(r.Context(), id, req.AudioPath, lang, req.DurationSec, segments)
	if err != nil {
		log.Printf("transcript result content=%s: %v", id, err)
		storeErr(w, err)
		return
	}
	if !stored {
		log.Printf("transcript result content=%s: ignored (audio replaced or job not claimed)", id)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"stored": stored})
}

// cleanSegments trims lines, drops empty ones and checks the timings, or
// returns why the result can't be stored.
func cleanSegments(in []domain.TranscriptSegment) ([]domain.TranscriptSegment, string) {
	if len(in) > maxTranscriptSegments {
		return nil, "too many segments"
	}
	out := make([]domain.TranscriptSegment, 0, len(in))
	last := 0.0
	for _, sg := range in {
		sg.Text = strings.TrimSpace(sg.Text)
		if sg.Text == "" {
			continue
		}
		if len([]rune(sg.Text)) > maxSegmentChars {
			return nil, "a segment is too long"
		}
		if math.IsNaN(sg.Start) || math.IsNaN(sg.End) || sg.Start < 0 || sg.End < sg.Start || sg.Start+0.5 < last {
			return nil, "segment times must be in order"
		}
		last = sg.Start
		out = append(out, sg)
	}
	return out, ""
}

// handleTranscriptFail records a failed attempt.
func (s *Server) handleTranscriptFail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AudioPath string `json:"audioPath"`
		Error     string `json:"error"`
		Retryable bool   `json:"retryable"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	msg := sanitizeText(req.Error, 500)
	if msg == "" {
		msg = "The transcriber couldn't process this recording."
	}
	id := chi.URLParam(r, "id")
	recorded, err := s.Store.FailTranscript(r.Context(), id, req.AudioPath, msg, req.Retryable,
		transcriptMaxAttempts, transcriptRetryDelay)
	if err != nil {
		storeErr(w, err)
		return
	}
	log.Printf("transcript failed content=%s retryable=%v recorded=%v: %s", id, req.Retryable, recorded, msg)
	writeJSON(w, http.StatusOK, map[string]bool{"recorded": recorded})
}

// studioResponse is a post as the Command Center editor gets it: signed
// media links plus its transcript.
func (s *Server) studioResponse(ctx context.Context, item domain.ContentItem) domain.ContentItem {
	item = s.signContent(item)
	if item.Type == domain.ContentAudio {
		t, err := s.Store.Transcript(ctx, item.ID)
		if err != nil {
			log.Printf("transcript content=%s: %v", item.ID, err)
		}
		item.Transcript = t
	}
	return item
}

// withPublicTranscript adds a finished transcript to a live post, without
// the review details.
func (s *Server) withPublicTranscript(ctx context.Context, item domain.ContentItem) domain.ContentItem {
	if item.Type != domain.ContentAudio {
		return item
	}
	t, err := s.Store.Transcript(ctx, item.ID)
	if err != nil {
		log.Printf("transcript content=%s: %v", item.ID, err)
	}
	if t != nil && t.Status == domain.TranscriptDone && len(t.Segments) > 0 {
		t.Status, t.Edited, t.Error = "", false, ""
		item.Transcript = t
	}
	return item
}

// studioEditableItem loads a post the caller may change (see studioItem
// and authorCanEdit), writing the error otherwise.
func (s *Server) studioEditableItem(w http.ResponseWriter, r *http.Request) (domain.ContentItem, bool) {
	item, c, ok := s.studioItem(w, r)
	if !ok {
		return item, false
	}
	if c.Role != domain.RoleAdmin && !authorCanEdit(item.Status) {
		writeErr(w, http.StatusConflict, "This post is waiting for review or live, so it can't be changed now.")
		return item, false
	}
	return item, true
}

// handleStudioTranscriptSave saves corrected transcript lines.
func (s *Server) handleStudioTranscriptSave(w http.ResponseWriter, r *http.Request) {
	item, ok := s.studioEditableItem(w, r)
	if !ok {
		return
	}
	var req struct {
		Texts []string `json:"texts"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	for i, t := range req.Texts {
		req.Texts[i] = strings.TrimSpace(t)
		if len([]rune(req.Texts[i])) > maxSegmentChars {
			writeErr(w, http.StatusBadRequest, "A line is too long.")
			return
		}
	}
	if _, err := s.Store.EditTranscript(r.Context(), item.ID, req.Texts); err != nil {
		if err == postgres.ErrTranscriptNotReady {
			writeErr(w, http.StatusConflict, "The transcript changed since you opened it. Reload and try again.")
			return
		}
		storeErr(w, err)
		return
	}
	if c := auth.FromContext(r.Context()); c.Role == domain.RoleAdmin {
		s.logAdminAction(r.Context(), "content.transcript_edit", "content", item.ID, map[string]any{"title": item.Title})
	}
	writeJSON(w, http.StatusOK, s.studioResponse(r.Context(), item))
}

// handleStudioTranscriptRetry makes a new transcript from the post's
// current audio (replacing any corrections).
func (s *Server) handleStudioTranscriptRetry(w http.ResponseWriter, r *http.Request) {
	item, ok := s.studioEditableItem(w, r)
	if !ok {
		return
	}
	if s.TranscriberToken == "" {
		writeErr(w, http.StatusServiceUnavailable, "Transcripts aren't set up on this server yet.")
		return
	}
	if item.Type != domain.ContentAudio || item.AudioURL == "" {
		writeErr(w, http.StatusBadRequest, "Upload the audio first.")
		return
	}
	if err := s.Store.QueueTranscript(r.Context(), item.ID, item.AudioURL); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.studioResponse(r.Context(), item))
}

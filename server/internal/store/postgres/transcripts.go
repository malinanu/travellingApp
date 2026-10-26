package postgres

// Audio guide transcripts (migration 050): a queue the transcriber service
// claims from, and the transcripts it sends back.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// QueueTranscript asks for a fresh transcript of a post's audio upload,
// replacing whatever transcript the post had.
func (s *Store) QueueTranscript(ctx context.Context, contentID, audioPath string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO content_transcripts (content_id, audio_path) VALUES ($1, $2)
		ON CONFLICT (content_id) DO UPDATE SET audio_path=EXCLUDED.audio_path, status='queued',
			attempts=0, error='', lease_until=NULL, language='', duration_sec=0, segments='[]',
			text='', edited=false, created_at=now(), updated_at=now()`, contentID, audioPath)
	return err
}

func (s *Store) DeleteTranscript(ctx context.Context, contentID string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM content_transcripts WHERE content_id=$1`, contentID)
	return err
}

// Transcript is a post's transcript, or nil when it has none.
func (s *Store) Transcript(ctx context.Context, contentID string) (*domain.ContentTranscript, error) {
	var t domain.ContentTranscript
	var segments []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT status, language, duration_sec, segments, edited, error, updated_at
		FROM content_transcripts WHERE content_id=$1`, contentID).
		Scan(&t.Status, &t.Language, &t.DurationSec, &segments, &t.Edited, &t.Error, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(segments, &t.Segments)
	if t.Segments == nil {
		t.Segments = []domain.TranscriptSegment{}
	}
	return &t, nil
}

func secs(d time.Duration) float64 { return d.Seconds() }

// ClaimTranscript hands the oldest waiting transcript to the transcriber for
// [lease]: a queued one past any retry delay, or one a crashed transcriber
// held past its lease. A post that has already taken [maxAttempts] tries
// that never finished (e.g. the transcriber is killed for memory each time)
// is marked failed instead of handed out again. ok is false when there's
// nothing to do.
func (s *Store) ClaimTranscript(ctx context.Context, lease time.Duration, maxAttempts int) (job domain.TranscriptJob, ok bool, err error) {
	if _, err = s.Pool.Exec(ctx, `
		UPDATE content_transcripts SET status='failed', lease_until=NULL, updated_at=now(),
			error='The transcriber stopped while working on this recording too many times.'
		WHERE status='working' AND lease_until <= now() AND attempts >= $1`, maxAttempts); err != nil {
		return job, false, err
	}
	err = s.Pool.QueryRow(ctx, `
		UPDATE content_transcripts SET status='working', attempts=attempts+1,
			lease_until=now() + make_interval(secs => $1), updated_at=now()
		WHERE content_id = (
			SELECT content_id FROM content_transcripts
			WHERE (status='queued' AND (lease_until IS NULL OR lease_until <= now()))
			   OR (status='working' AND lease_until <= now())
			ORDER BY created_at
			LIMIT 1 FOR UPDATE SKIP LOCKED)
		RETURNING content_id, audio_path, attempts`, secs(lease)).
		Scan(&job.ContentID, &job.AudioPath, &job.Attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return job, false, nil
	}
	return job, err == nil, err
}

// CompleteTranscript stores a finished transcript. It reports false (and
// stores nothing) when the post's audio was replaced or the job is no
// longer being worked on.
func (s *Store) CompleteTranscript(ctx context.Context, contentID, audioPath, language string, durationSec float64, segments []domain.TranscriptSegment) (bool, error) {
	raw, _ := json.Marshal(segments)
	tag, err := s.Pool.Exec(ctx, `
		UPDATE content_transcripts SET status='done', language=$3, duration_sec=$4, segments=$5,
			text=$6, error='', lease_until=NULL, edited=false, updated_at=now()
		WHERE content_id=$1 AND audio_path=$2 AND status='working'`,
		contentID, audioPath, language, durationSec, raw, joinSegments(segments))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// FailTranscript records a failed attempt: a retryable one is queued again
// after [backoff] while attempts remain, anything else is marked failed.
// It reports false when the job wasn't being worked on (stale report).
func (s *Store) FailTranscript(ctx context.Context, contentID, audioPath, message string, retryable bool, maxAttempts int, backoff time.Duration) (bool, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE content_transcripts SET
			status = CASE WHEN $3 AND attempts < $4 THEN 'queued' ELSE 'failed' END,
			lease_until = CASE WHEN $3 AND attempts < $4 THEN now() + make_interval(secs => $5) ELSE NULL END,
			error=$6, updated_at=now()
		WHERE content_id=$1 AND audio_path=$2 AND status='working'`,
		contentID, audioPath, retryable, maxAttempts, secs(backoff), message)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// ErrTranscriptNotReady means there's no finished transcript to edit.
var ErrTranscriptNotReady = errors.New("transcript isn't ready")

// EditTranscript replaces the text of a finished transcript's lines, one
// string per line in order; the timings stay.
func (s *Store) EditTranscript(ctx context.Context, contentID string, texts []string) (*domain.ContentTranscript, error) {
	t, err := s.Transcript(ctx, contentID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.Status != domain.TranscriptDone || len(texts) != len(t.Segments) {
		return nil, ErrTranscriptNotReady
	}
	for i := range t.Segments {
		t.Segments[i].Text = texts[i]
	}
	raw, _ := json.Marshal(t.Segments)
	if _, err := s.Pool.Exec(ctx, `
		UPDATE content_transcripts SET segments=$2, text=$3, edited=true, updated_at=now()
		WHERE content_id=$1 AND status='done'`, contentID, raw, joinSegments(t.Segments)); err != nil {
		return nil, err
	}
	return s.Transcript(ctx, contentID)
}

func joinSegments(segments []domain.TranscriptSegment) string {
	parts := make([]string, 0, len(segments))
	for _, sg := range segments {
		if t := strings.TrimSpace(sg.Text); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}

package httpapi

// The thumbnail worker generates image derivatives (a ~320px grid/list
// thumbnail and a tiny ~24px inline blur placeholder â€” see
// internal/media.GenerateThumbnails) off the upload-confirm request path.
// handleMediaConfirm just marks a freshly-uploaded image's thumb_status
// "pending"; this ticker-driven sweep does the actual fetch-from-R2,
// CPU-bound decode/resize, and re-upload, following the same shape as
// weather_scheduler.go/holiday_scheduler.go/br_cert_scheduler.go (no
// job-queue abstraction in this codebase â€” state lives in Postgres, not
// in-memory, so it survives a restart mid-sweep).

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"log"
	"path"
	"strings"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/media"
)

// errThumbnailDecode marks a permanent failure â€” the source image itself
// couldn't be decoded (corrupt/unsupported upload), so retrying it on a
// later sweep would just fail the same way forever. Any other failure
// (fetching the original from R2, uploading a derivative, the DB write) is
// transient infrastructure trouble, not a property of the image â€” those
// are left "pending" so the next sweep retries them instead of getting
// permanently stuck on what might just be a momentary R2 blip.
type errThumbnailDecode struct{ err error }

func (e *errThumbnailDecode) Error() string { return e.err.Error() }
func (e *errThumbnailDecode) Unwrap() error { return e.err }

// thumbnailWorkerInterval is much snappier than the other schedulers' â€”
// unlike weather/holiday/BR-cert, there's no external API to pace against,
// just local CPU (a decode+resize is well under a second), so there's no
// reason to make a freshly-uploaded photo wait any longer than this for
// its thumbnail to appear.
const thumbnailWorkerInterval = 10 * time.Second

// thumbnailBatchSize caps how much one tick takes on; a bigger backlog just
// spans more ticks rather than blocking one tick indefinitely.
const thumbnailBatchSize = 20

// thumbnailCallSpacing paces a sweep's decode/resize work â€” unlike the
// other schedulers this isn't pacing against an external rate limit, just
// against pegging this container's CPU with one tight back-to-back burst.
const thumbnailCallSpacing = 150 * time.Millisecond

// StartThumbnailWorker blocks until ctx is cancelled; call with `go`.
func (s *Server) StartThumbnailWorker(ctx context.Context) {
	s.runThumbnailSweep(ctx)
	ticker := time.NewTicker(thumbnailWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runThumbnailSweep(ctx)
		}
	}
}

func (s *Server) runThumbnailSweep(ctx context.Context) {
	if s.Media == nil {
		return // uploads unconfigured
	}
	pending, err := s.Store.MediaPendingThumbnails(ctx, thumbnailBatchSize)
	if err != nil {
		log.Printf("thumbnail worker: list pending: %v", err)
		return
	}

	done, failed, retrying := 0, 0, 0
	for i, m := range pending {
		if i > 0 {
			time.Sleep(thumbnailCallSpacing)
		}
		err := s.generateAndStoreThumbnails(ctx, m)
		if err == nil {
			done++
			continue
		}
		log.Printf("thumbnail worker: %s: %v", m.ID, err)
		var decodeErr *errThumbnailDecode
		if errors.As(err, &decodeErr) {
			if serr := s.Store.SetMediaThumbnailError(ctx, m.ID); serr != nil {
				log.Printf("thumbnail worker: record error for %s: %v", m.ID, serr)
			}
			failed++
		} else {
			// Transient (fetch/upload/DB) failure â€” left "pending", picked
			// up again next tick.
			retrying++
		}
	}
	log.Printf("thumbnail worker: %d done, %d failed, %d retrying (of %d pending)", done, failed, retrying, len(pending))
}

// generateAndStoreThumbnails fetches m's original bytes, generates both
// derivatives, uploads the 320px thumb to R2 (same key convention as the
// old synchronous path), and saves the tiny blur placeholder inline.
func (s *Server) generateAndStoreThumbnails(ctx context.Context, m domain.Media) error {
	body, _, err := s.Media.Get(ctx, m.ObjectKey)
	if err != nil {
		return err
	}
	defer body.Close()

	thumbs, err := media.GenerateThumbnails(body)
	if err != nil {
		return &errThumbnailDecode{err}
	}

	thumbKey := strings.TrimSuffix(m.ObjectKey, path.Ext(m.ObjectKey)) + "_thumb.jpg"
	if err := s.Media.Put(ctx, thumbKey, "image/jpeg", bytes.NewReader(thumbs.Thumb)); err != nil {
		return err
	}

	micro := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumbs.Micro)
	return s.Store.SetMediaThumbnails(ctx, m.ID, thumbKey, micro)
}

package httpapi

// Inspiration Board extraction worker. A traveller saves a reel/post link
// (inspirations.go); this worker has it read by s.Reels (internal/reelextract:
// yt-dlp + Gemini, which can move to its own server), finds each place on
// Google (internal/googleplaces, cached in geo_places), and tells everyone
// who saved the link that it's ready.
//
// Jobs live in link_extractions, so they survive restarts, and one result
// is shared by every traveller who saves the same link. Same ticker-sweep
// shape as br_cert_scheduler.go.

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/googleplaces"
	"github.com/spotto/server/internal/reelextract"
	"github.com/spotto/server/internal/store/postgres"
)

const (
	inspirationWorkerInterval = 5 * time.Second
	inspirationJobTimeout     = 5 * time.Minute
	inspirationMaxAttempts    = 3
)

// StartInspirationWorker blocks until ctx is cancelled; call with `go`.
func (s *Server) StartInspirationWorker(ctx context.Context) {
	if n, err := s.Store.ResetStuckExtractions(ctx, 10*time.Minute); err != nil {
		log.Printf("inspiration worker: reset stuck jobs: %v", err)
	} else if n > 0 {
		log.Printf("inspiration worker: requeued %d interrupted jobs", n)
	}
	ticker := time.NewTicker(inspirationWorkerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runInspirationSweep(ctx)
		}
	}
}

func (s *Server) runInspirationSweep(ctx context.Context) {
	for i := 0; i < 3; i++ { // a few per tick so a burst drains quickly
		job, err := s.Store.ClaimPendingExtraction(ctx)
		if errors.Is(err, postgres.ErrNotFound) {
			return
		}
		if err != nil {
			log.Printf("inspiration worker: claim job: %v", err)
			return
		}
		s.processExtraction(ctx, job)
	}
}

func (s *Server) processExtraction(parent context.Context, job domain.LinkExtraction) {
	ctx, cancel := context.WithTimeout(parent, inspirationJobTimeout)
	defer cancel()
	start := time.Now()

	res, err := s.Reels.Extract(ctx, job.CanonicalURL, job.Platform)
	job.Caption, job.Author, job.ThumbnailURL, job.Model = res.Caption, res.Author, res.ThumbnailURL, res.Model
	job.VideoBytes, job.VideoSecs = res.VideoBytes, res.VideoSecs
	if err != nil {
		job.ProcessingMs = int(time.Since(start).Milliseconds())
		log.Printf("inspiration worker: %s %s: %v", job.Platform, job.CanonicalURL, err)
		var xe *reelextract.Error
		if !errors.As(err, &xe) {
			xe = &reelextract.Error{Message: reelextract.MsgAnalysisFail, Retryable: true}
		}
		if xe.Retryable {
			s.retryOrFail(ctx, job, xe.Message)
		} else {
			s.finishExtraction(ctx, job, domain.ExtractionFailed, xe.Message, nil)
		}
		return
	}

	results := make([]domain.ExtractedPlace, 0, len(res.Places))
	for _, vp := range res.Places {
		ep := domain.ExtractedPlace{
			Name: vp.Name, Kind: vp.Kind, Area: vp.Area, City: vp.City, Country: vp.Country,
			Evidence: vp.Evidence, AtSec: vp.AtSec, Confidence: vp.Confidence,
		}
		s.resolveExtractedPlace(ctx, &ep, res.RegionHint)
		results = append(results, ep)
	}
	job.ProcessingMs = int(time.Since(start).Milliseconds())

	status := domain.ExtractionDone
	if res.PartialReason != "" {
		status = domain.ExtractionPartial
	}
	// One line per job: what it cost, for deciding whether extraction
	// needs its own server.
	log.Printf("inspiration worker: done %s %s places=%d video=%.1fMB/%.0fs took=%dms",
		job.Platform, job.CanonicalURL, len(results), float64(job.VideoBytes)/(1<<20), job.VideoSecs, job.ProcessingMs)
	s.finishExtraction(ctx, job, status, res.PartialReason, results)
}

// retryOrFail schedules another attempt with backoff, or gives up after
// inspirationMaxAttempts.
func (s *Server) retryOrFail(ctx context.Context, job domain.LinkExtraction, msg string) {
	if job.Attempts < inspirationMaxAttempts {
		delay := time.Duration(job.Attempts) * 2 * time.Minute
		if err := s.Store.RetryExtractionLater(context.WithoutCancel(ctx), job.CanonicalURL, msg, delay); err != nil {
			log.Printf("inspiration worker: requeue %s: %v", job.CanonicalURL, err)
		}
		return
	}
	s.finishExtraction(ctx, job, domain.ExtractionFailed, msg, nil)
}

func (s *Server) finishExtraction(ctx context.Context, job domain.LinkExtraction, status, msg string, results []domain.ExtractedPlace) {
	job.Status, job.Error, job.Results = status, msg, results
	ctx = context.WithoutCancel(ctx) // record the outcome even if the job ran out of time
	if err := s.Store.FinishExtraction(ctx, job); err != nil {
		log.Printf("inspiration worker: save result %s: %v", job.CanonicalURL, err)
		return
	}
	s.notifyExtractionReady(ctx, job)
}

// notifyExtractionReady pushes the outcome to everyone who saved the link.
func (s *Server) notifyExtractionReady(ctx context.Context, job domain.LinkExtraction) {
	links, err := s.Store.LinksForExtraction(ctx, job.CanonicalURL)
	if err != nil {
		log.Printf("inspiration worker: list links %s: %v", job.CanonicalURL, err)
		return
	}
	title, body := extractionPushText(job)
	for _, l := range links {
		s.SendUserPushNotification(ctx, l.UserID, title, body,
			map[string]string{"type": "inspiration_ready", "linkId": l.ID})
	}
}

func extractionPushText(job domain.LinkExtraction) (string, string) {
	from := ""
	if job.Author != "" {
		from = " from " + job.Author
	}
	n := len(job.Results)
	switch {
	case job.Status == domain.ExtractionFailed:
		return "We couldn't read that link", job.Error
	case n == 0:
		return "No places found", "We watched the reel" + from + " but couldn't spot any named places."
	case n == 1:
		return "1 place found", "We found a place in the reel" + from + ". Tap to save it."
	default:
		return fmt.Sprintf("%d places found", n), fmt.Sprintf("We found %d places in the reel%s. Tap to save the ones you like.", n, from)
	}
}

// resolveExtractedPlace finds the place on Google (cached in geo_places) and,
// when it is also in the Spotto catalog, links that too. Failures leave the
// place unresolved; they never fail the whole job.
func (s *Server) resolveExtractedPlace(ctx context.Context, ep *domain.ExtractedPlace, regionHint string) {
	if s.Places == nil {
		return
	}
	query := placeQuery(*ep, regionHint)
	key := strings.ToLower(query)

	geoID, cached, err := s.Store.GeoPlaceQuery(ctx, key)
	if err != nil {
		log.Printf("inspiration worker: query cache: %v", err)
		return
	}
	if !cached {
		geoID, err = s.searchGeoPlace(ctx, query, *ep)
		if err != nil {
			log.Printf("inspiration worker: places search %q: %v", query, err)
			return // not cached: a transient error shouldn't stick
		}
		if err := s.Store.SaveGeoPlaceQuery(ctx, key, geoID); err != nil {
			log.Printf("inspiration worker: save query cache: %v", err)
		}
	}
	if geoID == "" {
		return
	}
	ep.GeoPlaceID = geoID
	gp, err := s.Store.GeoPlaceByID(ctx, geoID)
	if err != nil {
		return
	}
	if ep.Kind == "" {
		ep.Kind = gp.Kind
	}
	ep.PlaceID = s.matchCatalogPlace(ctx, gp.Name, gp.Lat, gp.Lng)
}

// searchGeoPlace asks Google and stores the hit. It returns "" for no
// confident match: Google always returns *something* for a text query, so a
// result whose name has nothing in common with what the reel said is a miss.
func (s *Server) searchGeoPlace(ctx context.Context, query string, ep domain.ExtractedPlace) (string, error) {
	p, err := s.Places.SearchText(ctx, query)
	if errors.Is(err, googleplaces.ErrNoResult) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if nameSimilarity(ep.Name, p.Name) < 0.34 {
		return "", nil
	}
	kind := googleplaces.Kind(p.PrimaryType, p.Types)
	if kind == "" {
		kind = ep.Kind
	}
	return s.Store.UpsertGeoPlace(ctx, domain.GeoPlace{
		ID: newID("gp"), GooglePlaceID: p.ID, Name: p.Name, Lat: p.Lat, Lng: p.Lng,
		Address: p.Address, City: p.City, CountryCode: p.CountryCode, Kind: kind,
		Rating: p.Rating, RatingCount: p.RatingCount, PhotoName: p.PhotoName,
		MapsURL: p.MapsURL, Types: p.Types,
	})
}

// matchCatalogPlace returns the Spotto catalog place at the same spot with a
// similar name, or "". Both checks are required: a nearby place with another
// name is a different business, and a same-named place elsewhere is too.
func (s *Server) matchCatalogPlace(ctx context.Context, name string, lat, lng float64) string {
	const box = 0.0015 // ~165 m
	near, err := s.Store.CatalogPlacesInBox(ctx, lat-box, lat+box, lng-box, lng+box)
	if err != nil {
		log.Printf("inspiration worker: catalog lookup: %v", err)
		return ""
	}
	best, bestScore := "", 0.5
	for _, p := range near {
		if geo.HaversineKm(lat, lng, p.Lat, p.Lng) > 0.15 {
			continue
		}
		if sc := nameSimilarity(name, p.Name); sc >= bestScore {
			best, bestScore = p.ID, sc
		}
	}
	return best
}

// placeQuery is the Text Search query for a place: its name plus where the
// reel says it is, falling back to the reel's overall region.
func placeQuery(ep domain.ExtractedPlace, regionHint string) string {
	parts := []string{ep.Name}
	if ep.Area != "" && !strings.EqualFold(ep.Area, ep.Name) {
		parts = append(parts, ep.Area)
	}
	if ep.City != "" {
		parts = append(parts, ep.City)
	}
	if ep.Country != "" {
		parts = append(parts, ep.Country)
	}
	if ep.City == "" && ep.Country == "" && regionHint != "" {
		parts = append(parts, regionHint)
	}
	return strings.Join(parts, ", ")
}

// Words that don't tell two places apart.
var nameStopWords = map[string]bool{
	"the": true, "a": true, "an": true, "and": true, "of": true, "at": true, "by": true, "&": true,
	"restaurant": true, "cafe": true, "café": true, "coffee": true, "bar": true, "hotel": true,
	"resort": true, "house": true, "shop": true, "kitchen": true, "spa": true, "club": true,
}

func nameTokens(s string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if !nameStopWords[f] {
			out[f] = true
		}
	}
	return out
}

// nameSimilarity scores two place names 0..1: 1 when one name contains the
// other (ignoring case and punctuation), else the share of distinctive words
// they have in common.
func nameSimilarity(a, b string) float64 {
	na, nb := squash(a), squash(b)
	if na == "" || nb == "" {
		return 0
	}
	if strings.Contains(na, nb) || strings.Contains(nb, na) {
		return 1
	}
	ta, tb := nameTokens(a), nameTokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	common := 0
	for t := range ta {
		if tb[t] {
			common++
		}
	}
	smaller := len(ta)
	if len(tb) < smaller {
		smaller = len(tb)
	}
	return float64(common) / float64(smaller)
}

func squash(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

package httpapi

// Inspiration Board extraction worker. A traveller saves a reel/post link
// (inspirations.go); this worker reads it (inspiration_sources.go), has
// Gemini watch the video and read the caption (internal/gemini/video.go), finds each
// place on Google (internal/googleplaces, cached in geo_places), and tells
// everyone who saved the link that it's ready.
//
// Jobs live in link_extractions, so they survive restarts, and one result
// is shared by every traveller who saves the same link. Same ticker-sweep
// shape as br_cert_scheduler.go.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/gemini"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/googleplaces"
	"github.com/spotto/server/internal/store/postgres"
)

const (
	inspirationWorkerInterval = 5 * time.Second
	inspirationJobTimeout     = 5 * time.Minute
	inspirationMaxAttempts    = 3
	// Reels longer than this are read from the caption only: a long video
	// costs far more to analyse and rarely is a "N places to visit" reel.
	inspirationMaxVideoSecs = 180
	inspirationMaxVideoMB   = 45
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

// Messages travellers see on a link card.
const (
	msgNotConfigured = "Reading reels isn't set up on this server yet."
	msgCantOpen      = "We couldn't open this link. It may be private, deleted, or blocked from downloading."
	msgNothingToRead = "We couldn't watch this video and it has no caption to read."
	msgCaptionOnly   = "We couldn't watch the video, so these places come from the caption only."
	msgAnalysisFail  = "We couldn't analyse this video right now. Tap Retry to try again."
)

func (s *Server) processExtraction(parent context.Context, job domain.LinkExtraction) {
	ctx, cancel := context.WithTimeout(parent, inspirationJobTimeout)
	defer cancel()

	if s.GeminiAPIKey == "" {
		s.finishExtraction(ctx, job, domain.ExtractionFailed, msgNotConfigured, nil)
		return
	}
	meta, err := s.linkMetadata(ctx, job)
	if errors.Is(err, errNoYtdlp) {
		log.Printf("inspiration worker: yt-dlp not installed")
		s.finishExtraction(ctx, job, domain.ExtractionFailed, msgNotConfigured, nil)
		return
	}
	if err != nil {
		log.Printf("inspiration worker: metadata %s: %v", job.CanonicalURL, err)
		s.retryOrFail(ctx, job, msgCantOpen)
		return
	}
	job.Caption = meta.caption()
	job.Author = meta.author()
	job.ThumbnailURL = meta.Thumbnail

	var video *gemini.Video
	if meta.Duration <= inspirationMaxVideoSecs {
		video = s.linkVideo(ctx, job)
	}
	if video == nil && strings.TrimSpace(job.Caption) == "" {
		s.finishExtraction(ctx, job, domain.ExtractionFailed, msgNothingToRead, nil)
		return
	}

	model := s.GeminiVideoModel
	if model == "" {
		model = s.Gemini.ChatModel
	}
	found, err := s.aiExtractVideoPlaces(ctx, model, video, job.Caption)
	if err != nil {
		log.Printf("inspiration worker: gemini %s: %v", job.CanonicalURL, err)
		s.retryOrFail(ctx, job, msgAnalysisFail)
		return
	}
	job.Model = model

	results := make([]domain.ExtractedPlace, 0, len(found.Places))
	for _, vp := range found.Places {
		ep := domain.ExtractedPlace{
			Name: vp.Name, Kind: vp.Kind, Area: vp.Area, City: vp.City, Country: vp.Country,
			Evidence: vp.Evidence, AtSec: vp.AtSec, Confidence: vp.Confidence,
		}
		s.resolveExtractedPlace(ctx, &ep, found.RegionHint)
		results = append(results, ep)
	}

	status, msg := domain.ExtractionDone, ""
	if video == nil {
		status, msg = domain.ExtractionPartial, msgCaptionOnly
	}
	s.finishExtraction(ctx, job, status, msg, results)
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

func (s *Server) aiExtractVideoPlaces(ctx context.Context, model string, video *gemini.Video, caption string) (gemini.VideoPlaces, error) {
	start := time.Now()
	out, usage, err := s.Gemini.ExtractPlacesFromVideo(ctx, model, video, caption)
	kind := "video"
	if video == nil {
		kind = "chat"
	}
	s.recordAIUsage(ctx, aiFeatureInspiration, kind, model, 1, usage, start, err)
	return out, err
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

// ---- links ----

var (
	igPathRe      = regexp.MustCompile(`^/(?:[\w.]+/)?(reels?|p|tv)/([\w-]+)`)
	tiktokVideoRe = regexp.MustCompile(`^/(@[\w.-]+)/(?:video|photo)/(\d+)`)
	ytShortsRe    = regexp.MustCompile(`^/(?:shorts|live)/([\w-]{6,})`)
)

var errUnsupportedLink = errors.New("unsupported link")

// canonicalLink turns a shared link into one stable URL per post (dropping
// tracking parameters and share-sheet variations) and names its platform.
// Only these platforms are accepted, which also keeps yt-dlp from being
// pointed at arbitrary hosts.
func canonicalLink(raw string) (canonical, platform string, err error) {
	raw = strings.TrimSpace(raw)
	if raw != "" && !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", "", errUnsupportedLink
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	switch {
	case host == "instagram.com":
		m := igPathRe.FindStringSubmatch(u.Path)
		if m == nil {
			return "", "", errUnsupportedLink
		}
		kind := m[1]
		if kind == "reels" {
			kind = "reel"
		}
		return "https://www.instagram.com/" + kind + "/" + m[2] + "/", "instagram", nil
	case host == "tiktok.com":
		m := tiktokVideoRe.FindStringSubmatch(u.Path)
		if m == nil {
			return "", "", errUnsupportedLink
		}
		return "https://www.tiktok.com/" + m[1] + "/video/" + m[2], "tiktok", nil
	case host == "vm.tiktok.com" || host == "vt.tiktok.com":
		code := strings.Trim(u.Path, "/")
		if code == "" || strings.Contains(code, "/") {
			return "", "", errUnsupportedLink
		}
		return "https://" + host + "/" + code + "/", "tiktok", nil
	case host == "youtube.com" || host == "music.youtube.com":
		if m := ytShortsRe.FindStringSubmatch(u.Path); m != nil {
			return "https://www.youtube.com/shorts/" + m[1], "youtube", nil
		}
		if v := u.Query().Get("v"); u.Path == "/watch" && v != "" {
			return "https://www.youtube.com/watch?v=" + url.QueryEscape(v), "youtube", nil
		}
	case host == "youtu.be":
		if id := strings.Trim(u.Path, "/"); id != "" && !strings.Contains(id, "/") {
			return "https://www.youtube.com/watch?v=" + url.QueryEscape(id), "youtube", nil
		}
	}
	return "", "", errUnsupportedLink
}

// ---- yt-dlp ----

type ytdlpMeta struct {
	Description string  `json:"description"`
	Title       string  `json:"title"`
	Uploader    string  `json:"uploader"`
	UploaderID  string  `json:"uploader_id"`
	Channel     string  `json:"channel"`
	Duration    float64 `json:"duration"`
	Thumbnail   string  `json:"thumbnail"`
}

func (m ytdlpMeta) caption() string {
	if m.Description != "" {
		return m.Description
	}
	return m.Title
}

func (m ytdlpMeta) author() string {
	switch {
	case m.Channel != "":
		return m.Channel
	case m.Uploader != "":
		return m.Uploader
	default:
		return m.UploaderID
	}
}

func (s *Server) ytdlpArgs(extra ...string) []string {
	args := []string{"--no-warnings", "--no-playlist", "--socket-timeout", "20"}
	if s.YtdlpCookiesFile != "" {
		args = append(args, "--cookies", s.YtdlpCookiesFile)
	}
	return append(args, extra...)
}

func (s *Server) ytdlpMetadata(ctx context.Context, link string) (ytdlpMeta, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "yt-dlp", s.ytdlpArgs("--dump-json", "--skip-download", "--", link)...).Output()
	if err != nil {
		return ytdlpMeta{}, fmt.Errorf("yt-dlp metadata: %w%s", err, stderrOf(err))
	}
	var m ytdlpMeta
	if err := json.Unmarshal(out, &m); err != nil {
		return ytdlpMeta{}, fmt.Errorf("yt-dlp metadata: parse: %w", err)
	}
	return m, nil
}

// ytdlpDownload fetches the video as a single file (no ffmpeg merge needed).
func (s *Server) ytdlpDownload(ctx context.Context, link string) (*gemini.Video, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	dir, err := os.MkdirTemp("", "spotto-reel-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	limit := fmt.Sprintf("%dM", inspirationMaxVideoMB)
	format := fmt.Sprintf("best[ext=mp4][filesize<%[1]s]/best[ext=mp4][filesize_approx<%[1]s]/best[ext=mp4]/best", limit)
	args := s.ytdlpArgs("-f", format, "--max-filesize", limit, "-o", filepath.Join(dir, "video.%(ext)s"), "--", link)
	if out, err := exec.CommandContext(ctx, "yt-dlp", args...).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("yt-dlp download: %w: %s", err, lastLine(string(out)))
	}
	files, _ := filepath.Glob(filepath.Join(dir, "video.*"))
	if len(files) == 0 {
		return nil, errors.New("yt-dlp download: no file (over the size limit?)")
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		return nil, err
	}
	mime := "video/mp4"
	switch strings.ToLower(filepath.Ext(files[0])) {
	case ".webm":
		mime = "video/webm"
	case ".mov":
		mime = "video/quicktime"
	}
	return &gemini.Video{Data: data, MimeType: mime}, nil
}

func stderrOf(err error) string {
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		return ": " + lastLine(string(ee.Stderr))
	}
	return ""
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

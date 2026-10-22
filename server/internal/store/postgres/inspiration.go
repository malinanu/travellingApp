package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spotto/server/internal/domain"
)

// Inspiration Board storage: cached Google places, shared link extractions,
// each traveller's saved links and "save for later" places
// (migrations/039_inspiration.sql).

const geoPlaceCols = `id, google_place_id, name, lat, lng, address, city, country_code, kind,
	rating, rating_count, photo_name, maps_url, types, fetched_at`

func scanGeoPlace(row pgx.Row) (domain.GeoPlace, error) {
	var p domain.GeoPlace
	var types []byte
	err := row.Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Lat, &p.Lng, &p.Address, &p.City, &p.CountryCode,
		&p.Kind, &p.Rating, &p.RatingCount, &p.PhotoName, &p.MapsURL, &types, &p.FetchedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	_ = json.Unmarshal(types, &p.Types)
	if p.Types == nil {
		p.Types = []string{}
	}
	p.HasPhoto = p.PhotoName != ""
	return p, nil
}

func (s *Store) GeoPlaceByID(ctx context.Context, id string) (domain.GeoPlace, error) {
	return scanGeoPlace(s.Pool.QueryRow(ctx, `SELECT `+geoPlaceCols+` FROM geo_places WHERE id=$1`, id))
}

// GeoPlacesByIDs returns the geo places that exist among ids, keyed by id.
func (s *Store) GeoPlacesByIDs(ctx context.Context, ids []string) (map[string]domain.GeoPlace, error) {
	out := map[string]domain.GeoPlace{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+geoPlaceCols+` FROM geo_places WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		p, err := scanGeoPlace(rows)
		if err != nil {
			return nil, err
		}
		out[p.ID] = p
	}
	return out, rows.Err()
}

// UpsertGeoPlace stores a Google place, keyed by its Google id, and returns
// the stored row's id (the existing one when the place was already known).
func (s *Store) UpsertGeoPlace(ctx context.Context, p domain.GeoPlace) (string, error) {
	types, _ := json.Marshal(p.Types)
	if p.Types == nil {
		types = []byte("[]")
	}
	var id string
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO geo_places (id, google_place_id, name, lat, lng, address, city, country_code, kind,
			rating, rating_count, photo_name, maps_url, types, fetched_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,now())
		ON CONFLICT (google_place_id) DO UPDATE SET
			name=EXCLUDED.name, lat=EXCLUDED.lat, lng=EXCLUDED.lng, address=EXCLUDED.address,
			city=EXCLUDED.city, country_code=EXCLUDED.country_code, kind=EXCLUDED.kind,
			rating=EXCLUDED.rating, rating_count=EXCLUDED.rating_count, photo_name=EXCLUDED.photo_name,
			maps_url=EXCLUDED.maps_url, types=EXCLUDED.types, fetched_at=now()
		RETURNING id`,
		p.ID, p.GooglePlaceID, p.Name, p.Lat, p.Lng, p.Address, p.City, p.CountryCode, p.Kind,
		p.Rating, p.RatingCount, p.PhotoName, p.MapsURL, types).Scan(&id)
	return id, err
}

// GeoPlaceQuery returns the cached result of a text search: found reports
// whether the query was seen before, geoPlaceID is "" for a cached miss.
func (s *Store) GeoPlaceQuery(ctx context.Context, key string) (geoPlaceID string, found bool, err error) {
	var id *string
	err = s.Pool.QueryRow(ctx, `SELECT geo_place_id FROM geo_place_queries WHERE query_key=$1`, key).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	if id != nil {
		geoPlaceID = *id
	}
	return geoPlaceID, true, nil
}

// SaveGeoPlaceQuery caches a text search result; geoPlaceID "" records a miss.
func (s *Store) SaveGeoPlaceQuery(ctx context.Context, key, geoPlaceID string) error {
	var id *string
	if geoPlaceID != "" {
		id = &geoPlaceID
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO geo_place_queries (query_key, geo_place_id) VALUES ($1,$2)
		ON CONFLICT (query_key) DO UPDATE SET geo_place_id=EXCLUDED.geo_place_id, created_at=now()`, key, id)
	return err
}

// CatalogPlace is the location of a Spotto catalog place.
type CatalogPlace struct {
	ID   string
	Name string
	Lat  float64
	Lng  float64
}

// CatalogPlacesInBox returns catalog places inside a lat/lng box. Places keep
// their coordinates in the data JSON, so callers refine by real distance.
func (s *Store) CatalogPlacesInBox(ctx context.Context, minLat, maxLat, minLng, maxLng float64) ([]CatalogPlace, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, (data->>'lat')::float8, (data->>'lng')::float8 FROM places
		WHERE jsonb_typeof(data->'lat') = 'number' AND jsonb_typeof(data->'lng') = 'number'
		  AND (data->>'lat')::float8 BETWEEN $1 AND $2
		  AND (data->>'lng')::float8 BETWEEN $3 AND $4`, minLat, maxLat, minLng, maxLng)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CatalogPlace
	for rows.Next() {
		var p CatalogPlace
		if err := rows.Scan(&p.ID, &p.Name, &p.Lat, &p.Lng); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

const linkExtractionCols = `canonical_url, platform, status, error, caption, author, thumbnail_url,
	results, model, attempts, updated_at`

func scanLinkExtraction(row pgx.Row) (domain.LinkExtraction, error) {
	var e domain.LinkExtraction
	var results []byte
	err := row.Scan(&e.CanonicalURL, &e.Platform, &e.Status, &e.Error, &e.Caption, &e.Author,
		&e.ThumbnailURL, &results, &e.Model, &e.Attempts, &e.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, ErrNotFound
	}
	if err != nil {
		return e, err
	}
	_ = json.Unmarshal(results, &e.Results)
	if e.Results == nil {
		e.Results = []domain.ExtractedPlace{}
	}
	return e, nil
}

// EnsureLinkExtraction creates a pending extraction for a link unless one exists.
func (s *Store) EnsureLinkExtraction(ctx context.Context, canonicalURL, platform string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO link_extractions (canonical_url, platform) VALUES ($1,$2)
		ON CONFLICT (canonical_url) DO NOTHING`, canonicalURL, platform)
	return err
}

func (s *Store) LinkExtraction(ctx context.Context, canonicalURL string) (domain.LinkExtraction, error) {
	return scanLinkExtraction(s.Pool.QueryRow(ctx,
		`SELECT `+linkExtractionCols+` FROM link_extractions WHERE canonical_url=$1`, canonicalURL))
}

func (s *Store) LinkExtractionsByURLs(ctx context.Context, urls []string) (map[string]domain.LinkExtraction, error) {
	out := map[string]domain.LinkExtraction{}
	if len(urls) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx,
		`SELECT `+linkExtractionCols+` FROM link_extractions WHERE canonical_url = ANY($1)`, urls)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		e, err := scanLinkExtraction(rows)
		if err != nil {
			return nil, err
		}
		out[e.CanonicalURL] = e
	}
	return out, rows.Err()
}

// ClaimPendingExtraction marks the oldest due pending extraction as
// processing and returns it; ErrNotFound when there is nothing to do.
// SKIP LOCKED keeps two API instances from taking the same job.
func (s *Store) ClaimPendingExtraction(ctx context.Context) (domain.LinkExtraction, error) {
	return scanLinkExtraction(s.Pool.QueryRow(ctx, `
		UPDATE link_extractions SET status='processing', attempts=attempts+1, updated_at=now()
		WHERE canonical_url = (
			SELECT canonical_url FROM link_extractions
			WHERE status='pending' AND next_attempt_at <= now()
			ORDER BY next_attempt_at LIMIT 1 FOR UPDATE SKIP LOCKED)
		RETURNING `+linkExtractionCols))
}

// ResetStuckExtractions puts jobs left in processing by a crash or deploy
// back in the queue.
func (s *Store) ResetStuckExtractions(ctx context.Context, olderThan time.Duration) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE link_extractions SET status='pending', next_attempt_at=now()
		WHERE status='processing' AND updated_at < now() - make_interval(secs => $1)`, olderThan.Seconds())
	return tag.RowsAffected(), err
}

// FinishExtraction stores a finished job's outcome.
func (s *Store) FinishExtraction(ctx context.Context, e domain.LinkExtraction) error {
	results, _ := json.Marshal(e.Results)
	if e.Results == nil {
		results = []byte("[]")
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE link_extractions SET status=$2, error=$3, caption=$4, author=$5, thumbnail_url=$6,
			results=$7, model=$8, video_bytes=$9, video_secs=$10, processing_ms=$11, updated_at=now()
		WHERE canonical_url=$1`,
		e.CanonicalURL, e.Status, e.Error, e.Caption, e.Author, e.ThumbnailURL, results, e.Model,
		e.VideoBytes, e.VideoSecs, e.ProcessingMs)
	return err
}

// RetryExtractionLater puts a failed attempt back in the queue after delay.
func (s *Store) RetryExtractionLater(ctx context.Context, canonicalURL, errMsg string, delay time.Duration) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE link_extractions SET status='pending', error=$2,
			next_attempt_at=now() + make_interval(secs => $3), updated_at=now()
		WHERE canonical_url=$1`, canonicalURL, errMsg, delay.Seconds())
	return err
}

// RequeueExtraction starts a link over (a traveller's "Retry").
func (s *Store) RequeueExtraction(ctx context.Context, canonicalURL string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE link_extractions SET status='pending', error='', attempts=0, next_attempt_at=now(), updated_at=now()
		WHERE canonical_url=$1 AND status IN ('failed','partial')`, canonicalURL)
	return err
}

const inspirationLinkCols = `id, user_id, url, canonical_url, note, created_at`

func scanInspirationLink(row pgx.Row) (domain.InspirationLink, error) {
	var l domain.InspirationLink
	err := row.Scan(&l.ID, &l.UserID, &l.URL, &l.CanonicalURL, &l.Note, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

// CreateInspirationLink saves a link for a user. Saving the same link again
// returns the existing row with created=false.
func (s *Store) CreateInspirationLink(ctx context.Context, l domain.InspirationLink) (domain.InspirationLink, bool, error) {
	got, err := scanInspirationLink(s.Pool.QueryRow(ctx, `
		INSERT INTO inspiration_links (id, user_id, url, canonical_url, note) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (user_id, canonical_url) DO NOTHING
		RETURNING `+inspirationLinkCols, l.ID, l.UserID, l.URL, l.CanonicalURL, l.Note))
	if err == nil {
		return got, true, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return got, false, err
	}
	got, err = scanInspirationLink(s.Pool.QueryRow(ctx,
		`SELECT `+inspirationLinkCols+` FROM inspiration_links WHERE user_id=$1 AND canonical_url=$2`,
		l.UserID, l.CanonicalURL))
	return got, false, err
}

func (s *Store) InspirationLinks(ctx context.Context, userID string) ([]domain.InspirationLink, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+inspirationLinkCols+` FROM inspiration_links WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.InspirationLink{}
	for rows.Next() {
		l, err := scanInspirationLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) InspirationLink(ctx context.Context, userID, id string) (domain.InspirationLink, error) {
	return scanInspirationLink(s.Pool.QueryRow(ctx,
		`SELECT `+inspirationLinkCols+` FROM inspiration_links WHERE user_id=$1 AND id=$2`, userID, id))
}

func (s *Store) DeleteInspirationLink(ctx context.Context, userID, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM inspiration_links WHERE user_id=$1 AND id=$2`, userID, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CountInspirationLinksSince counts links a user saved since t (the daily limit).
func (s *Store) CountInspirationLinksSince(ctx context.Context, userID string, t time.Time) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx,
		`SELECT count(*) FROM inspiration_links WHERE user_id=$1 AND created_at >= $2`, userID, t).Scan(&n)
	return n, err
}

// LinksForExtraction lists every user link waiting on one extraction.
func (s *Store) LinksForExtraction(ctx context.Context, canonicalURL string) ([]domain.InspirationLink, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+inspirationLinkCols+` FROM inspiration_links WHERE canonical_url=$1`, canonicalURL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.InspirationLink
	for rows.Next() {
		l, err := scanInspirationLink(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

const savedPlaceCols = `id, user_id, COALESCE(place_id,''), COALESCE(geo_place_id,''), COALESCE(source_link_id,''), created_at`

func scanSavedPlace(row pgx.Row) (domain.SavedPlace, error) {
	var p domain.SavedPlace
	err := row.Scan(&p.ID, &p.UserID, &p.PlaceID, &p.GeoPlaceID, &p.SourceLinkID, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// AddSavedPlace saves a place for later; saving it again returns the
// existing entry.
func (s *Store) AddSavedPlace(ctx context.Context, p domain.SavedPlace) (domain.SavedPlace, error) {
	got, err := scanSavedPlace(s.Pool.QueryRow(ctx, `
		INSERT INTO saved_places (id, user_id, place_id, geo_place_id, source_link_id) VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT DO NOTHING
		RETURNING `+savedPlaceCols,
		p.ID, p.UserID, nullable(p.PlaceID), nullable(p.GeoPlaceID), nullable(p.SourceLinkID)))
	if !errors.Is(err, ErrNotFound) {
		return got, err
	}
	return scanSavedPlace(s.Pool.QueryRow(ctx, `
		SELECT `+savedPlaceCols+` FROM saved_places
		WHERE user_id=$1 AND (place_id=$2 OR geo_place_id=$3)`,
		p.UserID, nullable(p.PlaceID), nullable(p.GeoPlaceID)))
}

func (s *Store) SavedPlaces(ctx context.Context, userID string) ([]domain.SavedPlace, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+savedPlaceCols+` FROM saved_places WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SavedPlace{}
	for rows.Next() {
		p, err := scanSavedPlace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) DeleteSavedPlace(ctx context.Context, userID, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM saved_places WHERE user_id=$1 AND id=$2`, userID, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

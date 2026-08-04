package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/media"
	"github.com/spotto/server/internal/store/postgres"
)

func ptr[T any](v T) *T { return &v }

func vendorIDFor(placeID string) string {
	id := "ven_" + placeID
	if len(id) > 50 {
		id = id[:50]
	}
	return id
}

func randomID(prefix string) string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return prefix + "_" + hex.EncodeToString(b)
}

// categoryForSpot maps an imported OSM/Google spot's data.type + data.tags
// to a real vendor category id. Only food and cafe types correspond to an
// actual bookable business in the taxonomy; everything else (landmarks,
// viewpoints, temples, parks, wildlife, beaches) is non-commercial tourist
// geography with no equivalent category, so it's left blank — the same
// "unknown category" sentinel domain.CategoryByID/IsBookableCategory
// already fall back to gracefully, rather than the nonsensical "General".
func categoryForSpot(data map[string]any) string {
	t, _ := data["type"].(string)
	tagSet := map[string]bool{}
	if raw, ok := data["tags"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				tagSet[strings.ToLower(s)] = true
			}
		}
	}
	switch t {
	case "cafe":
		return "cafe"
	case "food":
		switch {
		case tagSet["bakery"] && !tagSet["restaurant"] && !tagSet["cafe"]:
			return "bakery"
		case tagSet["bar"] && !tagSet["restaurant"] && !tagSet["cafe"]:
			return "bar-pub"
		case tagSet["cafe"] && !tagSet["restaurant"]:
			return "cafe"
		default:
			return "restaurant"
		}
	default:
		return ""
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: importsrilanka <path-to-json>")
	}
	jsonPath := os.Args[1]

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		log.Fatalf("failed to read json: %v", err)
	}

	var spots []domain.Place
	if err := json.Unmarshal(data, &spots); err != nil {
		log.Fatalf("failed to parse json: %v", err)
	}

	cfg := config.Load()
	ctx := context.Background()

	store, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	if err := runImport(ctx, store, spots); err != nil {
		log.Fatalf("import failed: %v", err)
	}
	log.Println("Successfully imported Sri Lanka spots!")

	// Best-effort, outside the main transaction: give each vendor a cover
	// photo pulled from the place's real source photos. Never fails the
	// run — a bad/slow photo just gets skipped and logged, since this
	// step runs on every deploy and must not block the API from starting.
	uploadCoverPhotos(ctx, store, cfg, spots)
}

// runImport wipes existing places (and dependent rows) and reloads them —
// along with a per-place vendor + owner account — in ONE transaction, so a
// failure partway through leaves the previous data untouched instead of a
// half-migrated table. Every write is an idempotent upsert, so re-running
// this against unchanged data is a safe no-op, not an error.
func runImport(ctx context.Context, store *postgres.Store, spots []domain.Place) error {
	tx, err := store.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, stmt := range []string{
		`DELETE FROM reviews WHERE place_id IS NOT NULL`,
		`DELETE FROM claims WHERE place_id IS NOT NULL`,
		`DELETE FROM vibe_checkins WHERE place_id IS NOT NULL`,
		`DELETE FROM checkins WHERE place_id IS NOT NULL`,
		`DELETE FROM media WHERE owner_type = 'place'`,
		`DELETE FROM places`,
	} {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("wipe places %s: %w", stmt, err)
		}
	}
	log.Println("Wiped old places.")

	now := time.Now()
	hash, err := auth.HashPassword("spotto123")
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// central Sri Lanka — only used as a fallback if a place is somehow
	// missing lat/lng in its own data.
	const fallbackLat, fallbackLng = 7.8731, 80.7718

	for _, p := range spots {
		vendorID := vendorIDFor(p.ID)
		userID := "usr_" + p.ID
		if len(userID) > 50 {
			userID = userID[:50]
		}

		var coords struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		}
		lat, lng := fallbackLat, fallbackLng
		if err := json.Unmarshal(p.Data, &coords); err == nil && coords.Lat != 0 && coords.Lng != 0 {
			lat, lng = coords.Lat, coords.Lng
		}

		// Every vendor used to get a hardcoded region_id="colombo"/town="Sri
		// Lanka" regardless of where the place actually is — resolve the real
		// nearest region from its own coordinates instead, and mirror it into
		// the place's data blob too (self-contained JSONB, same as lat/lng),
		// so the traveller-facing API can expose it with no extra join.
		region := geo.NearestRegion(lat, lng)
		var data map[string]any
		if err := json.Unmarshal(p.Data, &data); err != nil || data == nil {
			data = map[string]any{}
		}
		data["region"] = region.Name
		data["regionId"] = region.ID
		category := categoryForSpot(data)
		enrichedData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal place data for %s: %w", p.ID, err)
		}
		p.Data = enrichedData

		// Extract the Google Place ID from the spot's data (if present) and
		// write it into the vendor row's first-class column so that
		// enrichWithPlaceDetails can look up live details without fishing
		// it out of the JSONB blob at runtime.
		var googlePlaceID *string
		if sid, ok := data["sourceId"].(string); ok && sid != "" {
			googlePlaceID = &sid
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO vendors (id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng, google_place_id)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, category=EXCLUDED.category, town=EXCLUDED.town, country=EXCLUDED.country,
				region_id=EXCLUDED.region_id, owner=EXCLUDED.owner, status=EXCLUDED.status,
				priority=EXCLUDED.priority, docs=EXCLUDED.docs, photos=EXCLUDED.photos,
				wifi_score=EXCLUDED.wifi_score, flags=EXCLUDED.flags, note=EXCLUDED.note,
				tier=EXCLUDED.tier, profile=EXCLUDED.profile, menu=EXCLUDED.menu,
				submitted_at=EXCLUDED.submitted_at, lat=EXCLUDED.lat, lng=EXCLUDED.lng,
				google_place_id=EXCLUDED.google_place_id`,
			vendorID, p.Name+" Vendor", category, region.Name, "LK", ptr(region.ID), "Admin",
			domain.VendorApproved, "normal", true, 0, nil, []byte(`[]`), "", "Premium",
			[]byte(`{}`), []byte(`{}`), now.AddDate(0, -1, 0), lat, lng, googlePlaceID,
		); err != nil {
			return fmt.Errorf("upsert vendor %s: %w", vendorID, err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO users (id, email, password_hash, role, name, country, travel_style, interests, vendor_id, extra, created_at, vendor_role)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
			ON CONFLICT (id) DO NOTHING`,
			userID, "import_"+p.ID+"@spotto.dev", hash, domain.RoleVendor, p.Name+" Owner", "Sri Lanka",
			[]byte(`[]`), []byte(`[]`), vendorID, []byte(`{}`), now.AddDate(0, -1, 0), domain.VendorRoleOwner,
		); err != nil {
			return fmt.Errorf("upsert user %s: %w", userID, err)
		}

		p.VendorID = &vendorID
		if _, err := tx.Exec(ctx, `
			INSERT INTO places (id, name, vendor_id, data) VALUES ($1,$2,$3,$4)
			ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, vendor_id=EXCLUDED.vendor_id, data=EXCLUDED.data`,
			p.ID, p.Name, p.VendorID, []byte(p.Data),
		); err != nil {
			return fmt.Errorf("upsert place %s: %w", p.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	log.Printf("Inserted %d spots (with vendor + owner accounts).\n", len(spots))
	return nil
}

// maxSourcePhotosPerPlace caps how many of a place's scraped Google Places
// photos get re-uploaded to R2 — Google Places itself returns at most a
// handful of photo references per place (sri_lanka_spots.json has up to 5),
// so this is "all of them", not an arbitrary truncation.
const maxSourcePhotosPerPlace = 5

// uploadCoverPhotos gives each vendor a real photo gallery pulled from its
// place's data.sourcePhotos (captured from Google Places when the dataset
// was fetched, but until now only sourcePhotos[0] was ever used — the rest
// were fetched and paid for, then discarded). Runs after runImport's
// transaction has already committed, so slow/failed network calls here
// can't hold the places table locked or block the API from starting —
// every failure is logged and skipped, never fatal. Idempotent: a vendor
// that already has a "cover" media row skips the fetch/upload, but every
// vendor still gets cover+gallery mirrored into places.data below — so a
// prior deploy that uploaded media without mirroring it (or a place row
// rebuilt by runImport since) gets self-healed on every run rather than
// only on first upload.
func uploadCoverPhotos(ctx context.Context, store *postgres.Store, cfg config.Config, spots []domain.Place) {
	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" || cfg.R2Bucket == "" {
		log.Println("importsrilanka: R2 not configured, skipping cover photo upload")
		return
	}
	r2 := media.NewR2(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket)
	client := &http.Client{Timeout: 20 * time.Second}

	uploaded, skipped, failed := 0, 0, 0
	for _, p := range spots {
		vendorID := vendorIDFor(p.ID)

		cover, gallery, err := existingVendorPhotos(ctx, store, vendorID)
		if err != nil {
			log.Printf("importsrilanka: check existing photos for %s: %v", vendorID, err)
			continue
		}
		switch {
		case cover != "":
			skipped++
		default:
			cover, gallery, err = uploadVendorPhotos(ctx, store, client, r2, vendorID, p.Data)
			if err != nil {
				log.Printf("importsrilanka: %v", err)
				failed++
				continue
			}
			if cover == "" {
				continue // no sourcePhotos to use
			}
			uploaded++
		}

		if err := mirrorPhotosIntoPlace(ctx, store, vendorID, cover, gallery); err != nil {
			log.Printf("importsrilanka: mirror photos into place for %s: %v", vendorID, err)
		}
	}
	log.Printf("importsrilanka: cover photos - uploaded %d, skipped %d (already had one), failed %d\n", uploaded, skipped, failed)
}

// existingVendorPhotos returns a vendor's already-uploaded cover + gallery
// media, if any — lets uploadCoverPhotos skip re-fetching from Google on
// every run while still re-mirroring into places.data (self-heal).
func existingVendorPhotos(ctx context.Context, store *postgres.Store, vendorID string) (cover string, gallery []string, err error) {
	rows, err := store.Pool.Query(ctx,
		`SELECT role, url FROM media WHERE owner_type='vendor' AND owner_id=$1 AND role IN ('cover','gallery') ORDER BY created_at ASC`,
		vendorID,
	)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()
	gallery = []string{}
	for rows.Next() {
		var role, url string
		if err := rows.Scan(&role, &url); err != nil {
			return "", nil, err
		}
		if role == "cover" {
			cover = url
		} else {
			gallery = append(gallery, url)
		}
	}
	return cover, gallery, rows.Err()
}

// uploadVendorPhotos fetches up to maxSourcePhotosPerPlace of
// placeData.sourcePhotos, uploads each to R2, and records it as a vendor
// media row (the first as role "cover", the rest as role "gallery" —
// matching the live vendor-upload convention in internal/http/vendor.go).
// A single photo failing (rejected type, oversize, network blip) doesn't
// sink the rest; returns ("", nil, nil) if the place has no source photos.
func uploadVendorPhotos(ctx context.Context, store *postgres.Store, client *http.Client, r2 *media.R2, vendorID string, placeData []byte) (cover string, gallery []string, err error) {
	var pd struct {
		SourcePhotos []string `json:"sourcePhotos"`
	}
	if err := json.Unmarshal(placeData, &pd); err != nil || len(pd.SourcePhotos) == 0 {
		return "", nil, nil
	}

	n := min(len(pd.SourcePhotos), maxSourcePhotosPerPlace)
	gallery = []string{}
	for i := 0; i < n; i++ {
		role := "gallery"
		if i == 0 {
			role = "cover"
		}
		url, uploadErr := uploadOnePhoto(ctx, store, client, r2, vendorID, role, pd.SourcePhotos[i])
		if uploadErr != nil {
			log.Printf("importsrilanka: photo %d for %s: %v", i, vendorID, uploadErr)
			continue
		}
		if role == "cover" {
			cover = url
		} else {
			gallery = append(gallery, url)
		}
	}
	if cover != "" || len(gallery) > 0 {
		count := len(gallery)
		if cover != "" {
			count++
		}
		if _, err := store.Pool.Exec(ctx, `UPDATE vendors SET photos = $2 WHERE id = $1`, vendorID, count); err != nil {
			log.Printf("importsrilanka: update photos count for %s: %v", vendorID, err)
		}
	}
	return cover, gallery, nil
}

// uploadOnePhoto fetches sourceURL, uploads it to R2, and records it as a
// vendor media row with the given role.
func uploadOnePhoto(ctx context.Context, store *postgres.Store, client *http.Client, r2 *media.R2, vendorID, role, sourceURL string) (string, error) {
	body, contentType, err := fetchImage(ctx, client, sourceURL)
	if err != nil {
		return "", fmt.Errorf("fetch photo for %s: %w", vendorID, err)
	}

	ext, kind, maxSize, ok := media.Resolve(contentType)
	if !ok || int64(len(body)) > maxSize {
		return "", fmt.Errorf("photo for %s rejected (type=%s size=%d)", vendorID, contentType, len(body))
	}

	key := fmt.Sprintf("vendor/%s/%s%s", vendorID, randomID("m"), ext)
	if err := putToR2(ctx, client, r2, key, contentType, body); err != nil {
		return "", fmt.Errorf("upload to R2 for %s: %w", vendorID, err)
	}

	url := r2.PublicURL(key)
	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO media (id, owner_type, owner_id, kind, role, object_key, url, content_type, size, width, height, uploaded_by, status, sort, created_at)
		VALUES ($1,'vendor',$2,$3,$4,$5,$6,$7,0,0,'system-import','ready',0,now())`,
		randomID("m"), vendorID, kind, role, key, url, contentType, len(body),
	); err != nil {
		return "", fmt.Errorf("insert media row for %s: %w", vendorID, err)
	}
	return url, nil
}

// mirrorPhotosIntoPlace writes cover/gallery into the vendor's
// place.data.cover/data.gallery, matching what syncVendorPlace does for the
// live HTTP vendor-media-upload path (server/internal/http/vendor.go). The
// import script is a separate main package without access to that server
// code, so it replicates the same JSON patch directly here.
func mirrorPhotosIntoPlace(ctx context.Context, store *postgres.Store, vendorID, cover string, gallery []string) error {
	var placeID string
	var raw []byte
	if err := store.Pool.QueryRow(ctx,
		`SELECT id, data FROM places WHERE vendor_id=$1`, vendorID,
	).Scan(&placeID, &raw); err != nil {
		return fmt.Errorf("find place for vendor %s: %w", vendorID, err)
	}

	data := map[string]any{}
	_ = json.Unmarshal(raw, &data)
	data["cover"] = cover
	data["gallery"] = gallery

	updated, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal place data for %s: %w", placeID, err)
	}
	if _, err := store.Pool.Exec(ctx, `UPDATE places SET data=$1 WHERE id=$2`, updated, placeID); err != nil {
		return fmt.Errorf("update place %s: %w", placeID, err)
	}
	return nil
}

func fetchImage(ctx context.Context, client *http.Client, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	const maxBody = 8<<20 + 1 // one byte over the 8MB image cap, to detect oversize
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return nil, "", err
	}
	contentType := resp.Header.Get("Content-Type")
	if idx := bytes.IndexByte([]byte(contentType), ';'); idx >= 0 {
		contentType = contentType[:idx]
	}
	return body, contentType, nil
}

func putToR2(ctx context.Context, client *http.Client, r2 *media.R2, key, contentType string, body []byte) error {
	putURL, err := r2.PresignPut(ctx, key, contentType, 2*time.Minute)
	if err != nil {
		return fmt.Errorf("presign: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, putURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", contentType)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

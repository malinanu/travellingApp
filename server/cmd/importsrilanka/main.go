package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
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
		`DELETE FROM vibe_votes WHERE place_id IS NOT NULL`,
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
		enrichedData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal place data for %s: %w", p.ID, err)
		}
		p.Data = enrichedData

		if _, err := tx.Exec(ctx, `
			INSERT INTO vendors (id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
			ON CONFLICT (id) DO UPDATE SET
				name=EXCLUDED.name, category=EXCLUDED.category, town=EXCLUDED.town, country=EXCLUDED.country,
				region_id=EXCLUDED.region_id, owner=EXCLUDED.owner, status=EXCLUDED.status,
				priority=EXCLUDED.priority, docs=EXCLUDED.docs, photos=EXCLUDED.photos,
				wifi_score=EXCLUDED.wifi_score, flags=EXCLUDED.flags, note=EXCLUDED.note,
				tier=EXCLUDED.tier, profile=EXCLUDED.profile, menu=EXCLUDED.menu,
				submitted_at=EXCLUDED.submitted_at, lat=EXCLUDED.lat, lng=EXCLUDED.lng`,
			vendorID, p.Name+" Vendor", "General", region.Name, "LK", ptr(region.ID), "Admin",
			domain.VendorApproved, "normal", true, 0, nil, []byte(`[]`), "", "Premium",
			[]byte(`{}`), []byte(`{}`), now.AddDate(0, -1, 0), lat, lng,
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

// uploadCoverPhotos gives each vendor a cover photo pulled from its place's
// data.sourcePhotos[0] (captured from Google Places when the dataset was
// fetched, never used until now). Runs after runImport's transaction has
// already committed, so slow/failed network calls here can't hold the
// places table locked or block the API from starting — every failure is
// logged and skipped, never fatal. Idempotent: a vendor that already has a
// "cover" media row skips the fetch/upload, but every vendor still gets its
// cover mirrored into places.data below — so a prior deploy that uploaded
// media without mirroring it (or a place row rebuilt by runImport since)
// gets self-healed on every run rather than only on first upload.
func uploadCoverPhotos(ctx context.Context, store *postgres.Store, cfg config.Config, spots []domain.Place) {
	if cfg.R2AccountID == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "" || cfg.R2Bucket == "" {
		log.Println("importsrilanka: R2 not configured, skipping cover photo upload")
		return
	}
	r2 := media.NewR2(cfg.R2AccountID, cfg.R2AccessKeyID, cfg.R2SecretAccessKey, cfg.R2Bucket, cfg.R2PublicBaseURL)
	client := &http.Client{Timeout: 20 * time.Second}

	uploaded, skipped, failed := 0, 0, 0
	for _, p := range spots {
		vendorID := vendorIDFor(p.ID)

		var coverURL string
		err := store.Pool.QueryRow(ctx,
			`SELECT url FROM media WHERE owner_type='vendor' AND owner_id=$1 AND role='cover' LIMIT 1`,
			vendorID,
		).Scan(&coverURL)
		switch {
		case err == nil:
			skipped++
		case errors.Is(err, pgx.ErrNoRows):
			coverURL, err = uploadOneCoverPhoto(ctx, store, client, r2, vendorID, p.Data)
			if err != nil {
				log.Printf("importsrilanka: %v", err)
				failed++
				continue
			}
			if coverURL == "" {
				continue // no sourcePhotos to use
			}
			uploaded++
		default:
			log.Printf("importsrilanka: check existing cover for %s: %v", vendorID, err)
			continue
		}

		if err := mirrorCoverIntoPlace(ctx, store, vendorID, coverURL); err != nil {
			log.Printf("importsrilanka: mirror cover into place for %s: %v", vendorID, err)
		}
	}
	log.Printf("importsrilanka: cover photos - uploaded %d, skipped %d (already had one), failed %d\n", uploaded, skipped, failed)
}

// uploadOneCoverPhoto fetches placeData.sourcePhotos[0], uploads it to R2,
// and records it as the vendor's cover media row. Returns ("", nil) if the
// place has no source photos to use.
func uploadOneCoverPhoto(ctx context.Context, store *postgres.Store, client *http.Client, r2 *media.R2, vendorID string, placeData []byte) (string, error) {
	var pd struct {
		SourcePhotos []string `json:"sourcePhotos"`
	}
	if err := json.Unmarshal(placeData, &pd); err != nil || len(pd.SourcePhotos) == 0 {
		return "", nil
	}

	body, contentType, err := fetchImage(ctx, client, pd.SourcePhotos[0])
	if err != nil {
		return "", fmt.Errorf("fetch cover photo for %s: %w", vendorID, err)
	}

	ext, kind, maxSize, ok := media.Resolve(contentType)
	if !ok || int64(len(body)) > maxSize {
		return "", fmt.Errorf("cover photo for %s rejected (type=%s size=%d)", vendorID, contentType, len(body))
	}

	key := fmt.Sprintf("vendor/%s/%s%s", vendorID, randomID("m"), ext)
	if err := putToR2(ctx, client, r2, key, contentType, body); err != nil {
		return "", fmt.Errorf("upload to R2 for %s: %w", vendorID, err)
	}

	url := r2.PublicURL(key)
	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO media (id, owner_type, owner_id, kind, role, object_key, url, content_type, size, width, height, uploaded_by, status, sort, created_at)
		VALUES ($1,'vendor',$2,$3,'cover',$4,$5,$6,$7,0,0,'system-import','ready',0,now())`,
		randomID("m"), vendorID, kind, key, url, contentType, len(body),
	); err != nil {
		return "", fmt.Errorf("insert media row for %s: %w", vendorID, err)
	}
	if _, err := store.Pool.Exec(ctx, `UPDATE vendors SET photos = 1 WHERE id = $1`, vendorID); err != nil {
		log.Printf("importsrilanka: update photos count for %s: %v", vendorID, err)
	}
	return url, nil
}

// mirrorCoverIntoPlace writes coverURL into the vendor's place.data.cover
// (and a single-entry gallery), matching what syncVendorPlace does for the
// live HTTP vendor-media-upload path (server/internal/http/vendor.go). The
// import script is a separate main package without access to that server
// code, so it replicates the same JSON patch directly here.
func mirrorCoverIntoPlace(ctx context.Context, store *postgres.Store, vendorID, coverURL string) error {
	var placeID string
	var raw []byte
	if err := store.Pool.QueryRow(ctx,
		`SELECT id, data FROM places WHERE vendor_id=$1`, vendorID,
	).Scan(&placeID, &raw); err != nil {
		return fmt.Errorf("find place for vendor %s: %w", vendorID, err)
	}

	data := map[string]any{}
	_ = json.Unmarshal(raw, &data)
	data["cover"] = coverURL
	data["gallery"] = []string{coverURL}

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

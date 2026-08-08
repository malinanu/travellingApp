package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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

	uploadCoverPhotos(ctx, store, cfg, spots, jsonPath)
}

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

		region := geo.NearestRegion(lat, lng)
		var data map[string]any
		if err := json.Unmarshal(p.Data, &data); err != nil || data == nil {
			data = map[string]any{}
		}
		data["region"] = region.Name
		data["regionId"] = region.ID
		category := categoryForSpot(data)

		// Extract formatted data fields into vendor profile
		profile := map[string]any{}
		if hours, ok := data["hours"]; ok {
			profile["hours"] = hours
		}
		if blurb, ok := data["blurb"]; ok {
			profile["blurb"] = blurb
		}
		if contact, ok := data["contact"]; ok {
			profile["contact"] = contact
		}
		profileBytes, _ := json.Marshal(profile)

		enrichedData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal place data for %s: %w", p.ID, err)
		}
		p.Data = enrichedData

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
			profileBytes, []byte(`{}`), now.AddDate(0, -1, 0), lat, lng, googlePlaceID,
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

const maxSourcePhotosPerPlace = 5

func uploadCoverPhotos(ctx context.Context, store *postgres.Store, cfg config.Config, spots []domain.Place, jsonPath string) {
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
			var coverThumb, coverMicro string
			var galleryThumbs []string
			cover, coverThumb, coverMicro, gallery, galleryThumbs, err = uploadVendorPhotos(ctx, store, client, r2, vendorID, p.Data, cfg.GoogleCalendarAPIKey, jsonPath)
			if err != nil {
				log.Printf("importsrilanka: %v", err)
				failed++
				continue
			}
			if cover == "" {
				continue
			}
			uploaded++

			if err := mirrorPhotosIntoPlace(ctx, store, vendorID, cover, coverThumb, coverMicro, gallery, galleryThumbs); err != nil {
				log.Printf("importsrilanka: mirror photos into place for %s: %v", vendorID, err)
			}
		}
	}
	log.Printf("importsrilanka: cover photos - uploaded %d, skipped %d (already had one), failed %d\n", uploaded, skipped, failed)
}

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

type PhotoResult struct {
	URL        string
	ThumbKey   string
	MicroThumb string
}

func uploadVendorPhotos(ctx context.Context, store *postgres.Store, client *http.Client, r2 *media.R2, vendorID string, placeData []byte, apiKey, jsonPath string) (string, string, string, []string, []string, error) {
	var pd struct {
		SourcePhotos []string `json:"sourcePhotos"`
	}
	if err := json.Unmarshal(placeData, &pd); err != nil || len(pd.SourcePhotos) == 0 {
		return "", "", "", nil, nil, nil
	}

	var cover, coverThumb, coverMicro string
	var gallery, galleryThumbs []string

	n := min(len(pd.SourcePhotos), maxSourcePhotosPerPlace)
	for i := 0; i < n; i++ {
		role := "gallery"
		if i == 0 {
			role = "cover"
		}
		res, uploadErr := uploadOnePhoto(ctx, store, client, r2, vendorID, role, pd.SourcePhotos[i], apiKey, jsonPath)
		if uploadErr != nil {
			log.Printf("importsrilanka: photo %d for %s: %v", i, vendorID, uploadErr)
			continue
		}
		if role == "cover" {
			cover = res.URL
			coverThumb = r2.PublicURL(res.ThumbKey)
			coverMicro = res.MicroThumb
		} else {
			gallery = append(gallery, res.URL)
			galleryThumbs = append(galleryThumbs, r2.PublicURL(res.ThumbKey))
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
	return cover, coverThumb, coverMicro, gallery, galleryThumbs, nil
}

func uploadOnePhoto(ctx context.Context, store *postgres.Store, client *http.Client, r2 *media.R2, vendorID, role, sourceURL, apiKey, jsonPath string) (PhotoResult, error) {
	body, contentType, err := fetchImage(ctx, client, sourceURL, apiKey, jsonPath)
	if err != nil {
		return PhotoResult{}, fmt.Errorf("fetch photo for %s: %w", vendorID, err)
	}

	ext, kind, maxSize, ok := media.Resolve(contentType)
	if !ok || int64(len(body)) > maxSize {
		return PhotoResult{}, fmt.Errorf("photo for %s rejected (type=%s size=%d)", vendorID, contentType, len(body))
	}

	thumbs, err := media.GenerateThumbnails(bytes.NewReader(body))
	if err != nil {
		return PhotoResult{}, fmt.Errorf("thumbnails for %s: %w", vendorID, err)
	}

	srcImg, _, err := image.DecodeConfig(bytes.NewReader(body))
	if err != nil {
		log.Printf("importsrilanka: warn: decode config for %s failed: %v", vendorID, err)
	}
	width, height := 0, 0
	if err == nil {
		width, height = srcImg.Width, srcImg.Height
	}

	key := fmt.Sprintf("vendor/%s/%s%s", vendorID, randomID("m"), ext)
	if err := putToR2(ctx, client, r2, key, contentType, body); err != nil {
		return PhotoResult{}, fmt.Errorf("upload to R2 for %s: %w", vendorID, err)
	}

	thumbKey := fmt.Sprintf("vendor/%s/%s_thumb.jpg", vendorID, randomID("m"))
	if err := putToR2(ctx, client, r2, thumbKey, "image/jpeg", thumbs.Thumb); err != nil {
		return PhotoResult{}, fmt.Errorf("upload thumb to R2 for %s: %w", vendorID, err)
	}

	microBase64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(thumbs.Micro)

	url := r2.PublicURL(key)
	if _, err := store.Pool.Exec(ctx, `
		INSERT INTO media (id, owner_type, owner_id, kind, role, object_key, url, content_type, size, width, height, thumb_key, micro_thumb, uploaded_by, status, sort, created_at)
		VALUES ($1,'vendor',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'system-import','ready',0,now())`,
		randomID("m"), vendorID, kind, role, key, url, contentType, len(body), width, height, thumbKey, microBase64,
	); err != nil {
		return PhotoResult{}, fmt.Errorf("insert media row for %s: %w", vendorID, err)
	}
	return PhotoResult{URL: url, ThumbKey: thumbKey, MicroThumb: microBase64}, nil
}

func mirrorPhotosIntoPlace(ctx context.Context, store *postgres.Store, vendorID, cover, coverThumb, coverMicro string, gallery, galleryThumbs []string) error {
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
	data["coverThumb"] = coverThumb
	data["coverMicro"] = coverMicro
	data["gallery"] = gallery
	data["galleryThumbs"] = galleryThumbs

	updated, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal place data for %s: %w", placeID, err)
	}
	if _, err := store.Pool.Exec(ctx, `UPDATE places SET data=$1 WHERE id=$2`, updated, placeID); err != nil {
		return fmt.Errorf("update place %s: %w", placeID, err)
	}
	return nil
}

func fetchImage(ctx context.Context, client *http.Client, urlStr, apiKey, jsonPath string) ([]byte, string, error) {
	if strings.HasPrefix(urlStr, "/images/") || strings.HasPrefix(urlStr, "images/") {
		baseDir := filepath.Dir(jsonPath)
		if strings.HasPrefix(urlStr, "/") {
			urlStr = urlStr[1:]
		}
		path := filepath.Join(baseDir, urlStr)
		body, err := os.ReadFile(path)
		if err != nil {
			return nil, "", err
		}
		ext := strings.ToLower(filepath.Ext(path))
		contentType := "image/jpeg"
		if ext == ".png" {
			contentType = "image/png"
		} else if ext == ".webp" {
			contentType = "image/webp"
		}
		return body, contentType, nil
	}

	if apiKey != "" && strings.Contains(urlStr, "google") {
		re := regexp.MustCompile(`([?&])key=[^&]*`)
		if re.MatchString(urlStr) {
			urlStr = re.ReplaceAllString(urlStr, "${1}key="+apiKey)
		} else {
			if strings.Contains(urlStr, "?") {
				urlStr += "&key=" + apiKey
			} else {
				urlStr += "?key=" + apiKey
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", err
	}
	if apiKey != "" && strings.Contains(urlStr, "google") {
		req.Header.Set("x-goog-api-key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status %d for %s", resp.StatusCode, urlStr)
	}
	const maxBody = 8<<20 + 1
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

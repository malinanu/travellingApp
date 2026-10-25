package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/geo"
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
	log.Println("Successfully imported Sri Lanka spots with media!")
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
		`DELETE FROM media WHERE owner_type = 'vendor'`,
		`DELETE FROM places`,
		`DELETE FROM vendors`,
		`DELETE FROM users WHERE email LIKE 'import_%@spotto.dev'`,
	} {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("wipe places %s: %w", stmt, err)
		}
	}
	log.Println("Wiped old places, vendors, media, and import users.")

	now := time.Now()
	// These owner accounts aren't meant for logging in: their emails are
	// guessable from the public dataset, so they get a random password
	// nobody knows. Real owners get in through vendor profile grants.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return fmt.Errorf("random password: %w", err)
	}
	hash, err := auth.HashPassword(hex.EncodeToString(secret))
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
		category := domain.CategoryForPlaceData(data)
		if category == "" {
			category = "uncategorized"
		}
		data["categoryId"] = category
		if c, ok := domain.CategoryByID(category); ok {
			data["color"] = domain.PinColorForSegment(c.Segment)
		}

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

		var googlePlaceID *string
		if sid, ok := data["sourceId"].(string); ok && sid != "" {
			googlePlaceID = &sid
		}

		// Handle uploaded media
		var cover, coverThumb, coverMicro string
		var gallery, galleryThumbs []string

		if uploadedPhotos, ok := data["uploadedPhotos"].([]interface{}); ok {
			for _, up := range uploadedPhotos {
				photo, ok := up.(map[string]interface{})
				if !ok {
					continue
				}

				role, _ := photo["role"].(string)
				objectKey, _ := photo["object_key"].(string)
				url, _ := photo["url"].(string)
				
				// Rewrite the old proxy URLs to the new public CDN URLs
				url = strings.ReplaceAll(url, "/media/raw/", "https://spotto-images.threelogiq.com/")
				
				contentType, _ := photo["content_type"].(string)
				width := int(photo["width"].(float64))
				height := int(photo["height"].(float64))
				size := int(photo["size"].(float64))
				thumbKey, _ := photo["thumb_key"].(string)
				microThumb, _ := photo["micro_thumb"].(string)

				kind := "image"

				if _, err := tx.Exec(ctx, `
					INSERT INTO media (id, owner_type, owner_id, kind, role, object_key, url, content_type, size, width, height, thumb_key, micro_thumb, uploaded_by, status, sort, created_at)
					VALUES ($1,'vendor',$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'system-import','ready',0,now())`,
					randomID("m"), vendorID, kind, role, objectKey, url, contentType, size, width, height, thumbKey, microThumb,
				); err != nil {
					return fmt.Errorf("insert media row for %s: %w", vendorID, err)
				}

				if role == "cover" {
					cover = url
					coverThumb = strings.Replace(url, objectKey, thumbKey, 1)
					coverMicro = microThumb
				} else {
					gallery = append(gallery, url)
					galleryThumbs = append(galleryThumbs, strings.Replace(url, objectKey, thumbKey, 1))
				}
			}
		}

		if cover != "" {
			data["cover"] = cover
			data["coverThumb"] = coverThumb
			data["coverMicro"] = coverMicro
		}
		if len(gallery) > 0 {
			data["gallery"] = gallery
			data["galleryThumbs"] = galleryThumbs
		}

		photosCount := len(gallery)
		if cover != "" {
			photosCount++
		}

		enrichedData, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal place data for %s: %w", p.ID, err)
		}
		p.Data = enrichedData

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
			vendorID, p.Name, category, region.Name, "LK", ptr(region.ID), "Admin",
			domain.VendorApproved, "normal", true, photosCount, nil, []byte(`[]`), "", "Premium",
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

package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/geo"
	"github.com/spotto/server/internal/store/postgres"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	store, err := postgres.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer store.Close()

	rows, err := store.Pool.Query(ctx, `SELECT id, vendor_id, data FROM places`)
	if err != nil {
		log.Fatalf("query places: %v", err)
	}
	defer rows.Close()

	type place struct {
		ID       string
		VendorID *string
		Data     []byte
	}
	var places []place
	for rows.Next() {
		var p place
		if err := rows.Scan(&p.ID, &p.VendorID, &p.Data); err != nil {
			log.Fatalf("scan place: %v", err)
		}
		places = append(places, p)
	}
	rows.Close()

	updatedCount := 0
	for _, p := range places {
		var data map[string]any
		if err := json.Unmarshal(p.Data, &data); err != nil || data == nil {
			continue
		}

		lat, okLat := data["lat"].(float64)
		lng, okLng := data["lng"].(float64)
		if !okLat || !okLng || lat == 0 || lng == 0 {
			continue
		}

		r := geo.NearestRegion(lat, lng)
		
		// Skip if region hasn't actually changed
		if data["regionId"] == r.ID {
			continue
		}

		data["region"] = r.Name
		data["regionId"] = r.ID

		b, _ := json.Marshal(data)

		// Update place
		_, err = store.Pool.Exec(ctx, `UPDATE places SET data = $1 WHERE id = $2`, b, p.ID)
		if err != nil {
			log.Fatalf("update place: %v", err)
		}

		// Update vendor
		if p.VendorID != nil {
			_, err = store.Pool.Exec(ctx, `UPDATE vendors SET region_id = $1, town = $2 WHERE id = $3`, r.ID, r.Name, *p.VendorID)
			if err != nil {
				log.Fatalf("update vendor: %v", err)
			}
		}
		updatedCount++
	}

	log.Printf("Successfully updated regions for %d spots to match the newly expanded map!\n", updatedCount)
}

// backfillcategories writes categoryId and the segment pin color onto
// places stored before those fields existed. The API also fills them in at
// serve time (normalizePlaceCategory), so this just makes the DB match.
package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
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

	rows, err := store.Pool.Query(ctx, `SELECT id, data FROM places`)
	if err != nil {
		log.Fatalf("query places: %v", err)
	}
	type place struct {
		ID   string
		Data []byte
	}
	var places []place
	for rows.Next() {
		var p place
		if err := rows.Scan(&p.ID, &p.Data); err != nil {
			log.Fatalf("scan place: %v", err)
		}
		places = append(places, p)
	}
	rows.Close()

	updated := 0
	for _, p := range places {
		var data map[string]any
		if err := json.Unmarshal(p.Data, &data); err != nil || data == nil {
			continue
		}
		id, _ := data["categoryId"].(string)
		cat, ok := domain.CategoryByID(id)
		if !ok {
			cat, ok = domain.CategoryByID(domain.CategoryForPlaceData(data))
			if !ok {
				continue
			}
			data["categoryId"] = cat.ID
		} else if c, _ := data["color"].(string); c != "" && c != "teal" {
			continue
		}
		data["color"] = domain.PinColorForSegment(cat.Segment)

		b, _ := json.Marshal(data)
		if _, err := store.Pool.Exec(ctx, `UPDATE places SET data = $1 WHERE id = $2`, b, p.ID); err != nil {
			log.Fatalf("update place %s: %v", p.ID, err)
		}
		updated++
	}
	log.Printf("Backfilled categoryId/color on %d places.", updated)
}

// Package importspots wipes the places table (and everything that
// references a place — reviews, claims, vibe votes, checkins, place media)
// and reloads it from the real-world dataset embedded in spots.json.
//
// This runs on EVERY server boot (see cmd/importspots and the Docker
// entrypoint), by explicit product decision while the deployment is still
// in a testing phase: any vendor listing, user-submitted spot, review, or
// media added between deploys is destroyed on the next one. Revisit this
// once real vendor/traveller data needs to survive a redeploy.
package importspots

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

//go:embed spots.json
var spotsJSON []byte

// Run replaces every row in places (plus rows in other tables that
// reference a place, since none of those have an ON DELETE CASCADE) with
// the places embedded in spots.json.
func Run(ctx context.Context, store *postgres.Store) error {
	var places []domain.Place
	if err := json.Unmarshal(spotsJSON, &places); err != nil {
		return fmt.Errorf("importspots: parse spots.json: %w", err)
	}

	tx, err := store.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("importspots: begin tx: %w", err)
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
			return fmt.Errorf("importspots: %s: %w", stmt, err)
		}
	}

	for _, p := range places {
		_, err := tx.Exec(ctx, `
			INSERT INTO places (id, name, vendor_id, data) VALUES ($1,$2,$3,$4)`,
			p.ID, p.Name, p.VendorID, []byte(p.Data))
		if err != nil {
			return fmt.Errorf("importspots: insert place %s: %w", p.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("importspots: commit: %w", err)
	}

	fmt.Printf("importspots: wiped places (+ dependent reviews/claims/vibe_votes/checkins/media), inserted %d places\n", len(places))
	return nil
}

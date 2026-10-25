package httpapi

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/spotto/server/internal/auth"
)

// presenceRetention is how long rough traveller areas are kept: a 90-day
// view on the geo page plus the 90 days before it, for growth.
const presenceRetention = 180 * 24 * time.Hour

// recordPresence notes the rough area a signed-in traveller is using the app
// from (see Store.RecordTravellerPresence). Best-effort: a failure is logged
// and never fails the request it rides on.
func (s *Server) recordPresence(r *http.Request, lat, lng float64) {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 || (lat == 0 && lng == 0) {
		return
	}
	userID := auth.FromContext(r.Context()).Subject
	if userID == "" {
		return
	}
	if err := s.Store.RecordTravellerPresence(r.Context(), userID, lat, lng); err != nil {
		log.Printf("record traveller presence: %v", err)
	}
}

// StartPresencePruner deletes rough traveller areas older than
// presenceRetention, now and then once a day.
func (s *Server) StartPresencePruner(ctx context.Context) {
	prune := func() {
		if n, err := s.Store.PruneTravellerPresence(ctx, time.Now().Add(-presenceRetention)); err != nil {
			log.Printf("prune traveller presence: %v", err)
		} else if n > 0 {
			log.Printf("pruned %d old traveller presence rows", n)
		}
	}
	prune()
	t := time.NewTicker(24 * time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			prune()
		}
	}
}

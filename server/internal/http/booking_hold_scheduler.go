package httpapi

import (
	"context"
	"log"
	"time"
)

const bookingHoldSchedulerInterval = time.Minute

// StartBookingHoldScheduler releases the seats of paid bookings whose
// payment didn't arrive before their hold ran out. Blocks until ctx is
// cancelled; call with `go`.
func (s *Server) StartBookingHoldScheduler(ctx context.Context) {
	ticker := time.NewTicker(bookingHoldSchedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n, err := s.Store.ExpireBookingHolds(ctx); err != nil {
				log.Printf("booking holds: %v", err)
			} else if n > 0 {
				log.Printf("booking holds: %d unpaid booking(s) expired", n)
			}
		}
	}
}

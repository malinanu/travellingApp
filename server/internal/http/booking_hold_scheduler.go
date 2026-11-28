package httpapi

import (
	"context"
	"log"
	"time"
)

const bookingHoldSchedulerInterval = time.Minute

// StartBookingHoldScheduler releases the seats of paid bookings whose
// payment didn't arrive before their hold ran out, and resends refunds that
// got stuck before reaching the gateway. Blocks until ctx is
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
			// Refunds recorded but never sent to the gateway (the process
			// stopped in between) are picked up here.
			if stuck, err := s.Store.StaleRequestedRefunds(ctx, staleRefundAge); err != nil {
				log.Printf("refund sweep: %v", err)
			} else {
				for _, rf := range stuck {
					s.processRefund(ctx, rf)
				}
			}
		}
	}
}

package httpapi

import (
	"context"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// Who cancelled a booking.
const (
	cancelByTraveller = "traveller"
	cancelByVendor    = "vendor"
	cancelByAdmin     = "admin"
)

// cancelBooking cancels b on behalf of `by` and, for a paid booking, issues
// the refund its terms allow: a traveller gets the cancellation policy, a
// vendor or admin cancellation always refunds in full. It returns the
// amount refunded (0 for a free or unpaid booking).
func (s *Server) cancelBooking(ctx context.Context, b domain.Booking, by, actor string) (postgres.Refund, error) {
	req := postgres.CancelRequest{BookingID: b.ID, By: by, Actor: actor, Now: time.Now(), FullRefund: by != cancelByTraveller}
	switch by {
	case cancelByTraveller:
		req.TravellerID = b.TravellerID
	case cancelByVendor:
		req.VendorID = b.VendorID
	}
	cancelled, refund, err := s.Store.CancelBookingWithRefund(ctx, req)
	if err != nil {
		return postgres.Refund{}, err
	}
	// No refund (nothing paid, or outside the refund window) is a zero
	// Refund: AmountMinor 0.
	var refunded postgres.Refund
	if refund != nil {
		s.processRefund(ctx, *refund)
		refunded = *refund
	}
	if by != cancelByTraveller {
		// The traveller didn't do this themselves: tell them.
		if me, err := s.Store.UserByID(ctx, cancelled.TravellerID); err == nil && me.Email != "" {
			name := cancelled.ServiceName
			s.Jobs.Enqueue(func() {
				s.Mail.Send(me.Email, "Your Spotto booking was cancelled", bookingCancelledEmailHTML(cancelled, name, refunded))
			})
		}
	}
	return refunded, nil
}

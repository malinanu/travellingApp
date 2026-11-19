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
func (s *Server) cancelBooking(ctx context.Context, b domain.Booking, by, actor string) (int64, error) {
	req := postgres.CancelRequest{BookingID: b.ID, By: by, Actor: actor, Now: time.Now(), FullRefund: by != cancelByTraveller}
	switch by {
	case cancelByTraveller:
		req.TravellerID = b.TravellerID
	case cancelByVendor:
		req.VendorID = b.VendorID
	}
	_, refund, err := s.Store.CancelBookingWithRefund(ctx, req)
	if err != nil {
		return 0, err
	}
	if refund == nil {
		return 0, nil
	}
	s.processRefund(ctx, *refund)
	return refund.AmountMinor, nil
}

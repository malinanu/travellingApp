package httpapi

import (
	"context"

	"github.com/spotto/server/internal/domain"
)

// Who cancelled a booking.
const (
	cancelByTraveller = "traveller"
	cancelByVendor    = "vendor"
	cancelByAdmin     = "admin"
)

// cancelBooking cancels b on behalf of `by`. The returned amount is what
// was refunded to the traveller (0 for a free booking).
func (s *Server) cancelBooking(ctx context.Context, b domain.Booking, by, actor string) (int64, error) {
	if by == cancelByTraveller {
		return 0, s.Store.CancelBookingAsTraveller(ctx, b.ID, b.TravellerID)
	}
	return 0, s.Store.CancelBookingAsVendor(ctx, b.ID, b.VendorID)
}

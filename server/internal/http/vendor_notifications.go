package httpapi

import (
	"context"
	"fmt"
	"time"

	"github.com/spotto/server/internal/domain"
)

// TriggerClaimNotification emits a notification when an offer is claimed.
func (s *Server) TriggerClaimNotification(ctx context.Context, vendorID, travellerName, offerTitle string) error {
	n := domain.VendorNotification{
		ID:     "vnotif_" + randomHex(8),
		Icon:   "Ticket",
		Tone:   "teal",
		Title:  "New offer claim!",
		Sub:    fmt.Sprintf("%s claimed: %s", travellerName, offerTitle),
		Time:   time.Now().Format("Jan 2, 3:04 PM"),
		Unread: true,
	}
	err := s.Store.UpsertVendorNotification(ctx, vendorID, n)
	if err == nil {
		go s.dispatchVendorPush(vendorID, n.Title, n.Sub, map[string]string{"type": "claim"})
	}
	return err
}

// TriggerReviewNotification emits a notification when a new review is posted.
func (s *Server) TriggerReviewNotification(ctx context.Context, vendorID, travellerName string, rating int) error {
	n := domain.VendorNotification{
		ID:     "vnotif_" + randomHex(8),
		Icon:   "Star",
		Tone:   "sun",
		Title:  fmt.Sprintf("New %d-star review", rating),
		Sub:    fmt.Sprintf("%s left a %d-star rating.", travellerName, rating),
		Time:   time.Now().Format("Jan 2, 3:04 PM"),
		Unread: true,
	}
	err := s.Store.UpsertVendorNotification(ctx, vendorID, n)
	if err == nil {
		go s.dispatchVendorPush(vendorID, n.Title, n.Sub, map[string]string{"type": "review"})
	}
	return err
}

// TriggerOfferLimitNotification emits an alert when an offer reaches its claim limit.
func (s *Server) TriggerOfferLimitNotification(ctx context.Context, vendorID, offerTitle string) error {
	n := domain.VendorNotification{
		ID:     "vnotif_" + randomHex(8),
		Icon:   "Zap",
		Tone:   "coral",
		Title:  "Offer claim limit reached",
		Sub:    fmt.Sprintf("'%s' has reached its claim limit.", offerTitle),
		Time:   time.Now().Format("Jan 2, 3:04 PM"),
		Unread: true,
	}
	err := s.Store.UpsertVendorNotification(ctx, vendorID, n)
	if err == nil {
		go s.dispatchVendorPush(vendorID, n.Title, n.Sub, map[string]string{"type": "limit"})
	}
	return err
}

func (s *Server) dispatchVendorPush(vendorID, title, body string, data map[string]string) {
	ctx := context.Background()
	team, err := s.Store.VendorTeamMembers(ctx, vendorID)
	if err != nil {
		return
	}
	for _, member := range team {
		s.SendUserPushNotification(ctx, member.ID, title, body, data)
	}
}

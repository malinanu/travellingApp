package httpapi

// Second scheduled background job in this codebase (see
// holiday_scheduler.go's doc comment for the distinction from
// internal/jobs.Queue's fire-and-forget work). Sends a reminder email
// (reusing the existing mail.Mailer â€” no traveller-facing notification
// system exists otherwise, see booking.go's design notes) for confirmed
// bookings starting soon.

import (
	"context"
	"fmt"
	"log"
	"time"
)

const bookingReminderSchedulerInterval = 1 * time.Hour

// bookingReminderWindow is how far ahead of a booking's start time this
// scheduler starts considering it "due" â€” a booking is only ever emailed
// once, gated by BookingsForReminder's reminder_sent_at IS NULL filter, so
// widening this window doesn't risk duplicate reminders on a slow tick.
const bookingReminderWindow = 24 * time.Hour

// StartBookingReminderScheduler periodically emails travellers whose
// confirmed booking starts within bookingReminderWindow and hasn't been
// reminded yet. Blocks until ctx is cancelled; call with `go`.
func (s *Server) StartBookingReminderScheduler(ctx context.Context) {
	s.runBookingReminderCheck(ctx)
	ticker := time.NewTicker(bookingReminderSchedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runBookingReminderCheck(ctx)
		}
	}
}

func (s *Server) runBookingReminderCheck(ctx context.Context) {
	now := time.Now()
	due, err := s.Store.BookingsForReminder(ctx, now, now.Add(bookingReminderWindow))
	if err != nil {
		log.Printf("booking reminder scheduler: list due bookings: %v", err)
		return
	}
	for _, b := range due {
		if b.TravellerEmail == "" {
			continue
		}
		subject := "Your Spotto booking is coming up"
		body := fmt.Sprintf("<p>Your booking for <strong>%s</strong> is coming up on %s.</p><p>Code: <strong>%s</strong></p>",
			b.ServiceName, b.StartAt.Format("Jan 2, 3:04 PM"), b.Code)
		if err := s.Mail.Send(b.TravellerEmail, subject, body); err != nil {
			log.Printf("booking reminder scheduler: send booking=%s: %v", b.ID, err)
			continue
		}
		if err := s.Store.MarkBookingReminderSent(ctx, b.ID); err != nil {
			log.Printf("booking reminder scheduler: mark sent booking=%s: %v", b.ID, err)
		}
	}
}

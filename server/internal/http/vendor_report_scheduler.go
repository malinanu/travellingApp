package httpapi

import (
	"context"
	"fmt"
	"log"
	"time"
)

const vendorReportSchedulerInterval = 12 * time.Hour

// StartVendorReportScheduler periodically generates and emails weekly performance
// reports to approved vendors. Blocks until ctx is cancelled; call with `go`.
func (s *Server) StartVendorReportScheduler(ctx context.Context) {
	s.runVendorReportCheck(ctx)
	ticker := time.NewTicker(vendorReportSchedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runVendorReportCheck(ctx)
		}
	}
}

func (s *Server) runVendorReportCheck(ctx context.Context) {
	now := time.Now()
	if now.Weekday() != time.Monday {
		return
	}

	vendors, err := s.Store.Vendors(ctx, "approved")
	if err != nil {
		log.Printf("vendor report scheduler: list approved vendors: %v", err)
		return
	}

	for _, v := range vendors {
		owner, err := s.Store.UserByID(ctx, v.Owner)
		if err != nil || owner.Email == "" {
			continue
		}

		subject := fmt.Sprintf("Weekly Performance Report â€” %s", v.Name)
		body := fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
				<h2>Weekly Digest for %s</h2>
				<p>Here is your weekly activity summary on Spotto:</p>
				<ul>
					<li><strong>Status:</strong> %s</li>
					<li><strong>Tier:</strong> %s</li>
					<li><strong>Town:</strong> %s</li>
				</ul>
				<p>Log in to your vendor dashboard for full analytics and live offer stats.</p>
				<p><a href="%s/dashboard" style="display: inline-block; background: #0d9488; color: #fff; padding: 10px 20px; border-radius: 12px; text-decoration: none; font-weight: bold;">Go to Dashboard</a></p>
			</div>
		`, v.Name, v.Status, v.Tier, v.Town, s.VendorAppURL)

		if err := s.Mail.Send(owner.Email, subject, body); err != nil {
			log.Printf("vendor report scheduler: send vendor=%s: %v", v.ID, err)
			continue
		}
	}
}

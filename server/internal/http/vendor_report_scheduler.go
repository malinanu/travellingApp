package httpapi

import (
	"context"
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

		subject := "Weekly Performance Report — " + v.Name
		body := vendorReportEmailHTML(v, s.VendorAppURL+"/dashboard")

		if err := s.Mail.Send(owner.Email, subject, body); err != nil {
			log.Printf("vendor report scheduler: send vendor=%s: %v", v.ID, err)
			continue
		}
	}
}

package httpapi

// The BR-cert scheduler is the background worker behind Business
// Registration certificate verification: a vendor uploads their
// certificate (PDF or image, media role "br_cert" â€” see media.go), which
// queues an OCR pre-fill pass (internal/gemini) of the printed certificate
// number, which in turn queues an eROC registry check (internal/eroc) once
// a number is set â€” either the OCR guess or whatever the vendor typed/
// edited themselves. Both steps run here, off the request path, following
// the same ticker-sweep shape as weather_scheduler.go/holiday_scheduler.go
// (this codebase has no job-queue abstraction) so state lives in Postgres
// columns, not in-memory, and survives a restart mid-sweep.

import (
	"context"
	"errors"
	"io"
	"log"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/eroc"
	"github.com/spotto/server/internal/gemini"
)

const brCertSchedulerInterval = 2 * time.Minute

// brCertGeminiCallSpacing keeps a sweep comfortably under Gemini 2.5
// Flash's free-tier 10 requests/minute cap.
const brCertGeminiCallSpacing = 7 * time.Second

// brCertVerifyCallSpacing paces eROC calls gently even though it has no
// documented rate limit.
const brCertVerifyCallSpacing = 1 * time.Second

// brCertMaxVerifyAttempts bounds retries on transient eROC failures
// (network errors, non-200s) before giving up and surfacing the vendor for
// manual review â€” a permanently malformed/unreachable case must not retry
// forever.
const brCertMaxVerifyAttempts = 5

// StartBRCertScheduler blocks until ctx is cancelled; call with `go`.
func (s *Server) StartBRCertScheduler(ctx context.Context) {
	s.runBRCertSweep(ctx)
	ticker := time.NewTicker(brCertSchedulerInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runBRCertSweep(ctx)
		}
	}
}

func (s *Server) runBRCertSweep(ctx context.Context) {
	s.runBRCertOCRSweep(ctx)
	s.runBRCertVerifySweep(ctx)
}

// runBRCertOCRSweep pre-fills BRCertNumber for vendors whose uploaded
// br_cert doc hasn't been read yet.
func (s *Server) runBRCertOCRSweep(ctx context.Context) {
	if s.GeminiAPIKey == "" {
		return // unconfigured â€” vendors can still type the number in manually
	}
	vendors, err := s.Store.VendorsPendingBRCertOCR(ctx)
	if err != nil {
		log.Printf("br-cert scheduler: list pending OCR: %v", err)
		return
	}

	found, none, failed := 0, 0, 0
	for i, v := range vendors {
		if i > 0 {
			time.Sleep(brCertGeminiCallSpacing)
		}
		candidate, err := s.ocrVendorBRCert(ctx, v.ID)
		if err != nil {
			failed++
			if err := s.Store.SetVendorBRCertOCRError(ctx, v.ID); err != nil {
				log.Printf("br-cert scheduler: record OCR error for vendor %s: %v", v.ID, err)
			}
			continue
		}
		if candidate == "" {
			none++
		} else {
			found++
		}
		if err := s.Store.SetVendorBRCertOCRResult(ctx, v.ID, candidate); err != nil {
			log.Printf("br-cert scheduler: save OCR result for vendor %s: %v", v.ID, err)
		}
	}
	log.Printf("br-cert scheduler: OCR %d found, %d found nothing, %d failed (of %d pending)",
		found, none, failed, len(vendors))
}

// ocrVendorBRCert fetches the vendor's uploaded br_cert media and asks
// Gemini for the printed certificate number.
func (s *Server) ocrVendorBRCert(ctx context.Context, vendorID string) (string, error) {
	list, err := s.Store.MediaByOwner(ctx, "vendor", vendorID)
	if err != nil {
		return "", err
	}
	var doc *domain.Media
	for i := range list {
		if list[i].Role == "br_cert" {
			doc = &list[i]
			break
		}
	}
	if doc == nil {
		return "", errors.New("br-cert scheduler: no br_cert media found")
	}
	body, _, err := s.Media.Get(ctx, doc.ObjectKey)
	if err != nil {
		return "", err
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	return gemini.ExtractCertNumber(ctx, s.GeminiAPIKey, data, doc.ContentType)
}

// runBRCertVerifySweep checks every vendor with a certificate number
// awaiting eROC verification.
func (s *Server) runBRCertVerifySweep(ctx context.Context) {
	vendors, err := s.Store.VendorsPendingBRCertVerify(ctx)
	if err != nil {
		log.Printf("br-cert scheduler: list pending verify: %v", err)
		return
	}

	verified, invalid, reviewing, retrying := 0, 0, 0, 0
	for i, v := range vendors {
		if i > 0 {
			time.Sleep(brCertVerifyCallSpacing)
		}
		result, err := eroc.VerifyCompany(ctx, v.BRCertNumber)
		switch {
		case errors.Is(err, eroc.ErrInvalid):
			invalid++
			if err := s.Store.SetVendorBRCertVerifyResult(ctx, v.ID, domain.BRCertVerifyInvalid, "", result.Message); err != nil {
				log.Printf("br-cert scheduler: save invalid result for vendor %s: %v", v.ID, err)
			}
		case err != nil:
			attempts := v.BRCertAttempts + 1
			if attempts >= brCertMaxVerifyAttempts {
				reviewing++
				if err := s.Store.SetVendorBRCertVerifyResult(ctx, v.ID, domain.BRCertVerifyManualReview, "", err.Error()); err != nil {
					log.Printf("br-cert scheduler: save manual-review result for vendor %s: %v", v.ID, err)
				}
				continue
			}
			retrying++
			if err := s.Store.SetVendorBRCertVerifyAttempt(ctx, v.ID, attempts, err.Error()); err != nil {
				log.Printf("br-cert scheduler: save verify attempt for vendor %s: %v", v.ID, err)
			}
		default:
			verified++
			if err := s.Store.SetVendorBRCertVerifyResult(ctx, v.ID, domain.BRCertVerifyVerified, result.CompanyName, result.Message); err != nil {
				log.Printf("br-cert scheduler: save verified result for vendor %s: %v", v.ID, err)
			}
		}
	}
	log.Printf("br-cert scheduler: verify %d verified, %d invalid, %d manual review, %d retrying (of %d pending)",
		verified, invalid, reviewing, retrying, len(vendors))
}

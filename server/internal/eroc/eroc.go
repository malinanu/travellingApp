// Package eroc calls Sri Lanka's DRC eROC company-verification endpoint —
// a public registry check, no API key required.
package eroc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const verifyURL = "https://erocapiv2.drc.gov.lk/api/v1/verify-company"

// ErrInvalid means the call succeeded but eROC reported the certificate
// number as unknown/invalid (status:false) — a definitive result. Callers
// must record it and stop, never retry.
var ErrInvalid = errors.New("eroc: certificate reported invalid")

type Result struct {
	Message         string
	CompanyName     string
	IncorporationAt string
	Postfix         string
}

// VerifyCompany calls eROC's verify-company endpoint for certNo. A non-nil
// error other than ErrInvalid (network failure, non-200, bad JSON) is
// transient — the caller applies its own bounded retry.
func VerifyCompany(ctx context.Context, certNo string) (*Result, error) {
	// Bounded independently of the caller's context — a slow/hanging eROC
	// response shouldn't hold the scheduler's sweep goroutine open
	// indefinitely.
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	body, err := json.Marshal(map[string]string{"certificateNo": certNo})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, verifyURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eroc: request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eroc: unexpected status %d", res.StatusCode)
	}

	var raw struct {
		Message         string `json:"message"`
		CompanyName     string `json:"company_name"`
		IncorporationAt string `json:"incorporation_at"`
		Status          bool   `json:"status"`
		Postfix         string `json:"postfix"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("eroc: decode response: %w", err)
	}

	result := &Result{
		Message:         raw.Message,
		CompanyName:     raw.CompanyName,
		IncorporationAt: raw.IncorporationAt,
		Postfix:         raw.Postfix,
	}
	if !raw.Status {
		return result, ErrInvalid
	}
	return result, nil
}

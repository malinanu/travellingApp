package placedetails

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// detailsResponse mirrors only the fields of Google's Place Details
// (Legacy) response (https://maps.googleapis.com/maps/api/place/details/json)
// that Details needs — see
// https://developers.google.com/maps/documentation/places/web-service/details.
type detailsResponse struct {
	Result struct {
		FormattedPhoneNumber         string `json:"formatted_phone_number"`
		Website                      string `json:"website"`
		PriceLevel                   *int   `json:"price_level"`
		WheelchairAccessibleEntrance *bool  `json:"wheelchair_accessible_entrance"`
	} `json:"result"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
}

// Fetch calls the Places Details endpoint for one Google place ID,
// requesting only the fields Details needs (field masking keeps the
// per-call cost down) — this is the only real network call in this
// package; internal/http wraps it with the two-tier cache.
func Fetch(ctx context.Context, apiKey, googlePlaceID string) (*Details, error) {
	q := url.Values{
		"place_id": {googlePlaceID},
		"fields":   {"formatted_phone_number,website,price_level,wheelchair_accessible_entrance"},
		"key":      {apiKey},
	}
	reqURL := "https://maps.googleapis.com/maps/api/place/details/json?" + q.Encode()

	// Bounded independently of the caller's own context — a slow/hanging
	// Google response shouldn't be able to hold this goroutine open
	// indefinitely just because the request's context has no deadline.
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("places details: unexpected status %d", res.StatusCode)
	}

	var parsed detailsResponse
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("places details: decode response: %w", err)
	}
	if parsed.Status != "OK" {
		return nil, fmt.Errorf("places details: status %s: %s", parsed.Status, parsed.ErrorMessage)
	}

	return &Details{
		Phone:                parsed.Result.FormattedPhoneNumber,
		Website:              parsed.Result.Website,
		WheelchairAccessible: parsed.Result.WheelchairAccessibleEntrance,
		PriceLevel:           priceLevelString(parsed.Result.PriceLevel),
	}, nil
}

// priceLevelString maps Google's 0-4 price_level scale to the app's
// existing $/$$/$$$ display convention.
func priceLevelString(level *int) string {
	if level == nil {
		return ""
	}
	switch *level {
	case 0:
		return "Free"
	case 1:
		return "$"
	case 2:
		return "$$"
	case 3:
		return "$$$"
	case 4:
		return "$$$$"
	default:
		return ""
	}
}

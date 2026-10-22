// Package googleplaces finds real-world places by name through Google
// Places API (New) Text Search, and fetches their photos. It works
// anywhere in the world, unlike the Sri Lanka-only Spotto catalog.
package googleplaces

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const defaultBaseURL = "https://places.googleapis.com"

// fieldMask asks only for what Spotto shows, which also keeps the request
// on a cheaper billing tier.
const fieldMask = "places.id,places.displayName,places.location,places.formattedAddress," +
	"places.rating,places.userRatingCount,places.primaryType,places.types,places.photos," +
	"places.googleMapsUri,places.addressComponents"

type Client struct {
	APIKey  string
	BaseURL string // tests point this at an httptest server
	HTTP    *http.Client
}

func New(apiKey string) *Client {
	return &Client{APIKey: apiKey, BaseURL: defaultBaseURL, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

// Place is one Text Search result.
type Place struct {
	ID          string
	Name        string
	Lat         float64
	Lng         float64
	Address     string
	City        string
	CountryCode string
	Rating      *float64
	RatingCount int
	PrimaryType string
	Types       []string
	PhotoName   string
	MapsURL     string
	Summary     string // Google's one-line description; SearchTextMany only
}

// ErrNoResult means Google found nothing for the query.
var ErrNoResult = errors.New("googleplaces: no result")

// SearchText returns the best match for a free-text query such as
// "Kata Noodle House, Kata Beach, Phuket, Thailand".
func (c *Client) SearchText(ctx context.Context, query string) (Place, error) {
	places, err := c.search(ctx, query, 1, fieldMask)
	if err != nil {
		return Place{}, err
	}
	return places[0], nil
}

// SearchTextMany returns up to limit results for a broader query such as
// "best viewpoints near Sigiriya, Sri Lanka", with Google's short editorial
// summary of each (a pricier field, so the single-result search skips it).
func (c *Client) SearchTextMany(ctx context.Context, query string, limit int) ([]Place, error) {
	return c.search(ctx, query, min(20, max(1, limit)), fieldMask+",places.editorialSummary")
}

func (c *Client) search(ctx context.Context, query string, pageSize int, mask string) ([]Place, error) {
	body, _ := json.Marshal(map[string]any{
		"textQuery":    query,
		"pageSize":     pageSize,
		"languageCode": "en",
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/places:searchText", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", c.APIKey)
	req.Header.Set("X-Goog-FieldMask", mask)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("googleplaces: search: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
		return nil, fmt.Errorf("googleplaces: search: HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(msg)))
	}
	var out struct {
		Places []struct {
			ID          string `json:"id"`
			DisplayName struct {
				Text string `json:"text"`
			} `json:"displayName"`
			Location struct {
				Latitude  float64 `json:"latitude"`
				Longitude float64 `json:"longitude"`
			} `json:"location"`
			FormattedAddress string   `json:"formattedAddress"`
			Rating           *float64 `json:"rating"`
			UserRatingCount  int      `json:"userRatingCount"`
			PrimaryType      string   `json:"primaryType"`
			Types            []string `json:"types"`
			GoogleMapsURI    string   `json:"googleMapsUri"`
			EditorialSummary struct {
				Text string `json:"text"`
			} `json:"editorialSummary"`
			Photos []struct {
				Name string `json:"name"`
			} `json:"photos"`
			AddressComponents []struct {
				LongText  string   `json:"longText"`
				ShortText string   `json:"shortText"`
				Types     []string `json:"types"`
			} `json:"addressComponents"`
		} `json:"places"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("googleplaces: decode: %w", err)
	}
	if len(out.Places) == 0 {
		return nil, ErrNoResult
	}
	places := make([]Place, 0, len(out.Places))
	for _, g := range out.Places {
		p := Place{
			ID: g.ID, Name: g.DisplayName.Text, Lat: g.Location.Latitude, Lng: g.Location.Longitude,
			Address: g.FormattedAddress, Rating: g.Rating, RatingCount: g.UserRatingCount,
			PrimaryType: g.PrimaryType, Types: g.Types, MapsURL: g.GoogleMapsURI,
			Summary: g.EditorialSummary.Text,
		}
		if len(g.Photos) > 0 {
			p.PhotoName = g.Photos[0].Name
		}
		for _, c := range g.AddressComponents {
			for _, t := range c.Types {
				switch t {
				case "country":
					p.CountryCode = c.ShortText
				case "locality":
					p.City = c.LongText
				case "administrative_area_level_1":
					if p.City == "" {
						p.City = c.LongText
					}
				}
			}
		}
		places = append(places, p)
	}
	return places, nil
}

// Photo fetches a place photo (as named in Place.PhotoName) at up to
// maxWidth pixels and returns its bytes and content type.
func (c *Client) Photo(ctx context.Context, photoName string, maxWidth int) ([]byte, string, error) {
	u := fmt.Sprintf("%s/v1/%s/media?maxWidthPx=%d&key=%s", c.BaseURL, photoName, maxWidth, url.QueryEscape(c.APIKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	res, err := c.HTTP.Do(req) // follows Google's redirect to the image
	if err != nil {
		return nil, "", fmt.Errorf("googleplaces: photo: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("googleplaces: photo: HTTP %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, 5<<20))
	if err != nil {
		return nil, "", err
	}
	return data, res.Header.Get("Content-Type"), nil
}

// Kind maps Google place types to Spotto's simple kinds:
// eat | drink | stay | shop | see | do. It returns "" when nothing fits.
func Kind(primary string, types []string) string {
	for _, t := range append([]string{primary}, types...) {
		switch {
		case t == "":
			continue
		case t == "bar" || t == "pub" || t == "night_club" || t == "wine_bar" || t == "cocktail_bar":
			return "drink"
		case strings.HasSuffix(t, "_restaurant") || t == "restaurant" || t == "cafe" || t == "coffee_shop" ||
			t == "bakery" || t == "food" || t == "meal_takeaway" || t == "meal_delivery" ||
			t == "food_court" || t == "ice_cream_shop" || t == "dessert_shop" || t == "tea_house":
			return "eat"
		case t == "lodging" || t == "hotel" || t == "resort_hotel" || t == "hostel" || t == "guest_house" ||
			t == "bed_and_breakfast" || t == "motel" || t == "campground":
			return "stay"
		case t == "shopping_mall" || t == "market" || strings.HasSuffix(t, "_store") || t == "store":
			return "shop"
		case t == "tourist_attraction" || t == "museum" || t == "beach" || t == "park" || t == "national_park" ||
			t == "hindu_temple" || t == "buddhist_temple" || t == "place_of_worship" || t == "church" ||
			t == "mosque" || t == "art_gallery" || t == "historical_landmark" || t == "monument" ||
			t == "observation_deck" || t == "zoo" || t == "aquarium" || t == "natural_feature":
			return "see"
		case t == "spa" || t == "gym" || t == "amusement_park" || t == "water_park" || t == "marina" ||
			t == "travel_agency" || t == "tour_agency" || t == "diving_center":
			return "do"
		}
	}
	return ""
}

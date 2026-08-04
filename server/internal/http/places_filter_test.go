package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func TestHandlePlacesFilters(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()

	p1Data, _ := json.Marshal(map[string]any{
		"name":        "Filter Test Cafe",
		"category":    "cafe",
		"description": "Artisanal coffee and co-working",
		"town":        "Weligama",
		"lat":         5.972,
		"lng":         80.428,
	})
	p2Data, _ := json.Marshal(map[string]any{
		"name":        "Filter Test Surf",
		"category":    "sports",
		"description": "Surfboard rentals and lessons",
		"town":        "Arugam Bay",
		"lat":         6.841,
		"lng":         81.836,
	})

	_ = store.UpsertPlace(ctx, domain.Place{ID: "p_filter_1", Name: "Filter Test Cafe", Data: p1Data})
	_ = store.UpsertPlace(ctx, domain.Place{ID: "p_filter_2", Name: "Filter Test Surf", Data: p2Data})

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	t.Cleanup(srv.Close)

	// Test category filter
	t.Run("Category filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/places?category=cafe", nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var list []map[string]any
		json.NewDecoder(res.Body).Decode(&list)
		found := false
		for _, item := range list {
			if item["id"] == "p_filter_1" {
				found = true
			}
			if item["id"] == "p_filter_2" {
				t.Errorf("p_filter_2 should have been filtered out by category=cafe")
			}
		}
		if !found {
			t.Errorf("expected p_filter_1 in category=cafe results")
		}
	})

	// Test search query filter
	t.Run("Search query filter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/places?q=Surfboard", nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200, got %d", res.StatusCode)
		}
		var list []map[string]any
		json.NewDecoder(res.Body).Decode(&list)
		found := false
		for _, item := range list {
			if item["id"] == "p_filter_2" {
				found = true
			}
			if item["id"] == "p_filter_1" {
				t.Errorf("p_filter_1 should have been filtered out by search q=Surfboard")
			}
		}
		if !found {
			t.Errorf("expected p_filter_2 in search results")
		}
	})
}

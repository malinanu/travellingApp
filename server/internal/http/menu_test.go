package httpapi_test

// End-to-end proof of the vendor-menu round trip that a vendor reported as
// "I edit the menu, nothing changes, and the app doesn't update":
//
//  1. An approved vendor's PUT /vendor/menu persists (GET returns it back).
//  2. The save mirrors the active preset into the traveller-facing place, so
//     GET /places (what the Flutter app reads) shows the new items — proving
//     syncVendorPlace runs on menu save and placeJSON serves the snapshot.
//
// Same real-Postgres pattern as the other integration tests; skips without a
// local database.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/store/postgres"
)

func TestVendorMenuRoundTripToTraveller(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	purgeMenuTestData(t, store, ctx)
	t.Cleanup(func() { purgeMenuTestData(t, store, ctx) })

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")

	vendor := domain.Vendor{
		ID: "ven_menu_test", Name: "Menu Test Cafe", Category: "cafe", Town: "Ella",
		Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now,
		Lat: 6.8667, Lng: 81.0466,
	}
	must(t, store.UpsertVendor(ctx, vendor))
	must(t, store.CreateUser(ctx, domain.User{
		ID: "usr_menu_owner", Email: "owner@menutest.dev", PasswordHash: pw,
		Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now,
	}))
	must(t, store.CreateUser(ctx, domain.User{
		ID: "usr_menu_traveller", Email: "traveller@menutest.dev", PasswordHash: pw,
		Role: domain.RoleTraveller, CreatedAt: now,
	}))

	ownerToken := loginAs(t, srv, "owner@menutest.dev", "testpass123")
	travellerToken := loginAs(t, srv, "traveller@menutest.dev", "testpass123")

	// The document the web Menu manager PUTs: one active preset, one section,
	// one item, no per-day restriction (so it shows every day).
	menuDoc := map[string]any{
		"activePresetId": "preset_main",
		"presets": []any{
			map[string]any{
				"id": "preset_main", "name": "Main menu",
				"sections": []any{
					map[string]any{
						"id": "sec_mains", "name": "Mains",
						"items": []any{
							map[string]any{"id": "itm_kottu", "name": "Kottu Roti", "price": 850, "desc": "Spicy", "icon": "UtensilsCrossed"},
						},
					},
				},
			},
		},
	}

	t.Run("approved vendor can save the menu", func(t *testing.T) {
		if got := doReq(t, srv, http.MethodPut, "/vendor/menu", ownerToken, menuDoc); got != http.StatusOK {
			t.Fatalf("PUT /vendor/menu: want 200, got %d", got)
		}
	})

	t.Run("GET /vendor/menu returns the saved document", func(t *testing.T) {
		var doc struct {
			ActivePresetID string `json:"activePresetId"`
			Presets        []struct {
				ID       string `json:"id"`
				Sections []struct {
					Name  string `json:"name"`
					Items []struct {
						Name  string `json:"name"`
						Price int    `json:"price"`
					} `json:"items"`
				} `json:"sections"`
			} `json:"presets"`
		}
		if got := doReqJSON(t, srv, http.MethodGet, "/vendor/menu", ownerToken, nil, &doc); got != http.StatusOK {
			t.Fatalf("GET /vendor/menu: want 200, got %d", got)
		}
		if len(doc.Presets) != 1 || len(doc.Presets[0].Sections) != 1 || len(doc.Presets[0].Sections[0].Items) != 1 {
			t.Fatalf("menu did not persist: %+v", doc)
		}
		if doc.Presets[0].Sections[0].Items[0].Name != "Kottu Roti" {
			t.Errorf("item name: want Kottu Roti, got %q", doc.Presets[0].Sections[0].Items[0].Name)
		}
	})

	t.Run("saved menu appears in the traveller place feed", func(t *testing.T) {
		var places []map[string]any
		if got := doReqJSON(t, srv, http.MethodGet, "/places", travellerToken, nil, &places); got != http.StatusOK {
			t.Fatalf("GET /places: want 200, got %d", got)
		}
		var place map[string]any
		for _, p := range places {
			if p["vendorId"] == vendor.ID {
				place = p
				break
			}
		}
		if place == nil {
			t.Fatal("vendor's place is not in the traveller feed — syncVendorPlace did not create/link it")
		}
		menu, ok := place["menu"].([]any)
		if !ok || len(menu) == 0 {
			t.Fatalf("place has no menu — syncVendorPlace did not propagate it; menu=%v", place["menu"])
		}
		sec, _ := menu[0].(map[string]any)
		if sec["name"] != "Mains" {
			t.Errorf("section name: want Mains, got %v", sec["name"])
		}
		items, _ := sec["items"].([]any)
		if len(items) == 0 {
			t.Fatal("section has no items in the traveller feed")
		}
		item, _ := items[0].(map[string]any)
		if item["name"] != "Kottu Roti" {
			t.Errorf("traveller-facing item name: want Kottu Roti, got %v", item["name"])
		}
	})
}

func purgeMenuTestData(t *testing.T, store *postgres.Store, ctx context.Context) {
	t.Helper()
	for _, q := range []string{
		`DELETE FROM places WHERE vendor_id LIKE 'ven_menu_%'`,
		`DELETE FROM users WHERE id LIKE 'usr_menu_%'`,
		`DELETE FROM vendors WHERE id LIKE 'ven_menu_%'`,
	} {
		if _, err := store.Pool.Exec(ctx, q); err != nil {
			t.Fatalf("purge (%s): %v", q, err)
		}
	}
}

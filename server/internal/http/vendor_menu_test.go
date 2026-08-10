package httpapi_test

// Integration tests for vendor menu presets + per-day item/section
// availability (vendor.go's effectiveMenuSections/availableToday). Same
// real-Postgres pattern as validate_test.go; skips automatically if it
// can't connect.

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
)

type menuTestItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type menuTestSection struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Items []menuTestItem `json:"items"`
}

func TestVendorMenuPresetsAndAvailability(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	testTOTPKey := make([]byte, 32)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: testTOTPKey, VendorAppURL: "http://localhost:3000"}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")
	vendor := domain.Vendor{ID: "ven_menu_test", Name: "Menu Test Vendor", Category: "cafe", Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, vendor))
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_menu_owner", Email: "owner@menutest.dev", PasswordHash: pw, Role: domain.RoleVendor, VendorID: &vendor.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))
	ownerToken := loginAs(t, srv, "owner@menutest.dev", "testpass123")

	weekdayKeys := [...]string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
	todayKey := weekdayKeys[now.Weekday()]
	var otherKey string
	for _, k := range weekdayKeys {
		if k != todayKey {
			otherKey = k
			break
		}
	}

	var placeID string
	getMenu := func(t *testing.T) []menuTestSection {
		t.Helper()
		if placeID == "" {
			store.Pool.QueryRow(ctx, `SELECT id FROM places WHERE vendor_id=$1`, vendor.ID).Scan(&placeID)
		}
		var out struct {
			Menu []menuTestSection `json:"menu"`
		}
		if got := doReqJSON(t, srv, http.MethodGet, "/places/"+placeID, ownerToken, nil, &out); got != http.StatusOK {
			t.Fatalf("GET /places/%s: want 200, got %d", placeID, got)
		}
		return out.Menu
	}

	t.Run("legacy flat sections still flatten unfiltered", func(t *testing.T) {
		status := doReq(t, srv, http.MethodPut, "/vendor/menu", ownerToken, map[string]any{
			"sections": []map[string]any{
				{"id": "s1", "name": "Coffee", "items": []map[string]any{
					{"id": "i1", "name": "Latte", "price": 500, "desc": "", "icon": "Coffee"},
				}},
			},
		})
		if status != http.StatusOK {
			t.Fatalf("PUT /vendor/menu (legacy): want 200, got %d", status)
		}
		menu := getMenu(t)
		if len(menu) != 1 || len(menu[0].Items) != 1 || menu[0].Items[0].Name != "Latte" {
			t.Fatalf("expected legacy single-section menu to flatten unfiltered, got %+v", menu)
		}
	})

	t.Run("only the active preset's sections appear", func(t *testing.T) {
		status := doReq(t, srv, http.MethodPut, "/vendor/menu", ownerToken, map[string]any{
			"activePresetId": "p_lunch",
			"presets": []map[string]any{
				{"id": "p_breakfast", "name": "Breakfast", "sections": []map[string]any{
					{"id": "s_b", "name": "Eggs", "items": []map[string]any{
						{"id": "i_b", "name": "Omelette", "price": 800, "desc": "", "icon": "EggFried"},
					}},
				}},
				{"id": "p_lunch", "name": "Lunch", "sections": []map[string]any{
					{"id": "s_l", "name": "Mains", "items": []map[string]any{
						{"id": "i_l", "name": "Sandwich", "price": 900, "desc": "", "icon": "Sandwich"},
					}},
				}},
			},
		})
		if status != http.StatusOK {
			t.Fatalf("PUT /vendor/menu (presets): want 200, got %d", status)
		}
		menu := getMenu(t)
		if len(menu) != 1 || menu[0].Name != "Mains" {
			t.Fatalf("expected only the Lunch preset's section, got %+v", menu)
		}
	})

	t.Run("unmatched activePresetId falls back to the first preset", func(t *testing.T) {
		status := doReq(t, srv, http.MethodPut, "/vendor/menu", ownerToken, map[string]any{
			"activePresetId": "does_not_exist",
			"presets": []map[string]any{
				{"id": "p_breakfast", "name": "Breakfast", "sections": []map[string]any{
					{"id": "s_b", "name": "Eggs", "items": []map[string]any{
						{"id": "i_b", "name": "Omelette", "price": 800, "desc": "", "icon": "EggFried"},
					}},
				}},
				{"id": "p_lunch", "name": "Lunch", "sections": []map[string]any{
					{"id": "s_l", "name": "Mains", "items": []map[string]any{
						{"id": "i_l", "name": "Sandwich", "price": 900, "desc": "", "icon": "Sandwich"},
					}},
				}},
			},
		})
		if status != http.StatusOK {
			t.Fatalf("PUT /vendor/menu: want 200, got %d", status)
		}
		menu := getMenu(t)
		if len(menu) != 1 || menu[0].Name != "Eggs" {
			t.Fatalf("expected fallback to the first preset (Breakfast), got %+v", menu)
		}
	})

	t.Run("item without today marked true is excluded, item with today marked true is kept", func(t *testing.T) {
		status := doReq(t, srv, http.MethodPut, "/vendor/menu", ownerToken, map[string]any{
			"activePresetId": "p_main",
			"presets": []map[string]any{
				{"id": "p_main", "name": "Main", "sections": []map[string]any{
					{"id": "s_specials", "name": "Specials", "items": []map[string]any{
						{"id": "i_today", "name": "Today Special", "price": 700, "desc": "", "icon": "Soup",
							"availableDays": map[string]any{todayKey: true}},
						{"id": "i_other", "name": "Other Day Special", "price": 700, "desc": "", "icon": "Soup",
							"availableDays": map[string]any{otherKey: true}},
					}},
				}},
			},
		})
		if status != http.StatusOK {
			t.Fatalf("PUT /vendor/menu: want 200, got %d", status)
		}
		menu := getMenu(t)
		if len(menu) != 1 || len(menu[0].Items) != 1 || menu[0].Items[0].Name != "Today Special" {
			t.Fatalf("expected only today's item to survive day filtering, got %+v", menu)
		}
	})

	t.Run("section-level availableDays excludes the whole section on a non-matching day", func(t *testing.T) {
		status := doReq(t, srv, http.MethodPut, "/vendor/menu", ownerToken, map[string]any{
			"activePresetId": "p_main",
			"presets": []map[string]any{
				{"id": "p_main", "name": "Main", "sections": []map[string]any{
					{"id": "s_other_only", "name": "Other Day Section", "availableDays": map[string]any{otherKey: true}, "items": []map[string]any{
						{"id": "i_x", "name": "X", "price": 100, "desc": "", "icon": "Coffee"},
					}},
					{"id": "s_everyday", "name": "Everyday", "items": []map[string]any{
						{"id": "i_everyday", "name": "Everyday Item", "price": 500, "desc": "", "icon": "Coffee"},
					}},
				}},
			},
		})
		if status != http.StatusOK {
			t.Fatalf("PUT /vendor/menu: want 200, got %d", status)
		}
		menu := getMenu(t)
		if len(menu) != 1 || menu[0].Name != "Everyday" {
			t.Fatalf("expected the non-matching-day section to be excluded entirely, got %+v", menu)
		}
	})

	t.Run("a section whose every item is filtered out for today is itself dropped", func(t *testing.T) {
		status := doReq(t, srv, http.MethodPut, "/vendor/menu", ownerToken, map[string]any{
			"activePresetId": "p_main",
			"presets": []map[string]any{
				{"id": "p_main", "name": "Main", "sections": []map[string]any{
					{"id": "s_all_other_day", "name": "Weekend Only", "items": []map[string]any{
						{"id": "i_weekend", "name": "Weekend Item", "price": 700, "desc": "", "icon": "Soup",
							"availableDays": map[string]any{otherKey: true}},
					}},
					{"id": "s_everyday", "name": "Everyday", "items": []map[string]any{
						{"id": "i_everyday", "name": "Everyday Item", "price": 500, "desc": "", "icon": "Coffee"},
					}},
				}},
			},
		})
		if status != http.StatusOK {
			t.Fatalf("PUT /vendor/menu: want 200, got %d", status)
		}
		menu := getMenu(t)
		if len(menu) != 1 || menu[0].Name != "Everyday" {
			t.Fatalf("expected the fully-filtered-out section to be dropped, got %+v", menu)
		}
	})

	// cleanup so repeat runs stay idempotent
	store.Pool.Exec(ctx, `DELETE FROM places WHERE vendor_id='ven_menu_test'`)
	store.Pool.Exec(ctx, `DELETE FROM users WHERE id='usr_menu_owner'`)
	store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id='ven_menu_test'`)
}

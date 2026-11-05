package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestVendorPackagesEndpoints(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)
	ctx := context.Background()

	tokens := auth.NewTokens("test-packages-secret-key-12345678")
	cfg := config.Config{
		CORSOrigins: []string{"*"},
	}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	t.Cleanup(srv.Close)

	adminUser := domain.User{
		ID:        "usr_pkg_admin",
		Email:     "pkgadmin@example.dev",
		Name:      "Package Admin",
		Role:      domain.RoleAdmin,
		CreatedAt: time.Now(),
	}
	travellerUser := domain.User{
		ID:        "usr_pkg_traveller",
		Email:     "pkgtrav@example.dev",
		Name:      "Package Traveller",
		Role:      domain.RoleTraveller,
		CreatedAt: time.Now(),
	}

	_ = store.DeleteUser(ctx, "usr_pkg_admin")
	_ = store.DeleteUser(ctx, "usr_pkg_traveller")

	if err := store.CreateUser(ctx, adminUser); err != nil {
		t.Fatalf("create admin user: %v", err)
	}
	if err := store.CreateUser(ctx, travellerUser); err != nil {
		t.Fatalf("create traveller user: %v", err)
	}

	t.Cleanup(func() {
		_ = store.DeleteUser(ctx, "usr_pkg_admin")
		_ = store.DeleteUser(ctx, "usr_pkg_traveller")
	})

	adminToken, err := tokens.Issue(adminUser)
	if err != nil {
		t.Fatalf("issue admin token: %v", err)
	}
	travellerToken, err := tokens.Issue(travellerUser)
	if err != nil {
		t.Fatalf("issue traveller token: %v", err)
	}

	// 1. Test public GET /packages
	t.Run("public GET /packages returns 3 tiers", func(t *testing.T) {
		res, err := http.Get(srv.URL + "/packages")
		if err != nil {
			t.Fatalf("GET /packages failed: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", res.StatusCode)
		}

		var cfg domain.PackagesConfig
		if err := json.NewDecoder(res.Body).Decode(&cfg); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(cfg.Packages) != 3 {
			t.Fatalf("expected 3 packages, got %d", len(cfg.Packages))
		}

		starter := cfg.Packages[0]
		growth := cfg.Packages[1]
		pro := cfg.Packages[2]

		if starter.MonthlyPriceUSD != 3.99 {
			t.Errorf("expected Starter monthly price 3.99, got %v", starter.MonthlyPriceUSD)
		}
		if growth.MonthlyPriceUSD != 11.99 {
			t.Errorf("expected Growth monthly price 11.99, got %v", growth.MonthlyPriceUSD)
		}
		if pro.MonthlyPriceUSD != 22.99 {
			t.Errorf("expected Pro monthly price 22.99, got %v", pro.MonthlyPriceUSD)
		}

		if growth.Badge != "Most Popular" {
			t.Errorf("expected Growth badge 'Most Popular', got %q", growth.Badge)
		}
		if !growth.Highlighted {
			t.Errorf("expected Growth tier to be highlighted")
		}

		// Verify feature pool presence
		if len(cfg.FeaturePool) < 15 {
			t.Errorf("expected at least 15 features in master pool, got %d", len(cfg.FeaturePool))
		}
	})

	// 2. Test public alias GET /api/v1/packages
	t.Run("public GET /api/v1/packages alias works", func(t *testing.T) {
		res, err := http.Get(srv.URL + "/api/v1/packages")
		if err != nil {
			t.Fatalf("GET /api/v1/packages failed: %v", err)
		}
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", res.StatusCode)
		}

		var cfg domain.PackagesConfig
		if err := json.NewDecoder(res.Body).Decode(&cfg); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if len(cfg.Packages) != 3 {
			t.Fatalf("expected 3 packages on alias endpoint, got %d", len(cfg.Packages))
		}
	})

	// 3. Test admin authorization checks on GET /admin/packages
	t.Run("admin auth checks on GET /admin/packages", func(t *testing.T) {
		// Unauthenticated
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/packages", nil)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", res.StatusCode)
		}

		// Non-admin role (traveller)
		req, _ = http.NewRequest(http.MethodGet, srv.URL+"/admin/packages", nil)
		req.Header.Set("Authorization", "Bearer "+travellerToken)
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 Forbidden, got %d", res.StatusCode)
		}

		// Admin role
		req, _ = http.NewRequest(http.MethodGet, srv.URL+"/admin/packages", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK for admin, got %d", res.StatusCode)
		}
	})

	// 4. Test admin PUT /admin/packages (drag-and-drop / limit & price update)
	t.Run("admin PUT /admin/packages saves updated config", func(t *testing.T) {
		// Get current config
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/packages", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var currentCfg domain.PackagesConfig
		_ = json.NewDecoder(res.Body).Decode(&currentCfg)
		res.Body.Close()

		// Simulate dragging a feature and updating prices
		currentCfg.Packages[0].MonthlyPriceUSD = 4.49 // update Starter price
		currentCfg.Packages[1].MonthlyPriceUSD = 12.99
		// Add an extra feature to Starter
		currentCfg.Packages[0].Features = append(currentCfg.Packages[0].Features, domain.PackageFeature{
			ID:          "feat_custom_dragged",
			Name:        "Custom Dragged Perk",
			Description: "Feature moved from pool to Starter tier via Kanban board",
			Category:    "marketing",
			Icon:        "Sparkles",
		})

		bodyBytes, _ := json.Marshal(currentCfg)
		putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/admin/packages", bytes.NewReader(bodyBytes))
		putReq.Header.Set("Authorization", "Bearer "+adminToken)
		putReq.Header.Set("Content-Type", "application/json")
		putRes, err := http.DefaultClient.Do(putReq)
		if err != nil {
			t.Fatal(err)
		}
		defer putRes.Body.Close()

		if putRes.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK on PUT, got %d", putRes.StatusCode)
		}

		// Verify via public GET /packages
		pubRes, err := http.Get(srv.URL + "/packages")
		if err != nil {
			t.Fatal(err)
		}
		defer pubRes.Body.Close()
		var updatedCfg domain.PackagesConfig
		_ = json.NewDecoder(pubRes.Body).Decode(&updatedCfg)

		if updatedCfg.Packages[0].MonthlyPriceUSD != 4.49 {
			t.Errorf("expected updated Starter price 4.49, got %v", updatedCfg.Packages[0].MonthlyPriceUSD)
		}

		hasCustomPerk := false
		for _, f := range updatedCfg.Packages[0].Features {
			if f.ID == "feat_custom_dragged" {
				hasCustomPerk = true
				break
			}
		}
		if !hasCustomPerk {
			t.Errorf("expected dragged feature feat_custom_dragged in Starter features")
		}
	})

	// 5. Test validation on PUT /admin/packages
	t.Run("admin PUT /admin/packages validation errors", func(t *testing.T) {
		// Empty packages
		badCfg := domain.PackagesConfig{Packages: []domain.VendorPackage{}}
		bodyBytes, _ := json.Marshal(badCfg)
		putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/admin/packages", bytes.NewReader(bodyBytes))
		putReq.Header.Set("Authorization", "Bearer "+adminToken)
		putReq.Header.Set("Content-Type", "application/json")
		putRes, err := http.DefaultClient.Do(putReq)
		if err != nil {
			t.Fatal(err)
		}
		putRes.Body.Close()
		if putRes.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for empty packages, got %d", putRes.StatusCode)
		}

		// Negative price
		badPriceCfg := domain.DefaultPackagesConfig()
		badPriceCfg.Packages[0].MonthlyPriceUSD = -10.0
		bodyBytes, _ = json.Marshal(badPriceCfg)
		putReq, _ = http.NewRequest(http.MethodPut, srv.URL+"/admin/packages", bytes.NewReader(bodyBytes))
		putReq.Header.Set("Authorization", "Bearer "+adminToken)
		putReq.Header.Set("Content-Type", "application/json")
		putRes, err = http.DefaultClient.Do(putReq)
		if err != nil {
			t.Fatal(err)
		}
		putRes.Body.Close()
		if putRes.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for negative price, got %d", putRes.StatusCode)
		}

		// Duplicate slug
		dupSlugCfg := domain.DefaultPackagesConfig()
		dupSlugCfg.Packages[1].Slug = dupSlugCfg.Packages[0].Slug
		bodyBytes, _ = json.Marshal(dupSlugCfg)
		putReq, _ = http.NewRequest(http.MethodPut, srv.URL+"/admin/packages", bytes.NewReader(bodyBytes))
		putReq.Header.Set("Authorization", "Bearer "+adminToken)
		putReq.Header.Set("Content-Type", "application/json")
		putRes, err = http.DefaultClient.Do(putReq)
		if err != nil {
			t.Fatal(err)
		}
		putRes.Body.Close()
		if putRes.StatusCode != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for duplicate slug, got %d", putRes.StatusCode)
		}
	})

	// 6. Test creating a 4th tier and shuffling package order
	t.Run("admin creates 4th package tier, shuffles package order, and verifies public API", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/packages", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var currentCfg domain.PackagesConfig
		_ = json.NewDecoder(res.Body).Decode(&currentCfg)
		res.Body.Close()

		// 1. Create a 4th tier (Enterprise)
		enterprise := domain.VendorPackage{
			ID:                 "pkg_enterprise",
			Slug:               "enterprise",
			Name:               "Enterprise",
			Tagline:            "For hotel chains and multi-location venues",
			MonthlyPriceUSD:    49.99,
			YearlyPriceUSD:     499.90,
			Badge:              "VIP Exclusive",
			Highlighted:        true,
			ActiveOffers:       -1,
			FlashSalesPerMonth: 10,
			TeamSeats:          20,
			Features: []domain.PackageFeature{
				{
					ID:          "feat_map_pin",
					Name:        "Verified Map Pin & Profile",
					Description: "Map pin",
					Category:    "discovery",
					Icon:        "MapPin",
				},
				{
					ID:          "feat_custom_enterprise",
					Name:        "Custom White-Glove Support",
					Description: "Dedicated account manager",
					Category:    "trust",
					Icon:        "ShieldCheck",
				},
			},
			SortOrder: 4,
		}

		// 2. Shuffle order: [Pro, Enterprise, Growth, Starter]
		var pro, growth, starter domain.VendorPackage
		for _, p := range currentCfg.Packages {
			switch p.Slug {
			case "pro":
				pro = p
			case "growth":
				growth = p
			case "starter":
				starter = p
			}
		}
		shuffledPackages := []domain.VendorPackage{pro, enterprise, growth, starter}
		currentCfg.Packages = shuffledPackages

		bodyBytes, _ := json.Marshal(currentCfg)
		putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/admin/packages", bytes.NewReader(bodyBytes))
		putReq.Header.Set("Authorization", "Bearer "+adminToken)
		putReq.Header.Set("Content-Type", "application/json")
		putRes, err := http.DefaultClient.Do(putReq)
		if err != nil {
			t.Fatal(err)
		}
		defer putRes.Body.Close()

		if putRes.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK on save shuffled packages, got %d", putRes.StatusCode)
		}

		// 3. Verify public GET /packages returns all 4 tiers in the exact shuffled sequence
		pubRes, err := http.Get(srv.URL + "/packages")
		if err != nil {
			t.Fatal(err)
		}
		defer pubRes.Body.Close()

		if pubRes.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK on public /packages, got %d", pubRes.StatusCode)
		}

		var pubCfg domain.PackagesConfig
		if err := json.NewDecoder(pubRes.Body).Decode(&pubCfg); err != nil {
			t.Fatalf("failed to decode public packages: %v", err)
		}

		if len(pubCfg.Packages) != 4 {
			t.Fatalf("expected 4 packages, got %d", len(pubCfg.Packages))
		}

		// Exact shuffled order check: [pro, enterprise, growth, starter]
		expectedOrder := []string{"pro", "enterprise", "growth", "starter"}
		for i, slug := range expectedOrder {
			if pubCfg.Packages[i].Slug != slug {
				t.Errorf("at index %d: expected slug %q, got %q", i, slug, pubCfg.Packages[i].Slug)
			}
		}

		// Verify 4th tier properties
		entTier := pubCfg.Packages[1]
		if entTier.MonthlyPriceUSD != 49.99 {
			t.Errorf("expected Enterprise monthly price 49.99, got %v", entTier.MonthlyPriceUSD)
		}
		if entTier.ActiveOffers != -1 {
			t.Errorf("expected unlimited active offers (-1), got %d", entTier.ActiveOffers)
		}
		if entTier.TeamSeats != 20 {
			t.Errorf("expected 20 team seats, got %d", entTier.TeamSeats)
		}

		// 4. Verify public alias /api/v1/packages also returns the 4 tiers in exact order
		aliasRes, err := http.Get(srv.URL + "/api/v1/packages")
		if err != nil {
			t.Fatal(err)
		}
		defer aliasRes.Body.Close()

		var aliasCfg domain.PackagesConfig
		_ = json.NewDecoder(aliasRes.Body).Decode(&aliasCfg)
		if len(aliasCfg.Packages) != 4 {
			t.Fatalf("expected 4 packages on alias, got %d", len(aliasCfg.Packages))
		}
		if aliasCfg.Packages[0].Slug != "pro" || aliasCfg.Packages[1].Slug != "enterprise" {
			t.Errorf("alias packages order mismatch: %v, %v", aliasCfg.Packages[0].Slug, aliasCfg.Packages[1].Slug)
		}
	})

	// 7. Test deleting a package tier
	t.Run("admin deletes a package tier and remaining packages are published", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/packages", nil)
		req.Header.Set("Authorization", "Bearer "+adminToken)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var currentCfg domain.PackagesConfig
		_ = json.NewDecoder(res.Body).Decode(&currentCfg)
		res.Body.Close()

		remaining := make([]domain.VendorPackage, 0)
		for _, p := range currentCfg.Packages {
			if p.Slug != "starter" {
				remaining = append(remaining, p)
			}
		}
		currentCfg.Packages = remaining

		bodyBytes, _ := json.Marshal(currentCfg)
		putReq, _ := http.NewRequest(http.MethodPut, srv.URL+"/admin/packages", bytes.NewReader(bodyBytes))
		putReq.Header.Set("Authorization", "Bearer "+adminToken)
		putReq.Header.Set("Content-Type", "application/json")
		putRes, err := http.DefaultClient.Do(putReq)
		if err != nil {
			t.Fatal(err)
		}
		defer putRes.Body.Close()

		if putRes.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK on save remaining packages, got %d", putRes.StatusCode)
		}

		pubRes, err := http.Get(srv.URL + "/packages")
		if err != nil {
			t.Fatal(err)
		}
		defer pubRes.Body.Close()

		var pubCfg domain.PackagesConfig
		_ = json.NewDecoder(pubRes.Body).Decode(&pubCfg)
		for _, p := range pubCfg.Packages {
			if p.Slug == "starter" {
				t.Errorf("expected starter to be deleted, but still found in public packages")
			}
		}
	})
}

func TestVendorPackagesPublicAndMemoryStore(t *testing.T) {
	tokens := auth.NewTokens("test-public-secret-key-12345678")
	cfg := config.Config{
		CORSOrigins: []string{"*"},
	}
	var store *postgres.Store
	router := httpapi.New(store, tokens, cfg).Router(cfg)

	// 1. GET /packages via httptest.NewRecorder
	req := httptest.NewRequest(http.MethodGet, "/packages", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	var pkgCfg domain.PackagesConfig
	if err := json.NewDecoder(rec.Body).Decode(&pkgCfg); err != nil {
		t.Fatalf("failed to decode packages: %v", err)
	}

	if len(pkgCfg.Packages) != 3 {
		t.Fatalf("expected 3 packages, got %d", len(pkgCfg.Packages))
	}

	starter := pkgCfg.Packages[0]
	growth := pkgCfg.Packages[1]
	pro := pkgCfg.Packages[2]

	if starter.MonthlyPriceUSD != 3.99 || growth.MonthlyPriceUSD != 11.99 || pro.MonthlyPriceUSD != 22.99 {
		t.Errorf("unexpected tier pricing: starter=%v, growth=%v, pro=%v", starter.MonthlyPriceUSD, growth.MonthlyPriceUSD, pro.MonthlyPriceUSD)
	}

	if growth.Badge != "Most Popular" {
		t.Errorf("expected Growth badge 'Most Popular', got %q", growth.Badge)
	}
	if !growth.Highlighted {
		t.Errorf("expected Growth tier to be highlighted")
	}

	// 2. GET /api/v1/packages alias via httptest.NewRecorder
	aliasReq := httptest.NewRequest(http.MethodGet, "/api/v1/packages", nil)
	aliasRec := httptest.NewRecorder()
	router.ServeHTTP(aliasRec, aliasReq)

	if aliasRec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on alias, got %d", aliasRec.Code)
	}

	var aliasCfg domain.PackagesConfig
	if err := json.NewDecoder(aliasRec.Body).Decode(&aliasCfg); err != nil {
		t.Fatalf("failed to decode alias response: %v", err)
	}
	if len(aliasCfg.Packages) != 3 {
		t.Fatalf("expected 3 packages from alias, got %d", len(aliasCfg.Packages))
	}

	// 3. In-memory store lifecycle
	ctx := context.Background()
	initial, err := store.GetVendorPackages(ctx)
	if err != nil {
		t.Fatalf("store.GetVendorPackages failed: %v", err)
	}
	if len(initial.Packages) != 3 {
		t.Fatalf("expected 3 default packages, got %d", len(initial.Packages))
	}

	// Modify and save
	initial.Packages[0].MonthlyPriceUSD = 4.99
	if err := store.SaveVendorPackages(ctx, initial); err != nil {
		t.Fatalf("store.SaveVendorPackages failed: %v", err)
	}

	retrieved, err := store.GetVendorPackages(ctx)
	if err != nil {
		t.Fatalf("retrieval failed: %v", err)
	}
	if retrieved.Packages[0].MonthlyPriceUSD != 4.99 {
		t.Errorf("expected 4.99, got %v", retrieved.Packages[0].MonthlyPriceUSD)
	}

	// Re-verify GET /packages reflects in-memory update
	req2 := httptest.NewRequest(http.MethodGet, "/packages", nil)
	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, req2)
	var updatedPkgCfg domain.PackagesConfig
	_ = json.NewDecoder(rec2.Body).Decode(&updatedPkgCfg)
	if updatedPkgCfg.Packages[0].MonthlyPriceUSD != 4.99 {
		t.Errorf("expected 4.99 in HTTP response, got %v", updatedPkgCfg.Packages[0].MonthlyPriceUSD)
	}
}

func TestVendorPackagesAdminValidationAndExclusions(t *testing.T) {
	tokens := auth.NewTokens("test-admin-secret-key-12345678")
	cfg := config.Config{CORSOrigins: []string{"*"}}
	var store *postgres.Store
	srv := httpapi.New(store, tokens, cfg)

	// 1. Direct handler test for negative price
	t.Run("rejects negative price", func(t *testing.T) {
		badCfg := domain.DefaultPackagesConfig()
		badCfg.Packages[0].MonthlyPriceUSD = -5.0
		b, _ := json.Marshal(badCfg)

		req := httptest.NewRequest(http.MethodPut, "/admin/packages", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.HandleAdminSavePackagesForTest(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request, got %d", rec.Code)
		}
	})

	// 2. Direct handler test for empty packages
	t.Run("rejects empty packages", func(t *testing.T) {
		emptyCfg := domain.PackagesConfig{Packages: []domain.VendorPackage{}}
		b, _ := json.Marshal(emptyCfg)

		req := httptest.NewRequest(http.MethodPut, "/admin/packages", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.HandleAdminSavePackagesForTest(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected 400 Bad Request for empty packages, got %d", rec.Code)
		}
	})

	// 3. Sanitizes excluded features that are also in included features
	t.Run("sanitizes conflicting excluded features", func(t *testing.T) {
		validCfg := domain.DefaultPackagesConfig()
		// Add "feat_stamp_cards" to Starter features
		validCfg.Packages[0].Features = append(validCfg.Packages[0].Features, domain.PackageFeature{
			ID:          "feat_stamp_cards",
			Name:        "Digital Stamp Cards",
			Description: "Loyalty stamps",
			Category:    "loyalty",
			Icon:        "Award",
		})
		// And ensure "feat_stamp_cards" was in ExcludedFeatures
		validCfg.Packages[0].ExcludedFeatures = []string{"feat_stamp_cards", "feat_geo_heatmap"}

		b, _ := json.Marshal(validCfg)
		req := httptest.NewRequest(http.MethodPut, "/admin/packages", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.HandleAdminSavePackagesForTest(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rec.Code, rec.Body.String())
		}

		var savedCfg domain.PackagesConfig
		if err := json.NewDecoder(rec.Body).Decode(&savedCfg); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Verify feat_stamp_cards was removed from ExcludedFeatures
		for _, ex := range savedCfg.Packages[0].ExcludedFeatures {
			if ex == "feat_stamp_cards" {
				t.Errorf("expected feat_stamp_cards to be removed from ExcludedFeatures")
			}
		}
		if len(savedCfg.Packages[0].ExcludedFeatures) != 1 || savedCfg.Packages[0].ExcludedFeatures[0] != "feat_geo_heatmap" {
			t.Errorf("expected only feat_geo_heatmap in ExcludedFeatures, got %v", savedCfg.Packages[0].ExcludedFeatures)
		}
	})
}


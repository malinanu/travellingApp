package httpapi_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
)

func TestMultiVendorSponsorships(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	run := time.Now().UnixNano()
	vendorID := fmt.Sprintf("ven_multi_%d", run)
	userID := fmt.Sprintf("usr_multi_%d", run)
	adminID := fmt.Sprintf("usr_admin_%d", run)
	pw, _ := auth.HashPassword("testpass123")

	// Create vendor & user
	must(t, store.UpsertVendor(ctx, domain.Vendor{
		ID:          vendorID,
		Name:        "Malin Multi Cafe",
		Category:    "Cafe",
		Town:        "Ahangama",
		Status:      domain.VendorApproved,
		Tier:        "Free",
		SubmittedAt: time.Now(),
	}))
	must(t, store.CreateUser(ctx, domain.User{
		ID:           userID,
		VendorID:     &vendorID,
		Email:        fmt.Sprintf("vendor%d@test.com", run),
		PasswordHash: pw,
		Role:         domain.RoleVendor,
		CreatedAt:    time.Now(),
	}))
	must(t, store.CreateUser(ctx, domain.User{
		ID:           adminID,
		Email:        fmt.Sprintf("admin%d@test.com", run),
		PasswordHash: pw,
		Role:         domain.RoleAdmin,
		CreatedAt:    time.Now(),
	}))

	vendorToken := loginAs(t, srv, fmt.Sprintf("vendor%d@test.com", run), "testpass123")
	adminToken := loginAs(t, srv, fmt.Sprintf("admin%d@test.com", run), "testpass123")

	// 1. Vendor proposes Perk 1: "Free Tea and Cake"
	code, perk1 := doJSON(t, http.MethodPost, srv.URL+"/vendor/sponsorships", vendorToken, map[string]any{
		"title":             "Afternoon Tea Set",
		"rewardDescription": "Free Tea and Cake",
		"rewardLimit":       10,
	})
	if code != http.StatusCreated {
		t.Fatalf("expected 201 created for perk 1, got %d %v", code, perk1)
	}
	perk1ID, ok := perk1["id"].(string)
	if !ok || perk1ID == "" {
		t.Fatalf("expected valid id for perk 1, got %v", perk1)
	}
	if perk1["status"] != "pending" {
		t.Fatalf("expected pending status, got %v", perk1["status"])
	}

	// 2. Admin approves Perk 1
	code, resp := doJSON(t, http.MethodPost, srv.URL+"/admin/quests/sponsorships/"+perk1ID+"/status", adminToken, map[string]string{
		"status": "approved",
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200 approving perk 1, got %d %v", code, resp)
	}

	// Verify perk 1 is approved
	spons1, err := store.GetVendorSponsorship(ctx, perk1ID)
	if err != nil || spons1.Status != "approved" {
		t.Fatalf("expected perk 1 to be approved, got %v (err: %v)", spons1, err)
	}

	// 3. Vendor proposes Perk 2: "Free Fresh King Coconut"
	code, perk2 := doJSON(t, http.MethodPost, srv.URL+"/vendor/sponsorships", vendorToken, map[string]any{
		"title":             "Fresh Coconut",
		"rewardDescription": "Free Fresh King Coconut",
		"rewardLimit":       20,
	})
	if code != http.StatusCreated {
		t.Fatalf("expected 201 created for perk 2, got %d %v", code, perk2)
	}
	perk2ID, ok := perk2["id"].(string)
	if !ok || perk2ID == "" {
		t.Fatalf("expected valid id for perk 2, got %v", perk2)
	}

	// CRITICAL VERIFICATION: Perk 1 MUST STILL BE APPROVED!
	spons1After, err := store.GetVendorSponsorship(ctx, perk1ID)
	if err != nil || spons1After.Status != "approved" {
		t.Fatalf("REGRESSION: perk 1 was overwritten or reset! status=%s, err=%v", spons1After.Status, err)
	}

	// 4. Vendor lists all sponsorships
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/vendor/sponsorships", nil)
	req.Header.Set("Authorization", "Bearer "+vendorToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var vendorPerks []domain.VendorSponsorship
	if err := json.NewDecoder(res.Body).Decode(&vendorPerks); err != nil {
		t.Fatal(err)
	}
	if len(vendorPerks) != 2 {
		t.Fatalf("expected vendor to have 2 perks, got %d", len(vendorPerks))
	}

	// 5. Admin approves Perk 2 as well
	doJSON(t, http.MethodPost, srv.URL+"/admin/quests/sponsorships/"+perk2ID+"/status", adminToken, map[string]string{
		"status": "approved",
	})

	// 6. Admin schedules a quest day linking specifically to Perk 2
	schedDate := "2026-12-05"
	code, schedResp := doJSON(t, http.MethodPost, srv.URL+"/admin/quests/schedules", adminToken, map[string]any{
		"date":          schedDate,
		"vendorId":      vendorID,
		"sponsorshipId": perk2ID,
		"goal":          3,
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200 setting quest schedule, got %d %v", code, schedResp)
	}

	// 7. Verify schedule links to Perk 2
	sched, err := store.GetQuestSchedule(ctx, schedDate)
	if err != nil {
		t.Fatalf("failed to get quest schedule: %v", err)
	}
	if sched.VendorID != vendorID {
		t.Fatalf("expected vendor %s, got %s", vendorID, sched.VendorID)
	}
	if sched.SponsorshipID != perk2ID {
		t.Fatalf("expected sponsorship ID %s, got %s", perk2ID, sched.SponsorshipID)
	}
	if sched.RewardDescription != "Free Fresh King Coconut" {
		t.Fatalf("expected 'Free Fresh King Coconut', got %s", sched.RewardDescription)
	}
	if sched.Goal != 3 {
		t.Fatalf("expected goal 3, got %d", sched.Goal)
	}

	// 8. Test Toggle active/pause on perk 1 without resetting approved status
	code, updateResp := doJSON(t, http.MethodPut, srv.URL+"/vendor/sponsorships/"+perk1ID, vendorToken, map[string]any{
		"title":             spons1After.Title,
		"rewardDescription": spons1After.RewardDescription,
		"rewardLimit":       spons1After.RewardLimit,
		"isActive":          false,
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200 updating perk 1 active state, got %d %v", code, updateResp)
	}
	if updateResp["status"] != "approved" {
		t.Fatalf("expected status to remain 'approved' when only isActive toggled, got %v", updateResp["status"])
	}
	if updateResp["isActive"] != false {
		t.Fatalf("expected isActive to be false, got %v", updateResp["isActive"])
	}

	// 9. Legacy PUT /vendor/sponsorship MUST NOT overwrite existing approved perks
	code, legacyResp := doJSON(t, http.MethodPut, srv.URL+"/vendor/sponsorship", vendorToken, map[string]any{
		"rewardDescription": "Free Dessert Slice",
		"rewardLimit":       5,
		"isActive":          true,
	})
	if code != http.StatusOK {
		t.Fatalf("expected 200 on legacy upsert, got %d %v", code, legacyResp)
	}

	// Perk 2 was approved and MUST STILL BE APPROVED
	spons2Check, err := store.GetVendorSponsorship(ctx, perk2ID)
	if err != nil || spons2Check.Status != "approved" {
		t.Fatalf("REGRESSION: legacy PUT overwrote approved perk 2! status=%s, err=%v", spons2Check.Status, err)
	}

	// 10. Deleting a perk scheduled for an upcoming date must be rejected
	reqDel, _ := http.NewRequest(http.MethodDelete, srv.URL+"/vendor/sponsorships/"+perk2ID, nil)
	reqDel.Header.Set("Authorization", "Bearer "+vendorToken)
	resDel, err := http.DefaultClient.Do(reqDel)
	if err != nil {
		t.Fatal(err)
	}
	defer resDel.Body.Close()
	if resDel.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when attempting to delete a scheduled perk, got %d", resDel.StatusCode)
	}
}

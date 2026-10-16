package httpapi_test

// Integration test for recorded quest completions against the real Postgres
// store (self-skips without a local DB, like validate_test.go).

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

func TestQuestCompletionIsRecordedOnce(t *testing.T) {
	store := mustConnect(t)
	defer store.Close()
	ctx := context.Background()

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{}
	srv := httptest.NewServer(httpapi.New(store, tokens, cfg).Router(cfg))
	defer srv.Close()

	// Fresh ids per run so earlier runs' check-ins don't count.
	run := time.Now().UnixNano()
	travellerID := fmt.Sprintf("usr_qc_traveller_%d", run)
	pw, _ := auth.HashPassword("testpass123")
	must(t, store.CreateUser(ctx, domain.User{ID: travellerID, Email: fmt.Sprintf("qc%d@edgecase.test", run), Name: "Quest Tester", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: time.Now()}))
	must(t, store.CreateUser(ctx, domain.User{ID: fmt.Sprintf("usr_qc_admin_%d", run), Email: fmt.Sprintf("qcadmin%d@edgecase.test", run), PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: time.Now()}))
	traveller := loginAs(t, srv, fmt.Sprintf("qc%d@edgecase.test", run), "testpass123")
	admin := loginAs(t, srv, fmt.Sprintf("qcadmin%d@edgecase.test", run), "testpass123")

	_, quest := doJSON(t, http.MethodGet, srv.URL+"/me/quest", traveller, nil)
	goal := int(quest["goal"].(float64))
	if quest["isComplete"] == true || quest["completedAt"] != nil {
		t.Fatalf("fresh traveller already complete: %v", quest)
	}

	var last map[string]any
	for i := 0; i < goal; i++ {
		code, body := doJSON(t, http.MethodPost, srv.URL+"/checkins", traveller, map[string]string{"placeId": fmt.Sprintf("plc_qc_%d_%d", run, i)})
		if code != http.StatusCreated {
			t.Fatalf("check-in %d: %d %v", i, code, body)
		}
		last, _ = body["quest"].(map[string]any)
	}
	if last == nil || last["isComplete"] != true || last["completedAt"] == nil {
		t.Fatalf("check-in response after reaching the goal should carry a completed quest, got %v", last)
	}

	// A repeat check-in and a later GET keep the first completion time.
	doJSON(t, http.MethodPost, srv.URL+"/checkins", traveller, map[string]string{"placeId": fmt.Sprintf("plc_qc_%d_extra", run)})
	_, again := doJSON(t, http.MethodGet, srv.URL+"/me/quest", traveller, nil)
	if again["completedAt"] != last["completedAt"] {
		t.Fatalf("completedAt changed: %v -> %v", last["completedAt"], again["completedAt"])
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/admin/quests/completions?travellerId="+travellerID, nil)
	req.Header.Set("Authorization", "Bearer "+admin)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var list []domain.QuestCompletion
	if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].TravellerName != "Quest Tester" || list[0].RedeemedAt != nil {
		t.Fatalf("admin completions = %+v, want one unredeemed row for Quest Tester", list)
	}
}

package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

func TestMasterCommandCenterFeaturesCatalog(t *testing.T) {
	features := httpapi.MasterCommandCenterFeatures
	if len(features) < 15 {
		t.Fatalf("expected at least 15 Command Center features, got %d", len(features))
	}

	seenFeatures := make(map[string]bool)
	seenPermissions := make(map[string]bool)

	for _, f := range features {
		if f.ID == "" || f.Name == "" || f.Description == "" || f.Category == "" || f.Icon == "" {
			t.Errorf("feature %+v has missing required fields", f)
		}
		if seenFeatures[f.ID] {
			t.Errorf("duplicate feature id: %s", f.ID)
		}
		seenFeatures[f.ID] = true

		if len(f.SubPermissions) == 0 {
			t.Errorf("feature %s has no sub-permissions broken down", f.ID)
		}

		for _, p := range f.SubPermissions {
			if p.ID == "" || p.Name == "" || p.Description == "" || p.Risk == "" {
				t.Errorf("permission %+v has missing required fields in feature %s", p, f.ID)
			}
			if p.FeatureID != f.ID {
				t.Errorf("permission %s featureId mismatch: want %s, got %s", p.ID, f.ID, p.FeatureID)
			}
			if seenPermissions[p.ID] {
				t.Errorf("duplicate permission id: %s", p.ID)
			}
			seenPermissions[p.ID] = true

			switch p.Risk {
			case "low", "medium", "high", "critical":
			default:
				t.Errorf("invalid risk level %s on permission %s", p.Risk, p.ID)
			}
		}
	}

	if !seenFeatures["access_control"] {
		t.Fatalf("expected 'access_control' feature in MasterCommandCenterFeatures catalog")
	}
	if !seenPermissions["access_control.view"] || !seenPermissions["access_control.manage"] {
		t.Fatalf("expected 'access_control.view' and 'access_control.manage' permissions")
	}
}

func TestAdminAccessControlCatalogEndpoint(t *testing.T) {
	// Catalog endpoint works with any admin session
	cfg := config.Config{
		CORSOrigins:      []string{"http://localhost:3000"},
		SuperAdminEmails: []string{"super@admin.dev"},
	}
	tokens := auth.NewTokens("test-secret-access-control")
	handler := httpapi.New(nil, tokens, cfg).Router(cfg)

	// Without auth -> 401
	req := httptest.NewRequest(http.MethodGet, "/admin/access-control/catalog", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("unauthenticated: want 401, got %d", rec.Code)
	}
}

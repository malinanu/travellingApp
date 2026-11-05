package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleConfig_MirissaEnabled(t *testing.T) {
	srv := &Server{
		DefaultLocationMirissa: true,
	}

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()

	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp PublicConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DefaultLocation == nil {
		t.Fatal("expected DefaultLocation to be non-nil when enabled")
	}
	if !resp.DefaultLocation.Enabled {
		t.Errorf("expected Enabled to be true")
	}
	if resp.DefaultLocation.Name != "Mirissa" {
		t.Errorf("expected Name to be Mirissa, got %q", resp.DefaultLocation.Name)
	}
	if resp.DefaultLocation.Lat != 5.9483 || resp.DefaultLocation.Lng != 80.4552 {
		t.Errorf("expected Mirissa coordinates 5.9483, 80.4552, got %f, %f", resp.DefaultLocation.Lat, resp.DefaultLocation.Lng)
	}
	if resp.DefaultLocation.Zoom != 14.5 {
		t.Errorf("expected Zoom 14.5, got %f", resp.DefaultLocation.Zoom)
	}
}

func TestHandleConfig_MirissaDisabled(t *testing.T) {
	srv := &Server{
		DefaultLocationMirissa: false,
	}

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()

	srv.handleConfig(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var resp PublicConfigResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DefaultLocation != nil {
		t.Fatalf("expected DefaultLocation to be nil when disabled, got %+v", resp.DefaultLocation)
	}
}

func TestNewGuideLocation_DefaultMirissa(t *testing.T) {
	no := false

	// When defaultMirissa is true and location is not known:
	locEnabled := newGuideLocation(0, 0, &no, true)
	if !locEnabled.Known {
		t.Errorf("expected Known=true with defaultMirissa=true")
	}
	if locEnabled.Area != "Mirissa" {
		t.Errorf("expected Area=Mirissa, got %q", locEnabled.Area)
	}
	if locEnabled.Lat != 5.9483 || locEnabled.Lng != 80.4552 {
		t.Errorf("expected Mirissa coordinates, got %f, %f", locEnabled.Lat, locEnabled.Lng)
	}

	// When defaultMirissa is false or omitted and location is not known:
	locDisabled := newGuideLocation(0, 0, &no, false)
	if locDisabled.Known {
		t.Errorf("expected Known=false with defaultMirissa=false")
	}

	locOmitted := newGuideLocation(0, 0, &no)
	if locOmitted.Known {
		t.Errorf("expected Known=false with defaultMirissa omitted")
	}
}

func TestHandleAdminSetUserDemoPrivilege_Validation(t *testing.T) {
	srv := &Server{}

	for _, tc := range []struct {
		name string
		body string
		code int
	}{
		{"omitted_reason", `{"demoPrivilege": true}`, http.StatusBadRequest},
		{"empty_reason", `{"demoPrivilege": true, "reason": ""}`, http.StatusBadRequest},
		{"whitespace_reason", `{"demoPrivilege": true, "reason": "   \n\t "}`, http.StatusBadRequest},
		{"malformed_json", `{"demoPrivilege": true, `, http.StatusBadRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPut, "/admin/users/usr_123/demo-privilege", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.handleAdminSetUserDemoPrivilege(w, r)
			if w.Code != tc.code {
				t.Fatalf("expected status %d, got %d: %s", tc.code, w.Code, w.Body.String())
			}
		})
	}
}


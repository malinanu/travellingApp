package geo

import "testing"

func TestFindRegion(t *testing.T) {
	tests := []struct {
		query    string
		wantID   string
		wantName string
	}{
		{"mirissa", "mirissa", "Mirissa"},
		{"Mirissa Beach", "mirissa", "Mirissa"},
		{"I'm planning to go mirissa next week, give me places", "mirissa", "Mirissa"},
		{"arugambay", "arugam", "Arugam Bay"},
		{"next month I'm planning to visit arugambay. give me the places to visit", "arugam", "Arugam Bay"},
		{"galle", "galle", "Galle"},
		{"places in Galle, Sri Lanka", "galle", "Galle"},
		{"top sights in Weligama", "weligama", "Weligama"},
		{"Ella", "ella", "Ella"},
		{"unknown-non-existent-city", "", ""},
	}

	for _, tt := range tests {
		r := FindRegion(tt.query)
		if tt.wantID == "" {
			if r != nil {
				t.Errorf("FindRegion(%q) = %v, want nil", tt.query, r)
			}
			continue
		}
		if r == nil {
			t.Fatalf("FindRegion(%q) = nil, want %s", tt.query, tt.wantID)
		}
		if r.ID != tt.wantID || r.Name != tt.wantName {
			t.Errorf("FindRegion(%q) = %s (%s), want %s (%s)", tt.query, r.ID, r.Name, tt.wantID, tt.wantName)
		}
	}
}


package domain

import "testing"

func TestCategoryForPlaceData(t *testing.T) {
	cases := []struct {
		name string
		data map[string]any
		want string
	}{
		{"cafe tag", map[string]any{"tags": []any{"Coffee Shop"}}, "coffee"},
		{"restaurant tag", map[string]any{"tags": []any{"Sri Lankan Restaurant"}}, "restaurants"},
		{"hotel tag", map[string]any{"tags": []any{"Resort Hotel"}}, "hotels"},
		{"park tag", map[string]any{"tags": []any{"National Park"}}, "parks"},
		{"cafe type fallback", map[string]any{"type": "cafe"}, "coffee"},
		{"unknown", map[string]any{}, "attractions"},
	}
	for _, c := range cases {
		if got := CategoryForPlaceData(c.data); got != c.want {
			t.Errorf("%s: got %q want %q", c.name, got, c.want)
		}
	}
}

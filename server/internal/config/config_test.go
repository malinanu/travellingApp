package config

import (
	"testing"
)

func TestResolveVendorAppURL(t *testing.T) {
	tests := []struct {
		name         string
		nextPublic   string
		vendorApp    string
		want         string
	}{
		{
			name:       "prefers NEXT_PUBLIC_VENDOR_URL when both are set",
			nextPublic: "https://spotto-vendor.threelogiq.com",
			vendorApp:  "https://vendor.threelogiq.com",
			want:       "https://spotto-vendor.threelogiq.com",
		},
		{
			name:       "strips trailing slashes from NEXT_PUBLIC_VENDOR_URL",
			nextPublic: "https://spotto-vendor.threelogiq.com///",
			vendorApp:  "https://vendor.threelogiq.com",
			want:       "https://spotto-vendor.threelogiq.com",
		},
		{
			name:       "falls back to VENDOR_APP_URL when NEXT_PUBLIC is empty",
			nextPublic: "",
			vendorApp:  "https://vendor.threelogiq.com",
			want:       "https://vendor.threelogiq.com",
		},
		{
			name:       "strips trailing slashes from VENDOR_APP_URL fallback",
			nextPublic: "",
			vendorApp:  "https://vendor.threelogiq.com/",
			want:       "https://vendor.threelogiq.com",
		},
		{
			name:       "defaults to localhost:3000 when both are empty",
			nextPublic: "",
			vendorApp:  "",
			want:       "http://localhost:3000",
		},
		{
			name:       "handles whitespace-only inputs",
			nextPublic: "   ",
			vendorApp:  " \t ",
			want:       "http://localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveVendorAppURL(tt.nextPublic, tt.vendorApp)
			if got != tt.want {
				t.Errorf("ResolveVendorAppURL(%q, %q) = %q; want %q", tt.nextPublic, tt.vendorApp, got, tt.want)
			}
		})
	}
}


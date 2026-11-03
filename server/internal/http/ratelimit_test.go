package httpapi

import (
	"net/http"
	"testing"
)

func TestResolveProxyIP(t *testing.T) {
	tests := []struct {
		name string
		xff  string
		xri  string
		want string
	}{
		{
			name: "single public IP in XFF",
			xff:  "203.0.113.195",
			want: "203.0.113.195",
		},
		{
			name: "docker bridge appended after public IP",
			xff:  "203.0.113.195, 172.18.0.1",
			want: "203.0.113.195",
		},
		{
			name: "localhost appended after public IP",
			xff:  "203.0.113.195, 127.0.0.1",
			want: "203.0.113.195",
		},
		{
			name: "multiple hops with private bridge",
			xff:  "198.51.100.2, 10.0.0.1, 172.17.0.2",
			want: "198.51.100.2",
		},
		{
			name: "only private IP in XFF",
			xff:  "172.18.0.5",
			want: "172.18.0.5",
		},
		{
			name: "X-Real-IP used when XFF empty",
			xff:  "",
			xri:  "198.51.100.44",
			want: "198.51.100.44",
		},
		{
			name: "both empty",
			xff:  "",
			xri:  "",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveProxyIP(tc.xff, tc.xri)
			if got != tc.want {
				t.Errorf("resolveProxyIP(%q, %q) = %q, want %q", tc.xff, tc.xri, got, tc.want)
			}
		})
	}
}

func TestClientIP(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.RemoteAddr = "192.0.2.1:12345"

	if got := clientIP(req); got != "192.0.2.1" {
		t.Errorf("clientIP with RemoteAddr = %q, want %q", got, "192.0.2.1")
	}

	req.Header.Set("X-Forwarded-For", "203.0.113.50, 127.0.0.1")
	if got := clientIP(req); got != "203.0.113.50" {
		t.Errorf("clientIP with XFF = %q, want %q", got, "203.0.113.50")
	}

	req.Header.Del("X-Forwarded-For")
	req.Header.Set("X-Real-IP", "203.0.113.77")
	if got := clientIP(req); got != "203.0.113.77" {
		t.Errorf("clientIP with X-Real-IP = %q, want %q", got, "203.0.113.77")
	}
}

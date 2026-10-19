package reelextract

import "testing"

func TestCanonical(t *testing.T) {
	cases := []struct {
		in, want, platform string
	}{
		{"https://www.instagram.com/reels/DbQSwgwMM_n/", "https://www.instagram.com/reel/DbQSwgwMM_n/", "instagram"},
		{"https://instagram.com/reel/DbQSwgwMM_n/?igsh=abc&utm_source=ig_web", "https://www.instagram.com/reel/DbQSwgwMM_n/", "instagram"},
		{"instagram.com/p/Cx1y2Z3/", "https://www.instagram.com/p/Cx1y2Z3/", "instagram"},
		{"https://www.instagram.com/some.user/reel/Abc_12-3/", "https://www.instagram.com/reel/Abc_12-3/", "instagram"},
		{"https://www.tiktok.com/@phuket.eats/video/7312345678901234567?is_from_webapp=1", "https://www.tiktok.com/@phuket.eats/video/7312345678901234567", "tiktok"},
		{"https://vm.tiktok.com/ZMabc123/", "https://vm.tiktok.com/ZMabc123/", "tiktok"},
		{"https://youtube.com/shorts/aBcDeFgHiJk?si=x", "https://www.youtube.com/shorts/aBcDeFgHiJk", "youtube"},
		{"https://youtu.be/aBcDeFgHiJk", "https://www.youtube.com/watch?v=aBcDeFgHiJk", "youtube"},
	}
	for _, c := range cases {
		got, platform, err := Canonical(c.in)
		if err != nil || got != c.want || platform != c.platform {
			t.Errorf("Canonical(%q) = %q, %q, %v; want %q, %q", c.in, got, platform, err, c.want, c.platform)
		}
	}
	for _, bad := range []string{
		"", "https://instagram.com/", "https://instagram.com.evil.example/reel/abc/",
		"ftp://instagram.com/reel/abc/", "https://example.com/reel/abc/", "https://www.instagram.com/explore/",
	} {
		if got, _, err := Canonical(bad); err == nil {
			t.Errorf("Canonical(%q) = %q, want an error", bad, got)
		}
	}
}

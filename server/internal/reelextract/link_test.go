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
		{"https://www.tiktok.com/@phuket.eats/video/7312345678901234567?is_from_webapp=1", "https://www.tiktok.com/@/video/7312345678901234567", "tiktok"},
		{"https://m.tiktok.com/v/7312345678901234567.html", "https://www.tiktok.com/@/video/7312345678901234567", "tiktok"},
		{"https://www.tiktok.com/@phuket.eats/photo/7312345678901234999", "https://www.tiktok.com/@/photo/7312345678901234999", "tiktok"},
		{"https://vm.tiktok.com/ZMabc123/", "https://vm.tiktok.com/ZMabc123/", "tiktok"},
		{"https://youtube.com/shorts/aBcDeFgHiJk?si=x", "https://www.youtube.com/shorts/aBcDeFgHiJk", "youtube"},
		{"https://youtu.be/aBcDeFgHiJk", "https://www.youtube.com/watch?v=aBcDeFgHiJk", "youtube"},
		{"https://www.facebook.com/reel/4720710134913664", "https://www.facebook.com/watch/?v=4720710134913664", "facebook"},
		{"https://m.facebook.com/reel/4720710134913664/?mibextid=abc", "https://www.facebook.com/watch/?v=4720710134913664", "facebook"},
		{"https://www.facebook.com/phuketist/videos/save-these-15-places-/4720710134913664/", "https://www.facebook.com/watch/?v=4720710134913664", "facebook"},
		{"https://www.facebook.com/phuketist/videos/4720710134913664", "https://www.facebook.com/watch/?v=4720710134913664", "facebook"},
		{"https://web.facebook.com/watch/?v=4720710134913664&ref=sharing", "https://www.facebook.com/watch/?v=4720710134913664", "facebook"},
		{"https://www.facebook.com/share/r/1AbCdEfGh/", "https://www.facebook.com/share/r/1AbCdEfGh/", "facebook"},
		{"https://fb.watch/qWeRtY12/", "https://fb.watch/qWeRtY12/", "facebook"},
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
		"https://www.facebook.com/phuketist", "https://www.facebook.com/login/?next=%2Freel%2F1", "https://facebook.com.evil.example/reel/1",
		"https://www.facebook.com/watch/?v=notanid",
	} {
		if got, _, err := Canonical(bad); err == nil {
			t.Errorf("Canonical(%q) = %q, want an error", bad, got)
		}
	}
}

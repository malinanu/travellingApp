package reelextract

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	igPathRe       = regexp.MustCompile(`^/(?:[\w.]+/)?(reels?|p|tv)/([\w-]+)`)
	tiktokPostRe   = regexp.MustCompile(`^/(?:@[\w.-]*/)?(video|photo)/(\d+)`)
	tiktokMobileRe = regexp.MustCompile(`^/v/(\d+)(?:\.html)?$`)
	ytShortsRe     = regexp.MustCompile(`^/(?:shorts|live)/([\w-]{6,})`)
	fbVideoRe      = regexp.MustCompile(`^/(?:reel|reels)/(\d+)|^/[^/]+/videos/(?:[^/]+/)?(\d+)`)
	fbShareRe      = regexp.MustCompile(`^/share/(?:r|v)/[\w-]+/?$`)
)

// ErrUnsupportedLink means the link isn't a video on a supported platform.
var ErrUnsupportedLink = errors.New("unsupported link")

// Canonical turns a shared link into one stable URL per post (dropping
// tracking parameters and share-sheet variations) and names its platform.
// Only these platforms are accepted, which also keeps yt-dlp from being
// pointed at arbitrary hosts.
func Canonical(raw string) (canonical, platform string, err error) {
	raw = strings.TrimSpace(raw)
	if raw != "" && !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return "", "", ErrUnsupportedLink
	}
	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	host = strings.TrimPrefix(host, "m.")
	host = strings.TrimPrefix(host, "web.")
	switch {
	case host == "instagram.com":
		m := igPathRe.FindStringSubmatch(u.Path)
		if m == nil {
			return "", "", ErrUnsupportedLink
		}
		kind := m[1]
		if kind == "reels" {
			kind = "reel"
		}
		return "https://www.instagram.com/" + kind + "/" + m[2] + "/", "instagram", nil
	case host == "tiktok.com":
		// The username is left out: it can change, the post id can't, and
		// yt-dlp accepts tiktok.com/@/video/{id}.
		if m := tiktokPostRe.FindStringSubmatch(u.Path); m != nil {
			return "https://www.tiktok.com/@/" + m[1] + "/" + m[2], "tiktok", nil
		}
		if m := tiktokMobileRe.FindStringSubmatch(u.Path); m != nil {
			return "https://www.tiktok.com/@/video/" + m[1], "tiktok", nil
		}
		return "", "", ErrUnsupportedLink
	case host == "vm.tiktok.com" || host == "vt.tiktok.com":
		code := strings.Trim(u.Path, "/")
		if code == "" || strings.Contains(code, "/") {
			return "", "", ErrUnsupportedLink
		}
		return "https://" + host + "/" + code + "/", "tiktok", nil
	case host == "youtube.com" || host == "music.youtube.com":
		if m := ytShortsRe.FindStringSubmatch(u.Path); m != nil {
			return "https://www.youtube.com/shorts/" + m[1], "youtube", nil
		}
		if v := u.Query().Get("v"); u.Path == "/watch" && v != "" {
			return "https://www.youtube.com/watch?v=" + url.QueryEscape(v), "youtube", nil
		}
	case host == "facebook.com":
		// Reels, page videos and watch links of one video share its id, so
		// they all canonicalise to the watch link.
		if m := fbVideoRe.FindStringSubmatch(u.Path); m != nil {
			id := m[1] + m[2]
			return "https://www.facebook.com/watch/?v=" + id, "facebook", nil
		}
		if v := u.Query().Get("v"); strings.TrimSuffix(u.Path, "/") == "/watch" && isDigits(v) {
			return "https://www.facebook.com/watch/?v=" + v, "facebook", nil
		}
		// A share link that couldn't be resolved: yt-dlp can follow it.
		if fbShareRe.MatchString(u.Path) {
			return "https://www.facebook.com" + strings.TrimSuffix(u.Path, "/") + "/", "facebook", nil
		}
	case host == "fb.watch":
		if code := strings.Trim(u.Path, "/"); code != "" && !strings.Contains(code, "/") {
			return "https://fb.watch/" + code + "/", "facebook", nil
		}
	case host == "youtu.be":
		if id := strings.Trim(u.Path, "/"); id != "" && !strings.Contains(id, "/") {
			return "https://www.youtube.com/watch?v=" + url.QueryEscape(id), "youtube", nil
		}
	}
	return "", "", ErrUnsupportedLink
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

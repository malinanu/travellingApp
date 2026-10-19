package reelextract

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

var (
	igPathRe      = regexp.MustCompile(`^/(?:[\w.]+/)?(reels?|p|tv)/([\w-]+)`)
	tiktokVideoRe = regexp.MustCompile(`^/(@[\w.-]+)/(?:video|photo)/(\d+)`)
	ytShortsRe    = regexp.MustCompile(`^/(?:shorts|live)/([\w-]{6,})`)
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
		m := tiktokVideoRe.FindStringSubmatch(u.Path)
		if m == nil {
			return "", "", ErrUnsupportedLink
		}
		return "https://www.tiktok.com/" + m[1] + "/video/" + m[2], "tiktok", nil
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
	case host == "youtu.be":
		if id := strings.Trim(u.Path, "/"); id != "" && !strings.Contains(id, "/") {
			return "https://www.youtube.com/watch?v=" + url.QueryEscape(id), "youtube", nil
		}
	}
	return "", "", ErrUnsupportedLink
}

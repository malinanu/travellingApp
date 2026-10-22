package reelextract

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Share sheets often hand out short links (vm.tiktok.com/…) that only
// redirect to the real post. Resolving them when a link is added means the
// same video always maps to one canonical link, and one extraction.

// shortLinkHosts redirect to a post; resolveHosts are where they may land.
var shortLinkHosts = map[string]bool{
	"vm.tiktok.com": true,
	"vt.tiktok.com": true,
	"fb.watch":      true,
}

var resolveHosts = map[string]bool{
	"tiktok.com": true, "www.tiktok.com": true, "m.tiktok.com": true,
	"vm.tiktok.com": true, "vt.tiktok.com": true,
	"facebook.com": true, "www.facebook.com": true, "m.facebook.com": true,
	"web.facebook.com": true, "fb.watch": true,
}

// isShortLink reports links that only redirect to a post: short-link
// hosts, and Facebook's share sheet links (facebook.com/share/r/…).
func isShortLink(u *url.URL) bool {
	host := strings.ToLower(u.Hostname())
	if shortLinkHosts[host] {
		return true
	}
	return resolveHosts[host] && strings.Contains(host, "facebook.com") && fbShareRe.MatchString(u.Path)
}

const maxRedirects = 5

var errHostNotAllowed = errors.New("redirect to a host that isn't a supported video site")

// ResolveShortLink follows a share short link to the post it points at.
// Anything that isn't a short link, or that fails to resolve, comes back
// unchanged: the caller canonicalises it either way.
func ResolveShortLink(ctx context.Context, raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !isShortLink(u) {
		return raw
	}
	client := &http.Client{Timeout: 5 * time.Second}
	if resolved, err := followRedirects(ctx, client, u.String(), func(h string) bool { return resolveHosts[h] }); err == nil {
		return resolved
	}
	return raw
}

// followRedirects returns where link ends up, following at most
// maxRedirects hops and only to hosts allowed() accepts.
func followRedirects(ctx context.Context, client *http.Client, link string, allowed func(host string) bool) (string, error) {
	c := *client
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return errors.New("too many redirects")
		}
		if !allowed(strings.ToLower(req.URL.Hostname())) {
			return errHostNotAllowed
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return "", err
	}
	// A browser user agent gets the full /@user/video/{id} URL rather than
	// an app-store or mobile page.
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0 Safari/537.36")
	res, err := c.Do(req)
	if err != nil {
		return "", err
	}
	res.Body.Close()
	return res.Request.URL.String(), nil
}

// CanonicalLink resolves a share short link and canonicalises the result.
// When the redirect lands somewhere unusable (Facebook often sends
// logged-out visitors to its login page), the original link is
// canonicalised instead.
func CanonicalLink(ctx context.Context, raw string) (canonical, platform string, err error) {
	if resolved := ResolveShortLink(ctx, raw); resolved != raw {
		if c, p, err := Canonical(resolved); err == nil {
			return c, p, nil
		}
	}
	return Canonical(raw)
}

package reelextract

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFollowRedirects(t *testing.T) {
	var target *httptest.Server
	target = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/short":
			http.Redirect(w, r, target.URL+"/@phuket.eats/video/7312345678901234567?_r=1", http.StatusMovedPermanently)
		case "/loop":
			http.Redirect(w, r, "/loop", http.StatusFound)
		case "/away":
			http.Redirect(w, r, "http://169.254.169.254/latest/meta-data", http.StatusFound)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer target.Close()
	testHost, _ := url.Parse(target.URL)
	onlyTest := func(h string) bool { return h == testHost.Hostname() }
	ctx := context.Background()

	got, err := followRedirects(ctx, target.Client(), target.URL+"/short", onlyTest)
	if err != nil || !strings.HasSuffix(got, "/@phuket.eats/video/7312345678901234567?_r=1") {
		t.Errorf("short link: %q, %v", got, err)
	}
	if _, err := followRedirects(ctx, target.Client(), target.URL+"/loop", onlyTest); err == nil {
		t.Error("redirect loop: want an error after the hop limit")
	}
	if got, err := followRedirects(ctx, target.Client(), target.URL+"/away", onlyTest); err == nil {
		t.Errorf("redirect to another host was followed to %q", got)
	}
}

func TestResolveShortLinkLeavesOtherLinksAlone(t *testing.T) {
	for _, link := range []string{
		"https://www.instagram.com/reel/abc/",
		"https://www.tiktok.com/@a/video/1",
		"not a url %%",
	} {
		if got := ResolveShortLink(context.Background(), link); got != link {
			t.Errorf("ResolveShortLink(%q) = %q, want it unchanged", link, got)
		}
	}
}

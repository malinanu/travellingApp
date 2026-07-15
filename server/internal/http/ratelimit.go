package httpapi

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// loginLimiter is a simple per-key sliding-window attempt counter — enough
// to blunt brute-force/credential-stuffing against a single-instance
// deployment without pulling in an external dependency (Redis etc.). Keys
// are typically "clientIP:identity" so one attacker can't lock out
// unrelated users, but also can't hammer one account from a single source
// indefinitely.
type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*attemptWindow
	max      int
	window   time.Duration
}

type attemptWindow struct {
	count   int
	resetAt time.Time
}

func newLoginLimiter(max int, window time.Duration) *loginLimiter {
	l := &loginLimiter{attempts: make(map[string]*attemptWindow), max: max, window: window}
	go l.cleanupLoop()
	return l
}

// allow reports whether key is still under the attempt limit.
func (l *loginLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.attempts[key]
	if !ok || time.Now().After(w.resetAt) {
		return true
	}
	return w.count < l.max
}

// recordFailure counts one more failed attempt against key, starting a
// fresh window if the previous one expired.
func (l *loginLimiter) recordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.attempts[key]
	if !ok || time.Now().After(w.resetAt) {
		w = &attemptWindow{resetAt: time.Now().Add(l.window)}
		l.attempts[key] = w
	}
	w.count++
}

// reset clears key's counter — call on a successful attempt.
func (l *loginLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}

func (l *loginLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for k, w := range l.attempts {
			if now.After(w.resetAt) {
				delete(l.attempts, k)
			}
		}
		l.mu.Unlock()
	}
}

// realIP replaces chi's stock middleware.RealIP, which trusts the
// left-most (client-supplied, spoofable) X-Forwarded-For entry. That let an
// attacker send a different X-Forwarded-For value on every request to get a
// fresh rate-limit bucket each time, defeating loginLimit/totpLimit
// entirely. Instead, this trusts only the right-most entry — the one
// CloudPanel's reverse proxy itself appends via
// `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for` — which
// reflects what the proxy actually saw on its own client-facing socket and
// can't be overridden by anything the client puts in the header.
func realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Deliberately X-Forwarded-For only, not X-Real-IP: we know
		// CloudPanel's nginx appends to the former (see comment above), but
		// we have no such guarantee for the latter — trusting it unverified
		// would reopen the same spoofing hole under a different header
		// name. Absent X-Forwarded-For, r.RemoteAddr (the direct TCP peer,
		// i.e. nginx itself) is used as-is: a safe, if coarse, fallback.
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			if ip := strings.TrimSpace(parts[len(parts)-1]); ip != "" {
				r.RemoteAddr = net.JoinHostPort(ip, "0")
			}
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the request's client IP from r.RemoteAddr, which realIP
// has already resolved to the trusted proxy-observed address.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeTooManyRequests(w http.ResponseWriter) {
	writeErr(w, http.StatusTooManyRequests, "too many attempts — try again later")
}

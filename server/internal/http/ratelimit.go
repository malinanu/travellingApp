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

// resolveProxyIP extracts the real client IP from reverse-proxy headers (X-Forwarded-For or X-Real-IP).
// If X-Forwarded-For has multiple entries, it inspects from right to left, skipping internal/private
// network hops (e.g. docker bridge, loopback) to find the public IP seen by the outer proxy.
func resolveProxyIP(xff, xri string) string {
	if xff != "" {
		parts := strings.Split(xff, ",")
		var candidate string
		for i := len(parts) - 1; i >= 0; i-- {
			ipStr := strings.TrimSpace(parts[i])
			if ipStr == "" {
				continue
			}
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if candidate == "" {
				candidate = ipStr
			}
			if !ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified() {
				return ipStr
			}
		}
		if candidate != "" {
			return candidate
		}
	}
	if xri = strings.TrimSpace(xri); xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return xri
		}
	}
	return ""
}

// realIP resolves r.RemoteAddr to the actual client IP observed by upstream proxies.
func realIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := resolveProxyIP(r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP")); ip != "" {
			r.RemoteAddr = net.JoinHostPort(ip, "0")
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP extracts the request's client IP from r.RemoteAddr or proxy headers.
func clientIP(r *http.Request) string {
	if ip := resolveProxyIP(r.Header.Get("X-Forwarded-For"), r.Header.Get("X-Real-IP")); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func writeTooManyRequests(w http.ResponseWriter) {
	writeErr(w, http.StatusTooManyRequests, "too many attempts — try again later")
}

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string]*requestWindow
	max      int
	window   time.Duration
}

type requestWindow struct {
	count   int
	resetAt time.Time
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	l := &rateLimiter{requests: make(map[string]*requestWindow), max: max, window: window}
	go l.cleanupLoop()
	return l
}

func (l *rateLimiter) allowInc(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	w, ok := l.requests[key]
	if !ok || now.After(w.resetAt) {
		l.requests[key] = &requestWindow{
			count:   1,
			resetAt: now.Add(l.window),
		}
		return true
	}
	if w.count >= l.max {
		return false
	}
	w.count++
	return true
}

func (l *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for k, w := range l.requests {
			if now.After(w.resetAt) {
				delete(l.requests, k)
			}
		}
		l.mu.Unlock()
	}
}

type dynamicRateLimiter struct {
	mu       sync.Mutex
	requests map[string]*requestWindow
	maxLimit int
	window   time.Duration
}

func newDynamicRateLimiter(initialMax int, window time.Duration) *dynamicRateLimiter {
	l := &dynamicRateLimiter{
		requests: make(map[string]*requestWindow),
		maxLimit: initialMax,
		window:   window,
	}
	go l.cleanupLoop()
	return l
}

func (l *dynamicRateLimiter) setMax(max int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxLimit = max
}

func (l *dynamicRateLimiter) allowInc(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	w, ok := l.requests[key]
	if !ok || now.After(w.resetAt) {
		l.requests[key] = &requestWindow{
			count:   1,
			resetAt: now.Add(l.window),
		}
		return true
	}
	if w.count >= l.maxLimit {
		return false
	}
	w.count++
	return true
}

func (l *dynamicRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for k, w := range l.requests {
			if now.After(w.resetAt) {
				delete(l.requests, k)
			}
		}
		l.mu.Unlock()
	}
}

func (s *Server) GlobalRateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.globalLimit == nil {
			next.ServeHTTP(w, r)
			return
		}
		ip := clientIP(r)
		if !s.globalLimit.allowInc(ip) {
			writeTooManyRequests(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

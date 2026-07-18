package httpapi

import (
	"sync"
	"time"
)

// authCodeStore holds short-lived, single-use exchange codes used to hand
// a login result (session token, or a 2FA-pending token) to the frontend
// after an OAuth redirect, without putting the token itself in a URL —
// which would otherwise sit in browser history and server access logs.
// The stored payload is the exact JSON body /auth/login would have
// returned, so the frontend's post-login handling doesn't need to know
// whether the session came from a password or a Google sign-in.
type authCodeStore struct {
	mu    sync.Mutex
	codes map[string]authCodeEntry
}

type authCodeEntry struct {
	payload   []byte
	expiresAt time.Time
}

func newAuthCodeStore() *authCodeStore {
	s := &authCodeStore{codes: make(map[string]authCodeEntry)}
	go s.cleanupLoop()
	return s
}

func (s *authCodeStore) put(code string, payload []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code] = authCodeEntry{payload: payload, expiresAt: time.Now().Add(2 * time.Minute)}
}

// take returns the payload for code and deletes it — a code can be
// exchanged exactly once.
func (s *authCodeStore) take(code string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.codes[code]
	delete(s.codes, code)
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.payload, true
}

func (s *authCodeStore) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, e := range s.codes {
			if now.After(e.expiresAt) {
				delete(s.codes, k)
			}
		}
		s.mu.Unlock()
	}
}

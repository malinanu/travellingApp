package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spotto/server/internal/config"
)

func TestMaxBytesMiddleware(t *testing.T) {
	s := &Server{}
	router := s.Router(config.Config{})

	// Create a valid JSON payload larger than 10MB
	largePayload := append([]byte(`{"email":"`), bytes.Repeat([]byte("a"), 11*1024*1024)...)
	largePayload = append(largePayload, []byte(`"}`)...)

	req := httptest.NewRequest("POST", "/auth/login", bytes.NewReader(largePayload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected status 413 Payload Too Large, got %d (body: %s)", w.Code, w.Body.String())
	}
}

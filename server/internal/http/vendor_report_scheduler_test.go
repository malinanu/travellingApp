package httpapi_test

import (
	"context"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

func TestVendorReportScheduler(t *testing.T) {
	store := mustConnect(t)
	t.Cleanup(store.Close)

	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{}
	srv := httpapi.New(store, tokens, cfg)

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	srv.StartVendorReportScheduler(ctx)
}

package httpapi

import (
	"log"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/payments"
	"github.com/spotto/server/internal/payments/fake"
	"github.com/spotto/server/internal/payments/payhere"
)

// selectGateway picks the payment gateway: PayHere when configured, the
// fake gateway outside production, none in production otherwise.
func selectGateway(cfg config.Config) payments.Gateway {
	if cfg.PayHereMerchantID != "" && cfg.PayHereMerchantSecret != "" {
		return payhere.New(payhere.Config{MerchantID: cfg.PayHereMerchantID,
			MerchantSecret: cfg.PayHereMerchantSecret, Sandbox: cfg.PayHereSandbox})
	}
	if cfg.IsProduction() {
		log.Printf("payments: no gateway configured (PAYHERE_*); paid bookings are disabled")
		return nil
	}
	secret := cfg.FakeGatewaySecret
	if secret == "" {
		secret = "dev-fake-gateway-secret"
	}
	return fake.New(secret, cfg.PublicAPIURL)
}

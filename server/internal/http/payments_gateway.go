package httpapi

import (
	"log"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/payments"
	"github.com/spotto/server/internal/payments/fake"
	"github.com/spotto/server/internal/payments/payhere"
)

// selectGateway picks the payment gateway: PayHere when configured; the
// fake gateway only outside production and only when FAKE_GATEWAY_SECRET
// is set; otherwise none, and paid bookings are refused.
func selectGateway(cfg config.Config) payments.Gateway {
	if cfg.PayHereMerchantID != "" && cfg.PayHereMerchantSecret != "" {
		if cfg.PublicAPIURL == "" {
			log.Printf("payments: PayHere needs PUBLIC_API_URL for its notify URL; paid bookings are disabled")
			return nil
		}
		return payhere.New(payhere.Config{MerchantID: cfg.PayHereMerchantID,
			MerchantSecret: cfg.PayHereMerchantSecret, Sandbox: cfg.PayHereSandbox})
	}
	if cfg.FakeGatewaySecret != "" && !cfg.IsProduction() {
		log.Printf("payments: using the FAKE gateway (FAKE_GATEWAY_SECRET is set): no real money moves")
		return fake.New(cfg.FakeGatewaySecret, cfg.PublicAPIURL)
	}
	log.Printf("payments: no gateway configured (PAYHERE_*); paid bookings are disabled")
	return nil
}

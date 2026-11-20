package httpapi

import (
	"context"
	"errors"
	"log"
	"net/url"
	"time"

	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/payments"
	"github.com/spotto/server/internal/payments/fake"
	"github.com/spotto/server/internal/payments/payhere"
	"github.com/spotto/server/internal/payments/paypal"
	"github.com/spotto/server/internal/store/postgres"
)

// buildGateways registers every configured payment gateway: PayHere (the
// default, in LKR) and PayPal (for travellers paying from abroad), each
// when its credentials are set; the fake gateway only outside production
// and only when FAKE_GATEWAY_SECRET is set. With none, paid bookings are
// refused.
func buildGateways(cfg config.Config) *payments.Registry {
	var gws []payments.Gateway
	if cfg.PayHereMerchantID != "" && cfg.PayHereMerchantSecret != "" {
		if cfg.PublicAPIURL == "" {
			log.Printf("payments: PayHere needs PUBLIC_API_URL for its notify URL; PayHere is disabled")
		} else {
			gws = append(gws, payhere.New(payhere.Config{MerchantID: cfg.PayHereMerchantID,
				MerchantSecret: cfg.PayHereMerchantSecret, Sandbox: cfg.PayHereSandbox}))
		}
	}
	if cfg.PayPalClientID != "" && cfg.PayPalClientSecret != "" {
		switch {
		case cfg.PayPalWebhookID == "":
			log.Printf("payments: PayPal needs PAYPAL_WEBHOOK_ID to verify its notifications; PayPal is disabled")
		case cfg.PublicAPIURL == "":
			log.Printf("payments: PayPal needs PUBLIC_API_URL for its return URL; PayPal is disabled")
		case cfg.PayPalCurrency == domain.CurrencyLKR || !domain.TwoDecimalCurrency(cfg.PayPalCurrency):
			log.Printf("payments: PAYPAL_CURRENCY=%q isn't supported (use e.g. USD); PayPal is disabled", cfg.PayPalCurrency)
		default:
			ppCfg := paypal.Config{ClientID: cfg.PayPalClientID, ClientSecret: cfg.PayPalClientSecret,
				WebhookID: cfg.PayPalWebhookID, Sandbox: cfg.PayPalSandbox, Currency: cfg.PayPalCurrency}
			if cfg.PayPalAPIBaseURL != "" && !cfg.IsProduction() {
				if u, err := url.Parse(cfg.PayPalAPIBaseURL); err == nil && u.Hostname() != "" {
					ppCfg.APIBaseURL, ppCfg.RedirectHosts = cfg.PayPalAPIBaseURL, []string{u.Hostname()}
				}
			}
			gws = append(gws, paypal.New(ppCfg))
		}
	}
	if cfg.FakeGatewaySecret != "" && !cfg.IsProduction() {
		log.Printf("payments: using the FAKE gateway (FAKE_GATEWAY_SECRET is set): no real money moves")
		gws = append(gws, fake.New(cfg.FakeGatewaySecret, cfg.PublicAPIURL))
	}
	if len(gws) == 0 {
		log.Printf("payments: no gateway configured (PAYHERE_* or PAYPAL_*); paid bookings are disabled")
	}
	return payments.NewRegistry(gws...)
}

// fakeGateway is the fake gateway when it's registered.
func (s *Server) fakeGateway() (*fake.Gateway, bool) {
	g, ok := s.Payments.Get("fake")
	if !ok {
		return nil, false
	}
	f, ok := g.(*fake.Gateway)
	return f, ok
}

// gatewayQuote is one way to pay for an amount: the gateway and what it
// will charge.
type gatewayQuote struct {
	Gateway      payments.Gateway
	ChargeMinor  int64
	Currency     string
	FxRateMicros int64 // 0 when no conversion
}

var (
	errUnknownGateway = errors.New("unknown payment method")
	// errFxUnavailable: the gateway charges another currency and there's
	// no current exchange rate for it.
	errFxUnavailable = errors.New("exchange rate unavailable")
)

// quoteGateway works out what gateway name ("" for the default) would
// charge for amountMinor in currency.
func (s *Server) quoteGateway(ctx context.Context, name string, amountMinor int64, currency string) (gatewayQuote, error) {
	var gw payments.Gateway
	if name == "" {
		gw = s.Payments.Default()
		if gw == nil {
			return gatewayQuote{}, errPaymentsUnavailable
		}
	} else {
		var ok bool
		if gw, ok = s.Payments.Get(name); !ok {
			return gatewayQuote{}, errUnknownGateway
		}
	}
	charge := gw.ChargeCurrency()
	if charge == currency {
		return gatewayQuote{Gateway: gw, ChargeMinor: amountMinor, Currency: currency}, nil
	}
	if currency != domain.CurrencyLKR {
		return gatewayQuote{}, errFxUnavailable
	}
	rate, err := s.Store.FxRate(ctx, charge)
	if errors.Is(err, postgres.ErrNotFound) || (err == nil && rate.Stale(time.Now())) {
		return gatewayQuote{}, errFxUnavailable
	}
	if err != nil {
		return gatewayQuote{}, err
	}
	converted := domain.ConvertFromLKR(amountMinor, rate.LKRPerUnitMicros, s.FxMarginBps)
	if converted <= 0 {
		return gatewayQuote{}, errFxUnavailable
	}
	return gatewayQuote{Gateway: gw, ChargeMinor: converted, Currency: charge, FxRateMicros: rate.LKRPerUnitMicros}, nil
}

// errHoldLapsed: the booking stopped waiting for this payment, so it
// isn't captured and the traveller isn't charged.
var errHoldLapsed = errors.New("booking no longer waiting for payment")

// capturePayment takes the money for a payment the traveller approved on
// a capture-style gateway, then applies the gateway's answer. It's safe to
// call twice (the return page and the gateway's webhook can race): the
// gateway reports the same capture and the event is applied once.
func (s *Server) capturePayment(ctx context.Context, gw payments.Gateway, pay postgres.Payment) error {
	c, ok := gw.(payments.Capturer)
	if !ok {
		return nil
	}
	if pay.Status == "succeeded" {
		return nil
	}
	if pay.Status != "created" {
		return errHoldLapsed
	}
	b, err := s.Store.BookingByID(ctx, pay.BookingID)
	if err != nil {
		return err
	}
	// Don't take money for a booking that has expired or been cancelled:
	// the approval simply lapses at the gateway.
	if b.Status != domain.BookingAwaitingPayment || b.HoldExpiresAt == nil || !b.HoldExpiresAt.After(time.Now()) {
		return errHoldLapsed
	}
	ev, err := c.Capture(ctx, pay.GatewayCheckoutRef)
	if err != nil {
		return err
	}
	if ev.OrderID == "" {
		ev.OrderID = pay.GatewayOrderID
	}
	_, err = s.applyPaymentEvent(ctx, gw.Name(), ev)
	if errors.Is(err, postgres.ErrPaymentMismatch) {
		return nil // recorded for finance to look at
	}
	return err
}

// paymentOption is one way a traveller can pay, as the app shows it.
type paymentOption struct {
	Gateway           string `json:"gateway"`
	DisplayName       string `json:"displayName"`
	ChargeCurrency    string `json:"chargeCurrency"`
	ChargeAmountMinor int64  `json:"chargeAmountMinor"`
	// Converted is set when the gateway charges another currency than
	// the booking's, at the current exchange rate.
	Converted bool `json:"converted"`
}

// paymentOptions lists the gateways that can take amountMinor LKR right
// now, the default first. A gateway without a current exchange rate is
// left out.
func (s *Server) paymentOptions(ctx context.Context, amountMinor int64) []paymentOption {
	out := []paymentOption{}
	for _, gw := range s.Payments.Enabled() {
		q, err := s.quoteGateway(ctx, gw.Name(), amountMinor, domain.CurrencyLKR)
		if err != nil {
			continue
		}
		out = append(out, paymentOption{Gateway: gw.Name(), DisplayName: gw.DisplayName(), ChargeCurrency: q.Currency,
			ChargeAmountMinor: q.ChargeMinor, Converted: q.Currency != domain.CurrencyLKR})
	}
	return out
}

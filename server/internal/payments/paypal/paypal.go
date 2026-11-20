// Package paypal connects Spotto to PayPal through the Orders v2 REST API:
// Spotto creates an order, sends the traveller to PayPal to approve it,
// then captures it server to server. PayPal reports results through
// webhooks, which are verified by asking PayPal itself.
//
// PayPal can't charge LKR, so this gateway charges in another currency
// (USD by default); the caller converts the amount before CreateCheckout.
//
// UNVERIFIED: written against PayPal's Orders v2, Payments v2 and webhook
// verification APIs as known when this was written; the developer
// documentation couldn't be reached from the build environment. Confirm
// the flow in the PayPal sandbox before going live.
//
// Set up in the PayPal developer dashboard: a REST app (client id and
// secret) and a webhook pointing at {PUBLIC_API_URL}/payments/paypal/notify
// subscribed to CHECKOUT.ORDER.APPROVED, PAYMENT.CAPTURE.COMPLETED,
// PAYMENT.CAPTURE.PENDING, PAYMENT.CAPTURE.DENIED, PAYMENT.CAPTURE.DECLINED,
// PAYMENT.CAPTURE.REVERSED and CUSTOMER.DISPUTE.CREATED; its id is
// PAYPAL_WEBHOOK_ID.
package paypal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/spotto/server/internal/payments"
)

const (
	sandboxAPI = "https://api-m.sandbox.paypal.com"
	liveAPI    = "https://api-m.paypal.com"

	maxWebhookBody = 64 << 10
	maxAPIResponse = 1 << 20
)

type Config struct {
	ClientID     string
	ClientSecret string
	WebhookID    string
	Sandbox      bool
	// Currency is what PayPal charges in (two-decimal, e.g. USD).
	Currency string
	// APIBaseURL overrides the API host (tests).
	APIBaseURL string
	// RedirectHosts overrides the hosts travellers may be sent to (tests).
	RedirectHosts []string
	HTTPClient    *http.Client
}

type Gateway struct {
	cfg    Config
	base   string
	client *http.Client
	hosts  map[string]bool

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

func New(cfg Config) *Gateway {
	base := liveAPI
	hosts := []string{"www.paypal.com", "paypal.com"}
	if cfg.Sandbox {
		base = sandboxAPI
		hosts = []string{"www.sandbox.paypal.com", "sandbox.paypal.com"}
	}
	if cfg.APIBaseURL != "" {
		base = strings.TrimRight(cfg.APIBaseURL, "/")
	}
	if len(cfg.RedirectHosts) > 0 {
		hosts = cfg.RedirectHosts
	}
	if cfg.Currency == "" {
		cfg.Currency = "USD"
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	g := &Gateway{cfg: cfg, base: base, client: client, hosts: map[string]bool{}}
	for _, h := range hosts {
		g.hosts[strings.ToLower(h)] = true
	}
	return g
}

func (g *Gateway) Name() string           { return "paypal" }
func (g *Gateway) DisplayName() string    { return "PayPal" }
func (g *Gateway) ChargeCurrency() string { return g.cfg.Currency }

// RedirectAllowed only lets travellers be sent to PayPal's own checkout.
func (g *Gateway) RedirectAllowed(u *url.URL) bool {
	if u == nil || u.User != nil {
		return false
	}
	if u.Scheme != "https" && !(u.Scheme == "http" && g.cfg.APIBaseURL != "") {
		return false
	}
	return g.hosts[strings.ToLower(u.Hostname())]
}

// ---- API plumbing ----

// apiError is a PayPal API error response.
type apiError struct {
	Status  int
	Name    string `json:"name"`
	Message string `json:"message"`
	Details []struct {
		Issue string `json:"issue"`
	} `json:"details"`
}

func (e *apiError) Error() string {
	issue := ""
	if len(e.Details) > 0 {
		issue = " (" + e.Details[0].Issue + ")"
	}
	return fmt.Sprintf("paypal: HTTP %d %s%s: %s", e.Status, e.Name, issue, e.Message)
}

func (e *apiError) hasIssue(issue string) bool {
	for _, d := range e.Details {
		if d.Issue == issue {
			return true
		}
	}
	return false
}

func (g *Gateway) accessToken(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.token != "" && time.Now().Before(g.tokenExp) {
		return g.token, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.base+"/v1/oauth2/token",
		strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return "", err
	}
	req.SetBasicAuth(g.cfg.ClientID, g.cfg.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("paypal: token: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponse))
	if resp.StatusCode != http.StatusOK {
		// Never echo the body: it can describe the credentials.
		return "", fmt.Errorf("paypal: token: HTTP %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return "", errors.New("paypal: token: unreadable response")
	}
	g.token = tok.AccessToken
	// Renew a minute early so a token never expires mid-request.
	g.tokenExp = time.Now().Add(time.Duration(tok.ExpiresIn)*time.Second - time.Minute)
	return g.token, nil
}

// call makes one authenticated JSON request. requestID, when set, is sent
// as PayPal-Request-Id so a retried request isn't carried out twice.
func (g *Gateway) call(ctx context.Context, method, path, requestID string, in, out any) error {
	tok, err := g.accessToken(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			return err
		}
		body = bytes.NewReader(b)
	}
	return g.send(ctx, method, path, requestID, tok, body, out)
}

func (g *Gateway) send(ctx context.Context, method, path, requestID, tok string, body io.Reader, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, g.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Prefer", "return=representation")
	if requestID != "" {
		req.Header.Set("PayPal-Request-Id", requestID)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("paypal: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, maxAPIResponse))
	if resp.StatusCode == http.StatusUnauthorized {
		g.mu.Lock()
		g.token = ""
		g.mu.Unlock()
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		e := &apiError{Status: resp.StatusCode}
		json.Unmarshal(raw, e)
		return e
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("paypal: %s %s: unreadable response: %w", method, path, err)
		}
	}
	return nil
}

// ---- checkout ----

type money struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type capture struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Amount    money  `json:"amount"`
	CustomID  string `json:"custom_id"`
	InvoiceID string `json:"invoice_id"`
}

type order struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	Links         []link `json:"links"`
	PurchaseUnits []struct {
		ReferenceID string `json:"reference_id"`
		CustomID    string `json:"custom_id"`
		InvoiceID   string `json:"invoice_id"`
		Payments    struct {
			Captures []capture `json:"captures"`
		} `json:"payments"`
	} `json:"purchase_units"`
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

func (g *Gateway) CreateCheckout(ctx context.Context, req payments.CheckoutRequest) (payments.CheckoutSession, error) {
	if req.Currency != g.cfg.Currency {
		return payments.CheckoutSession{}, fmt.Errorf("paypal: asked to charge %s, configured for %s", req.Currency, g.cfg.Currency)
	}
	body := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{{
			"reference_id": req.OrderID,
			"custom_id":    req.OrderID,
			"invoice_id":   req.OrderID,
			"description":  truncate(req.Description, 127),
			"amount":       money{CurrencyCode: req.Currency, Value: payments.FormatAmount(req.AmountMinor)},
		}},
		"payment_source": map[string]any{"paypal": map[string]any{"experience_context": map[string]any{
			"brand_name":          "Spotto",
			"user_action":         "PAY_NOW",
			"shipping_preference": "NO_SHIPPING",
			"return_url":          req.ReturnURL,
			"cancel_url":          req.CancelURL,
		}}},
	}
	var o order
	if err := g.call(ctx, http.MethodPost, "/v2/checkout/orders", "order-"+req.OrderID, body, &o); err != nil {
		return payments.CheckoutSession{}, err
	}
	approve := ""
	for _, l := range o.Links {
		if l.Rel == "payer-action" || (l.Rel == "approve" && approve == "") {
			approve = l.Href
		}
	}
	if o.ID == "" || approve == "" {
		return payments.CheckoutSession{}, errors.New("paypal: order created without an approval link")
	}
	u, err := url.Parse(approve)
	if err != nil || !g.RedirectAllowed(u) {
		return payments.CheckoutSession{}, fmt.Errorf("paypal: unexpected approval link host")
	}
	return payments.CheckoutSession{URL: approve, Method: payments.CheckoutGet, GatewayRef: o.ID}, nil
}

// Capture takes the money for an order the traveller approved. Capturing
// an order twice is safe: the second attempt reports the first capture.
func (g *Gateway) Capture(ctx context.Context, orderRef string) (payments.NotifyEvent, error) {
	if orderRef == "" || strings.ContainsAny(orderRef, "/?#") {
		return payments.NotifyEvent{}, errors.New("paypal: bad order id")
	}
	var o order
	err := g.call(ctx, http.MethodPost, "/v2/checkout/orders/"+orderRef+"/capture", "capture-"+orderRef, map[string]any{}, &o)
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.hasIssue("ORDER_ALREADY_CAPTURED") {
		o = order{}
		err = g.call(ctx, http.MethodGet, "/v2/checkout/orders/"+orderRef, "", nil, &o)
	}
	if err != nil {
		return payments.NotifyEvent{}, err
	}
	for _, pu := range o.PurchaseUnits {
		for _, c := range pu.Payments.Captures {
			if c.CustomID == "" {
				c.CustomID = pu.CustomID
			}
			if c.InvoiceID == "" {
				c.InvoiceID = pu.InvoiceID
			}
			if c.CustomID == "" && c.InvoiceID == "" {
				c.CustomID = pu.ReferenceID
			}
			return captureEvent(c)
		}
	}
	return payments.NotifyEvent{}, fmt.Errorf("paypal: order %s has no capture (status %s)", o.ID, o.Status)
}

func captureStatus(s string) (payments.EventStatus, bool) {
	switch s {
	case "COMPLETED":
		return payments.EventSucceeded, true
	case "PENDING":
		return payments.EventPending, true
	case "DECLINED", "DENIED", "FAILED":
		return payments.EventFailed, true
	}
	return "", false
}

func captureEvent(c capture) (payments.NotifyEvent, error) {
	status, ok := captureStatus(c.Status)
	if !ok || c.ID == "" {
		return payments.NotifyEvent{}, payments.ErrIgnoredEvent
	}
	orderID := c.CustomID
	if orderID == "" {
		orderID = c.InvoiceID
	}
	amount, err := payments.ParseAmount(c.Amount.Value)
	if err != nil && status == payments.EventSucceeded {
		return payments.NotifyEvent{}, fmt.Errorf("paypal: capture %s: %w", c.ID, err)
	}
	// Keyed by capture and status, so the capture Spotto made itself and
	// PayPal's webhook about it are recognised as the same event.
	return payments.NotifyEvent{
		EventKey:         "paypal:capture:" + c.ID + ":" + strings.ToLower(c.Status),
		OrderID:          orderID,
		GatewayPaymentID: c.ID,
		Status:           status,
		AmountMinor:      amount,
		Currency:         c.Amount.CurrencyCode,
		Raw:              map[string]string{"capture_id": c.ID, "status": c.Status, "amount": c.Amount.Value, "currency": c.Amount.CurrencyCode},
	}, nil
}

// ---- webhooks ----

type webhookEvent struct {
	ID        string          `json:"id"`
	EventType string          `json:"event_type"`
	Resource  json.RawMessage `json:"resource"`
}

// VerifyNotify checks a webhook with PayPal's verification API and maps
// it to a payment event. Errors other than ErrBadSignature and
// ErrIgnoredEvent are temporary (PayPal unreachable): PayPal should retry.
func (g *Gateway) VerifyNotify(r *http.Request) (payments.NotifyEvent, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody+1))
	if err != nil || len(raw) > maxWebhookBody || !json.Valid(raw) {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	h := map[string]string{
		"auth_algo":         r.Header.Get("PAYPAL-AUTH-ALGO"),
		"cert_url":          r.Header.Get("PAYPAL-CERT-URL"),
		"transmission_id":   r.Header.Get("PAYPAL-TRANSMISSION-ID"),
		"transmission_sig":  r.Header.Get("PAYPAL-TRANSMISSION-SIG"),
		"transmission_time": r.Header.Get("PAYPAL-TRANSMISSION-TIME"),
	}
	for _, v := range h {
		if v == "" {
			return payments.NotifyEvent{}, payments.ErrBadSignature
		}
	}
	if g.cfg.WebhookID == "" {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	h["webhook_id"] = g.cfg.WebhookID
	// The event goes back byte for byte, spliced in rather than re-encoded.
	head, _ := json.Marshal(h)
	body := append(head[:len(head)-1:len(head)-1], []byte(`,"webhook_event":`)...)
	body = append(body, raw...)
	body = append(body, '}')
	tok, err := g.accessToken(r.Context())
	if err != nil {
		return payments.NotifyEvent{}, err
	}
	var res struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := g.send(r.Context(), http.MethodPost, "/v1/notifications/verify-webhook-signature", "", tok, bytes.NewReader(body), &res); err != nil {
		var apiErr *apiError
		if errors.As(err, &apiErr) && apiErr.Status >= 400 && apiErr.Status < 500 && apiErr.Status != http.StatusUnauthorized && apiErr.Status != http.StatusTooManyRequests {
			return payments.NotifyEvent{}, payments.ErrBadSignature
		}
		return payments.NotifyEvent{}, err
	}
	if res.VerificationStatus != "SUCCESS" {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}

	var ev webhookEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return payments.NotifyEvent{}, payments.ErrBadSignature
	}
	return parseEvent(ev)
}

func parseEvent(ev webhookEvent) (payments.NotifyEvent, error) {
	switch ev.EventType {
	case "CHECKOUT.ORDER.APPROVED":
		var o order
		if err := json.Unmarshal(ev.Resource, &o); err != nil || o.ID == "" {
			return payments.NotifyEvent{}, payments.ErrIgnoredEvent
		}
		orderID := ""
		if len(o.PurchaseUnits) > 0 {
			orderID = o.PurchaseUnits[0].CustomID
		}
		return payments.NotifyEvent{EventKey: "paypal:event:" + ev.ID, OrderID: orderID, Status: payments.EventApproved,
			CheckoutRef: o.ID, Raw: map[string]string{"event": ev.EventType, "order": o.ID}}, nil
	case "PAYMENT.CAPTURE.COMPLETED", "PAYMENT.CAPTURE.PENDING", "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED":
		var c capture
		if err := json.Unmarshal(ev.Resource, &c); err != nil {
			return payments.NotifyEvent{}, payments.ErrIgnoredEvent
		}
		return captureEvent(c)
	case "PAYMENT.CAPTURE.REVERSED":
		// The money was pulled back (usually a bank chargeback).
		var res struct {
			ID        string `json:"id"`
			CustomID  string `json:"custom_id"`
			InvoiceID string `json:"invoice_id"`
		}
		json.Unmarshal(ev.Resource, &res)
		orderID := res.CustomID
		if orderID == "" {
			orderID = res.InvoiceID
		}
		if orderID == "" {
			return payments.NotifyEvent{}, payments.ErrIgnoredEvent
		}
		return payments.NotifyEvent{EventKey: "paypal:event:" + ev.ID, OrderID: orderID, Status: payments.EventChargeback,
			Raw: map[string]string{"event": ev.EventType, "resource": res.ID}}, nil
	case "CUSTOMER.DISPUTE.CREATED":
		var d struct {
			DisputeID    string `json:"dispute_id"`
			Transactions []struct {
				SellerTransactionID string `json:"seller_transaction_id"`
				InvoiceNumber       string `json:"invoice_number"`
				Custom              string `json:"custom"`
			} `json:"disputed_transactions"`
		}
		json.Unmarshal(ev.Resource, &d)
		if len(d.Transactions) == 0 {
			return payments.NotifyEvent{}, payments.ErrIgnoredEvent
		}
		t := d.Transactions[0]
		orderID := t.Custom
		if orderID == "" {
			orderID = t.InvoiceNumber
		}
		if orderID == "" {
			return payments.NotifyEvent{}, payments.ErrIgnoredEvent
		}
		return payments.NotifyEvent{EventKey: "paypal:event:" + ev.ID, OrderID: orderID, GatewayPaymentID: t.SellerTransactionID,
			Status: payments.EventChargeback, Raw: map[string]string{"event": ev.EventType, "dispute": d.DisputeID}}, nil
	}
	return payments.NotifyEvent{}, payments.ErrIgnoredEvent
}

// ---- refunds ----

// Refund pays back (part of) a capture.
func (g *Gateway) Refund(ctx context.Context, req payments.RefundRequest) (payments.RefundResult, error) {
	if req.GatewayPaymentID == "" || strings.ContainsAny(req.GatewayPaymentID, "/?#") {
		return payments.RefundResult{}, errors.New("paypal: refund needs the capture id")
	}
	body := map[string]any{
		"amount":        money{CurrencyCode: req.Currency, Value: payments.FormatAmount(req.AmountMinor)},
		"note_to_payer": truncate(req.Reason, 255),
	}
	var res struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	key := req.IdempotencyKey
	if key != "" {
		key = "refund-" + key
	}
	if err := g.call(ctx, http.MethodPost, "/v2/payments/captures/"+req.GatewayPaymentID+"/refund", key, body, &res); err != nil {
		return payments.RefundResult{}, err
	}
	if res.Status == "FAILED" || res.Status == "CANCELLED" {
		return payments.RefundResult{}, fmt.Errorf("paypal: refund %s %s", res.ID, res.Status)
	}
	return payments.RefundResult{GatewayRefundID: res.ID}, nil
}

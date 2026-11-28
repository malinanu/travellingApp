// Package paypaltest is a stand-in for PayPal's REST API, for tests: it
// creates orders, lets a test approve them as the buyer would, captures,
// refunds, and verifies webhooks signed with ValidSignature.
package paypaltest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// ValidSignature is the webhook transmission signature the stand-in
// accepts; anything else fails verification.
const ValidSignature = "valid-test-signature"

type Order struct {
	ID        string
	CustomID  string
	Currency  string
	Value     string
	Approved  bool
	CaptureID string
	ReturnURL string
	CancelURL string
}

type Refund struct {
	CaptureID string
	Currency  string
	Value     string
	RequestID string
}

type Server struct {
	*httptest.Server

	mu         sync.Mutex
	orders     map[string]*Order
	refunds    []Refund
	refundKeys map[string]string
	n          int
	// CaptureCalls counts capture requests.
	CaptureCalls int
}

func New() *Server {
	s := &Server{orders: map[string]*Order{}, refundKeys: map[string]string{}}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	return s
}

// Host is the stand-in's host:port, to allow as a redirect target.
func (s *Server) Host() string { return strings.TrimPrefix(s.URL, "http://") }

// Approve plays the buyer approving the order on PayPal.
func (s *Server) Approve(orderID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if o, ok := s.orders[orderID]; ok {
		o.Approved = true
	}
}

func (s *Server) Order(id string) (Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok {
		return Order{}, false
	}
	return *o, true
}

// Orders lists the orders created so far.
func (s *Server) Orders() []Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Order{}
	for _, o := range s.orders {
		out = append(out, *o)
	}
	return out
}

func (s *Server) Refunds() []Refund {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Refund(nil), s.refunds...)
}

// WebhookRequest builds a webhook delivery of event (a PayPal event
// object) as PayPal would send it, signed or not.
func WebhookRequest(target string, event any, signed bool) *http.Request {
	b, _ := json.Marshal(event)
	req := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	sig := "forged"
	if signed {
		sig = ValidSignature
	}
	req.Header.Set("PAYPAL-AUTH-ALGO", "SHA256withRSA")
	req.Header.Set("PAYPAL-CERT-URL", "https://api.sandbox.paypal.com/v1/notifications/certs/CERT")
	req.Header.Set("PAYPAL-TRANSMISSION-ID", "tx-1")
	req.Header.Set("PAYPAL-TRANSMISSION-SIG", sig)
	req.Header.Set("PAYPAL-TRANSMISSION-TIME", "2026-11-20T10:00:00Z")
	return req
}

// CaptureCompletedEvent is the webhook PayPal sends once a capture
// completes.
func (s *Server) CaptureCompletedEvent(orderID string) map[string]any {
	o, _ := s.Order(orderID)
	return map[string]any{
		"id": "WH-" + o.CaptureID, "event_type": "PAYMENT.CAPTURE.COMPLETED",
		"resource": map[string]any{"id": o.CaptureID, "status": "COMPLETED", "custom_id": o.CustomID, "invoice_id": o.CustomID,
			"amount": map[string]string{"currency_code": o.Currency, "value": o.Value}},
	}
}

// ApprovedEvent is the webhook PayPal sends once the buyer approves.
func (s *Server) ApprovedEvent(orderID string) map[string]any {
	o, _ := s.Order(orderID)
	return map[string]any{
		"id": "WH-APPROVED-" + o.ID, "event_type": "CHECKOUT.ORDER.APPROVED",
		"resource": map[string]any{"id": o.ID, "status": "APPROVED", "purchase_units": []map[string]any{{"custom_id": o.CustomID}}},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func issue(w http.ResponseWriter, status int, name, issue string) {
	writeJSON(w, status, map[string]any{"name": name, "message": issue, "details": []map[string]string{{"issue": issue}}})
}

func (s *Server) orderJSON(o *Order) map[string]any {
	pu := map[string]any{"reference_id": o.CustomID, "custom_id": o.CustomID, "invoice_id": o.CustomID}
	status := "PAYER_ACTION_REQUIRED"
	if o.Approved {
		status = "APPROVED"
	}
	if o.CaptureID != "" {
		status = "COMPLETED"
		pu["payments"] = map[string]any{"captures": []map[string]any{{"id": o.CaptureID, "status": "COMPLETED",
			"amount": map[string]string{"currency_code": o.Currency, "value": o.Value}}}}
	}
	return map[string]any{"id": o.ID, "status": status, "purchase_units": []any{pu},
		"links": []map[string]string{{"rel": "payer-action", "href": s.URL + "/checkoutnow?token=" + o.ID}}}
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := r.URL.Path
	if path != "/v1/oauth2/token" && !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer test-token") {
		issue(w, http.StatusUnauthorized, "AUTHENTICATION_FAILURE", "INVALID_TOKEN")
		return
	}
	switch {
	case path == "/v1/oauth2/token":
		if u, p, ok := r.BasicAuth(); !ok || u == "" || p == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_token": "test-token", "expires_in": 32400})

	case path == "/v2/checkout/orders" && r.Method == http.MethodPost:
		var in struct {
			PurchaseUnits []struct {
				CustomID string `json:"custom_id"`
				Amount   struct {
					CurrencyCode string `json:"currency_code"`
					Value        string `json:"value"`
				} `json:"amount"`
			} `json:"purchase_units"`
			PaymentSource struct {
				PayPal struct {
					Ctx struct {
						ReturnURL string `json:"return_url"`
						CancelURL string `json:"cancel_url"`
					} `json:"experience_context"`
				} `json:"paypal"`
			} `json:"payment_source"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.PurchaseUnits) != 1 {
			issue(w, http.StatusBadRequest, "INVALID_REQUEST", "MALFORMED_REQUEST_JSON")
			return
		}
		s.n++
		pu := in.PurchaseUnits[0]
		o := &Order{ID: fmt.Sprintf("PPORDER%04d", s.n), CustomID: pu.CustomID, Currency: pu.Amount.CurrencyCode,
			Value: pu.Amount.Value, ReturnURL: in.PaymentSource.PayPal.Ctx.ReturnURL, CancelURL: in.PaymentSource.PayPal.Ctx.CancelURL}
		s.orders[o.ID] = o
		writeJSON(w, http.StatusOK, s.orderJSON(o))

	case strings.HasPrefix(path, "/v2/checkout/orders/") && strings.HasSuffix(path, "/capture"):
		s.CaptureCalls++
		id := strings.TrimSuffix(strings.TrimPrefix(path, "/v2/checkout/orders/"), "/capture")
		o, ok := s.orders[id]
		switch {
		case !ok:
			issue(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", "INVALID_RESOURCE_ID")
		case !o.Approved:
			issue(w, http.StatusUnprocessableEntity, "UNPROCESSABLE_ENTITY", "ORDER_NOT_APPROVED")
		case o.CaptureID != "":
			issue(w, http.StatusUnprocessableEntity, "UNPROCESSABLE_ENTITY", "ORDER_ALREADY_CAPTURED")
		default:
			o.CaptureID = "CAP" + o.ID
			writeJSON(w, http.StatusCreated, s.orderJSON(o))
		}

	case strings.HasPrefix(path, "/v2/checkout/orders/") && r.Method == http.MethodGet:
		o, ok := s.orders[strings.TrimPrefix(path, "/v2/checkout/orders/")]
		if !ok {
			issue(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", "INVALID_RESOURCE_ID")
			return
		}
		writeJSON(w, http.StatusOK, s.orderJSON(o))

	case path == "/v1/notifications/verify-webhook-signature":
		var in struct {
			TransmissionSig string          `json:"transmission_sig"`
			WebhookID       string          `json:"webhook_id"`
			Event           json.RawMessage `json:"webhook_event"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in.Event) == 0 || in.WebhookID == "" {
			issue(w, http.StatusBadRequest, "VALIDATION_ERROR", "INVALID_REQUEST")
			return
		}
		status := "FAILURE"
		if in.TransmissionSig == ValidSignature {
			status = "SUCCESS"
		}
		writeJSON(w, http.StatusOK, map[string]string{"verification_status": status})

	case strings.HasPrefix(path, "/v2/payments/captures/") && strings.HasSuffix(path, "/refund"):
		capID := strings.TrimSuffix(strings.TrimPrefix(path, "/v2/payments/captures/"), "/refund")
		reqID := r.Header.Get("PayPal-Request-Id")
		if prev, ok := s.refundKeys[reqID]; ok && reqID != "" {
			writeJSON(w, http.StatusOK, map[string]string{"id": prev, "status": "COMPLETED"})
			return
		}
		var in struct {
			Amount struct {
				CurrencyCode string `json:"currency_code"`
				Value        string `json:"value"`
			} `json:"amount"`
		}
		json.NewDecoder(r.Body).Decode(&in)
		s.refunds = append(s.refunds, Refund{CaptureID: capID, Currency: in.Amount.CurrencyCode, Value: in.Amount.Value, RequestID: reqID})
		id := fmt.Sprintf("REFUND%d", len(s.refunds))
		if reqID != "" {
			s.refundKeys[reqID] = id
		}
		writeJSON(w, http.StatusCreated, map[string]string{"id": id, "status": "COMPLETED"})

	default:
		issue(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", "UNKNOWN_PATH")
	}
}

package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/payments"
	"github.com/spotto/server/internal/payments/fake"
	"github.com/spotto/server/internal/store/postgres"
)

// checkoutInfo is what the app needs to send the traveller to pay.
type checkoutInfo struct {
	PaymentID string `json:"paymentId"`
	Gateway   string `json:"gateway"`
	// URL is a page on this API that forwards to the gateway's hosted
	// checkout; the app opens it in a web view or the browser.
	URL string `json:"url"`
	// ReturnURLPrefix is where the gateway sends the traveller back; the
	// app can close its web view when it sees it.
	ReturnURLPrefix string     `json:"returnUrlPrefix"`
	ExpiresAt       *time.Time `json:"expiresAt,omitempty"`
}

// bookingWithCheckout is a booking plus, for one awaiting payment, how to
// pay for it.
type bookingWithCheckout struct {
	domain.Booking
	Checkout *checkoutInfo `json:"checkout,omitempty"`
}

// publicBaseURL is this API's own public URL: PUBLIC_API_URL, or (outside
// production only) the request's own host.
func (s *Server) publicBaseURL(r *http.Request) string {
	if s.PublicAPIURL != "" {
		return s.PublicAPIURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// startCheckout opens a gateway payment for a booking awaiting payment.
func (s *Server) startCheckout(r *http.Request, b domain.Booking) (*checkoutInfo, error) {
	if s.Payments == nil {
		return nil, errPaymentsUnavailable
	}
	base := s.publicBaseURL(r)
	if f, ok := s.Payments.(*fake.Gateway); ok && f.CheckoutBaseURL == "" {
		f.CheckoutBaseURL = base
	}
	gw := s.Payments.Name()
	pay := postgres.Payment{ID: newID("pay"), BookingID: b.ID, Gateway: gw, AmountMinor: b.AmountMinor, Currency: b.Currency}
	pay.GatewayOrderID = pay.ID
	customer := payments.Customer{}
	if me, err := s.Store.UserByID(r.Context(), b.TravellerID); err == nil {
		first, last, _ := strings.Cut(strings.TrimSpace(me.Name), " ")
		customer = payments.Customer{FirstName: first, LastName: last, Email: me.Email, Phone: me.Phone}
	}
	returnURL := base + "/payments/return?booking=" + url.QueryEscape(b.ID)
	session, err := s.Payments.CreateCheckout(r.Context(), payments.CheckoutRequest{
		OrderID: pay.GatewayOrderID, AmountMinor: b.AmountMinor, Currency: b.Currency,
		Description: fmt.Sprintf("%s (%s)", b.ServiceName, b.Code), Customer: customer,
		ReturnURL: returnURL, CancelURL: returnURL + "&cancelled=1", NotifyURL: base + "/payments/" + gw + "/notify",
	})
	if err != nil {
		return nil, err
	}
	pay.CheckoutURL, pay.CheckoutFields = session.URL, session.Fields
	if err := s.Store.CreatePayment(r.Context(), pay); err != nil {
		return nil, err
	}
	return &checkoutInfo{PaymentID: pay.ID, Gateway: gw, URL: base + "/payments/checkout/" + pay.ID,
		ReturnURLPrefix: base + "/payments/return", ExpiresAt: b.HoldExpiresAt}, nil
}

var errPaymentsUnavailable = errors.New("payments unavailable")

var checkoutPageTmpl = template.Must(template.New("checkout").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Spotto · Secure payment</title>
<style>body{font-family:system-ui,sans-serif;background:#faf5ec;color:#123d3a;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}
main{max-width:360px;padding:24px;text-align:center}button{background:#f6b821;color:#123d3a;border:0;border-radius:12px;padding:14px 24px;font-weight:700;font-size:16px}</style>
</head><body><main>
<h1>Taking you to secure payment…</h1>
<p>You pay on {{.Gateway}}'s own page. Spotto never sees your card details.</p>
<form id="pay" method="post" action="{{.URL}}">{{range $k, $v := .Fields}}<input type="hidden" name="{{$k}}" value="{{$v}}">{{end}}
<button type="submit">Continue to payment</button></form>
<script nonce="{{.Nonce}}">document.getElementById('pay').submit();</script>
</main></body></html>`))

// handleCheckoutPage forwards the traveller to the gateway's hosted
// checkout with the payment's signed form.
func (s *Server) handleCheckoutPage(w http.ResponseWriter, r *http.Request) {
	pay, err := s.Store.PaymentByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || pay.Status != "created" {
		writeHTMLMessage(w, http.StatusGone, "This payment link has expired", "Go back to the Spotto app to try again.")
		return
	}
	b, err := s.Store.BookingByID(r.Context(), pay.BookingID)
	if err != nil || b.Status != domain.BookingAwaitingPayment || (b.HoldExpiresAt != nil && b.HoldExpiresAt.Before(time.Now())) {
		writeHTMLMessage(w, http.StatusGone, "This payment link has expired", "Go back to the Spotto app to try again.")
		return
	}
	nonce := randomNonce()
	action, _ := url.Parse(pay.CheckoutURL)
	h := w.Header()
	h.Set("Content-Security-Policy", fmt.Sprintf("default-src 'none'; style-src 'unsafe-inline'; script-src 'nonce-%s'; form-action %s://%s; frame-ancestors 'none'; base-uri 'none'", nonce, action.Scheme, action.Host))
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer")
	gw := "the payment provider"
	if pay.Gateway == "payhere" {
		gw = "PayHere"
	}
	if err := checkoutPageTmpl.Execute(w, map[string]any{"URL": pay.CheckoutURL, "Fields": pay.CheckoutFields, "Nonce": nonce, "Gateway": gw}); err != nil {
		log.Printf("checkout page: %v", err)
	}
}

func randomNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.StdEncoding.EncodeToString(b)
}

var messagePageTmpl = template.Must(template.New("msg").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Spotto</title>
<style>body{font-family:system-ui,sans-serif;background:#faf5ec;color:#123d3a;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}main{max-width:360px;padding:24px;text-align:center}</style>
</head><body><main><h1>{{.Title}}</h1><p>{{.Body}}</p></main></body></html>`))

func writeHTMLMessage(w http.ResponseWriter, status int, title, body string) {
	h := w.Header()
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; frame-ancestors 'none'; base-uri 'none'")
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	messagePageTmpl.Execute(w, map[string]string{"Title": title, "Body": body})
}

// handlePaymentReturn is where the gateway sends the traveller back. It
// doesn't decide anything: only the signed notification confirms a
// payment, and the app reads the result from the booking's status.
func (s *Server) handlePaymentReturn(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("cancelled") == "1" {
		writeHTMLMessage(w, http.StatusOK, "Payment cancelled", "Nothing was charged. Go back to the Spotto app to try again.")
		return
	}
	writeHTMLMessage(w, http.StatusOK, "Thanks! Back to Spotto", "We're confirming your payment. Your booking and its QR code are waiting in the Spotto app.")
}

// handlePaymentNotify takes the gateway's server-to-server result. Only a
// correctly signed notification is applied, and each one only once.
func (s *Server) handlePaymentNotify(w http.ResponseWriter, r *http.Request) {
	if s.Payments == nil || chi.URLParam(r, "gateway") != s.Payments.Name() {
		writeErr(w, http.StatusNotFound, "unknown gateway")
		return
	}
	ev, err := s.Payments.VerifyNotify(r)
	if err != nil {
		log.Printf("payments: rejected %s notification: %v", s.Payments.Name(), err)
		writeErr(w, http.StatusBadRequest, "invalid notification")
		return
	}
	out, err := s.applyPaymentEvent(r.Context(), ev)
	if err != nil && !errors.Is(err, postgres.ErrPaymentMismatch) {
		// Let the gateway retry later.
		writeErr(w, http.StatusInternalServerError, "could not record payment")
		return
	}
	if out.Problem != "" {
		log.Printf("payments: %s order %s: %s", ev.EventKey, ev.OrderID, out.Problem)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// applyPaymentEvent applies a verified notification and does the
// follow-up work outside the transaction: confirmation messages, or the
// refund when the booking couldn't be kept.
func (s *Server) applyPaymentEvent(ctx context.Context, ev payments.NotifyEvent) (postgres.PaymentOutcome, error) {
	out, err := s.Store.ApplyPaymentEvent(ctx, postgres.PaymentEvent{
		Gateway: s.Payments.Name(), EventKey: ev.EventKey, OrderID: ev.OrderID, GatewayPaymentID: ev.GatewayPaymentID,
		Status: string(ev.Status), AmountMinor: ev.AmountMinor, Currency: ev.Currency, Payload: ev.Raw,
	})
	if err != nil {
		return out, err
	}
	if out.Confirmed {
		s.notifyBookingConfirmed(ctx, out.Booking)
	}
	if out.Refund != nil {
		s.processRefund(ctx, *out.Refund)
	}
	return out, nil
}

// processRefund asks the gateway to pay a recorded refund. A gateway that
// can't refund automatically leaves it for the finance team ("manual").
func (s *Server) processRefund(ctx context.Context, rf postgres.Refund) {
	if s.Payments == nil || rf.GatewayPaymentID == "" || rf.Gateway != s.Payments.Name() {
		s.Store.SetRefundStatus(ctx, rf.ID, "manual", "")
		return
	}
	res, err := s.Payments.Refund(ctx, payments.RefundRequest{OrderID: rf.GatewayOrderID, GatewayPaymentID: rf.GatewayPaymentID,
		AmountMinor: rf.AmountMinor, Currency: rf.Currency, Reason: rf.Reason})
	switch {
	case errors.Is(err, payments.ErrRefundUnsupported):
		s.Store.SetRefundStatus(ctx, rf.ID, "manual", "")
	case err != nil:
		log.Printf("payments: refund %s failed: %v", rf.ID, err)
		s.Store.SetRefundStatus(ctx, rf.ID, "failed", "")
	default:
		s.Store.SetRefundStatus(ctx, rf.ID, "succeeded", res.GatewayRefundID)
	}
}

// notifyBookingConfirmed tells the traveller (email) and the vendor's team
// (dashboard notification and push) about a confirmed booking.
func (s *Server) notifyBookingConfirmed(ctx context.Context, b domain.Booking) {
	serviceName := b.ServiceName
	if serviceName == "" {
		if svc, err := s.Store.BookableServiceByID(ctx, b.ServiceID); err == nil {
			serviceName = svc.Name
		}
	}
	if me, err := s.Store.UserByID(ctx, b.TravellerID); err == nil && me.Email != "" {
		s.Jobs.Enqueue(func() {
			s.Mail.Send(me.Email, "Your Spotto booking is confirmed", bookingConfirmedEmailHTML(b, serviceName))
		})
	}
	sub := fmt.Sprintf("%s, party of %d", inSriLanka(b.StartAt).Format("Jan 2, 3:04 PM"), b.PartySize)
	if b.AmountMinor > 0 {
		sub += ", paid " + formatLKR(b.AmountMinor)
	}
	s.Store.UpsertVendorNotification(ctx, b.VendorID, domain.VendorNotification{
		ID: "notif_booking_" + b.ID, Icon: "Calendar", Tone: "teal",
		Title: "New booking: " + serviceName, Sub: sub, Time: "just now", Unread: true,
	})
	s.Jobs.Enqueue(func() {
		s.dispatchVendorPush(b.VendorID, "New booking: "+serviceName, sub, map[string]string{"type": "booking", "bookingId": b.ID})
	})
}

// formatLKR renders minor units as "LKR 4,500" (or "LKR 4,500.50").
func formatLKR(minor int64) string {
	whole, cents := minor/100, minor%100
	digits := fmt.Sprint(whole)
	var b strings.Builder
	for i, c := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if cents != 0 {
		return fmt.Sprintf("LKR %s.%02d", b.String(), cents)
	}
	return "LKR " + b.String()
}

// ---- traveller endpoints ----

// ownBooking loads one of the calling traveller's bookings.
func (s *Server) ownBooking(w http.ResponseWriter, r *http.Request) (domain.Booking, bool) {
	c := auth.FromContext(r.Context())
	b, err := s.Store.BookingByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || b.TravellerID != c.Subject {
		writeErr(w, http.StatusNotFound, "The requested item could not be found.")
		return b, false
	}
	return b, true
}

// handleBookingStatus is polled by the app while the traveller pays.
func (s *Server) handleBookingStatus(w http.ResponseWriter, r *http.Request) {
	b, ok := s.ownBooking(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": b.ID, "code": b.Code, "status": b.Status, "paymentStatus": b.PaymentStatus,
		"holdExpiresAt": b.HoldExpiresAt, "amountMinor": b.AmountMinor, "currency": b.Currency,
	})
}

// handleBookingCheckout opens a new payment attempt for a booking still
// holding its seats (after a failed payment, or if the app was closed).
func (s *Server) handleBookingCheckout(w http.ResponseWriter, r *http.Request) {
	b, ok := s.ownBooking(w, r)
	if !ok {
		return
	}
	if b.Status != domain.BookingAwaitingPayment || b.HoldExpiresAt == nil || !b.HoldExpiresAt.After(time.Now()) {
		writeErr(w, http.StatusConflict, "This booking is no longer waiting for payment. Book again to pick a new time.")
		return
	}
	co, err := s.startCheckout(r, b)
	if err != nil {
		writePaymentStartErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bookingWithCheckout{Booking: b, Checkout: co})
}

func writePaymentStartErr(w http.ResponseWriter, err error) {
	if errors.Is(err, errPaymentsUnavailable) {
		writeErr(w, http.StatusServiceUnavailable, "Online payment isn't available right now.")
		return
	}
	log.Printf("payments: start checkout: %v", err)
	writeErr(w, http.StatusBadGateway, "Couldn't start the payment. Please try again.")
}

// handleCancelPreview tells the traveller what they'd get back before they
// confirm a cancellation.
func (s *Server) handleCancelPreview(w http.ResponseWriter, r *http.Request) {
	b, ok := s.ownBooking(w, r)
	if !ok {
		return
	}
	now := time.Now()
	cancellable := (b.Status == domain.BookingConfirmed || b.Status == domain.BookingAwaitingPayment) && b.StartAt.After(now)
	refund := int64(0)
	if cancellable {
		refund = domain.RefundFor(b, now, false)
	}
	freeUntil := b.StartAt.Add(-time.Duration(b.FreeCancelHours) * time.Hour)
	writeJSON(w, http.StatusOK, map[string]any{
		"cancellable": cancellable, "refundMinor": refund, "currency": b.Currency,
		"paidMinor": b.AmountMinor - b.RefundedMinor, "fullRefund": refund > 0 && refund == b.AmountMinor-b.RefundedMinor,
		"freeCancelUntil": freeUntil,
	})
}

// ---- fake gateway (development only) ----

var fakePayTmpl = template.Must(template.New("fakepay").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Test payment</title>
<style>body{font-family:system-ui,sans-serif;background:#faf5ec;color:#123d3a;display:flex;min-height:100vh;align-items:center;justify-content:center;margin:0}main{max-width:360px;padding:24px;text-align:center}button{border:0;border-radius:12px;padding:14px 24px;font-weight:700;font-size:16px;margin:6px}</style>
</head><body><main><h1>Test payment</h1><p>No real money moves. Order {{.Order}}: {{.Currency}} {{.Amount}}</p>
<form method="post" action="/payments/fake/complete">
<input type="hidden" name="order_id" value="{{.Order}}"><input type="hidden" name="amount" value="{{.Amount}}"><input type="hidden" name="currency" value="{{.Currency}}"><input type="hidden" name="return_url" value="{{.Return}}"><input type="hidden" name="cancel_url" value="{{.Cancel}}">
<button name="status" value="succeeded" style="background:#f6b821">Pay</button><button name="status" value="failed">Decline</button></form>
</main></body></html>`))

// handleFakePay is the fake gateway's "hosted checkout" (development only).
func (s *Server) handleFakePay(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.Payments.(*fake.Gateway); !ok || s.IsProduction {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	r.ParseForm()
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fakePayTmpl.Execute(w, map[string]string{"Order": r.PostForm.Get("order_id"), "Amount": r.PostForm.Get("amount"),
		"Currency": r.PostForm.Get("currency"), "Return": r.PostForm.Get("return_url"), "Cancel": r.PostForm.Get("cancel_url")})
}

// handleFakeComplete plays the gateway's part: it signs and applies the
// notification, then sends the traveller back (development only).
func (s *Server) handleFakeComplete(w http.ResponseWriter, r *http.Request) {
	g, ok := s.Payments.(*fake.Gateway)
	if !ok || s.IsProduction {
		writeErr(w, http.StatusNotFound, "not found")
		return
	}
	r.ParseForm()
	status := r.PostForm.Get("status")
	fields := map[string]string{"order_id": r.PostForm.Get("order_id"), "payment_id": "fakepay_" + randomHexID(),
		"status": status, "amount": r.PostForm.Get("amount"), "currency": r.PostForm.Get("currency"), "event_id": randomHexID()}
	fields["signature"] = g.Sign(fields)
	form := url.Values{}
	for k, v := range fields {
		form.Set(k, v)
	}
	req, _ := http.NewRequestWithContext(r.Context(), http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if ev, err := g.VerifyNotify(req); err == nil {
		s.applyPaymentEvent(r.Context(), ev)
	}
	target := r.PostForm.Get("return_url")
	if status != "succeeded" {
		target = r.PostForm.Get("cancel_url")
	}
	if u, err := url.Parse(target); err != nil || (u.Host != "" && u.Host != r.Host && !strings.HasPrefix(target, s.publicBaseURL(r))) {
		target = "/payments/return"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func randomHexID() string { return strings.TrimPrefix(newID("x"), "x_") }

package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// Payment is one gateway checkout for a booking.
type Payment struct {
	ID               string            `json:"id"`
	BookingID        string            `json:"bookingId"`
	Gateway          string            `json:"gateway"`
	GatewayOrderID   string            `json:"gatewayOrderId"`
	GatewayPaymentID string            `json:"gatewayPaymentId"`
	AmountMinor      int64             `json:"amountMinor"`
	Currency         string            `json:"currency"`
	Status           string            `json:"status"`
	CheckoutURL      string            `json:"-"`
	CheckoutFields   map[string]string `json:"-"`
	CreatedAt        time.Time         `json:"createdAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
}

const paymentCols = `id, booking_id, gateway, gateway_order_id, gateway_payment_id, amount_minor, currency, status, checkout_url, checkout_fields, created_at, updated_at`

func scanPayment(row pgx.Row) (Payment, error) {
	var p Payment
	var fields []byte
	err := row.Scan(&p.ID, &p.BookingID, &p.Gateway, &p.GatewayOrderID, &p.GatewayPaymentID, &p.AmountMinor,
		&p.Currency, &p.Status, &p.CheckoutURL, &fields, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err == nil && len(fields) > 0 {
		json.Unmarshal(fields, &p.CheckoutFields)
	}
	return p, err
}

func (s *Store) CreatePayment(ctx context.Context, p Payment) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO payments (id, booking_id, gateway, gateway_order_id, amount_minor, currency, status, checkout_url, checkout_fields)
		VALUES ($1,$2,$3,$4,$5,$6,'created',$7,$8)`,
		p.ID, p.BookingID, p.Gateway, p.GatewayOrderID, p.AmountMinor, p.Currency, p.CheckoutURL, jsonb(p.CheckoutFields))
	return err
}

func (s *Store) PaymentByID(ctx context.Context, id string) (Payment, error) {
	return scanPayment(s.Pool.QueryRow(ctx, `SELECT `+paymentCols+` FROM payments WHERE id=$1`, id))
}

func (s *Store) PaymentsByBooking(ctx context.Context, bookingID string) ([]Payment, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+paymentCols+` FROM payments WHERE booking_id=$1 ORDER BY created_at DESC`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Payment{}
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PaymentEvent is a verified gateway notification to apply.
type PaymentEvent struct {
	Gateway          string
	EventKey         string
	OrderID          string
	GatewayPaymentID string
	Status           string // succeeded | pending | failed | cancelled | chargeback
	AmountMinor      int64
	Currency         string
	Payload          map[string]string
}

// PaymentOutcome says what applying a notification did.
type PaymentOutcome struct {
	Duplicate bool // already seen; nothing changed
	// Confirmed is set when this notification confirmed the booking.
	Confirmed bool
	Booking   domain.Booking
	// Refund is set when the money must go back (the booking was cancelled
	// or its seats were taken while the traveller paid).
	Refund *Refund
	// Problem explains a notification that verified but wasn't applied.
	Problem string
}

var ErrPaymentMismatch = errors.New("payment amount or currency doesn't match")

// ApplyPaymentEvent records a gateway notification once and applies it:
// a successful payment confirms its booking (re-checking capacity if the
// hold had lapsed) and credits the vendor's net to the ledger, payable once
// the booking has taken place. Everything happens in one transaction, so a
// crash can't leave a paid booking without its ledger entry.
func (s *Store) ApplyPaymentEvent(ctx context.Context, ev PaymentEvent) (PaymentOutcome, error) {
	var out PaymentOutcome
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)

	eventID := newStoreID("pev")
	tag, err := tx.Exec(ctx, `
		INSERT INTO payment_events (id, gateway, event_key, status, payload)
		VALUES ($1,$2,$3,$4,$5) ON CONFLICT (event_key) DO NOTHING`,
		eventID, ev.Gateway, ev.EventKey, ev.Status, jsonb(ev.Payload))
	if err != nil {
		return out, err
	}
	if tag.RowsAffected() == 0 {
		out.Duplicate = true
		return out, tx.Commit(ctx)
	}
	setOutcome := func(msg string) error {
		out.Problem = msg
		_, err := tx.Exec(ctx, `UPDATE payment_events SET outcome=$2 WHERE id=$1`, eventID, msg)
		return err
	}

	// Lock the booking before its payment, the same order cancellation and
	// hold expiry use, so they can't deadlock against a notification.
	var bookingID string
	err = tx.QueryRow(ctx, `SELECT booking_id FROM payments WHERE gateway=$1 AND gateway_order_id=$2`, ev.Gateway, ev.OrderID).Scan(&bookingID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := setOutcome("unknown order"); err != nil {
			return out, err
		}
		return out, tx.Commit(ctx)
	}
	if err != nil {
		return out, err
	}
	b, err := scanBooking(tx.QueryRow(ctx, `SELECT `+bookingCols+bookingFrom+`WHERE b.id=$1 FOR UPDATE OF b`, bookingID))
	if err != nil {
		return out, err
	}
	pay, err := scanPayment(tx.QueryRow(ctx, `SELECT `+paymentCols+` FROM payments WHERE gateway=$1 AND gateway_order_id=$2 FOR UPDATE`, ev.Gateway, ev.OrderID))
	if err != nil {
		return out, err
	}
	if _, err := tx.Exec(ctx, `UPDATE payment_events SET payment_id=$2 WHERE id=$1`, eventID, pay.ID); err != nil {
		return out, err
	}
	out.Booking = b

	switch ev.Status {
	case "succeeded":
		if ev.AmountMinor != pay.AmountMinor || ev.Currency != pay.Currency {
			if err := setOutcome(fmt.Sprintf("amount mismatch: notified %d %s, expected %d %s", ev.AmountMinor, ev.Currency, pay.AmountMinor, pay.Currency)); err != nil {
				return out, err
			}
			if err := tx.Commit(ctx); err != nil {
				return out, err
			}
			return out, ErrPaymentMismatch
		}
		if pay.Status == "succeeded" {
			return out, tx.Commit(ctx) // a different notification for a payment already applied
		}
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='succeeded', gateway_payment_id=$2, updated_at=now() WHERE id=$1`, pay.ID, ev.GatewayPaymentID); err != nil {
			return out, err
		}
		pay.Status, pay.GatewayPaymentID = "succeeded", ev.GatewayPaymentID

		if b.PaymentStatus == domain.PaymentPaid || b.PaymentStatus == domain.PaymentPartiallyRefunded || b.PaymentStatus == domain.PaymentRefunded {
			// The booking was already paid through another checkout: this
			// second payment goes straight back.
			r, err := insertRefund(ctx, tx, b, pay, pay.AmountMinor, "duplicate payment", "system")
			if err != nil {
				return out, err
			}
			out.Refund = &r
			if err := setOutcome("duplicate payment refunded"); err != nil {
				return out, err
			}
			return out, tx.Commit(ctx)
		}

		holdLapsed := b.Status == domain.BookingAwaitingPayment && (b.HoldExpiresAt == nil || !b.HoldExpiresAt.After(time.Now()))
		canConfirm := b.Status == domain.BookingAwaitingPayment && !holdLapsed
		if b.Status == domain.BookingExpired || holdLapsed {
			// The hold lapsed while the traveller was paying: honour the
			// payment if the slot still has room.
			ok, err := capacityStillFree(ctx, tx, b)
			if err != nil {
				return out, err
			}
			canConfirm = ok
		}
		if !canConfirm {
			// Cancelled, or its seats went to someone else: record the
			// payment and refund it in full.
			if _, err := tx.Exec(ctx, `UPDATE bookings SET payment_status='paid',
				status = CASE WHEN status='awaiting_payment' THEN 'expired' ELSE status END WHERE id=$1`, b.ID); err != nil {
				return out, err
			}
			b.PaymentStatus = domain.PaymentPaid
			r, err := insertRefund(ctx, tx, b, pay, pay.AmountMinor, "booking no longer available when payment arrived", "system")
			if err != nil {
				return out, err
			}
			out.Refund = &r
			if _, err := tx.Exec(ctx, `UPDATE bookings SET refunded_minor=amount_minor, payment_status='refunded' WHERE id=$1`, b.ID); err != nil {
				return out, err
			}
			if err := setOutcome("booking unavailable, refunded"); err != nil {
				return out, err
			}
			return out, tx.Commit(ctx)
		}

		if _, err := tx.Exec(ctx, `
			UPDATE bookings SET status='confirmed', payment_status='paid', hold_expires_at=NULL WHERE id=$1`, b.ID); err != nil {
			return out, err
		}
		bookingID := b.ID
		if _, err := insertLedgerEntry(ctx, tx, LedgerEntry{
			ID: newStoreID("led"), VendorID: b.VendorID, BookingID: &bookingID, Kind: LedgerBookingNet,
			AmountMinor: b.VendorNetMinor, Currency: b.Currency, PayableAt: b.EndAt,
			Memo: "Booking " + b.Code, CreatedBy: "system",
		}); err != nil {
			return out, err
		}
		b.Status, b.PaymentStatus, b.HoldExpiresAt = domain.BookingConfirmed, domain.PaymentPaid, nil
		out.Booking, out.Confirmed = b, true
	case "failed", "cancelled":
		if pay.Status == "created" {
			if _, err := tx.Exec(ctx, `UPDATE payments SET status=$2, updated_at=now() WHERE id=$1`, pay.ID, ev.Status); err != nil {
				return out, err
			}
		}
		// The booking keeps its hold so the traveller can try again.
	case "chargeback":
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='chargeback', updated_at=now() WHERE id=$1`, pay.ID); err != nil {
			return out, err
		}
		if err := setOutcome("chargeback: review with the vendor"); err != nil {
			return out, err
		}
	}
	return out, tx.Commit(ctx)
}

// capacityStillFree re-checks, inside tx, whether booking b (currently
// not holding capacity) still fits.
func capacityStillFree(ctx context.Context, tx pgx.Tx, b domain.Booking) (bool, error) {
	var capacity int
	var unit string
	if err := tx.QueryRow(ctx, `SELECT capacity, capacity_unit FROM bookable_services WHERE id=$1 FOR UPDATE`, b.ServiceID).Scan(&capacity, &unit); err != nil {
		return false, err
	}
	var existing int
	if b.OccurrenceID != nil {
		var occCapacity *int
		var status string
		if err := tx.QueryRow(ctx, `SELECT capacity, status FROM service_occurrences WHERE id=$1`, *b.OccurrenceID).Scan(&occCapacity, &status); err != nil {
			return false, err
		}
		if status != "scheduled" {
			return false, nil
		}
		if occCapacity != nil {
			capacity = *occCapacity
		}
		if err := tx.QueryRow(ctx, heldOccurrenceCapacitySQL, *b.OccurrenceID, unit).Scan(&existing); err != nil {
			return false, err
		}
	} else if err := tx.QueryRow(ctx, heldCapacitySQL, b.ServiceID, b.StartAt, b.EndAt, unit).Scan(&existing); err != nil {
		return false, err
	}
	need := 1
	if unit == string(domain.CapacityGuests) {
		need = b.PartySize
	}
	return existing+need <= capacity, nil
}

// ExpireBookingHolds releases the seats of paid bookings whose payment
// didn't arrive in time, returning how many expired.
func (s *Store) ExpireBookingHolds(ctx context.Context) (int, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		UPDATE bookings SET status='expired'
		WHERE status='awaiting_payment' AND hold_expires_at < now()
		RETURNING id`)
	if err != nil {
		return 0, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) > 0 {
		if _, err := tx.Exec(ctx, `UPDATE payments SET status='cancelled', updated_at=now() WHERE booking_id = ANY($1) AND status='created'`, ids); err != nil {
			return 0, err
		}
	}
	return len(ids), tx.Commit(ctx)
}

// ---- refunds ----

// Refund is money going back to a traveller.
type Refund struct {
	ID              string    `json:"id"`
	BookingID       string    `json:"bookingId"`
	PaymentID       *string   `json:"paymentId,omitempty"`
	AmountMinor     int64     `json:"amountMinor"`
	Currency        string    `json:"currency"`
	Reason          string    `json:"reason"`
	Status          string    `json:"status"`
	GatewayRefundID string    `json:"gatewayRefundId"`
	RequestedBy     string    `json:"requestedBy"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	// Joined for display and for calling the gateway.
	BookingCode      string `json:"bookingCode,omitempty"`
	VendorName       string `json:"vendorName,omitempty"`
	TravellerName    string `json:"travellerName,omitempty"`
	TravellerEmail   string `json:"travellerEmail,omitempty"`
	Gateway          string `json:"gateway,omitempty"`
	GatewayOrderID   string `json:"-"`
	GatewayPaymentID string `json:"-"`
}

func insertRefund(ctx context.Context, tx pgx.Tx, b domain.Booking, pay Payment, amount int64, reason, by string) (Refund, error) {
	r := Refund{ID: newStoreID("rfd"), BookingID: b.ID, AmountMinor: amount, Currency: b.Currency, Reason: reason,
		Status: "requested", RequestedBy: by, CreatedAt: time.Now(), BookingCode: b.Code,
		Gateway: pay.Gateway, GatewayOrderID: pay.GatewayOrderID, GatewayPaymentID: pay.GatewayPaymentID}
	var payID *string
	if pay.ID != "" {
		payID = &pay.ID
	}
	r.PaymentID = payID
	_, err := tx.Exec(ctx, `
		INSERT INTO refunds (id, booking_id, payment_id, amount_minor, currency, reason, status, requested_by)
		VALUES ($1,$2,$3,$4,$5,$6,'requested',$7)`, r.ID, r.BookingID, payID, amount, r.Currency, reason, by)
	return r, err
}

// StaleRequestedRefunds returns refunds still "requested" after olderThan:
// the process stopped between recording them and asking the gateway.
func (s *Store) StaleRequestedRefunds(ctx context.Context, olderThan time.Duration) ([]Refund, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+refundCols+refundFrom+`
		WHERE r.status='requested' AND r.created_at < now() - $1::interval ORDER BY r.created_at LIMIT 100`,
		olderThan.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Refund{}
	for rows.Next() {
		r, err := scanRefund(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// LiveHoldCount is how many unpaid bookings a traveller is holding seats
// for right now.
func (s *Store) LiveHoldCount(ctx context.Context, travellerID string) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM bookings WHERE traveller_id=$1 AND status='awaiting_payment' AND hold_expires_at > now()`, travellerID).Scan(&n)
	return n, err
}

// RefundsForBooking lists one booking's refunds.
func (s *Store) RefundsForBooking(ctx context.Context, bookingID string) ([]Refund, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+refundCols+refundFrom+`WHERE r.booking_id=$1 ORDER BY r.created_at`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Refund{}
	for rows.Next() {
		r, err := scanRefund(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// SetRefundStatus records what the gateway (or the finance team) did with
// a refund.
func (s *Store) SetRefundStatus(ctx context.Context, id, status, gatewayRefundID string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE refunds SET status=$2, gateway_refund_id=CASE WHEN $3='' THEN gateway_refund_id ELSE $3 END, updated_at=now()
		WHERE id=$1`, id, status, gatewayRefundID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

const refundCols = `r.id, r.booking_id, r.payment_id, r.amount_minor, r.currency, r.reason, r.status, r.gateway_refund_id,
	r.requested_by, r.created_at, r.updated_at, COALESCE(b.code,''), COALESCE(v.name,''), COALESCE(u.name,''), COALESCE(u.email,''),
	COALESCE(p.gateway,''), COALESCE(p.gateway_order_id,''), COALESCE(p.gateway_payment_id,'')`
const refundFrom = ` FROM refunds r
	JOIN bookings b ON b.id = r.booking_id
	LEFT JOIN vendors v ON v.id = b.vendor_id
	LEFT JOIN users u ON u.id = b.traveller_id
	LEFT JOIN payments p ON p.id = r.payment_id `

func scanRefund(row pgx.Row) (Refund, error) {
	var r Refund
	err := row.Scan(&r.ID, &r.BookingID, &r.PaymentID, &r.AmountMinor, &r.Currency, &r.Reason, &r.Status, &r.GatewayRefundID,
		&r.RequestedBy, &r.CreatedAt, &r.UpdatedAt, &r.BookingCode, &r.VendorName, &r.TravellerName, &r.TravellerEmail,
		&r.Gateway, &r.GatewayOrderID, &r.GatewayPaymentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

func (s *Store) RefundByID(ctx context.Context, id string) (Refund, error) {
	return scanRefund(s.Pool.QueryRow(ctx, `SELECT `+refundCols+refundFrom+`WHERE r.id=$1`, id))
}

// Refunds lists refunds, newest first; status "" means any.
func (s *Store) Refunds(ctx context.Context, status string, limit int) ([]Refund, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+refundCols+refundFrom+`WHERE ($1='' OR r.status=$1) ORDER BY r.created_at DESC LIMIT $2`, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Refund{}
	for rows.Next() {
		r, err := scanRefund(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ---- cancellation ----

// CancelRequest describes who is cancelling which booking.
type CancelRequest struct {
	BookingID string
	// Exactly one of TravellerID / VendorID scopes the booking to its
	// owner; both empty means an admin cancelling any booking.
	TravellerID string
	VendorID    string
	By          string // traveller | vendor | admin
	Actor       string // who, for the refund record
	// FullRefund refunds everything regardless of the cancellation policy
	// (vendor and admin cancellations).
	FullRefund bool
	Now        time.Time
}

var ErrBookingStarted = errors.New("booking has already started")

// CancelBookingWithRefund cancels a confirmed or held booking and, if it
// was paid, works out the refund under the booking's own terms, records
// it, and takes the vendor's share of it back through the ledger, all in
// one transaction. The caller then asks the gateway to pay the refund.
func (s *Store) CancelBookingWithRefund(ctx context.Context, req CancelRequest) (domain.Booking, *Refund, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return domain.Booking{}, nil, err
	}
	defer tx.Rollback(ctx)
	b, err := scanBooking(tx.QueryRow(ctx, `SELECT `+bookingCols+bookingFrom+`WHERE b.id=$1 FOR UPDATE OF b`, req.BookingID))
	if err != nil {
		return b, nil, err
	}
	if (req.TravellerID != "" && b.TravellerID != req.TravellerID) || (req.VendorID != "" && b.VendorID != req.VendorID) {
		return b, nil, ErrNotFound
	}
	if b.Status != domain.BookingConfirmed && b.Status != domain.BookingAwaitingPayment {
		return b, nil, ErrInvalidTransition
	}
	if req.By == "traveller" && !b.StartAt.After(req.Now) {
		return b, nil, ErrBookingStarted
	}

	paymentStatus := b.PaymentStatus
	if b.PaymentStatus == domain.PaymentPending {
		paymentStatus = domain.PaymentFailed // never paid
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bookings SET status='cancelled', cancelled_at=$2, cancelled_by=$3, hold_expires_at=NULL, payment_status=$4
		WHERE id=$1`, b.ID, req.Now, req.By, paymentStatus); err != nil {
		return b, nil, err
	}
	if _, err := tx.Exec(ctx, `UPDATE payments SET status='cancelled', updated_at=now() WHERE booking_id=$1 AND status='created'`, b.ID); err != nil {
		return b, nil, err
	}
	b.Status, b.PaymentStatus, b.CancelledBy, b.HoldExpiresAt = domain.BookingCancelled, paymentStatus, req.By, nil
	b.CancelledAt = &req.Now

	// A cancelled booking won't take place: whatever the vendor keeps of it
	// is settled now, alongside the refund reversal, not on the old date.
	if _, err := tx.Exec(ctx, `UPDATE ledger_entries SET payable_at=LEAST(payable_at, $2)
		WHERE booking_id=$1 AND kind='booking_net' AND payout_id IS NULL`, b.ID, req.Now); err != nil {
		return b, nil, err
	}
	amount := domain.RefundFor(b, req.Now, req.FullRefund)
	if amount <= 0 {
		return b, nil, tx.Commit(ctx)
	}
	pay, err := scanPayment(tx.QueryRow(ctx, `SELECT `+paymentCols+` FROM payments WHERE booking_id=$1 AND status='succeeded' ORDER BY created_at LIMIT 1`, b.ID))
	if err != nil && !errors.Is(err, ErrNotFound) {
		return b, nil, err
	}
	refund, err := insertRefund(ctx, tx, b, pay, amount, "cancelled by "+req.By, req.Actor)
	if err != nil {
		return b, nil, err
	}
	reversal := domain.RefundReversal(b, amount)
	bookingID := b.ID
	if reversal > 0 {
		if _, err := insertLedgerEntry(ctx, tx, LedgerEntry{
			ID: newStoreID("led"), VendorID: b.VendorID, BookingID: &bookingID, Kind: LedgerRefundReversal,
			AmountMinor: -reversal, Currency: b.Currency, PayableAt: req.Now,
			Memo: "Refund on " + b.Code, CreatedBy: req.Actor,
		}); err != nil {
			return b, nil, err
		}
	}
	b.RefundedMinor += amount
	b.PaymentStatus = domain.PaymentRefunded
	if b.RefundedMinor < b.AmountMinor {
		b.PaymentStatus = domain.PaymentPartiallyRefunded
	}
	// Spotto gives back its commission on the refunded part too.
	b.CommissionMinor -= amount - reversal
	b.VendorNetMinor -= reversal
	if _, err := tx.Exec(ctx, `
		UPDATE bookings SET refunded_minor=$2, payment_status=$3, commission_minor=$4, vendor_net_minor=$5 WHERE id=$1`,
		b.ID, b.RefundedMinor, b.PaymentStatus, b.CommissionMinor, b.VendorNetMinor); err != nil {
		return b, nil, err
	}
	return b, &refund, tx.Commit(ctx)
}

// ---- check-in and per-guest fees ----

// chargePerGuestFee writes the vendor's per-guest fee for a free booking
// that has just been checked in or marked completed. A second call for the
// same booking does nothing.
func chargePerGuestFee(ctx context.Context, tx pgx.Tx, b domain.Booking, actor string) error {
	fee := domain.PerGuestFee(b)
	if fee <= 0 {
		return nil
	}
	bookingID := b.ID
	inserted, err := insertLedgerEntry(ctx, tx, LedgerEntry{
		ID: newStoreID("led"), VendorID: b.VendorID, BookingID: &bookingID, Kind: LedgerPerGuestFee,
		AmountMinor: -fee, Currency: b.Currency, PayableAt: time.Now(),
		Memo: fmt.Sprintf("%d guest(s) seated, booking %s", b.PartySize, b.Code), CreatedBy: actor,
	})
	if err != nil || !inserted {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE bookings SET commission_minor=$2 WHERE id=$1`, b.ID, fee)
	return err
}

// newStoreID is a short random id with a prefix, for rows the store
// creates itself.
func newStoreID(prefix string) string {
	return prefix + "_" + randomHex(6)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

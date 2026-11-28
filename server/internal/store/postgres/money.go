package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// ---- commission rules ----

const commissionRuleCols = `id, scope, scope_key, mode, rate_bps, fee_per_guest_minor, active_from, note, created_by, created_at`

func scanCommissionRule(row pgx.Row) (domain.CommissionRule, error) {
	var r domain.CommissionRule
	err := row.Scan(&r.ID, &r.Scope, &r.ScopeKey, &r.Mode, &r.RateBps, &r.FeePerGuestMinor,
		&r.ActiveFrom, &r.Note, &r.CreatedBy, &r.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrNotFound
	}
	return r, err
}

// ResolveCommissionRule finds the rule of the given mode that applies to a
// booking at `at`: the most specific scope with a rule in effect (vendor,
// then category, then booking type, then default), and within a scope the
// most recent active_from. Returns nil when no rule applies.
func (s *Store) ResolveCommissionRule(ctx context.Context, mode domain.CommissionMode, vendorID, category string, bookingType domain.BookingType, at time.Time) (*domain.CommissionRule, error) {
	r, err := scanCommissionRule(s.Pool.QueryRow(ctx, `
		SELECT `+commissionRuleCols+` FROM commission_rules
		WHERE mode=$1 AND active_from <= $5 AND (
			(scope='vendor' AND scope_key=$2) OR
			(scope='category' AND scope_key=$3) OR
			(scope='booking_type' AND scope_key=$4) OR
			scope='default')
		ORDER BY CASE scope WHEN 'vendor' THEN 0 WHEN 'category' THEN 1 WHEN 'booking_type' THEN 2 ELSE 3 END,
			active_from DESC
		LIMIT 1`, mode, vendorID, category, string(bookingType), at))
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// CommissionRules lists every rule, newest first within each scope.
func (s *Store) CommissionRules(ctx context.Context) ([]domain.CommissionRule, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+commissionRuleCols+` FROM commission_rules
		ORDER BY CASE scope WHEN 'default' THEN 0 WHEN 'booking_type' THEN 1 WHEN 'category' THEN 2 ELSE 3 END,
			scope_key, mode, active_from DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.CommissionRule{}
	for rows.Next() {
		r, err := scanCommissionRule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateCommissionRule adds a rule. Rules are never edited in place: a
// new rate is a new rule with a later active_from, so bookings keep
// pointing at the rate that applied to them.
func (s *Store) CreateCommissionRule(ctx context.Context, r domain.CommissionRule) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO commission_rules (id, scope, scope_key, mode, rate_bps, fee_per_guest_minor, active_from, note, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		r.ID, r.Scope, r.ScopeKey, r.Mode, r.RateBps, r.FeePerGuestMinor, r.ActiveFrom, r.Note, r.CreatedBy)
	return err
}

// DeleteCommissionRule removes a rule that hasn't taken effect yet. A rule
// already in effect is history (bookings were priced with it) and is
// superseded by adding a newer rule instead.
func (s *Store) DeleteCommissionRule(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM commission_rules WHERE id=$1 AND active_from > now()`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		var exists bool
		if err := s.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM commission_rules WHERE id=$1)`, id).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrRuleInEffect
		}
		return ErrNotFound
	}
	return nil
}

// ErrRuleInEffect is returned when deleting a rule that already applies.
var ErrRuleInEffect = errors.New("commission rule already in effect")

// ---- ledger ----

// LedgerEntry is one line of what Spotto and a vendor owe each other.
// A positive AmountMinor is owed to the vendor, a negative one by the
// vendor. An entry with no PayoutID is unsettled.
type LedgerEntry struct {
	ID          string    `json:"id"`
	VendorID    string    `json:"vendorId"`
	BookingID   *string   `json:"bookingId,omitempty"`
	PayoutID    *string   `json:"payoutId,omitempty"`
	Kind        string    `json:"kind"`
	AmountMinor int64     `json:"amountMinor"`
	Currency    string    `json:"currency"`
	PayableAt   time.Time `json:"payableAt"`
	Memo        string    `json:"memo"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	// Joined for display.
	BookingCode string `json:"bookingCode,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	VendorName  string `json:"vendorName,omitempty"`
}

const (
	LedgerBookingNet     = "booking_net"
	LedgerPerGuestFee    = "per_guest_fee"
	LedgerRefundReversal = "refund_reversal"
	LedgerAdjustment     = "adjustment"
)

// insertLedgerEntry writes e inside tx. A booking-driven entry that already
// exists for the same booking and kind is left alone (inserted=false), so
// retried webhooks and repeat scans can't count twice.
func insertLedgerEntry(ctx context.Context, tx pgx.Tx, e LedgerEntry) (inserted bool, err error) {
	if e.Currency == "" {
		e.Currency = domain.CurrencyLKR
	}
	if e.PayableAt.IsZero() {
		e.PayableAt = time.Now()
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO ledger_entries (id, vendor_id, booking_id, kind, amount_minor, currency, payable_at, memo, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (booking_id, kind) WHERE booking_id IS NOT NULL DO NOTHING`,
		e.ID, e.VendorID, e.BookingID, e.Kind, e.AmountMinor, e.Currency, e.PayableAt, e.Memo, e.CreatedBy)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// AddLedgerAdjustment records a manual correction by the finance team.
func (s *Store) AddLedgerAdjustment(ctx context.Context, e LedgerEntry) error {
	e.Kind, e.BookingID = LedgerAdjustment, nil
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := insertLedgerEntry(ctx, tx, e); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const ledgerCols = `l.id, l.vendor_id, l.booking_id, l.payout_id, l.kind, l.amount_minor, l.currency, l.payable_at,
	l.memo, l.created_by, l.created_at, COALESCE(b.code,''), COALESCE(sv.name,''), COALESCE(v.name,'')`
const ledgerFrom = ` FROM ledger_entries l
	LEFT JOIN bookings b ON b.id = l.booking_id
	LEFT JOIN bookable_services sv ON sv.id = b.service_id
	LEFT JOIN vendors v ON v.id = l.vendor_id `

func scanLedgerEntry(row pgx.Row) (LedgerEntry, error) {
	var e LedgerEntry
	err := row.Scan(&e.ID, &e.VendorID, &e.BookingID, &e.PayoutID, &e.Kind, &e.AmountMinor, &e.Currency,
		&e.PayableAt, &e.Memo, &e.CreatedBy, &e.CreatedAt, &e.BookingCode, &e.ServiceName, &e.VendorName)
	return e, err
}

// LedgerFilter narrows a ledger listing. Zero values mean "any".
type LedgerFilter struct {
	VendorID  string
	BookingID string
	From, To  time.Time
	Limit     int
}

// LedgerEntries lists entries newest first.
func (s *Store) LedgerEntries(ctx context.Context, f LedgerFilter) ([]LedgerEntry, error) {
	q := `SELECT ` + ledgerCols + ledgerFrom + `WHERE ($1='' OR l.vendor_id=$1)
		AND ($2::timestamptz IS NULL OR l.created_at >= $2) AND ($3::timestamptz IS NULL OR l.created_at < $3)
		AND ($5='' OR l.booking_id=$5)
		ORDER BY l.created_at DESC LIMIT $4`
	limit := f.Limit
	if limit <= 0 || limit > 5000 {
		limit = 500
	}
	rows, err := s.Pool.Query(ctx, q, f.VendorID, nullTime(f.From), nullTime(f.To), limit, f.BookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []LedgerEntry{}
	for rows.Next() {
		e, err := scanLedgerEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

// VendorBalance sums a vendor's unsettled ledger entries. Payable is what
// has become payable by now (the bookings have taken place); Pending is
// money for bookings still to come. A negative Payable means the vendor
// owes Spotto (per-guest fees or refunds larger than what's due to them).
type VendorBalance struct {
	PayableMinor int64  `json:"payableMinor"`
	PendingMinor int64  `json:"pendingMinor"`
	Currency     string `json:"currency"`
}

func (s *Store) VendorBalance(ctx context.Context, vendorID string) (VendorBalance, error) {
	b := VendorBalance{Currency: domain.CurrencyLKR}
	err := s.Pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount_minor) FILTER (WHERE payable_at <= now()), 0),
		       COALESCE(sum(amount_minor) FILTER (WHERE payable_at > now()), 0)
		FROM ledger_entries WHERE vendor_id=$1 AND payout_id IS NULL`, vendorID).
		Scan(&b.PayableMinor, &b.PendingMinor)
	return b, err
}

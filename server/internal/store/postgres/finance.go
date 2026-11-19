package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// ---- payout accounts ----

// PayoutAccount is where a vendor's payouts go. AccountNumberEnc is never
// sent to clients; AccountNumber is only filled for the finance team.
type PayoutAccount struct {
	VendorID         string     `json:"vendorId"`
	AccountHolder    string     `json:"accountHolder"`
	BankName         string     `json:"bankName"`
	Branch           string     `json:"branch"`
	AccountLast4     string     `json:"accountLast4"`
	AccountNumber    string     `json:"accountNumber,omitempty"`
	AccountNumberEnc string     `json:"-"`
	VerifiedAt       *time.Time `json:"verifiedAt,omitempty"`
	VerifiedBy       string     `json:"verifiedBy,omitempty"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (s *Store) PayoutAccount(ctx context.Context, vendorID string) (PayoutAccount, error) {
	var a PayoutAccount
	err := s.Pool.QueryRow(ctx, `
		SELECT vendor_id, account_holder, bank_name, branch, account_last4, account_number_enc, verified_at, verified_by, updated_at
		FROM vendor_payout_accounts WHERE vendor_id=$1`, vendorID).
		Scan(&a.VendorID, &a.AccountHolder, &a.BankName, &a.Branch, &a.AccountLast4, &a.AccountNumberEnc, &a.VerifiedAt, &a.VerifiedBy, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}

// UpsertPayoutAccount saves bank details. Any change clears verification:
// the finance team re-checks new details before money goes to them.
func (s *Store) UpsertPayoutAccount(ctx context.Context, a PayoutAccount) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendor_payout_accounts (vendor_id, account_holder, bank_name, branch, account_number_enc, account_last4, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
		ON CONFLICT (vendor_id) DO UPDATE SET
			account_holder=EXCLUDED.account_holder, bank_name=EXCLUDED.bank_name, branch=EXCLUDED.branch,
			account_number_enc=EXCLUDED.account_number_enc, account_last4=EXCLUDED.account_last4,
			verified_at=NULL, verified_by='', updated_at=now()`,
		a.VendorID, a.AccountHolder, a.BankName, a.Branch, a.AccountNumberEnc, a.AccountLast4)
	return err
}

func (s *Store) VerifyPayoutAccount(ctx context.Context, vendorID, by string) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE vendor_payout_accounts SET verified_at=now(), verified_by=$2 WHERE vendor_id=$1`, vendorID, by)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ---- payouts ----

type Payout struct {
	ID            string     `json:"id"`
	VendorID      string     `json:"vendorId"`
	VendorName    string     `json:"vendorName"`
	AmountMinor   int64      `json:"amountMinor"`
	Currency      string     `json:"currency"`
	PeriodEnd     time.Time  `json:"periodEnd"`
	Status        string     `json:"status"`
	ApprovedBy    string     `json:"approvedBy"`
	ApprovedAt    *time.Time `json:"approvedAt,omitempty"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	BankReference string     `json:"bankReference"`
	Note          string     `json:"note"`
	EntryCount    int        `json:"entryCount"`
	CreatedAt     time.Time  `json:"createdAt"`
	// Joined for the bank file.
	AccountHolder string `json:"accountHolder,omitempty"`
	BankName      string `json:"bankName,omitempty"`
	Branch        string `json:"branch,omitempty"`
	AccountLast4  string `json:"accountLast4,omitempty"`
	AccountEnc    string `json:"-"`
}

const payoutCols = `p.id, p.vendor_id, COALESCE(v.name,''), p.amount_minor, p.currency, p.period_end, p.status, p.approved_by,
	p.approved_at, p.paid_at, p.bank_reference, p.note, p.created_at,
	(SELECT count(*) FROM ledger_entries l WHERE l.payout_id = p.id),
	COALESCE(a.account_holder,''), COALESCE(a.bank_name,''), COALESCE(a.branch,''), COALESCE(a.account_last4,''), COALESCE(a.account_number_enc,'')`
const payoutFrom = ` FROM payouts p
	LEFT JOIN vendors v ON v.id = p.vendor_id
	LEFT JOIN vendor_payout_accounts a ON a.vendor_id = p.vendor_id `

func scanPayout(row pgx.Row) (Payout, error) {
	var p Payout
	err := row.Scan(&p.ID, &p.VendorID, &p.VendorName, &p.AmountMinor, &p.Currency, &p.PeriodEnd, &p.Status, &p.ApprovedBy,
		&p.ApprovedAt, &p.PaidAt, &p.BankReference, &p.Note, &p.CreatedAt, &p.EntryCount,
		&p.AccountHolder, &p.BankName, &p.Branch, &p.AccountLast4, &p.AccountEnc)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Store) PayoutByID(ctx context.Context, id string) (Payout, error) {
	return scanPayout(s.Pool.QueryRow(ctx, `SELECT `+payoutCols+payoutFrom+`WHERE p.id=$1`, id))
}

// Payouts lists payouts newest first; empty vendorID / status mean any.
func (s *Store) Payouts(ctx context.Context, vendorID, status string, limit int) ([]Payout, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+payoutCols+payoutFrom+`
		WHERE ($1='' OR p.vendor_id=$1) AND ($2='' OR p.status=$2) ORDER BY p.created_at DESC LIMIT $3`, vendorID, status, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Payout{}
	for rows.Next() {
		p, err := scanPayout(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PayoutRunResult says what a payout run did.
type PayoutRunResult struct {
	Created []Payout `json:"created"`
	// NoAccount lists vendors with money due but no verified bank details.
	NoAccount []string `json:"noAccount"`
	// BelowMinimum lists vendors whose payable balance (possibly negative:
	// fees owed) didn't reach the minimum payout.
	BelowMinimum []string `json:"belowMinimum"`
}

// RunPayouts creates a draft payout for every vendor whose unsettled,
// payable ledger entries (bookings that have taken place, fees, refunds,
// adjustments) add up to at least minMinor, attaching those entries to it.
// Fees a vendor owes are netted against what Spotto owes them; a vendor
// whose total is below the minimum or negative is left for a later run.
// Each vendor is handled in its own transaction with its entries locked,
// so two concurrent runs can't pay the same entry twice.
func (s *Store) RunPayouts(ctx context.Context, asOf time.Time, minMinor int64) (PayoutRunResult, error) {
	res := PayoutRunResult{Created: []Payout{}, NoAccount: []string{}, BelowMinimum: []string{}}
	rows, err := s.Pool.Query(ctx, `
		SELECT DISTINCT vendor_id FROM ledger_entries WHERE payout_id IS NULL AND payable_at <= $1`, asOf)
	if err != nil {
		return res, err
	}
	var vendors []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return res, err
		}
		vendors = append(vendors, id)
	}
	rows.Close()

	for _, vendorID := range vendors {
		p, outcome, err := s.runVendorPayout(ctx, vendorID, asOf, minMinor)
		if err != nil {
			return res, err
		}
		switch outcome {
		case "created":
			res.Created = append(res.Created, p)
		case "no_account":
			res.NoAccount = append(res.NoAccount, vendorID)
		case "below_minimum":
			res.BelowMinimum = append(res.BelowMinimum, vendorID)
		}
	}
	return res, nil
}

func (s *Store) runVendorPayout(ctx context.Context, vendorID string, asOf time.Time, minMinor int64) (Payout, string, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return Payout{}, "", err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		SELECT id, amount_minor FROM ledger_entries
		WHERE vendor_id=$1 AND payout_id IS NULL AND payable_at <= $2
		FOR UPDATE SKIP LOCKED`, vendorID, asOf)
	if err != nil {
		return Payout{}, "", err
	}
	var ids []string
	var total int64
	for rows.Next() {
		var id string
		var amount int64
		if err := rows.Scan(&id, &amount); err != nil {
			rows.Close()
			return Payout{}, "", err
		}
		ids, total = append(ids, id), total+amount
	}
	rows.Close()
	if len(ids) == 0 {
		return Payout{}, "", nil
	}
	if total < minMinor || total <= 0 {
		return Payout{}, "below_minimum", nil
	}
	var verified bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM vendor_payout_accounts WHERE vendor_id=$1 AND verified_at IS NOT NULL)`, vendorID).Scan(&verified); err != nil {
		return Payout{}, "", err
	}
	if !verified {
		return Payout{}, "no_account", nil
	}
	id := newStoreID("po")
	if _, err := tx.Exec(ctx, `
		INSERT INTO payouts (id, vendor_id, amount_minor, currency, period_end, status)
		VALUES ($1,$2,$3,$4,$5,'draft')`, id, vendorID, total, domain.CurrencyLKR, asOf); err != nil {
		return Payout{}, "", err
	}
	if _, err := tx.Exec(ctx, `UPDATE ledger_entries SET payout_id=$1 WHERE id = ANY($2)`, id, ids); err != nil {
		return Payout{}, "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return Payout{}, "", err
	}
	p, err := s.PayoutByID(ctx, id)
	return p, "created", err
}

// ErrPayoutState is returned when a payout isn't in a state that allows
// the requested step.
var ErrPayoutState = errors.New("payout can't do that in its current state")

// ApprovePayout moves a draft payout to approved (ready to transfer).
func (s *Store) ApprovePayout(ctx context.Context, id, by string) error {
	return s.payoutStep(ctx, `UPDATE payouts SET status='approved', approved_by=$2, approved_at=now(), updated_at=now()
		WHERE id=$1 AND status='draft'`, id, by)
}

// MarkPayoutPaid records the bank transfer of an approved payout.
func (s *Store) MarkPayoutPaid(ctx context.Context, id, bankReference string) error {
	return s.payoutStep(ctx, `UPDATE payouts SET status='paid', paid_at=now(), bank_reference=$2, updated_at=now()
		WHERE id=$1 AND status='approved'`, id, bankReference)
}

// FailPayout marks a draft or approved payout failed (or cancelled) and
// releases its ledger entries for the next run.
func (s *Store) FailPayout(ctx context.Context, id, status, note string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE payouts SET status=$2, note=$3, updated_at=now() WHERE id=$1 AND status IN ('draft','approved')`, id, status, note)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.payoutMissingOrState(ctx, id)
	}
	if _, err := tx.Exec(ctx, `UPDATE ledger_entries SET payout_id=NULL WHERE payout_id=$1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) payoutStep(ctx context.Context, sql, id, arg string) error {
	tag, err := s.Pool.Exec(ctx, sql, id, arg)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return s.payoutMissingOrState(ctx, id)
	}
	return nil
}

func (s *Store) payoutMissingOrState(ctx context.Context, id string) error {
	if _, err := s.PayoutByID(ctx, id); err != nil {
		return err
	}
	return ErrPayoutState
}

// PayoutEntries lists the ledger entries a payout settles.
func (s *Store) PayoutEntries(ctx context.Context, payoutID string) ([]LedgerEntry, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+ledgerCols+ledgerFrom+`WHERE l.payout_id=$1 ORDER BY l.created_at`, payoutID)
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

// ---- booking queries for the panels ----

// BookingFilter narrows booking listings; zero values mean "any".
type BookingFilter struct {
	VendorID string
	Status   string
	From, To time.Time // on start_at
	Query    string    // booking code, traveller name or email
	Limit    int
}

func (s *Store) Bookings(ctx context.Context, f BookingFilter) ([]domain.Booking, error) {
	limit := f.Limit
	if limit <= 0 || limit > 2000 {
		limit = 500
	}
	return s.bookingList(ctx, `SELECT `+bookingCols+bookingFrom+`
		WHERE ($1='' OR b.vendor_id=$1) AND ($2='' OR b.status=$2)
		  AND ($3::timestamptz IS NULL OR b.end_at > $3) AND ($4::timestamptz IS NULL OR b.start_at < $4)
		  AND ($5='' OR upper(b.code)=upper($5) OR u.name ILIKE '%'||$5||'%' OR u.email ILIKE '%'||$5||'%')
		ORDER BY b.start_at DESC LIMIT $6`,
		f.VendorID, f.Status, nullTime(f.From), nullTime(f.To), f.Query, limit)
}

// ---- finance summary ----

// FinanceSummary is the money picture for a period.
type FinanceSummary struct {
	From               time.Time       `json:"from"`
	To                 time.Time       `json:"to"`
	Bookings           int             `json:"bookings"`
	PaidBookings       int             `json:"paidBookings"`
	GrossMinor         int64           `json:"grossMinor"`      // what travellers paid
	RefundedMinor      int64           `json:"refundedMinor"`   // given back
	CommissionMinor    int64           `json:"commissionMinor"` // Spotto's share of paid bookings, after refunds
	GuestFeesMinor     int64           `json:"guestFeesMinor"`  // per-guest fees charged on free tables
	VendorNetMinor     int64           `json:"vendorNetMinor"`  // owed to vendors from paid bookings
	OpenRefunds        int             `json:"openRefunds"`     // waiting on the gateway or the finance team
	PendingPayouts     int             `json:"pendingPayouts"`  // draft or approved
	PendingPayoutMinor int64           `json:"pendingPayoutMinor"`
	Currency           string          `json:"currency"`
	ByCategory         []CategoryMoney `json:"byCategory"`
}

type CategoryMoney struct {
	Category        string `json:"category"`
	Bookings        int    `json:"bookings"`
	GrossMinor      int64  `json:"grossMinor"`
	CommissionMinor int64  `json:"commissionMinor"`
}

func (s *Store) FinanceSummary(ctx context.Context, from, to time.Time) (FinanceSummary, error) {
	f := FinanceSummary{From: from, To: to, Currency: domain.CurrencyLKR, ByCategory: []CategoryMoney{}}
	err := s.Pool.QueryRow(ctx, `
		SELECT count(*),
		       count(*) FILTER (WHERE payment_status IN ('paid','partially_refunded','refunded')),
		       COALESCE(sum(amount_minor) FILTER (WHERE payment_status IN ('paid','partially_refunded','refunded')), 0),
		       COALESCE(sum(refunded_minor), 0),
		       COALESCE(sum(commission_minor) FILTER (WHERE commission_mode='percent' AND payment_status IN ('paid','partially_refunded','refunded')), 0),
		       COALESCE(sum(commission_minor) FILTER (WHERE commission_mode='per_guest_fee'), 0),
		       COALESCE(sum(vendor_net_minor) FILTER (WHERE payment_status IN ('paid','partially_refunded')), 0)
		FROM bookings WHERE created_at >= $1 AND created_at < $2`, from, to).
		Scan(&f.Bookings, &f.PaidBookings, &f.GrossMinor, &f.RefundedMinor, &f.CommissionMinor, &f.GuestFeesMinor, &f.VendorNetMinor)
	if err != nil {
		return f, err
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM refunds WHERE status IN ('requested','manual','failed')`).Scan(&f.OpenRefunds); err != nil {
		return f, err
	}
	if err := s.Pool.QueryRow(ctx, `SELECT count(*), COALESCE(sum(amount_minor),0) FROM payouts WHERE status IN ('draft','approved')`).
		Scan(&f.PendingPayouts, &f.PendingPayoutMinor); err != nil {
		return f, err
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT v.category, count(*),
		       COALESCE(sum(b.amount_minor) FILTER (WHERE b.payment_status IN ('paid','partially_refunded','refunded')), 0),
		       COALESCE(sum(b.commission_minor) FILTER (WHERE b.payment_status IN ('paid','partially_refunded','refunded') OR b.commission_mode='per_guest_fee'), 0)
		FROM bookings b JOIN vendors v ON v.id = b.vendor_id
		WHERE b.created_at >= $1 AND b.created_at < $2
		GROUP BY v.category ORDER BY 4 DESC, 2 DESC`, from, to)
	if err != nil {
		return f, err
	}
	defer rows.Close()
	for rows.Next() {
		var c CategoryMoney
		if err := rows.Scan(&c.Category, &c.Bookings, &c.GrossMinor, &c.CommissionMinor); err != nil {
			return f, err
		}
		f.ByCategory = append(f.ByCategory, c)
	}
	return f, rows.Err()
}

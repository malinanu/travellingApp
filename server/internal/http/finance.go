package httpapi

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// minPayoutMinor is the smallest balance paid out (LKR 1,000): smaller
// amounts roll over to the next run rather than costing a bank transfer.
const minPayoutMinor = 100000

// Admin permissions for bookings and money (see MasterCommandCenterFeatures).
const (
	PermBookingsView     = "bookings.view"
	PermBookingsManage   = "bookings.manage"
	PermFinanceView      = "finance.view"
	PermFinancePayouts   = "finance.payouts"
	PermCommissionManage = "finance.commission"
)

// parseRange reads ?from=&to= (YYYY-MM-DD, Sri Lankan days; `to` is
// inclusive). Missing values fall back to the given defaults.
func parseRange(r *http.Request, defFrom, defTo time.Time) (time.Time, time.Time) {
	from, to := defFrom, defTo
	if d, err := parseLocalDate(r.URL.Query().Get("from")); err == nil {
		from = d
	}
	if d, err := parseLocalDate(r.URL.Query().Get("to")); err == nil {
		to = d.AddDate(0, 0, 1)
	}
	return from, to
}

func startOfLocalDay(t time.Time) time.Time {
	d, _ := parseLocalDate(inSriLanka(t).Format("2006-01-02"))
	return d
}

// nextPayoutRun is the next Monday 03:00 in Sri Lanka.
func nextPayoutRun(now time.Time) time.Time {
	local := inSriLanka(now)
	day := startOfLocalDay(now)
	days := (int(time.Monday) - int(local.Weekday()) + 7) % 7
	next := day.AddDate(0, 0, days).Add(3 * time.Hour)
	if !next.After(now) {
		next = next.AddDate(0, 0, 7)
	}
	return next
}

// StartPayoutScheduler drafts payouts every Monday at 03:00 Sri Lankan
// time. Drafts still need a finance admin to approve and mark them paid.
func (s *Server) StartPayoutScheduler(ctx context.Context) {
	for {
		wait := time.Until(nextPayoutRun(time.Now()))
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
			res, err := s.Store.RunPayouts(ctx, time.Now(), minPayoutMinor)
			if err != nil {
				log.Printf("payouts: run failed: %v", err)
				continue
			}
			log.Printf("payouts: %d drafted, %d waiting on bank details, %d below the minimum",
				len(res.Created), len(res.NoAccount), len(res.BelowMinimum))
		}
	}
}

// ---- vendor ----

// handleVendorEarnings is the vendor's money overview.
func (s *Server) handleVendorEarnings(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	bal, err := s.Store.VendorBalance(r.Context(), v.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	payouts, err := s.Store.Payouts(r.Context(), v.ID, "", 20)
	if err != nil {
		storeErr(w, err)
		return
	}
	for i := range payouts {
		payouts[i].AccountHolder, payouts[i].BankName, payouts[i].Branch = "", "", ""
	}
	account := map[string]any{"set": false}
	if a, err := s.Store.PayoutAccount(r.Context(), v.ID); err == nil {
		account = map[string]any{"set": true, "bankName": a.BankName, "accountLast4": a.AccountLast4,
			"accountHolder": a.AccountHolder, "verified": a.VerifiedAt != nil}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"payableMinor": bal.PayableMinor, "pendingMinor": bal.PendingMinor, "currency": bal.Currency,
		"owedToSpottoMinor": max(-bal.PayableMinor, 0), "minPayoutMinor": minPayoutMinor,
		"nextPayoutAt": nextPayoutRun(time.Now()), "payoutAccount": account, "payouts": payouts,
	})
}

// handleVendorStatement lists the vendor's ledger lines for a period, as
// JSON or (?format=csv) a spreadsheet.
func (s *Server) handleVendorStatement(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	now := time.Now()
	from, to := parseRange(r, startOfLocalDay(now).AddDate(0, -1, 0), startOfLocalDay(now).AddDate(0, 0, 1))
	entries, err := s.Store.LedgerEntries(r.Context(), postgres.LedgerFilter{VendorID: v.ID, From: from, To: to, Limit: 5000})
	if err != nil {
		storeErr(w, err)
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		writeLedgerCSV(w, fmt.Sprintf("spotto-statement-%s-%s.csv", inSriLanka(from).Format("2006-01-02"), inSriLanka(to.AddDate(0, 0, -1)).Format("2006-01-02")), entries, false)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"from": from, "to": to, "entries": entries})
}

var ledgerKindLabels = map[string]string{
	postgres.LedgerBookingNet:     "Booking paid (your share)",
	postgres.LedgerPerGuestFee:    "Per-guest fee",
	postgres.LedgerRefundReversal: "Refund to guest (your share)",
	postgres.LedgerAdjustment:     "Adjustment",
}

// csvSafe stops a spreadsheet treating a cell as a formula.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func writeLedgerCSV(w http.ResponseWriter, filename string, entries []postgres.LedgerEntry, withVendor bool) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	cw := csv.NewWriter(w)
	header := []string{"Date", "Type", "Booking", "Service", "Amount (LKR)", "Settled in payout", "Note"}
	if withVendor {
		header = append([]string{"Vendor"}, header...)
	}
	cw.Write(header)
	for _, e := range entries {
		payout := ""
		if e.PayoutID != nil {
			payout = *e.PayoutID
		}
		row := []string{inSriLanka(e.CreatedAt).Format("2006-01-02 15:04"), ledgerKindLabels[e.Kind], e.BookingCode,
			csvSafe(e.ServiceName), formatMinorPlain(e.AmountMinor), payout, csvSafe(e.Memo)}
		if withVendor {
			row = append([]string{csvSafe(e.VendorName)}, row...)
		}
		cw.Write(row)
	}
	cw.Flush()
}

// formatMinorPlain renders minor units as "-1234.50".
func formatMinorPlain(minor int64) string {
	sign := ""
	if minor < 0 {
		sign, minor = "-", -minor
	}
	return fmt.Sprintf("%s%d.%02d", sign, minor/100, minor%100)
}

func (s *Server) handleGetPayoutAccount(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	a, err := s.Store.PayoutAccount(r.Context(), v.ID)
	if errors.Is(err, postgres.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"set": false})
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"set": true, "accountHolder": a.AccountHolder, "bankName": a.BankName,
		"branch": a.Branch, "accountLast4": a.AccountLast4, "verified": a.VerifiedAt != nil, "updatedAt": a.UpdatedAt})
}

// handlePutPayoutAccount saves where payouts go (owner only). The account
// number is stored encrypted and must be re-verified after any change.
func (s *Server) handlePutPayoutAccount(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireApprovedVendor(w, r)
	if !ok {
		return
	}
	if len(s.PayoutKey) != 32 {
		writeErr(w, http.StatusServiceUnavailable, "Bank details can't be saved yet. Please contact Spotto support.")
		return
	}
	var req struct {
		AccountHolder string `json:"accountHolder"`
		BankName      string `json:"bankName"`
		Branch        string `json:"branch"`
		AccountNumber string `json:"accountNumber"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	number := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, req.AccountNumber)
	holder, bank := sanitizeText(req.AccountHolder, 120), sanitizeText(req.BankName, 120)
	if holder == "" || bank == "" || len(number) < 6 || len(number) > 20 {
		writeErr(w, http.StatusBadRequest, "Enter the account holder, bank and a 6 to 20 digit account number.")
		return
	}
	enc, err := auth.EncryptSecret(s.PayoutKey, number)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "An unexpected error occurred. Please try again later.")
		return
	}
	if err := s.Store.UpsertPayoutAccount(r.Context(), postgres.PayoutAccount{VendorID: v.ID, AccountHolder: holder,
		BankName: bank, Branch: sanitizeText(req.Branch, 120), AccountNumberEnc: enc, AccountLast4: number[len(number)-4:]}); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "verified": false, "accountLast4": number[len(number)-4:]})
}

// handleVendorBooking is one booking with its full detail.
func (s *Server) handleVendorBooking(w http.ResponseWriter, r *http.Request) {
	v, ok := s.requireVendor(w, r)
	if !ok {
		return
	}
	b, err := s.Store.BookingByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || b.VendorID != v.ID {
		writeErr(w, http.StatusNotFound, "The requested item could not be found.")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// ---- admin ----

func (s *Server) handleAdminBookings(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := postgres.BookingFilter{VendorID: q.Get("vendorId"), Status: q.Get("status"), Query: strings.TrimSpace(q.Get("q")), Limit: 500}
	if q.Get("from") != "" || q.Get("to") != "" {
		f.From, f.To = parseRange(r, time.Time{}, time.Time{})
	}
	if id := chi.URLParam(r, "id"); id != "" {
		f.VendorID = id
	}
	list, err := s.Store.Bookings(r.Context(), f)
	if err != nil {
		storeErr(w, err)
		return
	}
	if list == nil {
		list = []domain.Booking{}
	}
	writeJSON(w, http.StatusOK, list)
}

// handleAdminBooking is one booking with its payments, refunds and ledger.
func (s *Server) handleAdminBooking(w http.ResponseWriter, r *http.Request) {
	b, err := s.Store.BookingByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	pays, err := s.Store.PaymentsByBooking(r.Context(), b.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	refunds, err := s.Store.RefundsForBooking(r.Context(), b.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	ledger, err := s.Store.LedgerEntries(r.Context(), postgres.LedgerFilter{BookingID: b.ID, Limit: 100})
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"booking": b, "payments": pays, "refunds": refunds, "ledger": ledger})
}

func (s *Server) handleAdminCancelBooking(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "a reason is required")
		return
	}
	b, err := s.Store.BookingByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	c := auth.FromContext(r.Context())
	refund, err := s.cancelBooking(r.Context(), b, cancelByAdmin, "admin:"+c.Subject)
	if err != nil {
		writeCancelErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "booking.cancel", "booking", b.ID, map[string]any{"reason": reason, "refundMinor": refund.AmountMinor})
	writeJSON(w, http.StatusOK, map[string]any{"status": "cancelled", "refundMinor": refund.AmountMinor})
}

func (s *Server) handleAdminFinanceSummary(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from, to := parseRange(r, startOfLocalDay(now).AddDate(0, 0, -30), startOfLocalDay(now).AddDate(0, 0, 1))
	sum, err := s.Store.FinanceSummary(r.Context(), from, to)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

func (s *Server) handleAdminLedger(w http.ResponseWriter, r *http.Request) {
	f := postgres.LedgerFilter{VendorID: r.URL.Query().Get("vendorId"), Limit: 1000}
	if r.URL.Query().Get("from") != "" || r.URL.Query().Get("to") != "" {
		f.From, f.To = parseRange(r, time.Time{}, time.Time{})
	}
	entries, err := s.Store.LedgerEntries(r.Context(), f)
	if err != nil {
		storeErr(w, err)
		return
	}
	if r.URL.Query().Get("format") == "csv" {
		writeLedgerCSV(w, "spotto-ledger.csv", entries, true)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleAdminLedgerAdjustment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		VendorID    string `json:"vendorId"`
		AmountMinor int64  `json:"amountMinor"`
		Memo        string `json:"memo"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	memo := sanitizeText(req.Memo, 300)
	if req.VendorID == "" || req.AmountMinor == 0 || memo == "" {
		writeErr(w, http.StatusBadRequest, "vendorId, a non-zero amountMinor and a memo are required")
		return
	}
	if _, err := s.Store.VendorByID(r.Context(), req.VendorID); err != nil {
		storeErr(w, err)
		return
	}
	c := auth.FromContext(r.Context())
	e := postgres.LedgerEntry{ID: newID("led"), VendorID: req.VendorID, AmountMinor: req.AmountMinor,
		Currency: domain.CurrencyLKR, Memo: memo, CreatedBy: "admin:" + c.Subject}
	if err := s.Store.AddLedgerAdjustment(r.Context(), e); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "ledger.adjust", "vendor", req.VendorID, map[string]any{"amountMinor": req.AmountMinor, "memo": memo})
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) handleAdminRefunds(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.Refunds(r.Context(), r.URL.Query().Get("status"), 500)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// handleAdminRefundDone records that finance paid a manual refund back.
func (s *Server) handleAdminRefundDone(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reference string `json:"reference"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	rf, err := s.Store.RefundByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if rf.Status != "manual" && !refundNeedsAction(rf) {
		writeErr(w, http.StatusConflict, "only a manual, failed or stuck refund can be marked done by hand")
		return
	}
	ref := sanitizeText(req.Reference, 120)
	if err := s.Store.SetRefundStatus(r.Context(), rf.ID, "manual_done", ref); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "refund.manual_done", "refund", rf.ID, map[string]any{"reference": ref, "amountMinor": rf.AmountMinor})
	writeJSON(w, http.StatusOK, map[string]string{"status": "manual_done"})
}

// staleRefundAge is how long a refund may sit in "requested" (recorded,
// gateway not yet asked) before it counts as stuck.
const staleRefundAge = 10 * time.Minute

// refundNeedsAction reports whether a refund failed, or got stuck between
// being recorded and the gateway being asked.
func refundNeedsAction(rf postgres.Refund) bool {
	return rf.Status == "failed" || (rf.Status == "requested" && time.Since(rf.CreatedAt) > staleRefundAge)
}

// handleAdminRetryRefund asks the gateway again for a failed refund.
func (s *Server) handleAdminRetryRefund(w http.ResponseWriter, r *http.Request) {
	rf, err := s.Store.RefundByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if !refundNeedsAction(rf) {
		writeErr(w, http.StatusConflict, "only a failed or stuck refund can be retried")
		return
	}
	s.processRefund(r.Context(), rf)
	rf, _ = s.Store.RefundByID(r.Context(), rf.ID)
	s.logAdminAction(r.Context(), "refund.retry", "refund", rf.ID, map[string]any{"status": rf.Status})
	writeJSON(w, http.StatusOK, rf)
}

func (s *Server) handleAdminPayouts(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.Payouts(r.Context(), r.URL.Query().Get("vendorId"), r.URL.Query().Get("status"), 500)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAdminPayout(w http.ResponseWriter, r *http.Request) {
	p, err := s.Store.PayoutByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	entries, err := s.Store.PayoutEntries(r.Context(), p.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"payout": p, "entries": entries})
}

// handleAdminRunPayouts drafts payouts now instead of waiting for Monday.
func (s *Server) handleAdminRunPayouts(w http.ResponseWriter, r *http.Request) {
	res, err := s.Store.RunPayouts(r.Context(), time.Now(), minPayoutMinor)
	if err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "payouts.run", "payouts", "", map[string]any{"created": len(res.Created)})
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleAdminPayoutAction(w http.ResponseWriter, r *http.Request) {
	id, action := chi.URLParam(r, "id"), chi.URLParam(r, "action")
	var req struct {
		BankReference string `json:"bankReference"`
		Note          string `json:"note"`
	}
	if r.ContentLength != 0 && !readJSON(w, r, &req) {
		return
	}
	c := auth.FromContext(r.Context())
	var err error
	switch action {
	case "approve":
		err = s.Store.ApprovePayout(r.Context(), id, "admin:"+c.Subject)
	case "mark-paid":
		ref := sanitizeText(req.BankReference, 120)
		if ref == "" {
			writeErr(w, http.StatusBadRequest, "the bank transfer reference is required")
			return
		}
		err = s.Store.MarkPayoutPaid(r.Context(), id, ref)
	case "fail", "cancel":
		status := map[string]string{"fail": "failed", "cancel": "cancelled"}[action]
		err = s.Store.FailPayout(r.Context(), id, status, sanitizeText(req.Note, 300))
	default:
		writeErr(w, http.StatusNotFound, "unknown action")
		return
	}
	if errors.Is(err, postgres.ErrPayoutState) {
		writeErr(w, http.StatusConflict, "This payout can't do that in its current state.")
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "payout."+action, "payout", id, map[string]any{"bankReference": req.BankReference, "note": req.Note})
	p, _ := s.Store.PayoutByID(r.Context(), id)
	writeJSON(w, http.StatusOK, p)
}

// handleAdminPayoutsExport is the bank transfer file for approved payouts,
// with full account numbers (decrypted for this file only).
func (s *Server) handleAdminPayoutsExport(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "approved"
	}
	list, err := s.Store.Payouts(r.Context(), "", status, 1000)
	if err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "payouts.export", "payouts", "", map[string]any{"status": status, "count": len(list)})
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="spotto-payouts-`+status+`.csv"`)
	cw := csv.NewWriter(w)
	cw.Write([]string{"Payout", "Vendor", "Account holder", "Bank", "Branch", "Account number", "Amount (LKR)", "Status"})
	for _, p := range list {
		number := ""
		if len(s.PayoutKey) == 32 && p.AccountEnc != "" {
			if n, err := auth.DecryptSecret(s.PayoutKey, p.AccountEnc); err == nil {
				number = n
			}
		}
		cw.Write([]string{p.ID, csvSafe(p.VendorName), csvSafe(p.AccountHolder), csvSafe(p.BankName), csvSafe(p.Branch),
			number, formatMinorPlain(p.AmountMinor), p.Status})
	}
	cw.Flush()
}

// handleAdminPayoutAccount shows a vendor's bank details in full, for
// checking them against the vendor's documents.
func (s *Server) handleAdminPayoutAccount(w http.ResponseWriter, r *http.Request) {
	a, err := s.Store.PayoutAccount(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if len(s.PayoutKey) == 32 {
		a.AccountNumber, _ = auth.DecryptSecret(s.PayoutKey, a.AccountNumberEnc)
	}
	s.logAdminAction(r.Context(), "payout_account.view", "vendor", a.VendorID, nil)
	writeJSON(w, http.StatusOK, a)
}

func (s *Server) handleAdminVerifyPayoutAccount(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	if err := s.Store.VerifyPayoutAccount(r.Context(), id, "admin:"+c.Subject); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "payout_account.verify", "vendor", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "verified"})
}

func (s *Server) handleAdminCommissionRules(w http.ResponseWriter, r *http.Request) {
	rules, err := s.Store.CommissionRules(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Server) handleAdminCreateCommissionRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Scope            domain.CommissionScope `json:"scope"`
		ScopeKey         string                 `json:"scopeKey"`
		Mode             domain.CommissionMode  `json:"mode"`
		RateBps          int                    `json:"rateBps"`
		FeePerGuestMinor int64                  `json:"feePerGuestMinor"`
		ActiveFrom       string                 `json:"activeFrom"` // RFC3339 or YYYY-MM-DD; empty = now
		Note             string                 `json:"note"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	rule := domain.CommissionRule{ID: newID("cr"), Scope: req.Scope, ScopeKey: strings.TrimSpace(req.ScopeKey), Mode: req.Mode,
		RateBps: req.RateBps, FeePerGuestMinor: req.FeePerGuestMinor, Note: sanitizeText(req.Note, 300), ActiveFrom: time.Now()}
	if req.ActiveFrom != "" {
		if t, err := time.Parse(time.RFC3339, req.ActiveFrom); err == nil {
			rule.ActiveFrom = t
		} else if d, err := parseLocalDate(req.ActiveFrom); err == nil {
			rule.ActiveFrom = d
		} else {
			writeErr(w, http.StatusBadRequest, "activeFrom must be a date")
			return
		}
	}
	if msg := validateCommissionRule(&rule); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	c := auth.FromContext(r.Context())
	rule.CreatedBy = "admin:" + c.Subject
	if err := s.Store.CreateCommissionRule(r.Context(), rule); err != nil {
		if postgres.IsUniqueViolation(err) {
			writeErr(w, http.StatusConflict, "a rule for that scope already starts at that time")
			return
		}
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "commission_rule.create", "commission_rule", rule.ID, rule)
	writeJSON(w, http.StatusCreated, rule)
}

func validateCommissionRule(r *domain.CommissionRule) string {
	switch r.Scope {
	case domain.ScopeDefault:
		r.ScopeKey = ""
	case domain.ScopeBookingType:
		if !domain.ValidBookingType(domain.BookingType(r.ScopeKey)) {
			return "scopeKey must be a booking type"
		}
	case domain.ScopeCategory:
		if _, ok := domain.CategoryByID(r.ScopeKey); !ok {
			return "scopeKey must be a category id"
		}
	case domain.ScopeVendor:
		if r.ScopeKey == "" {
			return "scopeKey must be a vendor id"
		}
	default:
		return "scope must be default, booking_type, category or vendor"
	}
	switch r.Mode {
	case domain.CommissionPercent:
		if r.RateBps < 0 || r.RateBps > 5000 {
			return "rate must be between 0% and 50%"
		}
		r.FeePerGuestMinor = 0
	case domain.CommissionPerGuestFee:
		if r.FeePerGuestMinor < 0 || r.FeePerGuestMinor > 1000000 {
			return "fee per guest must be between LKR 0 and LKR 10,000"
		}
		r.RateBps = 0
	default:
		return "mode must be percent or per_guest_fee"
	}
	return ""
}

func (s *Server) handleAdminDeleteCommissionRule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	err := s.Store.DeleteCommissionRule(r.Context(), id)
	if errors.Is(err, postgres.ErrRuleInEffect) {
		writeErr(w, http.StatusConflict, "This rule is already in effect. Add a newer rule to change the rate.")
		return
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "commission_rule.delete", "commission_rule", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// vendorBookingsRange applies ?from=&to=&status= to the vendor bookings
// list when given (the calendar asks for one week at a time).
func vendorBookingsFilter(r *http.Request, vendorID string) (postgres.BookingFilter, bool) {
	q := r.URL.Query()
	if q.Get("from") == "" && q.Get("to") == "" && q.Get("status") == "" {
		return postgres.BookingFilter{}, false
	}
	f := postgres.BookingFilter{VendorID: vendorID, Status: q.Get("status"), Limit: 1000}
	if q.Get("from") != "" || q.Get("to") != "" {
		f.From, f.To = parseRange(r, time.Time{}, time.Time{})
	}
	if n, err := strconv.Atoi(q.Get("limit")); err == nil {
		f.Limit = n
	}
	return f, true
}

// ---- exchange rates for gateways that can't charge LKR ----

// fxRateView is an exchange rate as the finance page shows it.
type fxRateView struct {
	domain.FxRate
	Stale bool `json:"stale"`
}

// handleAdminFxRates lists the exchange rates, the gateways that need
// one, and the margin added on top.
func (s *Server) handleAdminFxRates(w http.ResponseWriter, r *http.Request) {
	rates, err := s.Store.FxRates(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	now := time.Now()
	views := make([]fxRateView, len(rates))
	for i, rt := range rates {
		views[i] = fxRateView{FxRate: rt, Stale: rt.Stale(now)}
	}
	type gatewayView struct {
		Name           string `json:"name"`
		DisplayName    string `json:"displayName"`
		ChargeCurrency string `json:"chargeCurrency"`
	}
	gws := []gatewayView{}
	for _, g := range s.Payments.Enabled() {
		gws = append(gws, gatewayView{Name: g.Name(), DisplayName: g.DisplayName(), ChargeCurrency: g.ChargeCurrency()})
	}
	writeJSON(w, http.StatusOK, map[string]any{"rates": views, "gateways": gws, "marginBps": s.FxMarginBps,
		"maxAgeHours": int(domain.FxRateMaxAge / time.Hour)})
}

// parseRateMicros reads a decimal LKR rate ("302.5") as millionths.
func parseRateMicros(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	whole, frac, _ := strings.Cut(s, ".")
	if whole == "" || len(whole) > 6 || len(frac) > 6 {
		return 0, false
	}
	for _, c := range whole + frac {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	frac += strings.Repeat("0", 6-len(frac))
	w, err1 := strconv.ParseInt(whole, 10, 64)
	f, err2 := strconv.ParseInt(frac, 10, 64)
	if err1 != nil || err2 != nil {
		return 0, false
	}
	return w*1_000_000 + f, true
}

// handleAdminSetFxRate sets how many LKR one unit of a currency costs.
func (s *Server) handleAdminSetFxRate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Currency   string `json:"currency"`
		LKRPerUnit string `json:"lkrPerUnit"` // decimal, e.g. "302.50"
	}
	if !readJSON(w, r, &req) {
		return
	}
	cur := strings.ToUpper(strings.TrimSpace(req.Currency))
	if cur == domain.CurrencyLKR || !domain.TwoDecimalCurrency(cur) {
		writeErr(w, http.StatusBadRequest, "Choose a supported currency other than LKR.")
		return
	}
	micros, ok := parseRateMicros(req.LKRPerUnit)
	// 1 to 100,000 LKR per unit covers every supported currency with room.
	if !ok || micros < 1_000_000 || micros > 100_000*1_000_000 {
		writeErr(w, http.StatusBadRequest, "Enter the rate as LKR per unit, e.g. 302.50.")
		return
	}
	setBy := ""
	if c := auth.FromContext(r.Context()); c != nil {
		setBy = c.Subject
	}
	prev, _ := s.Store.FxRate(r.Context(), cur)
	rate, err := s.Store.SetFxRate(r.Context(), cur, micros, setBy)
	if err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "fx_rate.set", "fx_rate", cur, map[string]any{"lkrPerUnitMicros": micros, "previousMicros": prev.LKRPerUnitMicros})
	writeJSON(w, http.StatusOK, fxRateView{FxRate: rate})
}

// handleAdminPaymentIssues lists gateway notifications that need a person:
// chargebacks, amount mismatches, unknown orders and auto-refunds.
func (s *Server) handleAdminPaymentIssues(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.PaymentIssues(r.Context(), 200)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

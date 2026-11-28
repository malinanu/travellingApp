package httpapi_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// financeFixture is a paid fixture plus a super-admin and a plain admin,
// with a payout encryption key.
func financeFixture(t *testing.T) (*bookingFixture, string, string) {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	superEmail := fmt.Sprintf("super_%d@fixture.test", time.Now().UnixNano())
	f := newBookingFixture(t, "gyms", func(c *config.Config) {
		c.SuperAdminEmails = []string{superEmail}
		c.PayoutEncryptionKey = key
	})
	pw, _ := auth.HashPassword("testpass123")
	adminEmail := "admin_" + f.suffix + "@fixture.test"
	superID, adminID := "usr_super_"+f.suffix, "usr_admin_"+f.suffix
	must(t, f.store.CreateUser(f.ctx, domain.User{ID: superID, Email: superEmail, PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: time.Now()}))
	must(t, f.store.CreateUser(f.ctx, domain.User{ID: adminID, Email: adminEmail, PasswordHash: pw, Role: domain.RoleAdmin, CreatedAt: time.Now()}))
	t.Cleanup(func() { f.store.Pool.Exec(f.ctx, `DELETE FROM users WHERE id IN ($1,$2)`, superID, adminID) })
	return f, loginAs(t, f.srv, superEmail, "testpass123"), loginAs(t, f.srv, adminEmail, "testpass123")
}

func TestPayoutRunNetsFeesAndPaysOnce(t *testing.T) {
	f, superTk, plainAdminTk := financeFixture(t)
	gw := fakeGateway(t, f)
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 10,
		PartySizeMin: 1, PartySizeMax: 4, PricingMode: domain.PricingPerPerson, PriceMinor: 500000, Currency: "LKR"})

	b, _ := f.bookPaid(t, f.travATk, svc.ID, slot(2), 2) // LKR 10,000, vendor net 9,000
	f.notify(t, gw, b.paymentID, "succeeded", "po_"+b.id, b.amount, true)
	// The booking hasn't happened yet, so nothing is payable.
	_, earn := doJSON(t, http.MethodGet, f.url("/vendor/earnings"), f.vendorTk, nil)
	if earn["payableMinor"].(float64) != 0 || earn["pendingMinor"].(float64) != 900000 {
		t.Fatalf("before the booking: %v", earn)
	}
	// Move it into the past, and add a per-guest fee the vendor owes.
	f.store.Pool.Exec(f.ctx, `UPDATE ledger_entries SET payable_at=now()-interval '1 day' WHERE booking_id=$1`, b.id)
	f.owe(t, -30000, "per-guest fees")

	// A plain admin without finance permissions is refused.
	if code, _ := doJSON(t, http.MethodPost, f.url("/admin/payouts/run"), plainAdminTk, nil); code != http.StatusForbidden {
		t.Errorf("admin without finance.payouts: want 403, got %d", code)
	}

	// No verified bank account yet: the vendor is flagged, not paid.
	_, run := doJSON(t, http.MethodPost, f.url("/admin/payouts/run"), superTk, nil)
	if !containsStr(run["noAccount"], f.vendor.ID) {
		t.Fatalf("vendor without bank details should be listed: %v", run)
	}

	if code, out := doJSON(t, http.MethodPut, f.url("/vendor/payout-account"), f.vendorTk, map[string]any{
		"accountHolder": "Fixture Pvt Ltd", "bankName": "Commercial Bank", "branch": "Kollupitiya", "accountNumber": "8001-2345-6789"}); code != http.StatusOK {
		t.Fatalf("save account: %d %v", code, out)
	}
	var enc string
	must(t, f.store.Pool.QueryRow(f.ctx, `SELECT account_number_enc FROM vendor_payout_accounts WHERE vendor_id=$1`, f.vendor.ID).Scan(&enc))
	if strings.Contains(enc, "800123456789") || enc == "" {
		t.Error("the account number must be stored encrypted")
	}
	_, acct := doJSON(t, http.MethodGet, f.url("/vendor/payout-account"), f.vendorTk, nil)
	if acct["accountLast4"] != "6789" || acct["verified"] != false || acct["accountNumber"] != nil {
		t.Errorf("vendor sees only the last 4, unverified: %v", acct)
	}
	_, full := doJSON(t, http.MethodGet, f.url("/admin/vendors/"+f.vendor.ID+"/payout-account"), superTk, nil)
	if full["accountNumber"] != "800123456789" {
		t.Errorf("finance sees the full number: %v", full)
	}
	doJSON(t, http.MethodPost, f.url("/admin/vendors/"+f.vendor.ID+"/payout-account/verify"), superTk, nil)

	_, run = doJSON(t, http.MethodPost, f.url("/admin/payouts/run"), superTk, nil)
	var payout map[string]any
	for _, p := range run["created"].([]any) {
		if p.(map[string]any)["vendorId"] == f.vendor.ID {
			payout = p.(map[string]any)
		}
	}
	if payout == nil || payout["amountMinor"].(float64) != 870000 || payout["entryCount"].(float64) != 2 {
		t.Fatalf("payout should net 9,000 - 300 = 8,700 over 2 entries: %v", run)
	}
	id := payout["id"].(string)
	// Running again pays nothing twice.
	_, run = doJSON(t, http.MethodPost, f.url("/admin/payouts/run"), superTk, nil)
	for _, p := range run["created"].([]any) {
		if p.(map[string]any)["vendorId"] == f.vendor.ID {
			t.Error("a second run must not pay the same entries again")
		}
	}

	if code, _ := doJSON(t, http.MethodPost, f.url("/admin/payouts/"+id+"/mark-paid"), superTk, map[string]any{"bankReference": "TRX1"}); code != http.StatusConflict {
		t.Errorf("paying a draft before approval: want 409, got %d", code)
	}
	doJSON(t, http.MethodPost, f.url("/admin/payouts/"+id+"/approve"), superTk, nil)

	req, _ := http.NewRequest(http.MethodGet, f.url("/admin/payouts-export.csv"), nil)
	req.Header.Set("Authorization", "Bearer "+superTk)
	res, err := http.DefaultClient.Do(req)
	must(t, err)
	csvBody, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(csvBody), "800123456789") || !strings.Contains(string(csvBody), "8700.00") {
		t.Errorf("bank file should carry the account and amount: %s", csvBody)
	}

	// Failing the payout releases its entries for the next run.
	doJSON(t, http.MethodPost, f.url("/admin/payouts/"+id+"/fail"), superTk, map[string]any{"note": "bank rejected"})
	_, earn = doJSON(t, http.MethodGet, f.url("/vendor/earnings"), f.vendorTk, nil)
	if earn["payableMinor"].(float64) != 870000 {
		t.Errorf("after a failed payout the money is payable again: %v", earn["payableMinor"])
	}
}

func TestPayoutNegativeBalanceStaysOwed(t *testing.T) {
	f, superTk, _ := financeFixture(t)
	f.owe(t, -40000, "per-guest fees")
	_, run := doJSON(t, http.MethodPost, f.url("/admin/payouts/run"), superTk, nil)
	if !containsStr(run["belowMinimum"], f.vendor.ID) {
		t.Errorf("a vendor owing fees isn't paid: %v", run)
	}
	_, earn := doJSON(t, http.MethodGet, f.url("/vendor/earnings"), f.vendorTk, nil)
	if earn["owedToSpottoMinor"].(float64) != 40000 {
		t.Errorf("vendor sees what they owe: %v", earn)
	}
}

func TestAdminBookingsAndFinanceSummary(t *testing.T) {
	f, superTk, plainAdminTk := financeFixture(t)
	gw := fakeGateway(t, f)
	svc := f.service(domain.BookableService{Type: domain.BookingTypeSession, DurationMins: 60, Capacity: 10,
		PartySizeMin: 1, PartySizeMax: 4, PricingMode: domain.PricingPerBooking, PriceMinor: 300000, Currency: "LKR"})
	b, _ := f.bookPaid(t, f.travATk, svc.ID, slot(3), 2)
	f.notify(t, gw, b.paymentID, "succeeded", "fs_"+b.id, b.amount, true)

	if code, _ := doJSON(t, http.MethodGet, f.url("/admin/bookings"), plainAdminTk, nil); code != http.StatusForbidden {
		t.Errorf("bookings need bookings.view: want 403, got %d", code)
	}
	req, _ := http.NewRequest(http.MethodGet, f.url("/admin/vendors/"+f.vendor.ID+"/bookings"), nil)
	req.Header.Set("Authorization", "Bearer "+superTk)
	res, err := http.DefaultClient.Do(req)
	must(t, err)
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if !strings.Contains(string(body), b.code) {
		t.Errorf("vendor bookings tab should list the booking: %s", body)
	}
	_, detail := doJSON(t, http.MethodGet, f.url("/admin/bookings/"+b.id), superTk, nil)
	if len(detail["payments"].([]any)) != 1 || len(detail["ledger"].([]any)) != 1 {
		t.Errorf("detail has its payment and ledger line: %v", detail)
	}
	_, sum := doJSON(t, http.MethodGet, f.url("/admin/finance/summary"), superTk, nil)
	if sum["grossMinor"].(float64) < 300000 || sum["commissionMinor"].(float64) < 30000 {
		t.Errorf("summary includes the paid booking: %v", sum)
	}
	code, out := doJSON(t, http.MethodPost, f.url("/admin/bookings/"+b.id+"/cancel"), superTk, map[string]any{"reason": "venue flooded"})
	if code != http.StatusOK || out["refundMinor"].(float64) != 300000 {
		t.Errorf("admin cancel refunds in full: %d %v", code, out)
	}

	// Commission rules: validated, and a rule in effect can't be deleted.
	if code, _ := doJSON(t, http.MethodPost, f.url("/admin/commission-rules"), superTk, map[string]any{
		"scope": "category", "scopeKey": "no-such-category", "mode": "percent", "rateBps": 900}); code != http.StatusBadRequest {
		t.Errorf("unknown category: want 400, got %d", code)
	}
	code, rule := doJSON(t, http.MethodPost, f.url("/admin/commission-rules"), superTk, map[string]any{
		"scope": "vendor", "scopeKey": f.vendor.ID, "mode": "percent", "rateBps": 700})
	if code != http.StatusCreated {
		t.Fatalf("create rule: %d %v", code, rule)
	}
	if code, _ := doJSON(t, http.MethodDelete, f.url("/admin/commission-rules/"+rule["id"].(string)), superTk, nil); code != http.StatusConflict {
		t.Errorf("deleting a rule in effect: want 409, got %d", code)
	}
}

// owe adds a payable ledger adjustment (negative: the vendor owes Spotto).
func (f *bookingFixture) owe(t *testing.T, amountMinor int64, memo string) {
	t.Helper()
	must(t, f.store.AddLedgerAdjustment(f.ctx, postgres.LedgerEntry{ID: fmt.Sprintf("led_test_%d", time.Now().UnixNano()),
		VendorID: f.vendor.ID, AmountMinor: amountMinor, Currency: "LKR", Memo: memo, CreatedBy: "test"}))
}

func containsStr(list any, s string) bool {
	items, _ := list.([]any)
	for _, i := range items {
		if i == s {
			return true
		}
	}
	return false
}

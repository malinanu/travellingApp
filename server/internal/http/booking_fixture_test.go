package httpapi_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/store/postgres"
)

// bookingFixture is one vendor (open every day, all day), its team owner,
// two travellers and a place, isolated by a per-test suffix and removed
// when the test ends.
type bookingFixture struct {
	t        *testing.T
	ctx      context.Context
	store    *postgres.Store
	srv      *httptest.Server
	api      *httpapi.Server
	suffix   string
	vendor   domain.Vendor
	placeID  string
	vendorTk string
	travATk  string
	travBTk  string
	travAID  string
	travBID  string
}

const allDayHours = `{"hours":{"monday":{"open":"00:00","close":"23:59"},"tuesday":{"open":"00:00","close":"23:59"},"wednesday":{"open":"00:00","close":"23:59"},"thursday":{"open":"00:00","close":"23:59"},"friday":{"open":"00:00","close":"23:59"},"saturday":{"open":"00:00","close":"23:59"},"sunday":{"open":"00:00","close":"23:59"}}}`

func allDayAvailability() *domain.ServiceAvailabilityDoc {
	doc := domain.ServiceAvailabilityDoc{}
	for _, day := range []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"} {
		doc[day] = []domain.ServiceAvailabilityWindow{{Start: "00:00", End: "23:59"}}
	}
	return &doc
}

// newBookingFixture builds the fixture. configure, when set, can adjust the
// server config (e.g. payment settings) before the server is created.
func newBookingFixture(t *testing.T, category string, configure func(*config.Config, *httpapi.Server)) *bookingFixture {
	t.Helper()
	store := mustConnect(t)
	ctx := context.Background()
	suffix := fmt.Sprintf("%s_%d", category, time.Now().UnixNano())
	tokens := auth.NewTokens("test-secret-" + t.Name())
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32), VendorAppURL: "http://localhost:3000"}
	api := httpapi.New(store, tokens, cfg)
	if configure != nil {
		configure(&cfg, api)
	}
	srv := httptest.NewServer(api.Router(cfg))

	f := &bookingFixture{t: t, ctx: ctx, store: store, srv: srv, api: api, suffix: suffix}
	now := time.Now()
	pw, _ := auth.HashPassword("testpass123")
	f.vendor = domain.Vendor{ID: "ven_" + suffix, Name: "Fixture " + suffix, Category: category,
		Status: domain.VendorApproved, Tier: "Free", Flags: []string{}, SubmittedAt: now}
	must(t, store.UpsertVendor(ctx, f.vendor))
	must(t, store.UpdateVendorProfile(ctx, f.vendor.ID, []byte(allDayHours)))
	vendorEmail := "v_" + suffix + "@fixture.test"
	f.travAID, f.travBID = "usr_ta_"+suffix, "usr_tb_"+suffix
	must(t, store.CreateUser(ctx, domain.User{ID: "usr_v_" + suffix, Email: vendorEmail, PasswordHash: pw, Role: domain.RoleVendor, VendorID: &f.vendor.ID, VendorRole: domain.VendorRoleOwner, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: f.travAID, Email: "ta_" + suffix + "@fixture.test", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))
	must(t, store.CreateUser(ctx, domain.User{ID: f.travBID, Email: "tb_" + suffix + "@fixture.test", PasswordHash: pw, Role: domain.RoleTraveller, CreatedAt: now}))
	f.placeID = "plc_" + suffix
	must(t, store.UpsertPlace(ctx, domain.Place{ID: f.placeID, Name: "Fixture place", VendorID: &f.vendor.ID, Data: []byte(`{}`)}))

	t.Cleanup(func() {
		srv.Close()
		for _, q := range []string{
			`DELETE FROM ledger_entries WHERE vendor_id=$1`,
			`DELETE FROM payouts WHERE vendor_id=$1`,
			`DELETE FROM refunds WHERE booking_id IN (SELECT id FROM bookings WHERE vendor_id=$1)`,
			`DELETE FROM payment_events WHERE payment_id IN (SELECT p.id FROM payments p JOIN bookings b ON b.id=p.booking_id WHERE b.vendor_id=$1)`,
			`DELETE FROM payments WHERE booking_id IN (SELECT id FROM bookings WHERE vendor_id=$1)`,
			`DELETE FROM bookings WHERE vendor_id=$1`,
			`DELETE FROM service_occurrences WHERE service_id IN (SELECT id FROM bookable_services WHERE vendor_id=$1)`,
			`DELETE FROM bookable_services WHERE vendor_id=$1`,
			`DELETE FROM commission_rules WHERE scope='vendor' AND scope_key=$1`,
			`DELETE FROM vendor_payout_accounts WHERE vendor_id=$1`,
			`DELETE FROM vendor_notifications WHERE vendor_id=$1`,
			`DELETE FROM places WHERE vendor_id=$1`,
			`DELETE FROM users WHERE vendor_id=$1`,
		} {
			// Tables from later migrations may not exist yet on an older
			// schema; ignore those errors so cleanup stays best-effort.
			store.Pool.Exec(ctx, q, f.vendor.ID)
		}
		store.Pool.Exec(ctx, `DELETE FROM users WHERE id IN ($1,$2)`, f.travAID, f.travBID)
		store.Pool.Exec(ctx, `DELETE FROM vendors WHERE id=$1`, f.vendor.ID)
		store.Close()
	})

	f.vendorTk = loginAs(t, srv, vendorEmail, "testpass123")
	f.travATk = loginAs(t, srv, "ta_"+suffix+"@fixture.test", "testpass123")
	f.travBTk = loginAs(t, srv, "tb_"+suffix+"@fixture.test", "testpass123")
	return f
}

// service creates an active service for the fixture vendor, open all day
// every day unless the caller sets its own availability.
func (f *bookingFixture) service(svc domain.BookableService) domain.BookableService {
	f.t.Helper()
	if svc.ID == "" {
		svc.ID = fmt.Sprintf("svc_%s_%d", f.suffix, time.Now().UnixNano())
	}
	svc.VendorID = f.vendor.ID
	svc.Active = true
	if svc.Name == "" {
		svc.Name = "Fixture service"
	}
	if svc.Availability == nil && svc.Type != domain.BookingTypeStay {
		svc.Availability = allDayAvailability()
	}
	if svc.Capacity == 0 {
		svc.Capacity = 1
	}
	must(f.t, f.store.UpsertBookableService(f.ctx, svc))
	return svc
}

func (f *bookingFixture) url(path string) string { return f.srv.URL + path }

// slot is an hour-aligned start time days from now, as RFC3339.
func slot(days int) string {
	return time.Now().Add(time.Duration(days) * 24 * time.Hour).Truncate(time.Hour).Format(time.RFC3339)
}

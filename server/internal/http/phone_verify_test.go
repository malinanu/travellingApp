package httpapi_test

// Integration tests for vendor-portal phone numbers: verifying your own
// number with a one-time code, and the non-blocking "already used" hints on
// the Account page and the Business Profile contact number.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	"github.com/spotto/server/internal/domain"
	httpapi "github.com/spotto/server/internal/http"
	"github.com/spotto/server/internal/phone"
	"github.com/spotto/server/internal/store/postgres"
)

// fakeSender records codes instead of sending them, or fails when told to.
type fakeSender struct {
	mu   sync.Mutex
	last struct{ channel, phone, code string }
	fail bool
}

func (f *fakeSender) Send(_ context.Context, channel, rawPhone, code string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		return fmt.Errorf("provider down")
	}
	f.last.channel, f.last.phone, f.last.code = channel, rawPhone, code
	return nil
}

func (f *fakeSender) code() string { return f.sent().code }

func (f *fakeSender) sent() struct{ channel, phone, code string } {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.last
}

func call(t *testing.T, srv *httptest.Server, method, path, token string, body any) (int, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, srv.URL+path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

// uniqueMobile returns a Sri Lankan mobile (9 subscriber digits) that no
// earlier test run has used.
func uniqueMobile(offset int64) string {
	return fmt.Sprintf("7%08d", (time.Now().UnixNano()/1000+offset)%100_000_000)
}

type phoneTestEnv struct {
	srv    *httptest.Server
	sender *fakeSender
	store  *postgres.Store
}

func newPhoneTestEnv(t *testing.T) phoneTestEnv {
	t.Helper()
	store := mustConnect(t)
	loadCategories(t, store)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32),
		SuperAdminEmails: []string{"super@phonetest.dev"}}
	api := httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg)
	sender := &fakeSender{}
	api.Verify = sender
	srv := httptest.NewServer(api.Router(cfg))
	t.Cleanup(srv.Close)
	return phoneTestEnv{srv: srv, sender: sender, store: store}
}

// registerVendor signs up a business owner and returns their token. The
// codes sent to them are forgotten afterwards, so test runs don't add up
// towards the real daily caps.
func (e phoneTestEnv) registerVendor(t *testing.T, label, mobile string) string {
	t.Helper()
	status, body := postJSON(t, e.srv.URL+"/auth/register", map[string]any{
		"email": fmt.Sprintf("%s-%d@example.com", label, time.Now().UnixNano()), "password": "Phone-Test-Pass-1",
		"name": label + " Owner", "role": "vendor", "businessName": label + " Cafe", "category": "coffee",
		"town": "Weligama", "phone": mobile, "lat": 5.97, "lng": 80.43,
	})
	if status != http.StatusCreated {
		t.Fatalf("register %s: %d %v", label, status, body)
	}
	token := body["token"].(string)
	id := e.userID(t, token)
	t.Cleanup(func() {
		e.store.Pool.Exec(context.Background(), `DELETE FROM phone_code_sends WHERE user_id=$1`, id)
	})
	return token
}

func (e phoneTestEnv) userID(t *testing.T, token string) string {
	t.Helper()
	_, me := call(t, e.srv, http.MethodGet, "/me", token, nil)
	return me["id"].(string)
}

// recordSends adds n codes sent in the last hour to userID / number.
func (e phoneTestEnv) recordSends(t *testing.T, userID, number string, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if _, err := e.store.Pool.Exec(context.Background(),
			`INSERT INTO phone_code_sends (user_id, phone_key, channel, sri_lankan) VALUES ($1,$2,'sms',$3)`,
			userID, phone.Key(number), phone.IsSriLankan(number)); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		e.store.Pool.Exec(context.Background(), `DELETE FROM phone_code_sends WHERE user_id=$1`, userID)
	})
}

func (e phoneTestEnv) setContact(t *testing.T, token, mobile string) map[string]any {
	t.Helper()
	_, me := call(t, e.srv, http.MethodGet, "/me", token, nil)
	status, body := call(t, e.srv, http.MethodPut, "/me/contact", token, map[string]any{"name": me["name"], "email": me["email"], "phone": mobile})
	if status != http.StatusOK {
		t.Fatalf("PUT /me/contact: %d %v", status, body)
	}
	return body
}

func (e phoneTestEnv) verify(t *testing.T, token string) {
	t.Helper()
	if status, body := call(t, e.srv, http.MethodPost, "/me/phone/verify", token, map[string]string{"channel": "sms"}); status != http.StatusOK {
		t.Fatalf("send code: %d %v", status, body)
	}
	if status, body := call(t, e.srv, http.MethodPost, "/me/phone/verify/confirm", token, map[string]string{"code": e.sender.code()}); status != http.StatusOK {
		t.Fatalf("confirm code: %d %v", status, body)
	}
}

func TestPhoneVerification(t *testing.T) {
	e := newPhoneTestEnv(t)
	mobile := uniqueMobile(1)
	token := e.registerVendor(t, "verify", "0"+mobile)

	if status, _ := call(t, e.srv, http.MethodPost, "/me/phone/verify", token, map[string]string{"channel": "fax"}); status != http.StatusBadRequest {
		t.Fatalf("unknown channel: %d, want 400", status)
	}
	status, body := call(t, e.srv, http.MethodPost, "/me/phone/verify", token, map[string]string{"channel": "whatsapp"})
	if sent := e.sender.sent(); status != http.StatusOK || sent.channel != "whatsapp" || sent.phone != "+94 "+mobile {
		t.Fatalf("send: %d %v (sent %+v)", status, body, sent)
	}
	if status, _ := call(t, e.srv, http.MethodPost, "/me/phone/verify", token, map[string]string{"channel": "sms"}); status != http.StatusTooManyRequests {
		t.Fatalf("resend within a minute: %d, want 429", status)
	}

	wrong := "000000"
	if e.sender.code() == wrong {
		wrong = "111111"
	}
	if status, _ := call(t, e.srv, http.MethodPost, "/me/phone/verify/confirm", token, map[string]string{"code": wrong}); status != http.StatusBadRequest {
		t.Fatalf("wrong code: %d, want 400", status)
	}
	status, body = call(t, e.srv, http.MethodPost, "/me/phone/verify/confirm", token, map[string]string{"code": e.sender.code()})
	if status != http.StatusOK || body["phoneVerified"] != true {
		t.Fatalf("right code: %d %v", status, body)
	}
	if _, me := call(t, e.srv, http.MethodGet, "/me", token, nil); me["phoneVerified"] != true {
		t.Fatalf("/me after verifying: %v", me)
	}

	// Same number, different spelling: still verified.
	if body := e.setContact(t, token, "+94 "+mobile[:2]+" "+mobile[2:]); body["phoneVerified"] != true || body["phone"] != "+94 "+mobile {
		t.Fatalf("respelled number: %v", body)
	}
	// A different number has to be verified again.
	if body := e.setContact(t, token, "0"+uniqueMobile(2)); body["phoneVerified"] != false {
		t.Fatalf("new number kept verification: %v", body)
	}
}

func TestPhoneInUseHints(t *testing.T) {
	e := newPhoneTestEnv(t)
	shared := uniqueMobile(3)
	tokenA := e.registerVendor(t, "hint-a", "+94 "+shared)
	tokenB := e.registerVendor(t, "hint-b", "0"+uniqueMobile(4))

	check := func(token, path, number string) map[string]any {
		t.Helper()
		status, body := call(t, e.srv, http.MethodGet, path+"?phone="+url.QueryEscape(number), token, nil)
		if status != http.StatusOK {
			t.Fatalf("GET %s: %d %v", path, status, body)
		}
		return body
	}

	// Account page: B typing A's number (spelled differently) is told it's
	// taken; A's own number isn't reported to A.
	if body := check(tokenB, "/me/phone/check", "0"+shared); body["usedByOtherAccount"] != true {
		t.Fatalf("B checking A's number: %v", body)
	}
	if body := check(tokenA, "/me/phone/check", shared); body["usedByOtherAccount"] != false {
		t.Fatalf("A checking own number: %v", body)
	}

	// Business Profile: B lists a contact number; A typing it is warned,
	// until A's own verified number is that number.
	contact := uniqueMobile(5)
	if status, body := call(t, e.srv, http.MethodPut, "/vendor/business", tokenB,
		map[string]any{"contact": map[string]string{"phone": "+94 " + contact}}); status != http.StatusOK {
		t.Fatalf("B sets contact phone: %d %v", status, body)
	}
	body := check(tokenA, "/vendor/business/phone-check", "0"+contact)
	other, _ := body["otherBusiness"].(map[string]any)
	if other["name"] != "hint-b Cafe" {
		t.Fatalf("A typing B's contact number: %v", body)
	}

	e.setContact(t, tokenA, "0"+contact)
	if body := check(tokenA, "/vendor/business/phone-check", "0"+contact); body["otherBusiness"] == nil {
		t.Fatalf("an unverified team number must not silence the warning: %v", body)
	}
	e.verify(t, tokenA)
	if body := check(tokenA, "/vendor/business/phone-check", "0"+contact); body["otherBusiness"] != nil {
		t.Fatalf("A's own verified number still warned: %v", body)
	}
}

// foreignMobile returns a UK mobile no earlier test run has used.
func foreignMobile(offset int64) string {
	return fmt.Sprintf("+44 7700 %06d", (time.Now().UnixNano()/1000+offset)%1_000_000)
}

// limitsFromNow sets the daily limits to what's been sent in the last 24
// hours plus extra, putting the old limits back afterwards.
func (e phoneTestEnv) limitsFromNow(t *testing.T, extraSriLanka, extraOther int) postgres.PhoneCodeLimits {
	t.Helper()
	ctx := context.Background()
	old, err := e.store.PhoneCodeLimits(ctx)
	if err != nil {
		t.Fatal(err)
	}
	used, err := e.store.PhoneCodesSentLast24h(ctx)
	if err != nil {
		t.Fatal(err)
	}
	lim := postgres.PhoneCodeLimits{SriLanka: used.SriLanka + extraSriLanka, Other: used.Other + extraOther}
	if err := e.store.SetPhoneCodeLimits(ctx, lim); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { e.store.SetPhoneCodeLimits(ctx, old) })
	return lim
}

func TestPhoneCodeLimits(t *testing.T) {
	e := newPhoneTestEnv(t)
	send := func(token string) (int, map[string]any) {
		return call(t, e.srv, http.MethodPost, "/me/phone/verify", token, map[string]string{"channel": "sms"})
	}

	t.Run("travellers can't use the phone endpoints", func(t *testing.T) {
		status, body := postJSON(t, e.srv.URL+"/auth/register", map[string]any{
			"email": fmt.Sprintf("phone-trav-%d@example.com", time.Now().UnixNano()), "password": "Phone-Test-Pass-1", "name": "Phone Traveller",
		})
		if status != http.StatusCreated {
			t.Fatalf("register traveller: %d %v", status, body)
		}
		tok := body["token"].(string)
		if status, _ := send(tok); status != http.StatusForbidden {
			t.Fatalf("traveller sending a code: %d, want 403", status)
		}
		if status, _ := call(t, e.srv, http.MethodGet, "/me/phone/check?phone=0771234567", tok, nil); status != http.StatusForbidden {
			t.Fatalf("traveller checking a number: %d, want 403", status)
		}
	})

	t.Run("per account", func(t *testing.T) {
		mobile := "0" + uniqueMobile(11)
		tok := e.registerVendor(t, "cap-user", mobile)
		e.recordSends(t, e.userID(t, tok), "0"+uniqueMobile(12), 5)
		if status, body := send(tok); status != http.StatusTooManyRequests || body["error"] != "Too many codes requested — try again later." {
			t.Fatalf("sixth code this hour: %d %v", status, body)
		}
	})

	t.Run("per number", func(t *testing.T) {
		mobile := "0" + uniqueMobile(13)
		e.recordSends(t, "usr_phonecap_other", "+94 "+mobile[1:], 5)
		tok := e.registerVendor(t, "cap-number", mobile)
		if status, body := send(tok); status != http.StatusTooManyRequests || body["error"] != "Too many codes have gone to this number today — try again tomorrow." {
			t.Fatalf("sixth code to one number today: %d %v", status, body)
		}
	})

	t.Run("a failed send doesn't count", func(t *testing.T) {
		tok := e.registerVendor(t, "cap-fail", "0"+uniqueMobile(14))
		e.sender.fail = true
		status, _ := send(tok)
		e.sender.fail = false
		if status != http.StatusBadGateway {
			t.Fatalf("send with the provider down: %d, want 502", status)
		}
		var n int
		e.store.Pool.QueryRow(context.Background(), `SELECT count(*) FROM phone_code_sends WHERE user_id=$1`, e.userID(t, tok)).Scan(&n)
		if n != 0 {
			t.Fatalf("failed send left %d rows", n)
		}
	})

	t.Run("daily limits, Sri Lankan and foreign apart", func(t *testing.T) {
		// One code left in each group.
		e.limitsFromNow(t, 1, 1)
		lkA := e.registerVendor(t, "cap-day-a", "0"+uniqueMobile(15))
		lkB := e.registerVendor(t, "cap-day-b", "0"+uniqueMobile(16))
		ukA := e.registerVendor(t, "cap-day-c", foreignMobile(1))
		ukB := e.registerVendor(t, "cap-day-d", foreignMobile(2))
		full := "We can't send more codes right now — please try again later."
		if status, body := send(lkA); status != http.StatusOK {
			t.Fatalf("last Sri Lankan code under the limit: %d %v", status, body)
		}
		if status, body := send(lkB); status != http.StatusTooManyRequests || body["error"] != full {
			t.Fatalf("Sri Lankan code over the limit: %d %v", status, body)
		}
		if status, body := send(ukA); status != http.StatusOK {
			t.Fatalf("foreign code while only the Sri Lankan limit is used up: %d %v", status, body)
		}
		if status, body := send(ukB); status != http.StatusTooManyRequests || body["error"] != full {
			t.Fatalf("foreign code over the limit: %d %v", status, body)
		}
	})

	t.Run("number checks per account", func(t *testing.T) {
		tok := e.registerVendor(t, "cap-check", "0"+uniqueMobile(17))
		for i := 0; i < 30; i++ {
			if status, body := call(t, e.srv, http.MethodGet, "/me/phone/check?phone=0771234567", tok, nil); status != http.StatusOK {
				t.Fatalf("check %d: %d %v", i+1, status, body)
			}
		}
		if status, _ := call(t, e.srv, http.MethodGet, "/me/phone/check?phone=0771234567", tok, nil); status != http.StatusTooManyRequests {
			t.Fatalf("31st check this hour: %d, want 429", status)
		}
	})
}

func TestPhoneCodeLimitSettings(t *testing.T) {
	e := newPhoneTestEnv(t)
	ctx := context.Background()
	purge := func() { e.store.Pool.Exec(ctx, `DELETE FROM users WHERE id LIKE 'usr_phset_%'`) }
	purge()
	t.Cleanup(purge)
	pw, _ := auth.HashPassword("testpass123")
	for _, u := range []domain.User{
		{ID: "usr_phset_super", Email: "super@phonetest.dev", Role: domain.RoleAdmin},
		{ID: "usr_phset_admin", Email: "admin@phonetest.dev", Role: domain.RoleAdmin},
	} {
		u.PasswordHash, u.CreatedAt = pw, time.Now()
		if err := e.store.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
	}
	superTok := loginAs(t, e.srv, "super@phonetest.dev", "testpass123")
	adminTok := loginAs(t, e.srv, "admin@phonetest.dev", "testpass123")
	old, _ := e.store.PhoneCodeLimits(ctx)
	t.Cleanup(func() { e.store.SetPhoneCodeLimits(ctx, old) })

	status, body := call(t, e.srv, http.MethodGet, "/admin/settings/phone-codes", adminTok, nil)
	if status != http.StatusOK || body["limitSriLanka"] == nil || body["sentOther"] == nil {
		t.Fatalf("admin reading limits: %d %v", status, body)
	}
	if status, _ := call(t, e.srv, http.MethodPut, "/admin/settings/phone-codes", adminTok, map[string]int{"limitSriLanka": 300, "limitOther": 20}); status != http.StatusForbidden {
		t.Fatalf("plain admin changing limits: %d, want 403", status)
	}
	if status, _ := call(t, e.srv, http.MethodPut, "/admin/settings/phone-codes", superTok, map[string]int{"limitSriLanka": 0, "limitOther": 20}); status != http.StatusBadRequest {
		t.Fatalf("zero limit: %d, want 400", status)
	}
	status, body = call(t, e.srv, http.MethodPut, "/admin/settings/phone-codes", superTok, map[string]int{"limitSriLanka": 300, "limitOther": 20})
	if status != http.StatusOK || body["limitSriLanka"] != float64(300) || body["limitOther"] != float64(20) {
		t.Fatalf("super-admin changing limits: %d %v", status, body)
	}
	if got, _ := e.store.PhoneCodeLimits(ctx); got != (postgres.PhoneCodeLimits{SriLanka: 300, Other: 20}) {
		t.Fatalf("limits not saved: %+v", got)
	}
	var audits int
	e.store.Pool.QueryRow(ctx, `SELECT count(*) FROM admin_audit_log WHERE action='settings.phone_codes' AND actor_id='usr_phset_super'`).Scan(&audits)
	if audits == 0 {
		t.Fatal("limit change not in the audit log")
	}

	// Past 80% of a limit, admins get a bell alert.
	e.recordSends(t, "usr_phset_sender", "0"+uniqueMobile(21), 4)
	e.limitsFromNow(t, 1, 1000)
	notifs, err := e.store.AdminNotifications(ctx, "usr_phset_admin")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range notifs {
		found = found || (n.ID == "phone-codes" && strings.Contains(n.Title, "Sri Lanka"))
	}
	if !found {
		t.Fatalf("no phone-code alert at over 80%%: %+v", notifs)
	}
}

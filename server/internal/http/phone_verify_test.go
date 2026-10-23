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
	"sync"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/config"
	httpapi "github.com/spotto/server/internal/http"
)

// fakeSender records codes instead of sending them.
type fakeSender struct {
	mu   sync.Mutex
	last struct{ channel, phone, code string }
}

func (f *fakeSender) Send(_ context.Context, channel, rawPhone, code string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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
}

func newPhoneTestEnv(t *testing.T) phoneTestEnv {
	t.Helper()
	store := mustConnect(t)
	loadCategories(t, store)
	cfg := config.Config{CORSOrigins: []string{"http://localhost:3000"}, TOTPEncryptionKey: make([]byte, 32)}
	api := httpapi.New(store, auth.NewTokens("test-secret-test-secret-test-secret-1234"), cfg)
	sender := &fakeSender{}
	api.Verify = sender
	srv := httptest.NewServer(api.Router(cfg))
	t.Cleanup(srv.Close)
	return phoneTestEnv{srv: srv, sender: sender}
}

// registerVendor signs up a business owner and returns their token.
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
	return body["token"].(string)
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

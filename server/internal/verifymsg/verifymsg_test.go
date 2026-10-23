package verifymsg

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type captured struct {
	path, auth, contentType string
	body                    []byte
}

func newTestProviders(t *testing.T, cfg Config, status int, reply string) (*Providers, *captured) {
	t.Helper()
	got := &captured{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.path, got.auth, got.contentType = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Content-Type")
		got.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(status)
		io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	p := New(cfg)
	p.whatsAppBase, p.textLKURL, p.twilioBase = srv.URL, srv.URL+"/api/v3/sms/send", srv.URL
	return p, got
}

var allConfigured = Config{
	TextLKAPIToken: "tl-token", TextLKSenderID: "Spotto",
	TwilioAccountSID: "AC123", TwilioAuthToken: "tw-secret", TwilioFromNumber: "+15005550006",
	WhatsAppPhoneNumberID: "1098", WhatsAppAccessToken: "wa-token", WhatsAppTemplate: "spotto_otp",
}

func TestSriLankanSMSGoesToTextLK(t *testing.T) {
	p, got := newTestProviders(t, allConfigured, 200, `{"status":"success"}`)
	if err := p.Send(context.Background(), ChannelSMS, "076 123 4567", "123456"); err != nil {
		t.Fatal(err)
	}
	var body map[string]string
	json.Unmarshal(got.body, &body)
	if got.path != "/api/v3/sms/send" || got.auth != "Bearer tl-token" {
		t.Fatalf("path %q auth %q", got.path, got.auth)
	}
	if body["recipient"] != "94761234567" || body["sender_id"] != "Spotto" || body["type"] != "plain" || !strings.Contains(body["message"], "123456") {
		t.Fatalf("body %v", body)
	}
}

func TestForeignSMSGoesToTwilio(t *testing.T) {
	p, got := newTestProviders(t, allConfigured, 201, `{"sid":"SM1"}`)
	if err := p.Send(context.Background(), ChannelSMS, "+44 7700 900123", "654321"); err != nil {
		t.Fatal(err)
	}
	form, _ := url.ParseQuery(string(got.body))
	if got.path != "/2010-04-01/Accounts/AC123/Messages.json" || !strings.HasPrefix(got.auth, "Basic ") {
		t.Fatalf("path %q auth %q", got.path, got.auth)
	}
	if form.Get("To") != "+447700900123" || form.Get("From") != "+15005550006" || !strings.Contains(form.Get("Body"), "654321") {
		t.Fatalf("form %v", form)
	}
}

func TestWhatsAppSendsTheCopyCodeTemplate(t *testing.T) {
	p, got := newTestProviders(t, allConfigured, 200, `{"messages":[{"id":"wamid.1"}]}`)
	if err := p.Send(context.Background(), ChannelWhatsApp, "0761234567", "111222"); err != nil {
		t.Fatal(err)
	}
	if got.path != "/v23.0/1098/messages" || got.auth != "Bearer wa-token" {
		t.Fatalf("path %q auth %q", got.path, got.auth)
	}
	var body struct {
		To       string `json:"to"`
		Template struct {
			Name       string `json:"name"`
			Components []struct {
				Type       string              `json:"type"`
				SubType    string              `json:"sub_type"`
				Index      string              `json:"index"`
				Parameters []map[string]string `json:"parameters"`
			} `json:"components"`
		} `json:"template"`
	}
	json.Unmarshal(got.body, &body)
	if body.To != "94761234567" || body.Template.Name != "spotto_otp" || len(body.Template.Components) != 2 {
		t.Fatalf("body %s", got.body)
	}
	button := body.Template.Components[1]
	if button.Type != "button" || button.SubType != "url" || button.Index != "0" || button.Parameters[0]["text"] != "111222" {
		t.Fatalf("button %+v", button)
	}
}

func TestProviderErrorsSurface(t *testing.T) {
	p, _ := newTestProviders(t, allConfigured, 200, `{"status":"error","message":"Insufficient balance"}`)
	if err := p.Send(context.Background(), ChannelSMS, "0761234567", "123456"); err == nil || !strings.Contains(err.Error(), "Insufficient balance") {
		t.Fatalf("err = %v", err)
	}
	p, _ = newTestProviders(t, allConfigured, 401, `{"error":"bad token"}`)
	if err := p.Send(context.Background(), ChannelWhatsApp, "0761234567", "123456"); err == nil {
		t.Fatal("want an error for HTTP 401")
	}
}

func TestUnconfiguredChannel(t *testing.T) {
	dev := New(Config{})
	if err := dev.Send(context.Background(), ChannelSMS, "0761234567", "123456"); err != nil {
		t.Fatalf("outside production an unconfigured channel logs the code: %v", err)
	}
	prod := New(Config{Production: true})
	if err := prod.Send(context.Background(), ChannelWhatsApp, "0761234567", "123456"); !errors.Is(err, ErrChannelUnavailable) {
		t.Fatalf("err = %v, want ErrChannelUnavailable", err)
	}
}

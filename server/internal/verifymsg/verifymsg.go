// Package verifymsg delivers phone-verification codes: WhatsApp through the
// Meta WhatsApp Cloud API, SMS through text.lk for Sri Lankan numbers and
// Twilio for everything else. Like internal/mail, a channel without
// credentials logs the code instead of sending it (outside production), so
// local and staging setups work before any provider account exists.
package verifymsg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spotto/server/internal/phone"
)

const (
	ChannelWhatsApp = "whatsapp"
	ChannelSMS      = "sms"
)

// ErrChannelUnavailable means the channel has no provider configured and
// this is a production server, where codes must never only be logged.
var ErrChannelUnavailable = errors.New("verification channel not configured")

// Sender delivers a verification code to a phone number.
type Sender interface {
	Send(ctx context.Context, channel, rawPhone, code string) error
}

type Config struct {
	Production bool

	TextLKAPIToken string
	TextLKSenderID string

	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	WhatsAppPhoneNumberID string
	WhatsAppAccessToken   string
	WhatsAppTemplate      string // an approved copy-code authentication template
	WhatsAppTemplateLang  string
	WhatsAppAPIVersion    string
}

type Providers struct {
	cfg    Config
	client *http.Client
	// Provider base URLs, swapped for an httptest server in tests.
	whatsAppBase, textLKURL, twilioBase string
}

func New(cfg Config) *Providers {
	if cfg.WhatsAppTemplateLang == "" {
		cfg.WhatsAppTemplateLang = "en_US"
	}
	if cfg.WhatsAppAPIVersion == "" {
		cfg.WhatsAppAPIVersion = "v23.0"
	}
	p := &Providers{
		cfg:          cfg,
		client:       &http.Client{Timeout: 15 * time.Second},
		whatsAppBase: "https://graph.facebook.com",
		textLKURL:    "https://app.text.lk/api/v3/sms/send",
		twilioBase:   "https://api.twilio.com",
	}
	fallback := "logged, not sent"
	if cfg.Production {
		fallback = "refused"
	}
	for name, ready := range map[string]bool{
		"WhatsApp (WHATSAPP_*)":                p.whatsAppReady(),
		"SMS to Sri Lankan numbers (TEXTLK_*)": p.textLKReady(),
		"SMS to other numbers (TWILIO_*)":      p.twilioReady(),
	} {
		if !ready {
			log.Printf("verifymsg: %s not configured — phone verification codes on it will be %s", name, fallback)
		}
	}
	return p
}

func (p *Providers) whatsAppReady() bool {
	return p.cfg.WhatsAppPhoneNumberID != "" && p.cfg.WhatsAppAccessToken != "" && p.cfg.WhatsAppTemplate != ""
}
func (p *Providers) textLKReady() bool {
	return p.cfg.TextLKAPIToken != "" && p.cfg.TextLKSenderID != ""
}
func (p *Providers) twilioReady() bool {
	return p.cfg.TwilioAccountSID != "" && p.cfg.TwilioAuthToken != "" && p.cfg.TwilioFromNumber != ""
}

// Message is the SMS text; WhatsApp uses its approved template's own copy.
func Message(code string) string {
	return fmt.Sprintf("Your Spotto verification code is %s. It expires in 10 minutes. Don't share it with anyone.", code)
}

func (p *Providers) Send(ctx context.Context, channel, rawPhone, code string) error {
	to := phone.E164(rawPhone)
	if to == "" {
		return errors.New("no phone number")
	}
	var ready bool
	var send func() error
	switch {
	case channel == ChannelWhatsApp:
		ready, send = p.whatsAppReady(), func() error { return p.sendWhatsApp(ctx, to, code) }
	case channel == ChannelSMS && phone.IsSriLankan(rawPhone):
		ready, send = p.textLKReady(), func() error { return p.sendTextLK(ctx, to, code) }
	case channel == ChannelSMS:
		ready, send = p.twilioReady(), func() error { return p.sendTwilio(ctx, to, code) }
	default:
		return fmt.Errorf("unknown channel %q", channel)
	}
	if !ready {
		if p.cfg.Production {
			return ErrChannelUnavailable
		}
		log.Printf("verifymsg: (%s not configured) verification code for %s is %s", channel, phone.Masked(rawPhone), code)
		return nil
	}
	if err := send(); err != nil {
		log.Printf("verifymsg: %s to %s failed: %v", channel, phone.Masked(rawPhone), err)
		return err
	}
	return nil
}

// sendWhatsApp sends the copy-code authentication template: the code goes
// in the body and again as the copy button's parameter.
func (p *Providers) sendWhatsApp(ctx context.Context, to, code string) error {
	param := []map[string]string{{"type": "text", "text": code}}
	body := map[string]any{
		"messaging_product": "whatsapp",
		"to":                strings.TrimPrefix(to, "+"),
		"type":              "template",
		"template": map[string]any{
			"name":     p.cfg.WhatsAppTemplate,
			"language": map[string]string{"code": p.cfg.WhatsAppTemplateLang},
			"components": []map[string]any{
				{"type": "body", "parameters": param},
				{"type": "button", "sub_type": "url", "index": "0", "parameters": param},
			},
		},
	}
	endpoint := fmt.Sprintf("%s/%s/%s/messages", p.whatsAppBase, p.cfg.WhatsAppAPIVersion, url.PathEscape(p.cfg.WhatsAppPhoneNumberID))
	return p.postJSON(ctx, endpoint, p.cfg.WhatsAppAccessToken, body)
}

// sendTextLK uses text.lk's v3 API, which takes the number as 94XXXXXXXXX.
func (p *Providers) sendTextLK(ctx context.Context, to, code string) error {
	body := map[string]string{
		"recipient": strings.TrimPrefix(to, "+"),
		"sender_id": p.cfg.TextLKSenderID,
		"type":      "plain",
		"message":   Message(code),
	}
	return p.postJSON(ctx, p.textLKURL, p.cfg.TextLKAPIToken, body)
}

func (p *Providers) sendTwilio(ctx context.Context, to, code string) error {
	form := url.Values{"To": {to}, "From": {p.cfg.TwilioFromNumber}, "Body": {Message(code)}}
	endpoint := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", p.twilioBase, url.PathEscape(p.cfg.TwilioAccountSID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(p.cfg.TwilioAccountSID, p.cfg.TwilioAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return p.do(req)
}

func (p *Providers) postJSON(ctx context.Context, endpoint, bearer string, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return p.do(req)
}

func (p *Providers) do(req *http.Request) error {
	res, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(res.Body, 2048))
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s: HTTP %d: %s", req.URL.Host, res.StatusCode, strings.TrimSpace(string(respBody)))
	}
	// text.lk answers 200 even for some failures, with {"status":"error"}.
	var status struct {
		Status  string `json:"status"`
		Message string `json:"message"`
	}
	if json.Unmarshal(respBody, &status) == nil && status.Status == "error" {
		return fmt.Errorf("%s: %s", req.URL.Host, status.Message)
	}
	return nil
}

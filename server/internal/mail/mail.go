// Package mail sends transactional email over plain SMTP — no proprietary
// provider SDK, so it works against Gmail app-password SMTP, SES SMTP,
// Mailgun SMTP, Mailtrap (for local dev), or any other SMTP-compatible
// relay by just pointing the env vars at it.
package mail

import (
	"fmt"
	"log"
	netmail "net/mail"
	"net/smtp"
)

type Config struct {
	Host string
	Port string
	User string
	Pass string
	From string
}

// Mailer sends HTML email. If Host/Port/From aren't configured, it logs
// what it would have sent instead of failing — so local dev without real
// SMTP credentials still works, just without actual delivery.
type Mailer struct {
	cfg     Config
	enabled bool
}

func New(cfg Config) *Mailer {
	enabled := cfg.Host != "" && cfg.Port != "" && cfg.From != ""
	if !enabled {
		log.Println("mail: SMTP not configured (SMTP_HOST/SMTP_PORT/EMAIL_FROM) — outgoing email will be logged, not sent")
	}
	return &Mailer{cfg: cfg, enabled: enabled}
}

// Send delivers an HTML email. When the mailer is disabled (no SMTP
// configured), it logs the send and returns nil rather than erroring —
// callers that email as a side effect of another action (e.g. a team
// invite that's still usable via its copy-link) shouldn't fail the whole
// request just because email isn't set up yet.
func (m *Mailer) Send(to, subject, bodyHTML string) error {
	if !m.enabled {
		log.Printf("mail: (SMTP disabled) would send to=%q subject=%q", to, subject)
		return nil
	}
	addr := m.cfg.Host + ":" + m.cfg.Port
	var a smtp.Auth
	if m.cfg.User != "" {
		a = smtp.PlainAuth("", m.cfg.User, m.cfg.Pass, m.cfg.Host)
	}
	msg := buildMessage(m.cfg.From, to, subject, bodyHTML)
	if err := smtp.SendMail(addr, a, envelopeAddress(m.cfg.From), []string{to}, msg); err != nil {
		log.Printf("mail: send to=%q failed: %v", to, err)
		return err
	}
	return nil
}

// envelopeAddress returns the bare address for SMTP's MAIL FROM. EMAIL_FROM
// may carry a display name ("Spotto <no-reply@spotto.dev>") for the From
// header, but MAIL FROM only accepts the address itself — Gmail rejects the
// full form with "555 5.5.2 Syntax error".
func envelopeAddress(from string) string {
	if a, err := netmail.ParseAddress(from); err == nil {
		return a.Address
	}
	return from
}

func buildMessage(from, to, subject, bodyHTML string) []byte {
	return []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=\"UTF-8\"\r\n\r\n%s\r\n",
		from, to, subject, bodyHTML))
}

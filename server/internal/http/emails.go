package httpapi

import (
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/mail"
)

// Branded bodies for every email the API sends, all laid out by
// mail.Render so they share the app's look.

// renderEmail renders e, falling back to a bare escaped message if the
// layout fails, so a template problem never silently drops an email.
func renderEmail(e mail.Email) string {
	body, err := mail.Render(e)
	if err == nil {
		return body
	}
	log.Printf("email render %q: %v", e.Title, err)
	var b strings.Builder
	b.WriteString("<p><strong>" + html.EscapeString(e.Title) + "</strong></p>")
	for _, p := range append(e.Intro, e.Code, e.Button.URL, e.Note) {
		if p != "" {
			b.WriteString("<p>" + html.EscapeString(p) + "</p>")
		}
	}
	return b.String()
}

func resetEmailHTML(link string) string {
	return renderEmail(mail.Email{
		Preheader: "Use this link within the next hour to choose a new password.",
		Title:     "Reset your password",
		Intro:     []string{"We got a request to reset the password for your Spotto account. Tap the button below to choose a new one."},
		Button:    mail.Button{Label: "Set a new password", URL: link},
		Note:      "This link expires in 1 hour and can only be used once. If you didn't ask for a reset, you can safely ignore this email; your password won't change.",
		Footer:    "You're receiving this because a password reset was requested for your account.",
	})
}

func bookingConfirmedEmailHTML(b domain.Booking, serviceName string) string {
	return renderEmail(mail.Email{
		Preheader: "Your booking code for " + serviceName + " is inside.",
		Title:     "You're booked in!",
		Intro:     []string{"Your booking for " + serviceName + " is confirmed. Show this code when you arrive:"},
		Code:      b.Code,
		Details:   bookingDetails(b, serviceName),
		Note:      "You can see and manage this booking any time in the Spotto app.",
		Footer:    "You're receiving this because you made a booking on Spotto.",
	})
}

func bookingReminderEmailHTML(b domain.Booking) string {
	return renderEmail(mail.Email{
		Preheader: b.ServiceName + " is coming up on " + b.StartAt.Format("Jan 2, 3:04 PM") + ".",
		Title:     "Your booking is coming up",
		Intro:     []string{"Just a reminder: " + b.ServiceName + " is coming up soon. Show this code when you arrive:"},
		Code:      b.Code,
		Details:   bookingDetails(b, b.ServiceName),
		Footer:    "You're receiving this because you have an upcoming booking on Spotto.",
	})
}

func bookingDetails(b domain.Booking, serviceName string) []mail.Detail {
	d := []mail.Detail{{Label: "Booking", Value: serviceName}}
	if !b.StartAt.IsZero() {
		d = append(d, mail.Detail{Label: "When", Value: b.StartAt.Format("Mon, Jan 2 · 3:04 PM")})
	}
	if b.PartySize > 0 {
		d = append(d, mail.Detail{Label: "Party size", Value: fmt.Sprint(b.PartySize)})
	}
	return d
}

func inviteEmailHTML(vendorName, role, code, link string) string {
	return renderEmail(mail.Email{
		Preheader: "Join " + vendorName + " on Spotto.",
		Title:     "You're invited to join " + vendorName,
		Intro:     []string{"You've been invited to join " + vendorName + " on Spotto as a " + role + "."},
		Button:    mail.Button{Label: "Accept invite", URL: link},
		Details:   []mail.Detail{{Label: "Business", Value: vendorName}, {Label: "Role", Value: role}, {Label: "Invite code", Value: code}},
		Note:      "This invite expires in 7 days. If you weren't expecting it, you can ignore this email.",
		Footer:    "You're receiving this because a Spotto business owner invited this address to their team.",
	})
}

func profileGrantEmailHTML(vendorName, pin string, durationMins int) string {
	return renderEmail(mail.Email{
		Preheader: "Your PIN to edit " + vendorName + "'s Business Profile.",
		Title:     "Your Business Profile access PIN",
		Intro:     []string{"You've been given temporary edit access to " + vendorName + "'s Business Profile. Enter this PIN on the Business Profile page:"},
		Code:      pin,
		Details: []mail.Detail{
			{Label: "Enter the PIN within", Value: fmt.Sprintf("%d minutes", int(profilePinTTL/time.Minute))},
			{Label: "Edit access lasts", Value: fmt.Sprintf("%d minutes", durationMins)},
		},
		Note:   "Don't share this PIN. If you weren't expecting it, let the business owner know.",
		Footer: "You're receiving this because a Spotto business owner granted you profile access.",
	})
}

func vendorReportEmailHTML(v domain.Vendor, dashboardURL string) string {
	return renderEmail(mail.Email{
		Preheader: "Your weekly Spotto summary for " + v.Name + ".",
		Title:     "Weekly digest for " + v.Name,
		Intro:     []string{"Here's your weekly activity summary on Spotto."},
		Details: []mail.Detail{
			{Label: "Status", Value: string(v.Status)},
			{Label: "Tier", Value: string(v.Tier)},
			{Label: "Town", Value: v.Town},
		},
		Button: mail.Button{Label: "Go to dashboard", URL: dashboardURL},
		Note:   "Open your vendor dashboard for full analytics and live offer stats.",
		Footer: "You're receiving this weekly report as the owner of a Spotto business.",
	})
}

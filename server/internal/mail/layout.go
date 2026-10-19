package mail

import (
	"bytes"
	"html/template"
)

// Email is the content of one branded Spotto email. Render lays it out in
// the app's "Sunny Postcard" look (app/lib/core/theme/palette.dart) so every
// transactional email reads as the same product. Every field is plain text
// and gets HTML-escaped; empty fields are left out.
type Email struct {
	// Preheader is the inbox preview line shown after the subject.
	Preheader string
	Title     string
	// Intro paragraphs, shown under the title.
	Intro []string
	// Code is shown large in a pill, e.g. a booking code or PIN.
	Code    string
	Button  Button
	Details []Detail
	// Note is small muted text under the main content (expiry, "if this
	// wasn't you", ...).
	Note string
	// Footer says why the recipient got this email.
	Footer string
}

type Button struct {
	Label string
	URL   string
}

type Detail struct {
	Label string
	Value string
}

// Render returns the email as a complete HTML document. It's built for
// email clients rather than browsers: table layout, inline styles only, no
// images (Gmail blocks them by default) and a light colour scheme so dark
// mode clients don't invert the brand colours.
func Render(e Email) (string, error) {
	var b bytes.Buffer
	if err := layoutTmpl.Execute(&b, e); err != nil {
		return "", err
	}
	return b.String(), nil
}

// Colours are the app palette (app/lib/core/theme/palette.dart): sand
// #FBF6ED, deep teal #0B4A54, teal #0B6F86, ink #103D3A, sun #F6B81E for
// the one primary action.
var layoutTmpl = template.Must(template.New("email").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
<meta name="supported-color-schemes" content="light">
<title>{{.Title}}</title>
</head>
<body style="margin:0;padding:0;background:#FBF6ED;">
{{- with .Preheader}}
<div style="display:none;max-height:0;overflow:hidden;opacity:0;mso-hide:all;">{{.}}</div>
{{- end}}
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="background:#FBF6ED;">
<tr><td align="center" style="padding:28px 16px;">
  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="max-width:560px;">
    <tr><td style="background:#0B4A54;border-radius:16px 16px 0 0;padding:22px 28px;">
      <span style="font-family:'Fredoka','Nunito','Arial Rounded MT Bold',Arial,sans-serif;font-size:26px;font-weight:700;color:#FBF6ED;letter-spacing:-0.3px;">Spotto</span><span style="font-family:Arial,sans-serif;font-size:26px;font-weight:700;color:#F6B81E;">.</span>
    </td></tr>
    <tr><td style="background:#FFFFFF;border:1px solid #E7E1D4;border-top:0;border-radius:0 0 16px 16px;padding:32px 28px 28px;font-family:'Nunito','Segoe UI',Arial,sans-serif;color:#103D3A;">
      <h1 style="margin:0 0 14px;font-family:'Fredoka','Nunito','Arial Rounded MT Bold',Arial,sans-serif;font-size:24px;line-height:1.25;font-weight:600;color:#103D3A;">{{.Title}}</h1>
      {{- range .Intro}}
      <p style="margin:0 0 14px;font-size:16px;line-height:1.6;color:#3A5A57;">{{.}}</p>
      {{- end}}
      {{- with .Code}}
      <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:8px 0 18px;"><tr>
        <td style="background:#DDEDEF;border-radius:12px;padding:14px 22px;font-family:'SFMono-Regular',Consolas,'Courier New',monospace;font-size:26px;font-weight:700;letter-spacing:4px;color:#0B4A54;">{{.}}</td>
      </tr></table>
      {{- end}}
      {{- if .Button.URL}}
      <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:10px 0 8px;"><tr>
        <td align="center" bgcolor="#F6B81E" style="border-radius:12px;">
          <a href="{{.Button.URL}}" target="_blank" style="display:inline-block;padding:14px 28px;font-family:'Nunito','Segoe UI',Arial,sans-serif;font-size:16px;font-weight:800;color:#103D3A;text-decoration:none;border-radius:12px;">{{.Button.Label}}</a>
        </td>
      </tr></table>
      <p style="margin:6px 0 18px;font-size:12px;line-height:1.5;color:#6C8582;">Button not working? Paste this link into your browser:<br><a href="{{.Button.URL}}" style="color:#0B6F86;word-break:break-all;">{{.Button.URL}}</a></p>
      {{- end}}
      {{- if .Details}}
      <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" style="margin:6px 0 18px;border-top:1px solid #E7E1D4;">
        {{- range .Details}}
        <tr>
          <td style="padding:10px 0;border-bottom:1px solid #E7E1D4;font-size:14px;color:#6C8582;">{{.Label}}</td>
          <td align="right" style="padding:10px 0;border-bottom:1px solid #E7E1D4;font-size:14px;font-weight:700;color:#103D3A;">{{.Value}}</td>
        </tr>
        {{- end}}
      </table>
      {{- end}}
      {{- with .Note}}
      <p style="margin:4px 0 0;font-size:13px;line-height:1.55;color:#6C8582;">{{.}}</p>
      {{- end}}
    </td></tr>
    <tr><td align="center" style="padding:20px 12px 0;font-family:'Nunito','Segoe UI',Arial,sans-serif;font-size:12px;line-height:1.6;color:#6C8582;">
      <strong style="color:#0B4A54;">Spotto</strong> &middot; Sri Lanka travel, local spots
      {{- with .Footer}}<br>{{.}}{{end}}
    </td></tr>
  </table>
</td></tr>
</table>
</body>
</html>
`))

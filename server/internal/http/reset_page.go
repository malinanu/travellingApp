package httpapi

import (
	"html/template"
	"log"
	"net/http"
)

// handleResetPasswordPage serves the browser page travellers land on from
// their password-reset email (they have no web app, unlike vendors and
// admins). GET renders the form; POST submits it as a plain HTML form and
// re-renders with the outcome, so the page needs no JavaScript. The token
// rides in a hidden field and goes through the same resetPassword checks
// as POST /auth/reset-password.
func (s *Server) handleResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	data := resetPageData{Token: r.URL.Query().Get("token")}
	status := http.StatusOK
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			status, data.Error = http.StatusBadRequest, "Could not read the form. Please try again."
		} else {
			data.Token = r.PostForm.Get("token")
			pw, confirm := r.PostForm.Get("password"), r.PostForm.Get("confirm")
			if pw != confirm {
				status, data.Error = http.StatusBadRequest, "The two passwords don't match."
			} else if st, msg := s.resetPassword(r.Context(), data.Token, pw); st != http.StatusOK {
				status, data.Error = st, msg
			} else {
				data.Done = true
			}
		}
	}
	if data.Token == "" && !data.Done {
		status, data.Error, data.Invalid = http.StatusBadRequest, "This reset link is incomplete. Request a new one from the app.", true
	} else if data.Error == "invalid or expired reset link" {
		data.Error, data.Invalid = "This reset link is invalid, already used, or expired. Request a new one from the app.", true
	}

	h := w.Header()
	// The global CSP is default-src 'none'; this page needs inline styles,
	// the app's Google Fonts and to post its form back to itself.
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline' https://fonts.googleapis.com; font-src https://fonts.gstatic.com; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	h.Set("Content-Type", "text/html; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("Referrer-Policy", "no-referrer") // keep the token out of Referer headers
	w.WriteHeader(status)
	if err := resetPageTmpl.Execute(w, data); err != nil {
		log.Printf("reset password page render: %v", err)
	}
}

type resetPageData struct {
	Token   string
	Error   string
	Invalid bool // token unusable: hide the form
	Done    bool
}

// Matches the app's "Sunny Postcard" theme (app/lib/core/theme/palette.dart)
// and the branded emails (internal/mail/layout.go).
var resetPageTmpl = template.Must(template.New("reset").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<meta name="color-scheme" content="light">
<title>Reset password · Spotto</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link rel="stylesheet" href="https://fonts.googleapis.com/css2?family=Fredoka:wght@500;600;700&family=Nunito:wght@400;600;700;800&display=swap">
<style>
  :root { --sand:#FBF6ED; --card:#FFFFFF; --ink:#103D3A; --ink-soft:#3A5A57; --muted:#6C8582; --line:#E7E1D4; --teal:#0B6F86; --deep:#0B4A54; --tint:#DDEDEF; --sun:#F6B81E; --err:#B2472F; --err-tint:#F8E6E1; --ok:#1B7A5A; }
  * { box-sizing: border-box; }
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center; padding:24px 16px; background:var(--sand); color:var(--ink); font:16px/1.55 "Nunito","Segoe UI",system-ui,sans-serif; }
  main { width:100%; max-width:420px; border-radius:20px; overflow:hidden; background:var(--card); border:1px solid var(--line); box-shadow:0 18px 40px -18px rgba(16,61,58,.35); }
  header { background:var(--deep); padding:20px 26px; }
  .brand { margin:0; font:700 26px/1 "Fredoka","Nunito",sans-serif; letter-spacing:-.01em; color:var(--sand); }
  .brand span { color:var(--sun); }
  .body { padding:28px 26px 26px; }
  h1 { margin:0 0 6px; font:600 24px/1.25 "Fredoka","Nunito",sans-serif; }
  .lead { margin:0 0 6px; color:var(--ink-soft); }
  label { display:block; font-weight:700; font-size:14px; margin:18px 0 6px; }
  input { width:100%; padding:13px 14px; font:inherit; color:var(--ink); background:var(--sand); border:1.5px solid var(--line); border-radius:12px; transition:border-color .15s, box-shadow .15s; }
  input:focus { outline:none; border-color:var(--teal); box-shadow:0 0 0 4px rgba(11,111,134,.15); background:#fff; }
  button { width:100%; margin-top:22px; padding:14px; font:800 16px "Nunito",sans-serif; color:var(--ink); background:var(--sun); border:0; border-radius:12px; cursor:pointer; box-shadow:0 6px 16px -6px rgba(246,184,30,.7); transition:transform .1s, filter .15s; }
  button:hover { filter:brightness(1.04); }
  button:active { transform:translateY(1px); }
  .hint { font-size:13px; color:var(--muted); margin:10px 0 0; }
  .err { margin:14px 0 0; padding:12px 14px; border-radius:12px; background:var(--err-tint); color:var(--err); font-weight:600; font-size:14px; }
  .done { text-align:center; padding:8px 0 4px; }
  .badge { width:64px; height:64px; margin:0 auto 16px; border-radius:50%; background:var(--tint); color:var(--teal); display:flex; align-items:center; justify-content:center; font:700 32px/1 "Nunito",sans-serif; }
  footer { padding:0 26px 22px; font-size:12px; color:var(--muted); text-align:center; }
</style>
</head>
<body>
<main>
  <header><p class="brand">Spotto<span>.</span></p></header>
  <div class="body">
  {{if .Done}}
    <div class="done">
      <div class="badge" aria-hidden="true">&#10003;</div>
      <h1>Password updated</h1>
      <p class="lead">Your new password is set. Head back to the Spotto app and sign in with it.</p>
    </div>
  {{else}}
    <h1>Set a new password</h1>
    <p class="lead">Choose a new password for your Spotto account.</p>
    {{with .Error}}<p class="err" role="alert">{{.}}</p>{{end}}
    {{if not .Invalid}}
    <form method="post" action="/reset-password">
      <input type="hidden" name="token" value="{{.Token}}">
      <label for="password">New password</label>
      <input id="password" name="password" type="password" autocomplete="new-password" minlength="10" maxlength="128" required>
      <label for="confirm">Confirm new password</label>
      <input id="confirm" name="confirm" type="password" autocomplete="new-password" minlength="10" maxlength="128" required>
      <p class="hint">At least 10 characters. Avoid all-digit or email-based passwords.</p>
      <button type="submit">Save password</button>
    </form>
    {{end}}
  {{end}}
  </div>
  <footer>Spotto · Sri Lanka travel, local spots</footer>
</main>
</body>
</html>
`))

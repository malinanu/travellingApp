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
	// The global CSP is default-src 'none'; this page needs inline styles
	// and to post its form back to itself, nothing else.
	h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
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

var resetPageTmpl = template.Must(template.New("reset").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex">
<title>Reset password · Spotto</title>
<style>
  :root { color-scheme: light dark; --bg:#fbf7f1; --card:#fff; --ink:#1d1b18; --muted:#6b645b; --line:#e4ddd2; --accent:#e8743b; --err:#b3261e; --ok:#1f7a4d; }
  @media (prefers-color-scheme: dark) { :root { --bg:#151412; --card:#1f1d1a; --ink:#f3eee7; --muted:#a39b90; --line:#36322d; --err:#f2b8b5; --ok:#7fd1a6; } }
  * { box-sizing: border-box; }
  body { margin:0; min-height:100vh; display:flex; align-items:center; justify-content:center; padding:16px; background:var(--bg); color:var(--ink); font:16px/1.5 system-ui,-apple-system,"Segoe UI",Roboto,sans-serif; }
  main { width:100%; max-width:380px; background:var(--card); border:1px solid var(--line); border-radius:16px; padding:28px 24px; }
  .brand { font-weight:800; letter-spacing:-.01em; color:var(--accent); margin:0 0 4px; }
  h1 { font-size:22px; margin:0 0 16px; }
  label { display:block; font-size:14px; color:var(--muted); margin:14px 0 6px; }
  input { width:100%; padding:12px; font:inherit; color:inherit; background:transparent; border:1px solid var(--line); border-radius:10px; }
  input:focus { outline:2px solid var(--accent); outline-offset:1px; }
  button { width:100%; margin-top:20px; padding:13px; font:inherit; font-weight:700; color:#fff; background:var(--accent); border:0; border-radius:10px; cursor:pointer; }
  .hint { font-size:13px; color:var(--muted); margin:8px 0 0; }
  .err { color:var(--err); margin:0 0 8px; }
  .ok { color:var(--ok); }
</style>
</head>
<body>
<main>
  <p class="brand">Spotto</p>
  {{if .Done}}
    <h1>Password updated</h1>
    <p class="ok">Your new password is set. Go back to the Spotto app and sign in with it.</p>
  {{else}}
    <h1>Set a new password</h1>
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
</main>
</body>
</html>
`))

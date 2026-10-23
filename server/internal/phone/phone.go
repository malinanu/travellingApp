// Package phone normalises the phone numbers people type into the vendor
// portal, so the same number written different ways ("+94 76 123 4567",
// "0761234567", "761234567") compares equal and is stored one way.
// Numbers are deliberately not unique: one person can hold a traveller and a
// business account with the same mobile.
package phone

import "strings"

func digits(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Key is the comparison form: digits only, with a leading Sri Lankan 94
// country code (when more than 9 digits remain) and then a leading trunk 0
// dropped, so local and international spellings of a Sri Lankan number meet.
// It matches sriLankanPhone() in web/vendor/app/login/page.tsx and KeySQL.
func Key(raw string) string {
	d := digits(raw)
	if strings.HasPrefix(d, "94") && len(d) > 9 {
		d = d[2:]
	}
	return strings.TrimPrefix(d, "0")
}

// IsSriLankan reports whether raw is a Sri Lankan number: written with the
// 94 country code, as a local 0 + 9 digits, or as the bare 9-digit number
// the registration wizard's fixed +94 prefix collects.
func IsSriLankan(raw string) bool {
	d := digits(raw)
	switch {
	case d == "":
		return false
	case strings.HasPrefix(d, "94") && len(d) == 11:
		return true
	case strings.HasPrefix(d, "0") && len(d) == 10:
		return true
	default:
		return len(d) == 9 && !strings.HasPrefix(strings.TrimSpace(raw), "+")
	}
}

// Normalize is the stored form: "+94 761234567" for Sri Lankan numbers,
// "+<digits>" for numbers written with a country code, "" for no number.
// Anything else (say a half-typed number the Account page autosaved) is kept
// as typed rather than guessed at.
func Normalize(raw string) string {
	d := digits(raw)
	switch {
	case d == "":
		return ""
	case IsSriLankan(raw):
		return "+94 " + Key(raw)
	case strings.HasPrefix(strings.TrimSpace(raw), "+"):
		return "+" + d
	default:
		return strings.TrimSpace(raw)
	}
}

// E164 is the number as messaging APIs want it ("+94761234567"), or "" when
// raw isn't a number a code can be sent to (no country code to go on).
func E164(raw string) string {
	n := Normalize(raw)
	if !strings.HasPrefix(n, "+") {
		return ""
	}
	return strings.ReplaceAll(n, " ", "")
}

// KeySQL is Key as a Postgres expression over col, so queries compare
// stored numbers (whatever their formatting) the same way Go does.
func KeySQL(col string) string {
	return `regexp_replace(regexp_replace(regexp_replace(coalesce(` + col + `, ''), '\D', '', 'g'), '^94(\d{8,})$', '\1'), '^0', '')`
}

// Masked shows only the last four digits, for "code sent to …" messages.
func Masked(raw string) string {
	n := Normalize(raw)
	d := digits(n)
	if len(d) <= 4 {
		return n
	}
	prefix := "+"
	if strings.HasPrefix(n, "+94 ") {
		prefix = "+94 "
	}
	return prefix + "•••• " + d[len(d)-4:]
}

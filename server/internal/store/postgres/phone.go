package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/phone"
)

// PhoneVerification is the one live verification code a user may have.
type PhoneVerification struct {
	UserID    string
	Phone     string
	Channel   string
	CodeHash  string
	ExpiresAt time.Time
	Attempts  int
	SentAt    time.Time
}

// UpsertPhoneVerification stores a freshly sent code, replacing any earlier
// one (and its attempt count) for the same user.
func (s *Store) UpsertPhoneVerification(ctx context.Context, v PhoneVerification) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO phone_verifications (user_id, phone, channel, code_hash, expires_at, attempts, sent_at)
		VALUES ($1,$2,$3,$4,$5,0,$6)
		ON CONFLICT (user_id) DO UPDATE SET phone=EXCLUDED.phone, channel=EXCLUDED.channel,
			code_hash=EXCLUDED.code_hash, expires_at=EXCLUDED.expires_at, attempts=0, sent_at=EXCLUDED.sent_at`,
		v.UserID, v.Phone, v.Channel, v.CodeHash, v.ExpiresAt, v.SentAt)
	return err
}

func (s *Store) PhoneVerificationByUser(ctx context.Context, userID string) (PhoneVerification, error) {
	var v PhoneVerification
	err := s.Pool.QueryRow(ctx, `
		SELECT user_id, phone, channel, code_hash, expires_at, attempts, sent_at
		FROM phone_verifications WHERE user_id=$1`, userID).
		Scan(&v.UserID, &v.Phone, &v.Channel, &v.CodeHash, &v.ExpiresAt, &v.Attempts, &v.SentAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	return v, err
}

func (s *Store) IncrementPhoneVerificationAttempts(ctx context.Context, userID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE phone_verifications SET attempts = attempts + 1 WHERE user_id=$1`, userID)
	return err
}

// MarkPhoneVerified records that the user confirmed phoneNumber — only if
// it is still their number — and drops the used code.
func (s *Store) MarkPhoneVerified(ctx context.Context, userID, phoneNumber string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `UPDATE users SET phone_verified_at = now() WHERE id=$1 AND phone=$2`, userID, phoneNumber); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM phone_verifications WHERE user_id=$1`, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PhoneUsedByOtherUser reports whether any account other than
// exceptUserID has this number (compared by phone.Key). key must be
// non-empty.
func (s *Store) PhoneUsedByOtherUser(ctx context.Context, key, exceptUserID string) (bool, error) {
	var used bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE id <> $2 AND phone <> '' AND `+phone.KeySQL("phone")+` = $1)`,
		key, exceptUserID).Scan(&used)
	return used, err
}

// VendorTeamHasVerifiedPhone reports whether someone on vendorID's team
// (owner, manager or staff) has verified this number as their own.
func (s *Store) VendorTeamHasVerifiedPhone(ctx context.Context, vendorID, key string) (bool, error) {
	var found bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM users WHERE vendor_id=$1 AND phone_verified_at IS NOT NULL
			AND `+phone.KeySQL("phone")+` = $2)`,
		vendorID, key).Scan(&found)
	return found, err
}

// OtherVendorWithContactPhone finds another business whose public contact
// number (profile.contact.phone) is this number. found is false when none.
func (s *Store) OtherVendorWithContactPhone(ctx context.Context, key, exceptVendorID string) (name string, found bool, err error) {
	err = s.Pool.QueryRow(ctx, `
		SELECT name FROM vendors
		WHERE id <> $2 AND `+phone.KeySQL("profile->'contact'->>'phone'")+` = $1
		ORDER BY name LIMIT 1`,
		key, exceptVendorID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return name, true, nil
}

// PhoneSendLimits caps how many verification codes one account and one
// number get. Zero means no cap for that field. The platform-wide daily
// limits live in platform_settings (see PhoneCodeLimits).
type PhoneSendLimits struct {
	PerUserHour  int
	PerUserDay   int
	PerNumberDay int
}

var (
	ErrPhoneSendUserLimit   = errors.New("too many codes for this account")
	ErrPhoneSendNumberLimit = errors.New("too many codes to this number today")
	ErrPhoneSendDailyLimit  = errors.New("daily code limit reached")
)

// Default platform-wide daily limits, used until a super-admin sets them.
const (
	DefaultPhoneCodesDailyLK   = 500
	DefaultPhoneCodesDailyIntl = 50
)

// PhoneCodeLimits are the platform-wide limits on codes sent in any 24
// hours, kept apart for Sri Lankan numbers (text.lk) and the rest
// (Twilio), which cost far more and are where SMS fraud aims.
type PhoneCodeLimits struct {
	SriLanka int `json:"limitSriLanka"`
	Other    int `json:"limitOther"`
}

type querier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func phoneCodeLimits(ctx context.Context, q querier) (PhoneCodeLimits, error) {
	lim := PhoneCodeLimits{SriLanka: DefaultPhoneCodesDailyLK, Other: DefaultPhoneCodesDailyIntl}
	err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT value::int FROM platform_settings WHERE key='phone_codes_daily_limit_lk'), $1),
		       COALESCE((SELECT value::int FROM platform_settings WHERE key='phone_codes_daily_limit_intl'), $2)`,
		lim.SriLanka, lim.Other).Scan(&lim.SriLanka, &lim.Other)
	return lim, err
}

func (s *Store) PhoneCodeLimits(ctx context.Context) (PhoneCodeLimits, error) {
	return phoneCodeLimits(ctx, s.Pool)
}

func (s *Store) SetPhoneCodeLimits(ctx context.Context, lim PhoneCodeLimits) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO platform_settings (key, value) VALUES
			('phone_codes_daily_limit_lk', to_jsonb($1::int)), ('phone_codes_daily_limit_intl', to_jsonb($2::int))
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, lim.SriLanka, lim.Other)
	return err
}

// PhoneCodeUsage is how many codes went out in the last 24 hours.
type PhoneCodeUsage struct {
	SriLanka int       `json:"sentSriLanka"`
	Other    int       `json:"sentOther"`
	LastSent time.Time `json:"-"`
}

func (s *Store) PhoneCodesSentLast24h(ctx context.Context) (PhoneCodeUsage, error) {
	var u PhoneCodeUsage
	err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE sri_lankan), count(*) FILTER (WHERE NOT sri_lankan),
		       COALESCE(max(sent_at), '1970-01-01')
		FROM phone_code_sends WHERE sent_at > now() - interval '1 day'`).Scan(&u.SriLanka, &u.Other, &u.LastSent)
	return u, err
}

// ReservePhoneCodeSend records a code about to be sent to phoneKey for
// userID if every limit still allows it, and returns the row's id so a send
// that fails can be released. The platform-wide limit is the one for the
// number's group (Sri Lankan or not). Reservations are serialised, so
// concurrent requests can't all slip under a limit.
func (s *Store) ReservePhoneCodeSend(ctx context.Context, userID, phoneKey, channel string, sriLankan bool, lim PhoneSendLimits) (int64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext('phone_code_sends'))`); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM phone_code_sends WHERE sent_at < now() - interval '2 days'`); err != nil {
		return 0, err
	}
	daily, err := phoneCodeLimits(ctx, tx)
	if err != nil {
		return 0, err
	}
	groupLimit := daily.Other
	if sriLankan {
		groupLimit = daily.SriLanka
	}
	var userHour, userDay, numberDay, groupDay int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE user_id=$1 AND sent_at > now() - interval '1 hour'),
		       count(*) FILTER (WHERE user_id=$1),
		       count(*) FILTER (WHERE phone_key=$2),
		       count(*) FILTER (WHERE sri_lankan=$3)
		FROM phone_code_sends WHERE sent_at > now() - interval '1 day'`, userID, phoneKey, sriLankan).
		Scan(&userHour, &userDay, &numberDay, &groupDay); err != nil {
		return 0, err
	}
	over := func(n, limit int) bool { return limit > 0 && n >= limit }
	switch {
	case over(userHour, lim.PerUserHour), over(userDay, lim.PerUserDay):
		return 0, ErrPhoneSendUserLimit
	case over(numberDay, lim.PerNumberDay):
		return 0, ErrPhoneSendNumberLimit
	case over(groupDay, groupLimit):
		return 0, ErrPhoneSendDailyLimit
	}
	var id int64
	if err := tx.QueryRow(ctx, `
		INSERT INTO phone_code_sends (user_id, phone_key, channel, sri_lankan) VALUES ($1,$2,$3,$4) RETURNING id`,
		userID, phoneKey, channel, sriLankan).Scan(&id); err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}

// ReleasePhoneCodeSend forgets a reservation whose message couldn't be sent.
func (s *Store) ReleasePhoneCodeSend(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM phone_code_sends WHERE id=$1`, id)
	return err
}

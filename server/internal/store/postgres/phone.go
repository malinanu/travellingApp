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

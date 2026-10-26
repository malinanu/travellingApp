package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

const sessionCols = `id, user_id, client, device_label, created_at, last_used_at, expires_at, revoked_at, rotated_at`

func scanSession(row pgx.Row) (domain.Session, error) {
	var s domain.Session
	err := row.Scan(&s.ID, &s.UserID, &s.Client, &s.DeviceLabel, &s.CreatedAt, &s.LastUsedAt, &s.ExpiresAt, &s.RevokedAt, &s.RotatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, ErrNotFound
	}
	return s, err
}

// CreateSession stores a new signed-in device under the hash of its refresh token.
func (s *Store) CreateSession(ctx context.Context, sess domain.Session, refreshHash string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO user_sessions (id, user_id, refresh_hash, client, device_label, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		sess.ID, sess.UserID, refreshHash, sess.Client, sess.DeviceLabel, sess.ExpiresAt)
	return err
}

// SessionByRefreshHash finds the session whose current refresh token hashes to hash.
func (s *Store) SessionByRefreshHash(ctx context.Context, hash string) (domain.Session, error) {
	return scanSession(s.Pool.QueryRow(ctx, `SELECT `+sessionCols+` FROM user_sessions WHERE refresh_hash = $1`, hash))
}

// SessionByPreviousHash finds the session whose refresh token used to hash to
// hash, before its last rotation.
func (s *Store) SessionByPreviousHash(ctx context.Context, hash string) (domain.Session, error) {
	return scanSession(s.Pool.QueryRow(ctx, `SELECT `+sessionCols+` FROM user_sessions WHERE previous_hash = $1`, hash))
}

// RotateSession swaps a session's refresh token for a new one and extends
// its expiry, but only if oldHash is still its current token and it isn't
// revoked. ok is false when another request rotated it first.
func (s *Store) RotateSession(ctx context.Context, id, oldHash, newHash string, expiresAt time.Time) (ok bool, err error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE user_sessions
		SET refresh_hash = $3, previous_hash = $2, rotated_at = now(), last_used_at = now(), expires_at = $4
		WHERE id = $1 AND refresh_hash = $2 AND revoked_at IS NULL`,
		id, oldHash, newHash, expiresAt)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// RevokeSession ends one session (if it isn't already).
func (s *Store) RevokeSession(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE user_sessions SET revoked_at = now() WHERE id = $1 AND revoked_at IS NULL`, id)
	return err
}

// RevokeOtherUserSessions ends every live session of userID except keepID
// (pass "" to end them all). Returns how many were ended.
func (s *Store) RevokeOtherUserSessions(ctx context.Context, userID, keepID string) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE user_sessions SET revoked_at = now()
		WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL`, userID, keepID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// RevokeAllUserSessions signs userID out everywhere: bumps token_version so
// every access token already issued stops working, ends every session so no
// refresh token works either, and drops push tokens so signed-out phones
// stop getting notifications. Returns how many sessions were ended.
func (s *Store) RevokeAllUserSessions(ctx context.Context, userID string) (int64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE users SET token_version = token_version + 1 WHERE id = $1`, userID)
	if err != nil {
		return 0, err
	}
	if tag.RowsAffected() == 0 {
		return 0, ErrNotFound
	}
	ended, err := tx.Exec(ctx, `UPDATE user_sessions SET revoked_at = now() WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM user_push_tokens WHERE user_id = $1`, userID); err != nil {
		return 0, err
	}
	return ended.RowsAffected(), tx.Commit(ctx)
}

// ActiveUserSessions lists userID's signed-in devices, most recently used first.
func (s *Store) ActiveUserSessions(ctx context.Context, userID string) ([]domain.Session, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+sessionCols+` FROM user_sessions
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > now()
		ORDER BY last_used_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Session{}
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// SearchUsers finds users by name, email or exact id (newest first; all
// users when q is empty), with their live session count and last activity.
func (s *Store) SearchUsers(ctx context.Context, q string, limit int) ([]domain.AdminUserSummary, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT u.id, COALESCE(u.name, ''), u.email, u.role, COALESCE(v.name, ''), u.publisher, u.created_at,
		       (SELECT COUNT(*) FROM user_sessions us
		        WHERE us.user_id = u.id AND us.revoked_at IS NULL AND us.expires_at > now()),
		       (SELECT MAX(us.last_used_at) FROM user_sessions us WHERE us.user_id = u.id)
		FROM users u
		LEFT JOIN vendors v ON v.id = u.vendor_id
		WHERE $1 = '' OR u.id = $1 OR u.email ILIKE '%' || $1 || '%' OR u.name ILIKE '%' || $1 || '%'
		ORDER BY u.created_at DESC
		LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AdminUserSummary{}
	for rows.Next() {
		var u domain.AdminUserSummary
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role, &u.VendorName, &u.Publisher, &u.CreatedAt, &u.ActiveSessions, &u.LastActiveAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

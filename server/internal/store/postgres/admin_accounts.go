package postgres

// Super-admin account surgery: moving an account between traveller and
// vendor, and permanently deleting a user or a business. Each runs in one
// transaction and also cleans up the rows that point at users/vendors
// without a foreign key (invites, grants, sponsorships, places), which a
// bare DELETE would leave dangling.

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// ErrNotOnTeam means the chosen new owner isn't a member of the business.
var ErrNotOnTeam = errors.New("not a member of this business")

// OwnerHandover promotes a team member to owner of VendorID, for when the
// current owner leaves (role change or delete).
type OwnerHandover struct {
	UserID   string
	VendorID string
}

// RoleChange describes a super-admin role change. VendorID nil means the
// account becomes a traveller with no business; NewVendor, when set, is
// created in the same transaction and must have ID == *VendorID.
type RoleChange struct {
	UserID     string
	Role       domain.Role
	VendorID   *string
	VendorRole string
	NewVendor  *domain.Vendor
	NewOwner   *OwnerHandover
}

func promoteOwner(ctx context.Context, tx pgx.Tx, h *OwnerHandover) error {
	if h == nil {
		return nil
	}
	var name string
	err := tx.QueryRow(ctx, `
		UPDATE users SET vendor_role='owner', token_version=token_version+1
		WHERE id=$1 AND vendor_id=$2 AND role='vendor' RETURNING name`,
		h.UserID, h.VendorID).Scan(&name)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotOnTeam
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE vendors SET owner=$2 WHERE id=$1`, h.VendorID, name)
	return err
}

// ChangeUserRole applies ch and bumps the user's token_version so tokens
// issued under the old role stop working.
func (s *Store) ChangeUserRole(ctx context.Context, ch RoleChange) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if ch.NewVendor != nil {
		if err := upsertVendor(ctx, tx, *ch.NewVendor); err != nil {
			return err
		}
	}
	if err := promoteOwner(ctx, tx, ch.NewOwner); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE users SET role=$2, vendor_id=$3, vendor_role=$4, token_version=token_version+1
		WHERE id=$1`, ch.UserID, ch.Role, ch.VendorID, ch.VendorRole)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// DeleteUserPermanently deletes the account and the rows that reference it
// without a foreign key. Everything with a foreign key (sessions, push
// tokens, trips, groups, payments, saved places, …) cascades. Reviews,
// claims and bookings they made stay, shown without a name.
func (s *Store) DeleteUserPermanently(ctx context.Context, userID string, newOwner *OwnerHandover) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := promoteOwner(ctx, tx, newOwner); err != nil {
		return err
	}
	for _, q := range []string{
		`DELETE FROM password_reset_tokens WHERE user_id=$1`,
		`DELETE FROM vendor_profile_grants WHERE grantee_id=$1`,
		`UPDATE vendor_invites SET revoked=true WHERE created_by=$1 AND accepted_at IS NULL`,
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return err
		}
	}
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// VendorQuestDaysFrom lists the daily quests vendorID sponsors on or after
// day ('YYYY-MM-DD') — these have to be reassigned before it can be deleted.
func (s *Store) VendorQuestDaysFrom(ctx context.Context, vendorID, day string) ([]string, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT date FROM daily_quests WHERE vendor_id=$1 AND date >= $2 ORDER BY date`, vendorID, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var days []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		days = append(days, d)
	}
	return days, rows.Err()
}

// DeletedVendor reports what DeleteVendorPermanently removed or detached,
// for the caller's follow-up (sessions, media) and the audit log.
type DeletedVendor struct {
	PlaceIDs        []string
	DetachedUserIDs []string
}

// DeleteVendorPermanently deletes the business, its map place and the rows
// that reference it without a foreign key. Team members are detached (and
// their tokens invalidated), not deleted. Quests it sponsored before today
// are dropped; callers must refuse while it sponsors today or later (see
// VendorQuestDaysFrom), otherwise the daily_quests foreign key fails.
func (s *Store) DeleteVendorPermanently(ctx context.Context, vendorID, today string) (DeletedVendor, error) {
	var out DeletedVendor
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)

	for _, q := range []struct {
		sql  string
		args []any
	}{
		{`DELETE FROM daily_quests WHERE vendor_id=$1 AND date < $2`, []any{vendorID, today}},
		{`DELETE FROM vendor_sponsorships WHERE vendor_id=$1`, []any{vendorID}},
		{`DELETE FROM vendor_invites WHERE vendor_id=$1`, []any{vendorID}},
		{`DELETE FROM vendor_profile_grants WHERE vendor_id=$1`, []any{vendorID}},
	} {
		if _, err := tx.Exec(ctx, q.sql, q.args...); err != nil {
			return out, err
		}
	}

	if out.PlaceIDs, err = collectIDs(ctx, tx, `DELETE FROM places WHERE vendor_id=$1 RETURNING id`, vendorID); err != nil {
		return out, err
	}
	for _, placeID := range out.PlaceIDs {
		if _, err := tx.Exec(ctx, `DELETE FROM saved_places WHERE place_id=$1`, placeID); err != nil {
			return out, err
		}
	}
	if out.DetachedUserIDs, err = collectIDs(ctx, tx, `
		UPDATE users SET vendor_id=NULL, vendor_role='', token_version=token_version+1
		WHERE vendor_id=$1 RETURNING id`, vendorID); err != nil {
		return out, err
	}

	tag, err := tx.Exec(ctx, `DELETE FROM vendors WHERE id=$1`, vendorID)
	if err != nil {
		return out, err
	}
	if tag.RowsAffected() == 0 {
		return out, ErrNotFound
	}
	return out, tx.Commit(ctx)
}

func collectIDs(ctx context.Context, tx pgx.Tx, sql string, args ...any) ([]string, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// MediaAllByOwner lists every media row an owner has, in any status — for
// removing the objects when the owner is deleted.
func (s *Store) MediaAllByOwner(ctx context.Context, ownerType, ownerID string) ([]domain.Media, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+mediaCols+` FROM media WHERE owner_type=$1 AND owner_id=$2`, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Media
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

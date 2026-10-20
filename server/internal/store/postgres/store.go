// Package postgres is the pgx-backed store used by the API and seed command.
package postgres

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spotto/server/internal/domain"
)

//go:embed migrations/*.sql
var migrations embed.FS

var ErrNotFound = errors.New("not found")

// BadReviewFlagThreshold is how many 1-2 star reviews a vendor accumulates
// before the admin map shows it red-flagged.
const BadReviewFlagThreshold = 5

type Store struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return &Store{Pool: pool}, nil
}

func (s *Store) Close() { s.Pool.Close() }

// Migrate applies embedded numbered migrations not yet recorded in schema_migrations.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.Pool.Exec(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		var done bool
		if err := s.Pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name=$1)`, name).Scan(&done); err != nil {
			return err
		}
		if done {
			continue
		}
		sql, err := migrations.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		tx, err := s.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			tx.Rollback(ctx)
			return fmt.Errorf("migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, name); err != nil {
			tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func jsonb(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ---- users ----

func (s *Store) CreateUser(ctx context.Context, u domain.User) error {
	vendorRole := u.VendorRole
	if vendorRole == "" {
		vendorRole = domain.VendorRoleOwner
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, role, name, country, travel_style, interests, vendor_id, extra, created_at, vendor_role)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		u.ID, u.Email, u.PasswordHash, u.Role, u.Name, u.Country, jsonb(u.TravelStyle),
		jsonb(u.Interests), u.VendorID, u.Extra, u.CreatedAt, vendorRole)
	return err
}

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	var travelStyle, interests []byte
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.Country,
		&travelStyle, &interests, &u.VendorID, &u.Extra, &u.CreatedAt,
		&u.TOTPSecret, &u.TOTPPendingSecret, &u.TOTPEnabled, &u.TokenVersion,
		&u.ShareLocation, &u.Lat, &u.Lng, &u.LocationUpdatedAt,
		&u.Phone, &u.VendorRole, &u.NotifyRedemptions, &u.NotifyReviews, &u.NotifyPayouts, &u.NotifyGrowthTips,
		&u.GoogleID, &u.AvatarURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	if err != nil {
		return u, err
	}
	json.Unmarshal(travelStyle, &u.TravelStyle)
	json.Unmarshal(interests, &u.Interests)
	return u, nil
}

const userCols = `id, email, password_hash, role, name, country, travel_style, interests, vendor_id, extra, created_at, totp_secret, totp_pending_secret, totp_enabled, token_version, share_location, lat, lng, location_updated_at, phone, vendor_role, notify_redemptions, notify_reviews, notify_payouts, notify_growth_tips, google_id, avatar_url`

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE lower(email)=lower($1)`, email))
}

func (s *Store) UserByID(ctx context.Context, id string) (domain.User, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
}

// UserByGoogleID looks up an account previously linked to a Google
// identity — used by the OAuth callback to find a returning user.
func (s *Store) UserByGoogleID(ctx context.Context, googleID string) (domain.User, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE google_id=$1`, googleID))
}

// LinkUserGoogleID records a Google identity against an existing account
// (matched by email) the first time it signs in with Google.
func (s *Store) LinkUserGoogleID(ctx context.Context, userID, googleID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET google_id=$2 WHERE id=$1`, userID, googleID)
	return err
}

func (s *Store) GetUserByAppleID(ctx context.Context, appleID string) (domain.User, error) {
	row := s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE apple_id = $1`, appleID)
	return scanUser(row)
}

func (s *Store) LinkUserAppleID(ctx context.Context, userID, appleID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET apple_id=$2 WHERE id=$1`, userID, appleID)
	return err
}

// UsersByIDs resolves group member IDs to full profiles (used to show real
// names/emails instead of bare IDs on the trip-invite sheet). Silently
// drops any ID that no longer matches a user rather than erroring — a
// group member who deleted their account shouldn't break the whole group.
func (s *Store) UsersByIDs(ctx context.Context, ids []string) ([]domain.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+userCols+` FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) UpdateUserProfile(ctx context.Context, id, name, country string, style, interests []string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET name=$2, country=$3, travel_style=$4, interests=$5 WHERE id=$1`,
		id, name, country, jsonb(style), jsonb(interests))
	return err
}

// UpdateUserPassword sets a new password hash and bumps token_version, since
// a password change — like a 2FA disable — should invalidate any session
// token issued before it (e.g. one leaked before the change).
func (s *Store) UpdateUserPassword(ctx context.Context, id, passwordHash string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET password_hash=$2, token_version=token_version+1 WHERE id=$1`,
		id, passwordHash)
	return err
}

// SetUserRole changes a user's role and bumps token_version, so tokens
// issued under the old role stop working.
func (s *Store) SetUserRole(ctx context.Context, id string, role domain.Role) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET role=$2, token_version=token_version+1 WHERE id=$1`, id, role)
	return err
}

// LockUsersByEmail makes every account whose email matches the LIKE
// pattern (except keepEmail) impossible to log into. '!locked' is not a
// bcrypt hash, so no password matches it. It also signs those accounts out
// everywhere. Accounts that are already locked are skipped, so it's cheap
// to run at every boot. Returns how many were newly locked.
func (s *Store) LockUsersByEmail(ctx context.Context, pattern, keepEmail string) (int64, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx,
		`UPDATE users SET password_hash='!locked', token_version=token_version+1
		WHERE email LIKE $1 AND lower(email) <> lower($2) AND password_hash <> '!locked'`,
		pattern, keepEmail)
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE user_sessions SET revoked_at=now() WHERE revoked_at IS NULL
		AND user_id IN (SELECT id FROM users WHERE email LIKE $1 AND lower(email) <> lower($2))`,
		pattern, keepEmail); err != nil {
		return 0, err
	}
	return tag.RowsAffected(), tx.Commit(ctx)
}

// SetShareLocation toggles the traveller's nearby-travellers consent. When
// turned off, any previously reported position is cleared immediately
// rather than left stale and merely unqueried.
func (s *Store) SetShareLocation(ctx context.Context, id string, share bool) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET share_location=$2, lat=CASE WHEN $2 THEN lat ELSE NULL END,
			lng=CASE WHEN $2 THEN lng ELSE NULL END,
			location_updated_at=CASE WHEN $2 THEN location_updated_at ELSE NULL END
		WHERE id=$1`,
		id, share)
	return err
}

// UpdateUserLocation records the traveller's current position. Callers
// must check share_location themselves first (see handleUpdateMyLocation) —
// this just writes whatever it's given.
func (s *Store) UpdateUserLocation(ctx context.Context, id string, lat, lng float64) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET lat=$2, lng=$3, location_updated_at=now() WHERE id=$1`,
		id, lat, lng)
	return err
}

// DeleteUser deletes a user record by ID.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	res, err := s.Pool.Exec(ctx, `DELETE FROM users WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// NearbyTraveller is the coarse, privacy-conscious shape returned by
// NearbyTravellers — never another traveller's exact lat/lng, just enough
// to show them as a nearby pin at a rounded distance.
type NearbyTraveller struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	TravelStyle []string `json:"travelStyle"`
	DistanceKm  float64  `json:"distanceKm"`
}

// NearbyTravellers finds other opted-in, recently-active travellers within
// radiusKm of (lat, lng), nearest first. "Recently active" means a location
// report in the last 15 minutes — older reports are treated as stale and
// excluded rather than shown at a wrong position.
func (s *Store) NearbyTravellers(ctx context.Context, callerID string, lat, lng, radiusKm float64) ([]NearbyTraveller, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, name, travel_style, distance_km FROM (
			SELECT id, name, travel_style,
				6371 * acos(LEAST(1, GREATEST(-1,
					cos(radians($1)) * cos(radians(lat)) * cos(radians(lng) - radians($2))
					+ sin(radians($1)) * sin(radians(lat))
				))) AS distance_km
			FROM users
			WHERE share_location = true
				AND id != $3
				AND lat IS NOT NULL AND lng IS NOT NULL
				AND location_updated_at > now() - interval '15 minutes'
		) nearby
		WHERE distance_km <= $4
		ORDER BY distance_km ASC`,
		lat, lng, callerID, radiusKm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []NearbyTraveller{}
	for rows.Next() {
		var t NearbyTraveller
		var travelStyle []byte
		if err := rows.Scan(&t.ID, &t.Name, &travelStyle, &t.DistanceKm); err != nil {
			return nil, err
		}
		json.Unmarshal(travelStyle, &t.TravelStyle)
		out = append(out, t)
	}
	return out, rows.Err()
}

// UpdateUserContact updates the vendor account settings page's editable
// fields. Callers must check email uniqueness themselves first (see
// handleUpdateMyContact) — this just writes whatever it's given.
func (s *Store) UpdateUserContact(ctx context.Context, id, name, email, phone string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET name=$2, email=$3, phone=$4 WHERE id=$1`,
		id, name, email, phone)
	return err
}

// UpdateNotificationPrefs saves the account page's notification toggles.
func (s *Store) UpdateNotificationPrefs(ctx context.Context, id string, redemptions, reviews, payouts, growthTips bool) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET notify_redemptions=$2, notify_reviews=$3, notify_payouts=$4, notify_growth_tips=$5 WHERE id=$1`,
		id, redemptions, reviews, payouts, growthTips)
	return err
}

// VendorTeamMembers lists everyone (owner + staff) currently attached to a
// vendor account, oldest first (so the owner — always created first —
// naturally sorts to the top).
func (s *Store) VendorTeamMembers(ctx context.Context, vendorID string) ([]domain.User, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+userCols+` FROM users WHERE vendor_id=$1 ORDER BY created_at ASC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// RemoveVendorTeamMember detaches a staff or manager member from the
// vendor account (clears vendor_id/role rather than deleting the account
// outright, so their login persists but loses all vendor access) — scoped
// to vendorID and vendor_role<>'owner' so an owner can't be removed this way.
func (s *Store) RemoveVendorTeamMember(ctx context.Context, vendorID, userID string) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE users SET vendor_id=NULL, vendor_role='' WHERE id=$1 AND vendor_id=$2 AND vendor_role<>$3`,
		userID, vendorID, domain.VendorRoleOwner)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CreateVendorInvite stores a freshly generated invite code.
func (s *Store) CreateVendorInvite(ctx context.Context, inv domain.VendorInvite) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendor_invites (id, vendor_id, code, email, role, created_by, created_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		inv.ID, inv.VendorID, inv.Code, inv.Email, inv.Role, inv.CreatedBy, inv.CreatedAt, inv.ExpiresAt)
	return err
}

func scanVendorInvite(row pgx.Row) (domain.VendorInvite, error) {
	var inv domain.VendorInvite
	err := row.Scan(&inv.ID, &inv.VendorID, &inv.Code, &inv.Email, &inv.Role, &inv.CreatedBy, &inv.CreatedAt,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.AcceptedUserID, &inv.Revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return inv, ErrNotFound
	}
	return inv, err
}

const vendorInviteCols = `id, vendor_id, code, email, role, created_by, created_at, expires_at, accepted_at, accepted_user_id, revoked`

func (s *Store) VendorInviteByCode(ctx context.Context, code string) (domain.VendorInvite, error) {
	return scanVendorInvite(s.Pool.QueryRow(ctx, `SELECT `+vendorInviteCols+` FROM vendor_invites WHERE code=$1`, code))
}

// VendorInvites lists still-pending (not accepted, not revoked) invites for
// a vendor — expired-but-otherwise-pending ones are included so the owner
// can see and clean them up, rather than having them vanish silently.
func (s *Store) VendorInvites(ctx context.Context, vendorID string) ([]domain.VendorInvite, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+vendorInviteCols+` FROM vendor_invites WHERE vendor_id=$1 AND accepted_at IS NULL AND revoked=false ORDER BY created_at DESC`,
		vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VendorInvite
	for rows.Next() {
		inv, err := scanVendorInvite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (s *Store) AcceptVendorInvite(ctx context.Context, id, userID string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE vendor_invites SET accepted_at=now(), accepted_user_id=$2 WHERE id=$1`, id, userID)
	return err
}

// RevokeVendorInvite is scoped to vendorID so one vendor's owner can't
// revoke another vendor's invite by guessing an id.
func (s *Store) RevokeVendorInvite(ctx context.Context, id, vendorID string) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE vendor_invites SET revoked=true WHERE id=$1 AND vendor_id=$2`, id, vendorID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CreateVendorProfileGrant stores a freshly generated PIN grant (hash only —
// the raw PIN is never persisted).
func (s *Store) CreateVendorProfileGrant(ctx context.Context, g domain.VendorProfileGrant) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendor_profile_grants (id, vendor_id, grantee_id, created_by, pin_hash, duration_mins, created_at, pin_expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		g.ID, g.VendorID, g.GranteeID, g.CreatedBy, g.PinHash, g.DurationMins, g.CreatedAt, g.PinExpiresAt)
	return err
}

const vendorProfileGrantCols = `id, vendor_id, grantee_id, created_by, pin_hash, duration_mins, created_at, pin_expires_at, redeemed_at, access_expires_at, revoked_at, revoked_by`

func scanVendorProfileGrant(row pgx.Row) (domain.VendorProfileGrant, error) {
	var g domain.VendorProfileGrant
	var revokedBy *string
	err := row.Scan(&g.ID, &g.VendorID, &g.GranteeID, &g.CreatedBy, &g.PinHash, &g.DurationMins, &g.CreatedAt,
		&g.PinExpiresAt, &g.RedeemedAt, &g.AccessExpiresAt, &g.RevokedAt, &revokedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, ErrNotFound
	}
	if revokedBy != nil {
		g.RevokedBy = *revokedBy
	}
	return g, err
}

func (s *Store) VendorProfileGrantByID(ctx context.Context, id string) (domain.VendorProfileGrant, error) {
	return scanVendorProfileGrant(s.Pool.QueryRow(ctx, `SELECT `+vendorProfileGrantCols+` FROM vendor_profile_grants WHERE id=$1`, id))
}

// RedeemVendorProfileGrant marks a grant as redeemed and starts its access
// window (redeemed_at + duration_mins) — scoped to id only; callers must
// have already verified the PIN and ownership before calling this.
func (s *Store) RedeemVendorProfileGrant(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE vendor_profile_grants
		SET redeemed_at=now(), access_expires_at=now() + make_interval(mins => duration_mins)
		WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ActiveVendorProfileGrant reports whether granteeID currently has a
// redeemed, unrevoked, unexpired grant on vendorID — the real authorization
// check used by requireBusinessProfileAccess.
func (s *Store) ActiveVendorProfileGrant(ctx context.Context, vendorID, granteeID string) (bool, error) {
	var exists bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM vendor_profile_grants
			WHERE vendor_id=$1 AND grantee_id=$2 AND revoked_at IS NULL
			  AND access_expires_at IS NOT NULL AND access_expires_at > now()
		)`, vendorID, granteeID).Scan(&exists)
	return exists, err
}

// LatestVendorProfileGrantForUser returns the most recent non-revoked grant
// for a grantee on a vendor (pending-PIN or active), for the Profile page's
// countdown/PIN-entry UI.
func (s *Store) LatestVendorProfileGrantForUser(ctx context.Context, vendorID, granteeID string) (domain.VendorProfileGrant, error) {
	return scanVendorProfileGrant(s.Pool.QueryRow(ctx, `
		SELECT `+vendorProfileGrantCols+` FROM vendor_profile_grants
		WHERE vendor_id=$1 AND grantee_id=$2 AND revoked_at IS NULL
		ORDER BY created_at DESC LIMIT 1`, vendorID, granteeID))
}

// VendorProfileGrants lists all grants for a vendor (any status), newest
// first, so an owner can see and revoke pending or active ones.
func (s *Store) VendorProfileGrants(ctx context.Context, vendorID string) ([]domain.VendorProfileGrant, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+vendorProfileGrantCols+` FROM vendor_profile_grants WHERE vendor_id=$1 ORDER BY created_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VendorProfileGrant
	for rows.Next() {
		g, err := scanVendorProfileGrant(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// RevokeVendorProfileGrant is scoped to vendorID so one vendor's owner can't
// revoke another vendor's grant by guessing an id.
func (s *Store) RevokeVendorProfileGrant(ctx context.Context, id, vendorID, revokedBy string) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE vendor_profile_grants SET revoked_at=now(), revoked_by=$3 WHERE id=$1 AND vendor_id=$2`, id, vendorID, revokedBy)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// AdminVendorGrantView is a joined, display-ready row for the platform
// admin's cross-vendor grant list — status is computed here rather than
// stored, since "active"/"pending"/"expired"/"revoked" all derive from the
// same timestamp columns.
type AdminVendorGrantView struct {
	domain.VendorProfileGrant
	VendorName  string `json:"vendorName"`
	GranteeName string `json:"granteeName"`
	OwnerName   string `json:"ownerName"`
	Status      string `json:"status"`
}

// AdminListVendorProfileGrants lists grants across all vendors, optionally
// filtered by computed status ("active", "pending", or "" for all).
func (s *Store) AdminListVendorProfileGrants(ctx context.Context, status string) ([]AdminVendorGrantView, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT g.id, g.vendor_id, g.grantee_id, g.created_by, g.pin_hash, g.duration_mins, g.created_at,
		       g.pin_expires_at, g.redeemed_at, g.access_expires_at, g.revoked_at, g.revoked_by,
		       v.name, grantee.name, owner.name
		FROM vendor_profile_grants g
		JOIN vendors v ON v.id = g.vendor_id
		JOIN users grantee ON grantee.id = g.grantee_id
		JOIN users owner ON owner.id = g.created_by
		ORDER BY g.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Initialised (not nil) so an empty result serialises to JSON [] rather
	// than null — the admin Access-grants page does grants.filter() and a
	// null crashes it. Matches AdminSearch/AdminNotifications.
	out := []AdminVendorGrantView{}
	for rows.Next() {
		var v AdminVendorGrantView
		var revokedBy *string
		if err := rows.Scan(&v.ID, &v.VendorID, &v.GranteeID, &v.CreatedBy, &v.PinHash, &v.DurationMins, &v.CreatedAt,
			&v.PinExpiresAt, &v.RedeemedAt, &v.AccessExpiresAt, &v.RevokedAt, &revokedBy,
			&v.VendorName, &v.GranteeName, &v.OwnerName); err != nil {
			return nil, err
		}
		if revokedBy != nil {
			v.RevokedBy = *revokedBy
		}
		v.Status = vendorGrantStatus(v.VendorProfileGrant)
		if status == "" || status == v.Status {
			out = append(out, v)
		}
	}
	return out, rows.Err()
}

func vendorGrantStatus(g domain.VendorProfileGrant) string {
	now := time.Now()
	switch {
	case g.RevokedAt != nil:
		return "revoked"
	case g.AccessExpiresAt != nil && g.AccessExpiresAt.After(now):
		return "active"
	case g.RedeemedAt == nil && g.PinExpiresAt.After(now):
		return "pending"
	default:
		return "expired"
	}
}

// AdminAdjustVendorProfileGrant extends (positive) or shortens (negative)
// whichever expiry is currently in play — access_expires_at once redeemed,
// otherwise pin_expires_at — clamped so it can never be pushed before now
// (use revoke for an immediate cutoff instead).
func (s *Store) AdminAdjustVendorProfileGrant(ctx context.Context, id string, minutesDelta int) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE vendor_profile_grants SET
			access_expires_at = CASE WHEN redeemed_at IS NOT NULL
				THEN GREATEST(now(), access_expires_at + make_interval(mins => $2)) END,
			pin_expires_at = CASE WHEN redeemed_at IS NULL
				THEN GREATEST(now(), pin_expires_at + make_interval(mins => $2))
				ELSE pin_expires_at END
		WHERE id=$1`, id, minutesDelta)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ---- password reset ----

// ResetToken is never returned with its raw token — only token_hash is
// stored/compared, same never-store-the-secret pattern as TOTP secrets.
type ResetToken struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

func (s *Store) CreatePasswordResetToken(ctx context.Context, id, userID, tokenHash string, expiresAt time.Time) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at) VALUES ($1,$2,$3,$4)`,
		id, userID, tokenHash, expiresAt)
	return err
}

func (s *Store) PasswordResetTokenByHash(ctx context.Context, tokenHash string) (ResetToken, error) {
	var t ResetToken
	err := s.Pool.QueryRow(ctx,
		`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash=$1`, tokenHash,
	).Scan(&t.ID, &t.UserID, &t.ExpiresAt, &t.UsedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) MarkPasswordResetTokenUsed(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE password_reset_tokens SET used_at=now() WHERE id=$1`, id)
	return err
}

// SetUserTOTPPending stores a freshly generated (not yet active) secret.
func (s *Store) SetUserTOTPPending(ctx context.Context, id, pendingSecret string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET totp_pending_secret=$2 WHERE id=$1`, id, pendingSecret)
	return err
}

// EnableUserTOTP promotes the pending secret to active and clears it.
func (s *Store) EnableUserTOTP(ctx context.Context, id, secret string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET totp_secret=$2, totp_pending_secret='', totp_enabled=true WHERE id=$1`,
		id, secret)
	return err
}

// DisableUserTOTP clears both secrets, turns 2FA off, and bumps
// token_version so any session token issued while 2FA was still active
// (e.g. one leaked before this call) stops being accepted immediately.
func (s *Store) DisableUserTOTP(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET totp_secret='', totp_pending_secret='', totp_enabled=false, token_version=token_version+1 WHERE id=$1`, id)
	return err
}

// UserTokenVersion fetches just the token_version column, for cheap
// per-request revocation checks in Server.Authenticated.
func (s *Store) UserTokenVersion(ctx context.Context, id string) (int, error) {
	var v int
	err := s.Pool.QueryRow(ctx, `SELECT token_version FROM users WHERE id=$1`, id).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	return v, err
}

// ---- vendors ----

const vendorCols = `id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng, br_cert_number, br_cert_ocr_status, br_cert_verify_status, br_cert_company_name, br_cert_message, br_cert_attempts, br_cert_verified_at, google_place_id`

func scanVendor(row pgx.Row) (domain.Vendor, error) {
	var v domain.Vendor
	var flags []byte
	err := row.Scan(&v.ID, &v.Name, &v.Category, &v.Town, &v.Country, &v.RegionID, &v.Owner, &v.Status,
		&v.Priority, &v.Docs, &v.Photos, &v.WifiScore, &flags, &v.Note, &v.Tier,
		&v.Profile, &v.Menu, &v.SubmittedAt, &v.Lat, &v.Lng,
		&v.BRCertNumber, &v.BRCertOCRStatus, &v.BRCertVerifyStatus, &v.BRCertCompanyName,
		&v.BRCertMessage, &v.BRCertAttempts, &v.BRCertVerifiedAt, &v.GooglePlaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	if err != nil {
		return v, err
	}
	json.Unmarshal(flags, &v.Flags)
	return v, nil
}

func (s *Store) UpsertVendor(ctx context.Context, v domain.Vendor) error {
	if v.Country == "" {
		// Matches migration 009's column default — callers that don't know
		// or haven't been updated to set this (seed/gendata scripts, older
		// code paths) shouldn't silently write an empty string instead of
		// the correct default.
		v.Country = "LK"
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendors (id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng,
			br_cert_number, br_cert_ocr_status, br_cert_verify_status, br_cert_company_name, br_cert_message, br_cert_attempts, br_cert_verified_at, google_place_id)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28)
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name, category=EXCLUDED.category, town=EXCLUDED.town, country=EXCLUDED.country,
			region_id=EXCLUDED.region_id, owner=EXCLUDED.owner, status=EXCLUDED.status,
			priority=EXCLUDED.priority, docs=EXCLUDED.docs, photos=EXCLUDED.photos,
			wifi_score=EXCLUDED.wifi_score, flags=EXCLUDED.flags, note=EXCLUDED.note,
			tier=EXCLUDED.tier, profile=EXCLUDED.profile, menu=EXCLUDED.menu,
			submitted_at=EXCLUDED.submitted_at, lat=EXCLUDED.lat, lng=EXCLUDED.lng,
			br_cert_number=EXCLUDED.br_cert_number, br_cert_ocr_status=EXCLUDED.br_cert_ocr_status,
			br_cert_verify_status=EXCLUDED.br_cert_verify_status, br_cert_company_name=EXCLUDED.br_cert_company_name,
			br_cert_message=EXCLUDED.br_cert_message, br_cert_attempts=EXCLUDED.br_cert_attempts,
			br_cert_verified_at=EXCLUDED.br_cert_verified_at, google_place_id=EXCLUDED.google_place_id`,
		v.ID, v.Name, v.Category, v.Town, v.Country, v.RegionID, v.Owner, v.Status, v.Priority,
		v.Docs, v.Photos, v.WifiScore, jsonb(v.Flags), v.Note, v.Tier,
		orEmptyObj(v.Profile), orEmptyObj(v.Menu), v.SubmittedAt, v.Lat, v.Lng,
		v.BRCertNumber, v.BRCertOCRStatus, v.BRCertVerifyStatus, v.BRCertCompanyName,
		v.BRCertMessage, v.BRCertAttempts, v.BRCertVerifiedAt, v.GooglePlaceID)
	return err
}

// VendorsForMap returns every vendor with lat/lng and a computed
// badReviewCount/flagged (>=5 reviews rated 1-2), for the admin geo map.
func (s *Store) VendorsForMap(ctx context.Context) ([]domain.Vendor, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT v.`+vendorColsPrefixed()+`,
			COALESCE(r.bad_count, 0) AS bad_count
		FROM vendors v
		LEFT JOIN (
			SELECT vendor_id, count(*) AS bad_count
			FROM reviews
			WHERE vendor_id IS NOT NULL AND rating <= 2
			GROUP BY vendor_id
		) r ON r.vendor_id = v.id
		WHERE v.lat != 0 OR v.lng != 0
		ORDER BY v.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Vendor
	for rows.Next() {
		v, badCount, err := scanVendorWithBadCount(rows)
		if err != nil {
			return nil, err
		}
		v.BadReviewCount = badCount
		v.Flagged = badCount >= BadReviewFlagThreshold
		out = append(out, v)
	}
	return out, rows.Err()
}

// VendorsNear returns vendors within radiusKm of (lat,lng) using the
// haversine formula — the only proximity query in the app, so a full GIS
// extension isn't warranted.
func (s *Store) VendorsNear(ctx context.Context, lat, lng, radiusKm float64) ([]domain.Vendor, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT v.`+vendorColsPrefixed()+`,
			COALESCE(r.bad_count, 0) AS bad_count
		FROM vendors v
		LEFT JOIN (
			SELECT vendor_id, count(*) AS bad_count
			FROM reviews
			WHERE vendor_id IS NOT NULL AND rating <= 2
			GROUP BY vendor_id
		) r ON r.vendor_id = v.id
		WHERE (v.lat != 0 OR v.lng != 0) AND (
			6371 * acos(least(1.0, greatest(-1.0,
				cos(radians($1)) * cos(radians(v.lat)) * cos(radians(v.lng) - radians($2))
				+ sin(radians($1)) * sin(radians(v.lat))
			)))
		) <= $3
		ORDER BY v.name`, lat, lng, radiusKm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Vendor
	for rows.Next() {
		v, badCount, err := scanVendorWithBadCount(rows)
		if err != nil {
			return nil, err
		}
		v.BadReviewCount = badCount
		v.Flagged = badCount >= BadReviewFlagThreshold
		out = append(out, v)
	}
	return out, rows.Err()
}

func vendorColsPrefixed() string {
	return "id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng, br_cert_number, br_cert_ocr_status, br_cert_verify_status, br_cert_company_name, br_cert_message, br_cert_attempts, br_cert_verified_at, google_place_id"
}

func scanVendorWithBadCount(row pgx.Row) (domain.Vendor, int, error) {
	var v domain.Vendor
	var flags []byte
	var badCount int
	err := row.Scan(&v.ID, &v.Name, &v.Category, &v.Town, &v.Country, &v.RegionID, &v.Owner, &v.Status,
		&v.Priority, &v.Docs, &v.Photos, &v.WifiScore, &flags, &v.Note, &v.Tier,
		&v.Profile, &v.Menu, &v.SubmittedAt, &v.Lat, &v.Lng,
		&v.BRCertNumber, &v.BRCertOCRStatus, &v.BRCertVerifyStatus, &v.BRCertCompanyName,
		&v.BRCertMessage, &v.BRCertAttempts, &v.BRCertVerifiedAt, &v.GooglePlaceID, &badCount)
	if err != nil {
		return v, 0, err
	}
	json.Unmarshal(flags, &v.Flags)
	return v, badCount, nil
}

func orEmptyObj(m json.RawMessage) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage(`{}`)
	}
	return m
}

func (s *Store) VendorByID(ctx context.Context, id string) (domain.Vendor, error) {
	return scanVendor(s.Pool.QueryRow(ctx, `SELECT `+vendorCols+` FROM vendors WHERE id=$1`, id))
}

func (s *Store) Vendors(ctx context.Context, status string) ([]domain.Vendor, error) {
	q := `SELECT ` + vendorCols + ` FROM vendors`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY submitted_at DESC`
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// AdminVendorStats is the per-vendor activity roll-up behind the admin
// Vendors page's monitor pane. Everything here is derived on read — there
// is no stats table — so it always reflects live data, unlike the
// vendor-facing dashboard/growth/reports blobs (see VendorBlob).
type AdminVendorStats struct {
	// Redemptions come from claims: Claimed is every claim raised against
	// the vendor, Redeemed only those actually validated at the counter.
	Claimed     int     `json:"claimed"`
	Redeemed    int     `json:"redeemed"`
	Redeemed30d int     `json:"redeemed30d"`
	Checkins    int     `json:"checkins"`
	Checkins30d int     `json:"checkins30d"`
	Reviews     int     `json:"reviews"`
	AvgRating   float64 `json:"avgRating"`
	BadReviews  int     `json:"badReviews"`
	Flagged     bool    `json:"flagged"`
	Offers      int     `json:"offers"`
	OffersLive  int     `json:"offersLive"`
	OffersDraft int     `json:"offersDraft"`
	// NOTE: feed_items are deliberately not counted here. That table has no
	// vendor_id — only a free-text `vendor` display label that does not
	// reliably equal vendors.name (the seed set pairs "Driftwood Coffee"
	// with a vendor actually named "Driftwood Coffee & Co-Work"), so a
	// per-vendor count would read 0 for almost everyone and look broken
	// rather than empty. Doing this properly needs a feed_items.vendor_id.
	TeamTotal      int `json:"teamTotal"`
	TeamManagers   int `json:"teamManagers"`
	TeamStaff      int `json:"teamStaff"`
	PendingInvites int `json:"pendingInvites"`
	// Subscription fields are nil for a vendor that has never had a
	// subscriptions row (the common case for Free-tier vendors).
	SubTier     *string    `json:"subTier,omitempty"`
	SubMRR      *int       `json:"subMrr,omitempty"`
	SubState    *string    `json:"subState,omitempty"`
	SubRenewsAt *time.Time `json:"subRenewsAt,omitempty"`
}

// VendorStats rolls up one vendor's activity for the admin monitor pane.
// Written as scalar subqueries against a single vendors row so the whole
// pane costs one round trip; every source table is already indexed on
// vendor_id (claims_vendor_idx, reviews_vendor_idx, and the offers FK).
//
// One join is not by vendor_id: checkins carry only a place_id, so they
// reach the vendor through places.vendor_id.
func (s *Store) VendorStats(ctx context.Context, vendorID string) (AdminVendorStats, error) {
	var st AdminVendorStats
	err := s.Pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM claims WHERE vendor_id = v.id),
			(SELECT count(*) FROM claims WHERE vendor_id = v.id AND redeemed_at IS NOT NULL),
			(SELECT count(*) FROM claims WHERE vendor_id = v.id AND redeemed_at >= now() - interval '30 days'),
			(SELECT count(*) FROM checkins c JOIN places p ON p.id = c.place_id WHERE p.vendor_id = v.id),
			(SELECT count(*) FROM checkins c JOIN places p ON p.id = c.place_id
				WHERE p.vendor_id = v.id AND c.created_at >= now() - interval '30 days'),
			(SELECT count(*) FROM reviews WHERE vendor_id = v.id),
			(SELECT COALESCE(avg(rating), 0) FROM reviews WHERE vendor_id = v.id),
			(SELECT count(*) FROM reviews WHERE vendor_id = v.id AND rating <= 2),
			(SELECT count(*) FROM offers WHERE vendor_id = v.id),
			(SELECT count(*) FROM offers WHERE vendor_id = v.id AND status = 'live'),
			(SELECT count(*) FROM offers WHERE vendor_id = v.id AND status = 'draft'),
			(SELECT count(*) FROM users WHERE vendor_id = v.id),
			(SELECT count(*) FROM users WHERE vendor_id = v.id AND vendor_role = 'manager'),
			(SELECT count(*) FROM users WHERE vendor_id = v.id AND vendor_role = 'staff'),
			(SELECT count(*) FROM vendor_invites WHERE vendor_id = v.id
				AND accepted_at IS NULL AND NOT revoked AND expires_at > now()),
			sub.tier, sub.mrr, sub.state, sub.renews_at
		FROM vendors v
		LEFT JOIN LATERAL (
			SELECT tier, mrr, state, renews_at FROM subscriptions WHERE vendor_id = v.id LIMIT 1
		) sub ON true
		WHERE v.id = $1`, vendorID).Scan(
		&st.Claimed, &st.Redeemed, &st.Redeemed30d,
		&st.Checkins, &st.Checkins30d,
		&st.Reviews, &st.AvgRating, &st.BadReviews,
		&st.Offers, &st.OffersLive, &st.OffersDraft,
		&st.TeamTotal, &st.TeamManagers, &st.TeamStaff, &st.PendingInvites,
		&st.SubTier, &st.SubMRR, &st.SubState, &st.SubRenewsAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return st, ErrNotFound
	}
	if err != nil {
		return st, err
	}
	// Same threshold the geo map flags on, so the two views agree.
	st.Flagged = st.BadReviews >= BadReviewFlagThreshold
	return st, nil
}

// AdminVendorSummary is the network-wide vendor roll-up behind the Vendors
// page's aggregate band — the same activity metrics as AdminVendorStats but
// summed across every vendor, plus a status breakdown. Derived on read; no
// stats table.
type AdminVendorSummary struct {
	// Vendor counts: Total plus one field per status the directory shows.
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Approved  int `json:"approved"`
	Edits     int `json:"edits"`
	Suspended int `json:"suspended"`
	Rejected  int `json:"rejected"`
	// Flagged counts vendors (not reviews) at/over BadReviewFlagThreshold,
	// matching the geo map's red-flag rule.
	Flagged     int     `json:"flagged"`
	Redeemed    int     `json:"redeemed"`
	Redeemed30d int     `json:"redeemed30d"`
	Checkins    int     `json:"checkins"`
	Checkins30d int     `json:"checkins30d"`
	Reviews     int     `json:"reviews"`
	AvgRating   float64 `json:"avgRating"`
	BadReviews  int     `json:"badReviews"`
	Offers      int     `json:"offers"`
	OffersLive  int     `json:"offersLive"`
	OffersDraft int     `json:"offersDraft"`
	ActiveSubs  int     `json:"activeSubs"`
	MRR         int     `json:"mrr"`
}

// VendorSummary rolls up every vendor's activity for the Vendors page's
// aggregate band. Like VendorStats it's a single round trip of scalar
// subqueries, but un-scoped (no per-vendor filter). Check-ins reach a vendor
// through places.vendor_id, same as the per-vendor query. Flagged reuses the
// bad-review-count-per-vendor shape from AdminNotifications (store.go ~1768).
func (s *Store) VendorSummary(ctx context.Context) (AdminVendorSummary, error) {
	var sm AdminVendorSummary
	err := s.Pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM vendors),
			(SELECT count(*) FROM vendors WHERE status = 'pending'),
			(SELECT count(*) FROM vendors WHERE status = 'approved'),
			(SELECT count(*) FROM vendors WHERE status = 'edits'),
			(SELECT count(*) FROM vendors WHERE status = 'suspended'),
			(SELECT count(*) FROM vendors WHERE status = 'rejected'),
			(SELECT count(*) FROM (
				SELECT vendor_id FROM reviews
				WHERE vendor_id IS NOT NULL AND rating <= 2
				GROUP BY vendor_id HAVING count(*) >= $1
			) f),
			(SELECT count(*) FROM claims WHERE vendor_id IS NOT NULL AND redeemed_at IS NOT NULL),
			(SELECT count(*) FROM claims WHERE vendor_id IS NOT NULL AND redeemed_at >= now() - interval '30 days'),
			(SELECT count(*) FROM checkins c JOIN places p ON p.id = c.place_id WHERE p.vendor_id IS NOT NULL),
			(SELECT count(*) FROM checkins c JOIN places p ON p.id = c.place_id
				WHERE p.vendor_id IS NOT NULL AND c.created_at >= now() - interval '30 days'),
			(SELECT count(*) FROM reviews WHERE vendor_id IS NOT NULL),
			(SELECT COALESCE(avg(rating), 0) FROM reviews WHERE vendor_id IS NOT NULL),
			(SELECT count(*) FROM reviews WHERE vendor_id IS NOT NULL AND rating <= 2),
			(SELECT count(*) FROM offers),
			(SELECT count(*) FROM offers WHERE status = 'live'),
			(SELECT count(*) FROM offers WHERE status = 'draft'),
			(SELECT count(*) FROM subscriptions WHERE state = 'active'),
			(SELECT COALESCE(sum(mrr), 0) FROM subscriptions WHERE state = 'active')`,
		BadReviewFlagThreshold).Scan(
		&sm.Total, &sm.Pending, &sm.Approved, &sm.Edits, &sm.Suspended, &sm.Rejected,
		&sm.Flagged,
		&sm.Redeemed, &sm.Redeemed30d,
		&sm.Checkins, &sm.Checkins30d,
		&sm.Reviews, &sm.AvgRating, &sm.BadReviews,
		&sm.Offers, &sm.OffersLive, &sm.OffersDraft,
		&sm.ActiveSubs, &sm.MRR)
	if err != nil {
		return sm, err
	}
	return sm, nil
}

func (s *Store) SetVendorStatus(ctx context.Context, id string, status domain.VendorStatus) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE vendors SET status=$2 WHERE id=$1`, id, status)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) UpdateVendorProfile(ctx context.Context, id string, profile json.RawMessage) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET profile=$2 WHERE id=$1`, id, profile)
	return err
}

// UpdateVendorCategory changes the business type shown on the map pin and
// used to gate the Menu tab (see isFoodCategory in the vendor web app) —
// a top-level vendors.category column, not part of the profile JSON blob.
func (s *Store) UpdateVendorCategory(ctx context.Context, id, category string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET category=$2 WHERE id=$1`, id, category)
	return err
}

// UpdateVendorIdentity corrects business name/town/map-pin location —
// otherwise set once at registration (see handleRegister) and never
// touched again by any other handler.
func (s *Store) UpdateVendorIdentity(ctx context.Context, id, name, town string, lat, lng float64) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE vendors SET name=$2, town=$3, lat=$4, lng=$5 WHERE id=$1`, id, name, town, lat, lng)
	return err
}

func (s *Store) UpdateVendorMenu(ctx context.Context, id string, menu json.RawMessage) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET menu=$2 WHERE id=$1`, id, menu)
	return err
}

// ---- media (R2 uploads) ----

const mediaCols = `id, owner_type, owner_id, kind, role, object_key, url, thumb_key, micro_thumb, thumb_status, content_type, size, width, height, uploaded_by, status, sort, created_at`

func scanMedia(row pgx.Row) (domain.Media, error) {
	var m domain.Media
	err := row.Scan(&m.ID, &m.OwnerType, &m.OwnerID, &m.Kind, &m.Role, &m.ObjectKey, &m.URL, &m.ThumbKey, &m.MicroThumb, &m.ThumbStatus,
		&m.ContentType, &m.Size, &m.Width, &m.Height, &m.UploadedBy, &m.Status, &m.Sort, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Store) InsertMedia(ctx context.Context, m domain.Media) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO media (`+mediaCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		m.ID, m.OwnerType, m.OwnerID, m.Kind, m.Role, m.ObjectKey, m.URL, m.ThumbKey, m.MicroThumb, m.ThumbStatus,
		m.ContentType, m.Size, m.Width, m.Height, m.UploadedBy, m.Status, m.Sort, m.CreatedAt)
	return err
}

// MediaPendingThumbnails lists the oldest images still awaiting background
// thumbnail generation, for the thumbnail worker's sweep (see
// internal/http/thumbnail_worker.go). limit caps how much one tick takes on.
func (s *Store) MediaPendingThumbnails(ctx context.Context, limit int) ([]domain.Media, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+mediaCols+` FROM media
		WHERE kind='image' AND status='ready' AND thumb_status='pending'
		ORDER BY created_at ASC LIMIT $1`, limit)
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

// SetMediaThumbnails records a completed background thumbnail pass.
func (s *Store) SetMediaThumbnails(ctx context.Context, id, thumbKey, microThumb string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE media SET thumb_key=$2, micro_thumb=$3, thumb_status='done' WHERE id=$1`,
		id, thumbKey, microThumb)
	return err
}

// SetMediaThumbnailError records a failed thumbnail pass (bad/undecodable
// image, fetch error) — not retried, since a decode failure is almost
// always deterministic rather than transient.
func (s *Store) SetMediaThumbnailError(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE media SET thumb_status='error' WHERE id=$1`, id)
	return err
}

func (s *Store) MediaByID(ctx context.Context, id string) (domain.Media, error) {
	return scanMedia(s.Pool.QueryRow(ctx, `SELECT `+mediaCols+` FROM media WHERE id=$1`, id))
}

// MediaByOwner lists an owner's media, cover first then by sort/newest — the
// order galleries render in.
func (s *Store) MediaByOwner(ctx context.Context, ownerType, ownerID string) ([]domain.Media, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+mediaCols+` FROM media
		WHERE owner_type=$1 AND owner_id=$2 AND status='ready'
		ORDER BY (role='cover') DESC, sort ASC, created_at DESC`, ownerType, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Media{}
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) DeleteMedia(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM media WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ClearOwnerRole removes existing media for an owner+role, used to enforce
// single-value roles (a vendor has one cover, one promo video; a user one
// avatar). Returns the removed rows so the caller can delete their R2
// objects.
func (s *Store) ClearOwnerRole(ctx context.Context, ownerType, ownerID, role string) ([]domain.Media, error) {
	rows, err := s.Pool.Query(ctx,
		`DELETE FROM media WHERE owner_type=$1 AND owner_id=$2 AND role=$3 RETURNING `+mediaCols,
		ownerType, ownerID, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Media{}
	for rows.Next() {
		m, err := scanMedia(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) SetUserAvatar(ctx context.Context, userID, url string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET avatar_url=$2 WHERE id=$1`, userID, url)
	return err
}

// SetVendorDocs flips the vendor's "has verification documents" flag, kept in
// sync as docs are uploaded/removed so the admin's docs indicator is honest.
func (s *Store) SetVendorDocs(ctx context.Context, vendorID string, has bool) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET docs=$2 WHERE id=$1`, vendorID, has)
	return err
}

// ---- BR certificate OCR pre-fill + eROC verification (see
// internal/gemini, internal/eroc, http.br_cert_scheduler.go) ----

// SetVendorBRCertNumber sets the vendor-confirmed/edited certificate
// number and re-queues verification — editing the number always
// invalidates any prior verification result.
func (s *Store) SetVendorBRCertNumber(ctx context.Context, vendorID, certNo string) error {
	verifyStatus := domain.BRCertVerifyNone
	if certNo != "" {
		verifyStatus = domain.BRCertVerifyPending
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE vendors SET br_cert_number=$2, br_cert_verify_status=$3,
			br_cert_company_name='', br_cert_message='', br_cert_attempts=0, br_cert_verified_at=NULL
		WHERE id=$1`, vendorID, certNo, verifyStatus)
	return err
}

// StartVendorBRCertOCR queues a freshly-uploaded br_cert doc for the
// scheduler's OCR sweep — leaves any already-set number/verify state
// untouched (a re-upload doesn't clobber a confirmed number).
func (s *Store) StartVendorBRCertOCR(ctx context.Context, vendorID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET br_cert_ocr_status=$2 WHERE id=$1`, vendorID, domain.BRCertOCRPending)
	return err
}

// ResetVendorBRCertOCR clears OCR status when the br_cert doc is deleted —
// there's nothing left to OCR, but a previously confirmed/verified number
// is preserved.
func (s *Store) ResetVendorBRCertOCR(ctx context.Context, vendorID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET br_cert_ocr_status=$2 WHERE id=$1`, vendorID, domain.BRCertOCRNone)
	return err
}

// SetVendorBRCertOCRResult records a completed OCR pass. candidate may be
// "" (Gemini found nothing plausible) — the number/verify-status columns
// only get filled in if the vendor doesn't already have a number set, so
// OCR never overwrites a value the vendor typed themselves while it ran.
func (s *Store) SetVendorBRCertOCRResult(ctx context.Context, vendorID, candidate string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE vendors SET
			br_cert_ocr_status = $2,
			br_cert_number = CASE WHEN br_cert_number = '' AND $3 <> '' THEN $3 ELSE br_cert_number END,
			br_cert_verify_status = CASE WHEN br_cert_number = '' AND $3 <> '' THEN $4 ELSE br_cert_verify_status END
		WHERE id = $1`, vendorID, domain.BRCertOCRDone, candidate, domain.BRCertVerifyPending)
	return err
}

// SetVendorBRCertOCRError records that the OCR pipeline itself failed
// (network/decode/rate-limit) — distinct from a clean run that simply
// found no number, which is SetVendorBRCertOCRResult with "".
func (s *Store) SetVendorBRCertOCRError(ctx context.Context, vendorID string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET br_cert_ocr_status=$2 WHERE id=$1`, vendorID, domain.BRCertOCRError)
	return err
}

// SetVendorBRCertVerifyResult records a definitive eROC outcome
// (verified or invalid) and resets the retry-attempt counter. verifiedAt
// is only stamped for a "verified" status.
func (s *Store) SetVendorBRCertVerifyResult(ctx context.Context, vendorID string, status domain.BRCertVerifyStatus, companyName, message string) error {
	if status == domain.BRCertVerifyVerified {
		_, err := s.Pool.Exec(ctx, `
			UPDATE vendors SET br_cert_verify_status=$2, br_cert_company_name=$3, br_cert_message=$4,
				br_cert_attempts=0, br_cert_verified_at=now()
			WHERE id=$1`, vendorID, status, companyName, message)
		return err
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE vendors SET br_cert_verify_status=$2, br_cert_company_name=$3, br_cert_message=$4, br_cert_attempts=0
		WHERE id=$1`, vendorID, status, companyName, message)
	return err
}

// SetVendorBRCertVerifyAttempt records a transient verify failure —
// verify_status stays "pending" so the scheduler retries it next sweep,
// up to br_cert_scheduler.go's brCertMaxVerifyAttempts cap.
func (s *Store) SetVendorBRCertVerifyAttempt(ctx context.Context, vendorID string, attempts int, message string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET br_cert_attempts=$2, br_cert_message=$3 WHERE id=$1`, vendorID, attempts, message)
	return err
}

// VendorsPendingBRCertOCR lists vendors whose uploaded br_cert doc hasn't
// been OCR'd yet, for the scheduler's OCR sweep.
func (s *Store) VendorsPendingBRCertOCR(ctx context.Context) ([]domain.Vendor, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+vendorCols+` FROM vendors WHERE br_cert_ocr_status=$1`, domain.BRCertOCRPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// VendorsPendingBRCertVerify lists vendors with a certificate number
// awaiting eROC verification, for the scheduler's verify sweep.
func (s *Store) VendorsPendingBRCertVerify(ctx context.Context) ([]domain.Vendor, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+vendorCols+` FROM vendors WHERE br_cert_verify_status=$1`, domain.BRCertVerifyPending)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Vendor
	for rows.Next() {
		v, err := scanVendor(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ---- vendor blobs (dashboard/growth/reports documents) ----

func (s *Store) UpsertVendorBlob(ctx context.Context, vendorID, kind string, data json.RawMessage) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendor_blobs (vendor_id, kind, data) VALUES ($1,$2,$3)
		ON CONFLICT (vendor_id, kind) DO UPDATE SET data=EXCLUDED.data`,
		vendorID, kind, data)
	return err
}

func (s *Store) VendorBlob(ctx context.Context, vendorID, kind string) (json.RawMessage, error) {
	var data json.RawMessage
	err := s.Pool.QueryRow(ctx,
		`SELECT data FROM vendor_blobs WHERE vendor_id=$1 AND kind=$2`, vendorID, kind).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return data, err
}

// ---- offers ----

const offerCols = `id, vendor_id, title, type, value, status, start_at, end_at, lim, views, updated_at`

// scanOffer does NOT populate Claimed/Redeemed — those are live-computed
// from the claims table (see ClaimCountsByOfferIDs) and merged in by
// OffersByVendor/OfferByID after scanning.
func scanOffer(row pgx.Row) (domain.Offer, error) {
	var o domain.Offer
	err := row.Scan(&o.ID, &o.VendorID, &o.Title, &o.Type, &o.Value, &o.Status,
		&o.StartAt, &o.EndAt, &o.Limit, &o.Views, &o.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

// UpsertOffer never writes Claimed/Redeemed (live-computed, not stored) or
// Views (only IncrementOfferViews touches that column, so an edit doesn't
// reset a offer's real view count).
func (s *Store) UpsertOffer(ctx context.Context, o domain.Offer) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO offers (id, vendor_id, title, type, value, status, start_at, end_at, lim, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9, now())
		ON CONFLICT (id) DO UPDATE SET
			title=EXCLUDED.title, type=EXCLUDED.type, value=EXCLUDED.value,
			status=EXCLUDED.status, start_at=EXCLUDED.start_at, end_at=EXCLUDED.end_at,
			lim=EXCLUDED.lim, updated_at=now()`,
		o.ID, o.VendorID, o.Title, o.Type, o.Value, o.Status, o.StartAt, o.EndAt, o.Limit)
	return err
}

// OfferCounts holds live claimed/redeemed totals for one offer.
type OfferCounts struct {
	Claimed  int
	Redeemed int
}

// ClaimCountsByOfferIDs computes claimed/redeemed directly from the claims
// table (grouped by offer_id) rather than trusting a stored counter,
// avoiding drift/double-counting. Call once per vendor with all of that
// vendor's offer IDs, not once per offer.
func (s *Store) ClaimCountsByOfferIDs(ctx context.Context, offerIDs []string) (map[string]OfferCounts, error) {
	if len(offerIDs) == 0 {
		return map[string]OfferCounts{}, nil
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT offer_id, count(*), count(*) FILTER (WHERE status='redeemed')
		FROM claims WHERE offer_id = ANY($1) GROUP BY offer_id`, offerIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]OfferCounts{}
	for rows.Next() {
		var id string
		var c OfferCounts
		if err := rows.Scan(&id, &c.Claimed, &c.Redeemed); err != nil {
			return nil, err
		}
		out[id] = c
	}
	return out, rows.Err()
}

func (s *Store) OffersByVendor(ctx context.Context, vendorID string) ([]domain.Offer, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+offerCols+` FROM offers WHERE vendor_id=$1 ORDER BY id`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Offer
	for rows.Next() {
		o, err := scanOffer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	ids := make([]string, len(out))
	for i, o := range out {
		ids[i] = o.ID
	}
	counts, err := s.ClaimCountsByOfferIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range out {
		c := counts[out[i].ID]
		out[i].Claimed, out[i].Redeemed = c.Claimed, c.Redeemed
	}
	return out, nil
}

func (s *Store) OfferByID(ctx context.Context, id string) (domain.Offer, error) {
	o, err := scanOffer(s.Pool.QueryRow(ctx, `SELECT `+offerCols+` FROM offers WHERE id=$1`, id))
	if err != nil {
		return o, err
	}
	counts, err := s.ClaimCountsByOfferIDs(ctx, []string{id})
	if err != nil {
		return o, err
	}
	c := counts[id]
	o.Claimed, o.Redeemed = c.Claimed, c.Redeemed
	return o, nil
}

// IncrementOfferViews records one real traveller impression.
func (s *Store) IncrementOfferViews(ctx context.Context, offerID string) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE offers SET views = views + 1 WHERE id=$1`, offerID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) SetOfferStatus(ctx context.Context, id, vendorID string, status domain.OfferStatus) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE offers SET status=$3, updated_at=now() WHERE id=$1 AND vendor_id=$2`, id, vendorID, status)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) DeleteOffer(ctx context.Context, id, vendorID string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM offers WHERE id=$1 AND vendor_id=$2`, id, vendorID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ---- places ----

func (s *Store) UpsertPlace(ctx context.Context, p domain.Place) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO places (id, name, vendor_id, data) VALUES ($1,$2,$3,$4)
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, vendor_id=EXCLUDED.vendor_id, data=EXCLUDED.data`,
		p.ID, p.Name, p.VendorID, orEmptyObj(p.Data))
	return err
}

// placeVisibleWhere hides a vendor's traveller-facing place while that vendor
// is not approved (suspended/rejected/pending/edits) — this is what an
// admin emergency takedown, and an ordinary suspend, rely on to pull a
// listing from the feed. Curated places (no vendor_id) always show.
const placeVisibleWhere = `(p.vendor_id IS NULL OR EXISTS (
	SELECT 1 FROM vendors v WHERE v.id = p.vendor_id AND v.status = 'approved'))`

func (s *Store) Places(ctx context.Context) ([]domain.Place, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT p.id, p.name, p.vendor_id, p.data, p.embedding FROM places p WHERE `+placeVisibleWhere+` ORDER BY p.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Place
	for rows.Next() {
		var p domain.Place
		var embedding []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.VendorID, &p.Data, &embedding); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(embedding, &p.Embedding)
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) PlaceByID(ctx context.Context, id string) (domain.Place, error) {
	var p domain.Place
	var embedding []byte
	err := s.Pool.QueryRow(ctx,
		`SELECT p.id, p.name, p.vendor_id, p.data, p.embedding FROM places p WHERE p.id=$1 AND `+placeVisibleWhere, id).
		Scan(&p.ID, &p.Name, &p.VendorID, &p.Data, &embedding)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	_ = json.Unmarshal(embedding, &p.Embedding)
	return p, nil
}

// PlaceByName resolves a free-text place name (e.g. "Ella") to the best-
// matching place — an exact case-insensitive match wins if one exists,
// otherwise the shortest substring match (so "Ella" beats "Ella Rock" when
// both match but neither is exact). Used by Local Guide's get_weather tool
// to answer questions about somewhere other than the traveller's own
// location — sri_lanka_spots.json's imported landmarks/towns already give
// this real coverage without needing a separate geocoding API.
func (s *Store) PlaceByName(ctx context.Context, name string) (domain.Place, error) {
	var p domain.Place
	var embedding []byte
	err := s.Pool.QueryRow(ctx, `
		SELECT p.id, p.name, p.vendor_id, p.data, p.embedding FROM places p
		WHERE p.name ILIKE $1 AND `+placeVisibleWhere+`
		ORDER BY (lower(p.name) = lower($2)) DESC, length(p.name) ASC
		LIMIT 1`, "%"+name+"%", name).
		Scan(&p.ID, &p.Name, &p.VendorID, &p.Data, &embedding)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	if err != nil {
		return p, err
	}
	_ = json.Unmarshal(embedding, &p.Embedding)
	return p, nil
}

// PlacesMissingEmbedding returns up to limit places whose semantic-search
// embedding hasn't been computed yet, or was computed with a different
// model than the current one (vectors from different models aren't
// comparable) — used by the background embedding worker
// (internal/http/embedding_worker.go) to backfill in small batches.
func (s *Store) PlacesMissingEmbedding(ctx context.Context, model string, limit int) ([]domain.Place, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, name, vendor_id, data FROM places
		 WHERE embedding IS NULL OR embedding_model IS DISTINCT FROM $1
		 LIMIT $2`, model, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Place
	for rows.Next() {
		var p domain.Place
		if err := rows.Scan(&p.ID, &p.Name, &p.VendorID, &p.Data); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// SetPlaceEmbedding stores a place's semantic-search embedding and the
// model that produced it.
func (s *Store) SetPlaceEmbedding(ctx context.Context, id string, embedding []float64, model string) error {
	b, err := json.Marshal(embedding)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `UPDATE places SET embedding=$2, embedding_model=$3 WHERE id=$1`, id, b, model)
	return err
}

// PlaceByVendorID looks up the (at most one, enforced by a partial unique
// index) Place row for a vendor — used by syncVendorPlace to find or
// lazily create the vendor's traveller-facing listing.
func (s *Store) PlaceByVendorID(ctx context.Context, vendorID string) (domain.Place, error) {
	var p domain.Place
	err := s.Pool.QueryRow(ctx,
		`SELECT id, name, vendor_id, data FROM places WHERE vendor_id=$1`, vendorID).
		Scan(&p.ID, &p.Name, &p.VendorID, &p.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// ---- claims ----

// claimCols joins traveller display fields for vendor-facing views.
const claimCols = `c.id, c.code, c.place_id, c.offer_id, c.offer_label, c.offer_value, c.vendor_id, c.traveller_id, c.status, c.claimed_at, c.redeemed_at,
	COALESCE(u.name,''), COALESCE(u.country,''), COALESCE(u.travel_style,'[]'::jsonb)`
const claimFrom = ` FROM claims c LEFT JOIN users u ON u.id = c.traveller_id `

func scanClaim(row pgx.Row) (domain.Claim, error) {
	var c domain.Claim
	var travelStyle []byte
	err := row.Scan(&c.ID, &c.Code, &c.PlaceID, &c.OfferID, &c.OfferLabel, &c.OfferValue,
		&c.VendorID, &c.TravellerID, &c.Status, &c.ClaimedAt, &c.RedeemedAt,
		&c.TravellerName, &c.TravellerCountry, &travelStyle)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	json.Unmarshal(travelStyle, &c.TravellerStyle)
	return c, nil
}

func (s *Store) CreateClaim(ctx context.Context, c domain.Claim) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO claims (id, code, place_id, offer_id, offer_label, offer_value, vendor_id, traveller_id, status, claimed_at, redeemed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		c.ID, c.Code, c.PlaceID, c.OfferID, c.OfferLabel, c.OfferValue, c.VendorID, c.TravellerID,
		c.Status, c.ClaimedAt, c.RedeemedAt)
	return err
}

// ClaimsByOfferID lists claims linked to one vendor Offer, most recent
// first — drives the vendor portal's offer detail drawer. Claims made
// before the offer_id column existed (or where the submitted offerId
// didn't resolve) never match here, by design.
func (s *Store) ClaimsByOfferID(ctx context.Context, offerID string) ([]domain.Claim, error) {
	return s.claimList(ctx,
		`SELECT `+claimCols+claimFrom+`WHERE c.offer_id=$1 ORDER BY c.claimed_at DESC`, offerID)
}

func (s *Store) ClaimByCode(ctx context.Context, code string) (domain.Claim, error) {
	return scanClaim(s.Pool.QueryRow(ctx,
		`SELECT `+claimCols+claimFrom+`WHERE upper(c.code)=upper($1)`, code))
}

func (s *Store) ClaimsByTraveller(ctx context.Context, travellerID string) ([]domain.Claim, error) {
	return s.claimList(ctx,
		`SELECT `+claimCols+claimFrom+`WHERE c.traveller_id=$1 ORDER BY c.claimed_at DESC`, travellerID)
}

// ClaimForTravellerPlace supports idempotent claiming per traveller+place.
func (s *Store) ClaimForTravellerPlace(ctx context.Context, travellerID, placeID string) (domain.Claim, error) {
	return scanClaim(s.Pool.QueryRow(ctx,
		`SELECT `+claimCols+claimFrom+`WHERE c.traveller_id=$1 AND c.place_id=$2 ORDER BY c.claimed_at DESC LIMIT 1`,
		travellerID, placeID))
}

func (s *Store) RedeemClaim(ctx context.Context, id string) (domain.Claim, error) {
	return scanClaim(s.Pool.QueryRow(ctx, `
		WITH upd AS (
			UPDATE claims SET status='redeemed', redeemed_at=now()
			WHERE id=$1 AND status='claimed'
			RETURNING *
		)
		SELECT `+claimCols+` FROM upd c LEFT JOIN users u ON u.id = c.traveller_id`, id))
}

func (s *Store) RedemptionsByVendor(ctx context.Context, vendorID string, after *string) ([]domain.Claim, error) {
	q := `SELECT ` + claimCols + claimFrom + `WHERE c.vendor_id=$1 AND c.status='redeemed'`
	args := []any{vendorID}
	if after != nil && *after != "" {
		q += ` AND c.redeemed_at > $2`
		args = append(args, *after)
	}
	q += ` ORDER BY c.redeemed_at DESC LIMIT 50`
	return s.claimList(ctx, q, args...)
}

func (s *Store) claimList(ctx context.Context, q string, args ...any) ([]domain.Claim, error) {
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Claim
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ---- bookable_services ----

// ErrServiceInactive/ErrBookingFull are returned by CreateBooking — see
// its doc comment for when each applies.
var ErrServiceInactive = errors.New("service inactive")
var ErrBookingFull = errors.New("no capacity for requested time")

const serviceCols = `id, vendor_id, type, name, description, duration_mins, capacity, party_size_min, party_size_max, guests_per_room, price_label, availability, active, created_at, updated_at`

func scanBookableService(row pgx.Row) (domain.BookableService, error) {
	var svc domain.BookableService
	var availability []byte
	err := row.Scan(&svc.ID, &svc.VendorID, &svc.Type, &svc.Name, &svc.Description,
		&svc.DurationMins, &svc.Capacity, &svc.PartySizeMin, &svc.PartySizeMax, &svc.GuestsPerRoom,
		&svc.PriceLabel, &availability, &svc.Active, &svc.CreatedAt, &svc.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return svc, ErrNotFound
	}
	if err != nil {
		return svc, err
	}
	if len(availability) > 0 {
		var doc domain.ServiceAvailabilityDoc
		if json.Unmarshal(availability, &doc) == nil {
			svc.Availability = &doc
		}
	}
	return svc, nil
}

// UpsertBookableService never writes CreatedAt (only the initial INSERT's
// default now() sets it) — same convention as UpsertOffer not touching
// live-computed/append-only columns.
func (s *Store) UpsertBookableService(ctx context.Context, svc domain.BookableService) error {
	var availability []byte
	if svc.Availability != nil {
		availability = jsonb(*svc.Availability)
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO bookable_services (id, vendor_id, type, name, description, duration_mins, capacity, party_size_min, party_size_max, guests_per_room, price_label, availability, active, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13, now())
		ON CONFLICT (id) DO UPDATE SET
			type=EXCLUDED.type, name=EXCLUDED.name, description=EXCLUDED.description,
			duration_mins=EXCLUDED.duration_mins, capacity=EXCLUDED.capacity,
			party_size_min=EXCLUDED.party_size_min, party_size_max=EXCLUDED.party_size_max,
			guests_per_room=EXCLUDED.guests_per_room, price_label=EXCLUDED.price_label,
			availability=EXCLUDED.availability, active=EXCLUDED.active, updated_at=now()`,
		svc.ID, svc.VendorID, svc.Type, svc.Name, svc.Description, svc.DurationMins, svc.Capacity,
		svc.PartySizeMin, svc.PartySizeMax, svc.GuestsPerRoom, svc.PriceLabel, availability, svc.Active)
	return err
}

func (s *Store) BookableServicesByVendor(ctx context.Context, vendorID string, activeOnly bool) ([]domain.BookableService, error) {
	q := `SELECT ` + serviceCols + ` FROM bookable_services WHERE vendor_id=$1`
	if activeOnly {
		q += ` AND active=true`
	}
	q += ` ORDER BY name`
	rows, err := s.Pool.Query(ctx, q, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.BookableService
	for rows.Next() {
		svc, err := scanBookableService(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, svc)
	}
	return out, rows.Err()
}

func (s *Store) BookableServiceByID(ctx context.Context, id string) (domain.BookableService, error) {
	return scanBookableService(s.Pool.QueryRow(ctx, `SELECT `+serviceCols+` FROM bookable_services WHERE id=$1`, id))
}

// DeleteBookableService cascades to that service's bookings (ON DELETE
// CASCADE) — acceptable for this phase; a vendor should set active=false
// instead of deleting a service with real booking history.
func (s *Store) DeleteBookableService(ctx context.Context, id, vendorID string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM bookable_services WHERE id=$1 AND vendor_id=$2`, id, vendorID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ---- bookings ----

const bookingCols = `b.id, b.code, b.vendor_id, b.service_id, b.type, b.place_id, b.traveller_id,
	b.start_at, b.end_at, b.party_size, b.status, b.note, b.created_at,
	b.cancelled_at, b.cancelled_by, b.checked_in_at, b.reminder_sent_at,
	COALESCE(sv.name,''), COALESCE(u.name,''), COALESCE(u.country,''), COALESCE(u.email,''), COALESCE(v.name,'')`
const bookingFrom = ` FROM bookings b
	LEFT JOIN bookable_services sv ON sv.id = b.service_id
	LEFT JOIN users u ON u.id = b.traveller_id
	LEFT JOIN vendors v ON v.id = b.vendor_id `

func scanBooking(row pgx.Row) (domain.Booking, error) {
	var b domain.Booking
	err := row.Scan(&b.ID, &b.Code, &b.VendorID, &b.ServiceID, &b.Type, &b.PlaceID, &b.TravellerID,
		&b.StartAt, &b.EndAt, &b.PartySize, &b.Status, &b.Note, &b.CreatedAt,
		&b.CancelledAt, &b.CancelledBy, &b.CheckedInAt, &b.ReminderSentAt,
		&b.ServiceName, &b.TravellerName, &b.TravellerCountry, &b.TravellerEmail, &b.VendorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, ErrNotFound
	}
	return b, err
}

func (s *Store) bookingList(ctx context.Context, q string, args ...any) ([]domain.Booking, error) {
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Booking
	for rows.Next() {
		b, err := scanBooking(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// OverlappingBookingsCount is the read-only version of the capacity check,
// used by the traveller-facing availability endpoint (not the atomic write
// path — see CreateBooking for that).
func (s *Store) OverlappingBookingsCount(ctx context.Context, serviceID string, startAt, endAt time.Time) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `
		SELECT count(*) FROM bookings
		WHERE service_id=$1 AND status='confirmed' AND start_at < $3 AND end_at > $2`,
		serviceID, startAt, endAt).Scan(&n)
	return n, err
}

// CreateBooking atomically checks capacity and inserts. A capacity check
// needs to serialize *multiple* concurrent inserts against the same
// service — done by locking the bookable_services row first (SELECT ...
// FOR UPDATE), then counting overlaps and inserting inside the same
// transaction. A second concurrent call for the same service blocks on the
// row lock until the first commits, then re-reads a consistent count — no
// double-booking race. Unlike RedeemClaim's single UPDATE...WHERE
// status='claimed' (safe because it's one row keyed by primary key), this
// needs the row lock because it's checking an aggregate over many rows.
func (s *Store) CreateBooking(ctx context.Context, b domain.Booking) (domain.Booking, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return b, err
	}
	defer tx.Rollback(ctx)

	var capacity int
	var active bool
	err = tx.QueryRow(ctx, `SELECT capacity, active FROM bookable_services WHERE id=$1 FOR UPDATE`, b.ServiceID).
		Scan(&capacity, &active)
	if errors.Is(err, pgx.ErrNoRows) {
		return b, ErrNotFound
	}
	if err != nil {
		return b, err
	}
	if !active {
		return b, ErrServiceInactive
	}

	var existing int
	if err := tx.QueryRow(ctx, `
		SELECT count(*) FROM bookings
		WHERE service_id=$1 AND status='confirmed' AND start_at < $3 AND end_at > $2`,
		b.ServiceID, b.StartAt, b.EndAt).Scan(&existing); err != nil {
		return b, err
	}
	if existing >= capacity {
		return b, ErrBookingFull
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO bookings (id, code, vendor_id, service_id, type, place_id, traveller_id, start_at, end_at, party_size, status, note)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'confirmed',$11)`,
		b.ID, b.Code, b.VendorID, b.ServiceID, b.Type, b.PlaceID, b.TravellerID, b.StartAt, b.EndAt, b.PartySize, b.Note); err != nil {
		return b, err
	}
	if err := tx.Commit(ctx); err != nil {
		return b, err
	}
	b.Status, b.CreatedAt = domain.BookingConfirmed, time.Now()
	return b, nil
}

func (s *Store) BookingsByVendor(ctx context.Context, vendorID string) ([]domain.Booking, error) {
	return s.bookingList(ctx, `SELECT `+bookingCols+bookingFrom+`WHERE b.vendor_id=$1 ORDER BY b.start_at DESC`, vendorID)
}

func (s *Store) BookingsByTraveller(ctx context.Context, travellerID string) ([]domain.Booking, error) {
	return s.bookingList(ctx, `SELECT `+bookingCols+bookingFrom+`WHERE b.traveller_id=$1 ORDER BY b.start_at DESC`, travellerID)
}

func (s *Store) BookingByID(ctx context.Context, id string) (domain.Booking, error) {
	return scanBooking(s.Pool.QueryRow(ctx, `SELECT `+bookingCols+bookingFrom+`WHERE b.id=$1`, id))
}

func (s *Store) BookingByCode(ctx context.Context, code string) (domain.Booking, error) {
	return scanBooking(s.Pool.QueryRow(ctx, `SELECT `+bookingCols+bookingFrom+`WHERE upper(b.code)=upper($1)`, code))
}

func (s *Store) CancelBookingAsTraveller(ctx context.Context, id, travellerID string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE bookings SET status='cancelled', cancelled_at=now(), cancelled_by='traveller'
		WHERE id=$1 AND traveller_id=$2 AND status='confirmed'`, id, travellerID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) CancelBookingAsVendor(ctx context.Context, id, vendorID string) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE bookings SET status='cancelled', cancelled_at=now(), cancelled_by='vendor'
		WHERE id=$1 AND vendor_id=$2 AND status='confirmed'`, id, vendorID)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// SetBookingStatus lets a vendor mark a booking no_show/completed/cancelled
// by hand — covers phone bookings or a no-show whose code was never
// scanned. Mirrors SetOfferStatus exactly.
func (s *Store) SetBookingStatus(ctx context.Context, id, vendorID string, status domain.BookingStatus) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE bookings SET status=$3 WHERE id=$1 AND vendor_id=$2`, id, vendorID, status)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CheckInBooking mirrors RedeemClaim's atomic CTE exactly (status flip
// guarded by the WHERE clause, race-safe by construction).
func (s *Store) CheckInBooking(ctx context.Context, id string) (domain.Booking, error) {
	return scanBooking(s.Pool.QueryRow(ctx, `
		WITH upd AS (
			UPDATE bookings SET checked_in_at=now(), status='completed'
			WHERE id=$1 AND status='confirmed'
			RETURNING *
		)
		SELECT `+bookingCols+`
		FROM upd b
		LEFT JOIN bookable_services sv ON sv.id = b.service_id
		LEFT JOIN users u ON u.id = b.traveller_id
		LEFT JOIN vendors v ON v.id = b.vendor_id`, id))
}

// BookingsForReminder returns confirmed bookings starting within
// [windowStart, windowEnd) that haven't had a reminder sent yet — drives
// booking_reminder_scheduler.go. reminder_sent_at IS NULL is the idempotency
// guard: a re-run only ever picks up rows not yet marked sent.
func (s *Store) BookingsForReminder(ctx context.Context, windowStart, windowEnd time.Time) ([]domain.Booking, error) {
	return s.bookingList(ctx, `
		SELECT `+bookingCols+bookingFrom+`
		WHERE b.status='confirmed' AND b.reminder_sent_at IS NULL
		AND b.start_at >= $1 AND b.start_at < $2`, windowStart, windowEnd)
}

func (s *Store) MarkBookingReminderSent(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE bookings SET reminder_sent_at=now() WHERE id=$1`, id)
	return err
}

// ---- trips / groups ----

func (s *Store) UpsertTrip(ctx context.Context, t domain.Trip) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO trips (owner_id, name, code, start_date, collab, days, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,now())
		ON CONFLICT (owner_id) DO UPDATE SET
			name=EXCLUDED.name, code=EXCLUDED.code, start_date=EXCLUDED.start_date,
			collab=EXCLUDED.collab, days=EXCLUDED.days, updated_at=now()`,
		t.OwnerID, t.Name, t.Code, t.StartDate, jsonb(t.Collab), orEmptyArr(t.Days))
	return err
}

func orEmptyArr(m json.RawMessage) json.RawMessage {
	if len(m) == 0 {
		return json.RawMessage(`[]`)
	}
	return m
}

func (s *Store) TripByOwner(ctx context.Context, ownerID string) (domain.Trip, error) {
	var t domain.Trip
	var collab []byte
	err := s.Pool.QueryRow(ctx,
		`SELECT owner_id, name, code, start_date, collab, days, updated_at FROM trips WHERE owner_id=$1`,
		ownerID).Scan(&t.OwnerID, &t.Name, &t.Code, &t.StartDate, &collab, &t.Days, &t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	if err != nil {
		return t, err
	}
	json.Unmarshal(collab, &t.Collab)
	return t, nil
}

func (s *Store) UpsertGroup(ctx context.Context, g domain.Group) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO groups (owner_id, name, code, member_ids) VALUES ($1,$2,$3,$4)
		ON CONFLICT (owner_id) DO UPDATE SET
			name=EXCLUDED.name, code=EXCLUDED.code, member_ids=EXCLUDED.member_ids`,
		g.OwnerID, g.Name, g.Code, jsonb(g.MemberIDs))
	return err
}

// IsGroupParticipant reports whether userID owns, or is listed as a member
// of, a group with this code.
func (s *Store) IsGroupParticipant(ctx context.Context, code, userID string) (bool, error) {
	var ok bool
	err := s.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM groups WHERE code=$1 AND (owner_id=$2 OR member_ids ? $2))`,
		code, userID).Scan(&ok)
	return ok, err
}

func (s *Store) GroupByOwner(ctx context.Context, ownerID string) (domain.Group, error) {
	var g domain.Group
	var members []byte
	err := s.Pool.QueryRow(ctx,
		`SELECT owner_id, name, code, member_ids FROM groups WHERE owner_id=$1`, ownerID).
		Scan(&g.OwnerID, &g.Name, &g.Code, &members)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, ErrNotFound
	}
	if err != nil {
		return g, err
	}
	json.Unmarshal(members, &g.MemberIDs)
	return g, nil
}

// ---- feed ----

func (s *Store) InsertFeedItem(ctx context.Context, f domain.FeedItem) (int64, error) {
	var id int64
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO feed_items (vendor, town, type, text, status, reason, resolved, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		f.Vendor, f.Town, f.Type, f.Text, f.Status, f.Reason, f.Resolved, f.CreatedAt).Scan(&id)
	return id, err
}

func (s *Store) FeedItems(ctx context.Context, afterID int64, status string, limit int) ([]domain.FeedItem, error) {
	q := `SELECT id, vendor, town, type, text, status, reason, resolved, created_at FROM feed_items WHERE id > $1`
	args := []any{afterID}
	if status != "" {
		q += ` AND status=$2`
		args = append(args, status)
	}
	q += fmt.Sprintf(` ORDER BY id DESC LIMIT %d`, limit)
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.FeedItem
	for rows.Next() {
		var f domain.FeedItem
		if err := rows.Scan(&f.ID, &f.Vendor, &f.Town, &f.Type, &f.Text, &f.Status,
			&f.Reason, &f.Resolved, &f.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) ResolveFeedItem(ctx context.Context, id int64, resolved string) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE feed_items SET resolved=$2, status='ok' WHERE id=$1`, id, resolved)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// PruneFeed keeps the newest max rows, preferring to drop resolved items first.
func (s *Store) PruneFeed(ctx context.Context, max int) error {
	_, err := s.Pool.Exec(ctx, `
		DELETE FROM feed_items WHERE id NOT IN (
			SELECT id FROM feed_items ORDER BY (resolved = '') DESC, id DESC LIMIT $1
		)`, max)
	return err
}

// ---- regions / subscriptions / audio guides ----

func (s *Store) UpsertRegion(ctx context.Context, r domain.Region) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO regions (id, name, x, y, lat, lng, travelers, vendors) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, x=EXCLUDED.x, y=EXCLUDED.y,
			lat=EXCLUDED.lat, lng=EXCLUDED.lng,
			travelers=EXCLUDED.travelers, vendors=EXCLUDED.vendors`,
		r.ID, r.Name, r.X, r.Y, r.Lat, r.Lng, r.Travelers, r.Vendors)
	return err
}

func (s *Store) Regions(ctx context.Context) ([]domain.Region, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, x, y, lat, lng, travelers, vendors FROM regions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Region
	for rows.Next() {
		var r domain.Region
		if err := rows.Scan(&r.ID, &r.Name, &r.X, &r.Y, &r.Lat, &r.Lng, &r.Travelers, &r.Vendors); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) UpsertSubscription(ctx context.Context, sub domain.Subscription) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO subscriptions (id, vendor_id, vendor, town, tier, mrr, renews_at, state, since)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (id) DO UPDATE SET vendor_id=EXCLUDED.vendor_id, vendor=EXCLUDED.vendor,
			town=EXCLUDED.town, tier=EXCLUDED.tier, mrr=EXCLUDED.mrr,
			renews_at=EXCLUDED.renews_at, state=EXCLUDED.state, since=EXCLUDED.since`,
		sub.ID, sub.VendorID, sub.Vendor, sub.Town, sub.Tier, sub.MRR, sub.RenewsAt, sub.State, sub.Since)
	return err
}

func (s *Store) Subscriptions(ctx context.Context) ([]domain.Subscription, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, vendor_id, vendor, town, tier, mrr, renews_at, state, since FROM subscriptions ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		if err := rows.Scan(&sub.ID, &sub.VendorID, &sub.Vendor, &sub.Town, &sub.Tier,
			&sub.MRR, &sub.RenewsAt, &sub.State, &sub.Since); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

func (s *Store) UpgradeSubscription(ctx context.Context, id string, mrr int) error {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE subscriptions SET tier='Premium', state='active', mrr=$2,
			renews_at=now() + interval '30 days'
		WHERE id=$1`, id, mrr)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

func (s *Store) UpsertAudioGuide(ctx context.Context, g domain.AudioGuide) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO audio_guides (id, site, region, lat, lng, lang, published, plays, depths, duration)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET site=EXCLUDED.site, region=EXCLUDED.region,
			lat=EXCLUDED.lat, lng=EXCLUDED.lng, lang=EXCLUDED.lang,
			published=EXCLUDED.published, plays=EXCLUDED.plays,
			depths=EXCLUDED.depths, duration=EXCLUDED.duration`,
		g.ID, g.Site, g.Region, g.Lat, g.Lng, g.Lang, g.Published, g.Plays,
		orEmptyArr(g.Depths), orEmptyObj(g.Duration))
	return err
}

func (s *Store) AudioGuides(ctx context.Context) ([]domain.AudioGuide, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, site, region, lat, lng, lang, published, plays, depths, duration FROM audio_guides ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AudioGuide
	for rows.Next() {
		var g domain.AudioGuide
		if err := rows.Scan(&g.ID, &g.Site, &g.Region, &g.Lat, &g.Lng, &g.Lang,
			&g.Published, &g.Plays, &g.Depths, &g.Duration); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) SetAudioGuidePublished(ctx context.Context, id string, published bool) error {
	tag, err := s.Pool.Exec(ctx, `UPDATE audio_guides SET published=$2 WHERE id=$1`, id, published)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// ---- reviews ----

const reviewCols = `r.id, r.place_id, r.vendor_id, r.traveller_id, r.rating, r.text, r.created_at, COALESCE(u.name,'')`
const reviewFrom = ` FROM reviews r LEFT JOIN users u ON u.id = r.traveller_id `

func scanReview(row pgx.Row) (domain.Review, error) {
	var rv domain.Review
	err := row.Scan(&rv.ID, &rv.PlaceID, &rv.VendorID, &rv.TravellerID, &rv.Rating, &rv.Text,
		&rv.CreatedAt, &rv.TravellerName)
	if errors.Is(err, pgx.ErrNoRows) {
		return rv, ErrNotFound
	}
	return rv, err
}

// ReviewIDForTravellerPlace returns the stored review id for a traveller's
// review of a place — used after UpsertReview (whose ON CONFLICT keeps the
// original id) so callers get the real id to attach photos to.
func (s *Store) ReviewIDForTravellerPlace(ctx context.Context, travellerID, placeID string) (string, error) {
	var id string
	err := s.Pool.QueryRow(ctx,
		`SELECT id FROM reviews WHERE place_id=$1 AND traveller_id=$2`, placeID, travellerID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return id, err
}

// ReviewPhotos batch-loads the attached photos (url + thumb key) for a set
// of reviews, keyed by review id — one query for a whole reviews list (no
// N+1). Returned URL/ThumbKey are unsigned/unresolved — handlePlaceReviews
// signs URL and resolves+signs ThumbKey into ThumbURL, same as
// signMediaList does for every other media listing endpoint.
func (s *Store) ReviewPhotos(ctx context.Context, reviewIDs []string) (map[string][]domain.ReviewPhoto, error) {
	out := map[string][]domain.ReviewPhoto{}
	if len(reviewIDs) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx,
		`SELECT owner_id, url, thumb_key, micro_thumb FROM media
		 WHERE owner_type='review' AND role='review' AND status='ready' AND owner_id = ANY($1)
		 ORDER BY created_at ASC`, reviewIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ownerID string
		var p domain.ReviewPhoto
		if err := rows.Scan(&ownerID, &p.URL, &p.ThumbKey, &p.MicroThumb); err != nil {
			return nil, err
		}
		out[ownerID] = append(out[ownerID], p)
	}
	return out, rows.Err()
}

// UpsertReview is idempotent per traveller+place (one review each, editable).
func (s *Store) UpsertReview(ctx context.Context, rv domain.Review) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO reviews (id, place_id, vendor_id, traveller_id, rating, text, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		ON CONFLICT (place_id, traveller_id) DO UPDATE SET
			rating=EXCLUDED.rating, text=EXCLUDED.text, created_at=EXCLUDED.created_at`,
		rv.ID, rv.PlaceID, rv.VendorID, rv.TravellerID, rv.Rating, rv.Text, rv.CreatedAt)
	return err
}

// DeleteReview deletes a review by ID (e.g. for content moderation).
func (s *Store) DeleteReview(ctx context.Context, id string) error {
	res, err := s.Pool.Exec(ctx, `DELETE FROM reviews WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ReviewsByPlace(ctx context.Context, placeID string) ([]domain.Review, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+reviewCols+reviewFrom+`WHERE r.place_id=$1 ORDER BY r.created_at DESC`, placeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Review
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

// ReviewsByVendorBad returns only the low (<=2) ratings for a vendor — the
// ones behind an admin-map flag.
func (s *Store) ReviewsByVendorBad(ctx context.Context, vendorID string) ([]domain.Review, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+reviewCols+reviewFrom+`WHERE r.vendor_id=$1 AND r.rating <= 2 ORDER BY r.created_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Review
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

// ReviewByID fetches one review — used to verify the author before letting a
// traveller attach photos to it.
func (s *Store) ReviewByID(ctx context.Context, id string) (domain.Review, error) {
	return scanReview(s.Pool.QueryRow(ctx, `SELECT `+reviewCols+reviewFrom+`WHERE r.id=$1`, id))
}

// ReviewsByVendor lists every review for a vendor (all ratings), newest
// first — the admin vendor-detail Reviews tab. ReviewsByVendorBad stays the
// geo map's flag source.
func (s *Store) ReviewsByVendor(ctx context.Context, vendorID string) ([]domain.Review, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT `+reviewCols+reviewFrom+`WHERE r.vendor_id=$1 ORDER BY r.created_at DESC`, vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Review
	for rows.Next() {
		rv, err := scanReview(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rv)
	}
	return out, rows.Err()
}

// VoidClaim deletes or voids a claim record by ID for admin correction.
func (s *Store) VoidClaim(ctx context.Context, claimID string) error {
	res, err := s.Pool.Exec(ctx, `DELETE FROM claims WHERE id=$1`, claimID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}


// TripsByOwner retrieves all trips owned by a traveler.
func (s *Store) TripsByOwner(ctx context.Context, ownerID string) ([]domain.Trip, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT name, code, start_date, collab, days FROM trips WHERE owner_id=$1 ORDER BY updated_at DESC`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Trip{}
	for rows.Next() {
		var t domain.Trip
		t.OwnerID = ownerID
		if err := rows.Scan(&t.Name, &t.Code, &t.StartDate, &t.Collab, &t.Days); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ---- vibes / checkins ----

// AddVibeCheckin always inserts a new row — unlike the old vibe_votes it
// replaces, a traveller's rating is never overwritten, only added to, so
// the full history stays available for future vibe-based recommendations.
func (s *Store) AddVibeCheckin(ctx context.Context, placeID, travellerID string, productive, loud, crowded int) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vibe_checkins (place_id, traveller_id, productive, loud, crowded)
		VALUES ($1,$2,$3,$4,$5)`, placeID, travellerID, productive, loud, crowded)
	return err
}

// VibeAggregate averages each traveller's most recent check-in at placeID
// from the last 30 days — a rolling "current vibe" reading, not an all-time
// blend that would never reflect a place's vibe actually changing.
func (s *Store) VibeAggregate(ctx context.Context, placeID string) (domain.VibeAggregate, error) {
	var agg domain.VibeAggregate
	err := s.Pool.QueryRow(ctx, `
		SELECT coalesce(avg(productive), 0), coalesce(avg(loud), 0), coalesce(avg(crowded), 0), count(*)
		FROM (
			SELECT DISTINCT ON (traveller_id) productive, loud, crowded
			FROM vibe_checkins
			WHERE place_id=$1 AND created_at > now() - interval '30 days'
			ORDER BY traveller_id, created_at DESC
		) latest`, placeID).Scan(&agg.Productive, &agg.Loud, &agg.Crowded, &agg.SampleSize)
	return agg, err
}

// MyLatestVibeCheckin returns one traveller's most recent check-in at a
// place, how ever old — unlike VibeAggregate this has no recency window,
// since "what I last rated it" should never just disappear.
func (s *Store) MyLatestVibeCheckin(ctx context.Context, placeID, travellerID string) (domain.VibeCheckin, error) {
	c := domain.VibeCheckin{PlaceID: placeID, TravellerID: travellerID}
	err := s.Pool.QueryRow(ctx, `
		SELECT productive, loud, crowded, created_at FROM vibe_checkins
		WHERE place_id=$1 AND traveller_id=$2
		ORDER BY created_at DESC LIMIT 1`, placeID, travellerID).
		Scan(&c.Productive, &c.Loud, &c.Crowded, &c.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// ---- Local Guide chat (see internal/http/guide.go) ----

// CreateGuideConversation starts a new conversation with the given id
// (caller-generated, see newID("gconv")).
func (s *Store) CreateGuideConversation(ctx context.Context, id, travellerID string) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO guide_conversations (id, traveller_id) VALUES ($1,$2)`, id, travellerID)
	return err
}

// LatestGuideConversation returns the traveller's most recently updated
// conversation, so the app can resume it on reopen.
func (s *Store) LatestGuideConversation(ctx context.Context, travellerID string) (domain.GuideConversation, error) {
	var c domain.GuideConversation
	err := s.Pool.QueryRow(ctx, `
		SELECT id, created_at, updated_at FROM guide_conversations
		WHERE traveller_id=$1 ORDER BY updated_at DESC LIMIT 1`, travellerID).
		Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// GuideConversationOwner returns the traveller_id a conversation belongs
// to, for an ownership check before appending to or reading it.
func (s *Store) GuideConversationOwner(ctx context.Context, conversationID string) (string, error) {
	var travellerID string
	err := s.Pool.QueryRow(ctx,
		`SELECT traveller_id FROM guide_conversations WHERE id=$1`, conversationID).Scan(&travellerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return travellerID, err
}

// InsertGuideMessage appends one turn and bumps the conversation's
// updated_at (so LatestGuideConversation reflects real recency).
func (s *Store) InsertGuideMessage(ctx context.Context, m domain.GuideMessage) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		INSERT INTO guide_messages (conversation_id, role, content, place_ids)
		VALUES ($1,$2,$3,$4)`,
		m.ConversationID, m.Role, m.Content, jsonb(m.PlaceIDs)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE guide_conversations SET updated_at=now() WHERE id=$1`, m.ConversationID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// GuideMessagesByConversation lists a conversation's turns oldest-first,
// for both replaying history to the traveller and as context for the LLM.
func (s *Store) GuideMessagesByConversation(ctx context.Context, conversationID string) ([]domain.GuideMessage, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT role, content, place_ids, created_at FROM guide_messages
		WHERE conversation_id=$1 ORDER BY created_at ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GuideMessage{}
	for rows.Next() {
		var m domain.GuideMessage
		var placeIDs []byte
		if err := rows.Scan(&m.Role, &m.Content, &placeIDs, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.ConversationID = conversationID
		json.Unmarshal(placeIDs, &m.PlaceIDs)
		out = append(out, m)
	}
	return out, rows.Err()
}

// GuideMessagesByTraveller returns all user-visible turns for a given traveller across all conversations
func (s *Store) GuideMessagesByTraveller(ctx context.Context, travellerID string) ([]domain.GuideMessage, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT m.conversation_id, m.role, m.content, m.place_ids, m.created_at
		FROM guide_messages m
		JOIN guide_conversations c ON m.conversation_id = c.id
		WHERE c.traveller_id = $1
		ORDER BY m.created_at ASC`, travellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GuideMessage{}
	for rows.Next() {
		var m domain.GuideMessage
		var placeIDs []byte
		if err := rows.Scan(&m.ConversationID, &m.Role, &m.Content, &placeIDs, &m.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(placeIDs, &m.PlaceIDs)
		out = append(out, m)
	}
	return out, rows.Err()
}

// InsertGuideRequest records telemetry for one POST /guide/chat call,
// success or failure — see recordGuideRequest in internal/http/guide.go.
func (s *Store) InsertGuideRequest(ctx context.Context, r domain.GuideRequest) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO guide_requests
			(conversation_id, traveller_id, status, error_message, latency_ms, tools_used, used_semantic_query, input_tokens, output_tokens)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		r.ConversationID, r.TravellerID, r.Status, r.ErrorMessage, r.LatencyMs,
		jsonb(r.ToolsUsed), r.UsedSemanticQuery, r.InputTokens, r.OutputTokens)
	return err
}

// GuideStats aggregates Local Guide usage/reliability for the admin
// dashboard — same scalar-correlated-subquery pattern as VendorSummary.
func (s *Store) GuideStats(ctx context.Context) (domain.GuideStats, error) {
	var st domain.GuideStats
	err := s.Pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM guide_conversations),
			(SELECT count(*) FROM guide_messages),
			(SELECT count(*) FROM guide_messages WHERE created_at >= date_trunc('day', now())),
			(SELECT count(*) FROM guide_messages WHERE created_at >= now() - interval '7 days'),
			(SELECT count(*) FROM guide_requests WHERE created_at >= now() - interval '7 days'),
			(SELECT count(*) FROM guide_requests WHERE created_at >= now() - interval '7 days' AND status = 'ok'),
			(SELECT count(*) FROM guide_requests WHERE created_at >= now() - interval '7 days' AND status = 'rate_limited'),
			(SELECT count(*) FROM guide_requests WHERE created_at >= now() - interval '7 days' AND status = 'error'),
			(SELECT COALESCE(avg(latency_ms), 0) FROM guide_requests WHERE created_at >= now() - interval '7 days'),
			(SELECT count(*) FROM guide_requests WHERE created_at >= now() - interval '7 days' AND used_semantic_query),
			(SELECT count(*) FROM places WHERE embedding IS NOT NULL),
			(SELECT count(*) FROM places)`).Scan(
		&st.TotalConversations, &st.TotalMessages, &st.MessagesToday, &st.Messages7d,
		&st.Requests7d, &st.OkCount7d, &st.RateLimitedCount7d, &st.ErrorCount7d,
		&st.AvgLatencyMs7d, &st.SemanticQueryCount7d, &st.PlacesEmbedded, &st.PlacesTotal,
	)
	return st, err
}

// GuideToolUsage breaks down tool-call counts over the trailing 7 days —
// the one place this needs an actual GROUP BY, since tool names live inside
// a JSONB array column rather than a scalar one.
func (s *Store) GuideToolUsage(ctx context.Context) ([]domain.ToolUsageCount, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT tool, count(*) FROM guide_requests, jsonb_array_elements_text(tools_used) AS tool
		WHERE created_at >= now() - interval '7 days'
		GROUP BY tool ORDER BY count(*) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ToolUsageCount{}
	for rows.Next() {
		var t domain.ToolUsageCount
		if err := rows.Scan(&t.Tool, &t.Count); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// RecentGuideUsers lists the most recently active Local Guide users
// for the admin dashboard's browsable list, along with their analytics.
func (s *Store) RecentGuideUsers(ctx context.Context, limit int) ([]domain.GuideUserSummary, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT
			u.id,
			COALESCE(u.name, ''),
			COALESCE((SELECT count(*) FROM guide_messages m 
				JOIN guide_conversations c ON m.conversation_id = c.id 
				WHERE c.traveller_id = u.id), 0) AS total_messages,
			COALESCE((SELECT count(*) FROM guide_requests r 
				WHERE r.traveller_id = u.id AND r.status = 'error'), 0) AS error_count,
			COALESCE((SELECT sum(input_tokens) FROM guide_requests r 
				WHERE r.traveller_id = u.id), 0) AS input_tokens,
			COALESCE((SELECT sum(output_tokens) FROM guide_requests r 
				WHERE r.traveller_id = u.id), 0) AS output_tokens,
			COALESCE((SELECT count(*) FROM trips t 
				WHERE t.owner_id = u.id), 0) AS trips_planned,
			COALESCE((SELECT max(created_at) FROM guide_requests r 
				WHERE r.traveller_id = u.id), u.created_at) AS last_active_at
		FROM users u
		WHERE EXISTS (SELECT 1 FROM guide_requests r WHERE r.traveller_id = u.id)
		ORDER BY last_active_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GuideUserSummary{}
	for rows.Next() {
		var g domain.GuideUserSummary
		if err := rows.Scan(&g.TravellerID, &g.TravellerName, &g.TotalMessages, &g.ErrorCount, &g.InputTokens, &g.OutputTokens, &g.TripsPlanned, &g.LastActiveAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// RecentGuideConversations lists the most recently updated conversations
// for the admin dashboard's browsable list.
func (s *Store) RecentGuideConversations(ctx context.Context, limit int) ([]domain.GuideConversationSummary, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT c.id, COALESCE(u.name, ''),
			(SELECT count(*) FROM guide_messages m WHERE m.conversation_id = c.id),
			COALESCE((SELECT content FROM guide_messages m
				WHERE m.conversation_id = c.id ORDER BY m.created_at DESC LIMIT 1), ''),
			c.updated_at
		FROM guide_conversations c
		LEFT JOIN users u ON u.id = c.traveller_id
		ORDER BY c.updated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.GuideConversationSummary{}
	for rows.Next() {
		var g domain.GuideConversationSummary
		if err := rows.Scan(&g.ID, &g.TravellerName, &g.MessageCount, &g.LastMessage, &g.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (s *Store) AddCheckin(ctx context.Context, placeID, travellerID string) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO checkins (place_id, traveller_id) VALUES ($1,$2)`, placeID, travellerID)
	return err
}

// GetDailyQuestProgress counts the distinct places a traveller has checked
// into since dayStart — the start of the UTC quest day, passed in rather than
// taken from Postgres CURRENT_DATE so it can't disagree with the UTC date the
// quest schedule is looked up by.
func (s *Store) GetDailyQuestProgress(ctx context.Context, travellerID string, dayStart time.Time) (int, error) {
	var count int
	err := s.Pool.QueryRow(ctx,
		`SELECT COUNT(DISTINCT place_id)
		 FROM checkins
		 WHERE traveller_id = $1
		 AND created_at >= $2`, travellerID, dayStart).Scan(&count)
	return count, err
}

// RecordQuestCompletion saves that a traveller finished a day's quest, once
// per traveller and day, and returns when it was first recorded.
func (s *Store) RecordQuestCompletion(ctx context.Context, c domain.QuestCompletion) (time.Time, error) {
	var at time.Time
	// DO UPDATE with a no-op set (rather than DO NOTHING) so RETURNING also
	// yields the existing row's completed_at on repeat calls.
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO quest_completions (traveller_id, date, vendor_id, goal, reward_description)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (traveller_id, date) DO UPDATE SET traveller_id = quest_completions.traveller_id
		RETURNING completed_at`,
		c.TravellerID, c.Date, c.VendorID, c.Goal, c.RewardDescription).Scan(&at)
	return at, err
}

// QuestClaimedAt is when a traveller's reward for date was scanned, or nil.
func (s *Store) QuestClaimedAt(ctx context.Context, travellerID, date string) (*time.Time, error) {
	var at time.Time
	err := s.Pool.QueryRow(ctx,
		`SELECT claimed_at FROM quest_claims WHERE traveller_id = $1 AND date = $2`, travellerID, date).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &at, nil
}

// ListQuestCompletions returns completions newest first, with the
// traveller's name/email, the sponsor's name, and when (if) the reward was
// redeemed.
func (s *Store) ListQuestCompletions(ctx context.Context, f domain.QuestCompletionFilter) ([]domain.QuestCompletion, error) {
	limit := f.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT qc.traveller_id, COALESCE(u.name, ''), COALESCE(u.email, ''), qc.date, qc.vendor_id,
		       COALESCE(v.name, 'Spotto'), qc.goal, qc.reward_description, qc.completed_at, cl.claimed_at
		FROM quest_completions qc
		LEFT JOIN users u ON u.id = qc.traveller_id
		LEFT JOIN vendors v ON v.id = qc.vendor_id
		LEFT JOIN quest_claims cl ON cl.traveller_id = qc.traveller_id AND cl.date = qc.date
		WHERE ($1 = '' OR qc.date = $1)
		  AND ($2 = '' OR qc.date >= $2)
		  AND ($3 = '' OR qc.vendor_id = $3)
		  AND ($4 = '' OR qc.traveller_id = $4)
		  AND ($5::timestamptz IS NULL OR qc.completed_at < $5)
		ORDER BY qc.completed_at DESC
		LIMIT $6`,
		f.Date, f.Since, f.VendorID, f.TravellerID, f.Before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.QuestCompletion{}
	for rows.Next() {
		var c domain.QuestCompletion
		if err := rows.Scan(&c.TravellerID, &c.TravellerName, &c.TravellerEmail, &c.Date, &c.VendorID,
			&c.SponsorName, &c.Goal, &c.RewardDescription, &c.CompletedAt, &c.RedeemedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// QuestCompletionsCount is how many travellers finished vendorID's quest on date.
func (s *Store) QuestCompletionsCount(ctx context.Context, vendorID, date string) (int, error) {
	var count int
	err := s.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM quest_completions WHERE vendor_id = $1 AND date = $2`, vendorID, date).Scan(&count)
	return count, err
}

// ---- vendor notifications ----

func (s *Store) UpsertVendorNotification(ctx context.Context, vendorID string, n domain.VendorNotification) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendor_notifications (id, vendor_id, icon, tone, title, sub, t, unread)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET icon=EXCLUDED.icon, tone=EXCLUDED.tone,
			title=EXCLUDED.title, sub=EXCLUDED.sub, t=EXCLUDED.t, unread=EXCLUDED.unread`,
		n.ID, vendorID, n.Icon, n.Tone, n.Title, n.Sub, n.Time, n.Unread)
	return err
}

func (s *Store) VendorNotifications(ctx context.Context, vendorID string) ([]domain.VendorNotification, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT id, icon, tone, title, sub, t, unread FROM vendor_notifications WHERE vendor_id=$1 ORDER BY id`,
		vendorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.VendorNotification
	for rows.Next() {
		var n domain.VendorNotification
		if err := rows.Scan(&n.ID, &n.Icon, &n.Tone, &n.Title, &n.Sub, &n.Time, &n.Unread); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (s *Store) MarkVendorNotificationsRead(ctx context.Context, vendorID string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE vendor_notifications SET unread=false WHERE vendor_id=$1`, vendorID)
	return err
}

// ---- admin audit log ----

func (s *Store) InsertAdminAuditLog(ctx context.Context, e domain.AdminAuditLog) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO admin_audit_log (actor_id, actor_email, actor_name, action, target_type, target_id, metadata, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		e.ActorID, e.ActorEmail, e.ActorName, e.Action, e.TargetType, e.TargetID, e.Metadata, e.CreatedAt)
	return err
}

// AdminAuditLogs lists entries most-recent-first, optionally filtered by
// targetType and/or actorID, cursor-paginated the same way FeedItems is
// (beforeID=0 starts from the newest row).
func (s *Store) AdminAuditLogs(ctx context.Context, beforeID int64, targetType, actorID string, limit int) ([]domain.AdminAuditLog, error) {
	q := `SELECT id, actor_id, actor_email, actor_name, action, target_type, target_id, metadata, created_at
		FROM admin_audit_log WHERE ($1 = 0 OR id < $1)`
	args := []any{beforeID}
	if targetType != "" {
		args = append(args, targetType)
		q += fmt.Sprintf(` AND target_type=$%d`, len(args))
	}
	if actorID != "" {
		args = append(args, actorID)
		q += fmt.Sprintf(` AND actor_id=$%d`, len(args))
	}
	q += fmt.Sprintf(` ORDER BY id DESC LIMIT %d`, limit)
	rows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.AdminAuditLog
	for rows.Next() {
		var e domain.AdminAuditLog
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorEmail, &e.ActorName, &e.Action,
			&e.TargetType, &e.TargetID, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// AuditLogsByTarget returns the admin-action history for one specific entity
// (e.g. a single vendor), newest first — powers the vendor-detail History
// tab. AdminAuditLogs filters by target *type* and actor, not target id.
func (s *Store) AuditLogsByTarget(ctx context.Context, targetType, targetID string, limit int) ([]domain.AdminAuditLog, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, actor_id, actor_email, actor_name, action, target_type, target_id, metadata, created_at
		FROM admin_audit_log
		WHERE target_type=$1 AND target_id=$2
		ORDER BY id DESC LIMIT $3`, targetType, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AdminAuditLog{}
	for rows.Next() {
		var e domain.AdminAuditLog
		if err := rows.Scan(&e.ID, &e.ActorID, &e.ActorEmail, &e.ActorName, &e.Action,
			&e.TargetType, &e.TargetID, &e.Metadata, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ---- admin search ----

// AdminSearchResults is a query-response shape (not a persisted entity),
// so it lives here rather than in package domain.
type AdminSearchResults struct {
	Vendors       []domain.Vendor       `json:"vendors"`
	FeedItems     []domain.FeedItem     `json:"feedItems"`
	Subscriptions []domain.Subscription `json:"subscriptions"`
}

// AdminSearch runs three independent ILIKE lookups (vendor/feed-item/
// subscription name) plus an exact feed-item-ID match so pasting a numeric
// ID still works — data volumes here don't warrant full-text-search infra.
func (s *Store) AdminSearch(ctx context.Context, q string, limit int) (AdminSearchResults, error) {
	out := AdminSearchResults{Vendors: []domain.Vendor{}, FeedItems: []domain.FeedItem{}, Subscriptions: []domain.Subscription{}}
	pattern := "%" + q + "%"

	vendorRows, err := s.Pool.Query(ctx,
		`SELECT `+vendorCols+` FROM vendors WHERE name ILIKE $1 ORDER BY name LIMIT $2`, pattern, limit)
	if err != nil {
		return out, err
	}
	for vendorRows.Next() {
		v, err := scanVendor(vendorRows)
		if err != nil {
			vendorRows.Close()
			return out, err
		}
		out.Vendors = append(out.Vendors, v)
	}
	vendorRows.Close()
	if err := vendorRows.Err(); err != nil {
		return out, err
	}

	feedIDMatch, _ := strconv.ParseInt(q, 10, 64)
	feedRows, err := s.Pool.Query(ctx, `
		SELECT id, vendor, town, type, text, status, reason, resolved, created_at FROM feed_items
		WHERE vendor ILIKE $1 OR id = $2 ORDER BY id DESC LIMIT $3`, pattern, feedIDMatch, limit)
	if err != nil {
		return out, err
	}
	for feedRows.Next() {
		var f domain.FeedItem
		if err := feedRows.Scan(&f.ID, &f.Vendor, &f.Town, &f.Type, &f.Text, &f.Status,
			&f.Reason, &f.Resolved, &f.CreatedAt); err != nil {
			feedRows.Close()
			return out, err
		}
		out.FeedItems = append(out.FeedItems, f)
	}
	feedRows.Close()
	if err := feedRows.Err(); err != nil {
		return out, err
	}

	subRows, err := s.Pool.Query(ctx,
		`SELECT id, vendor_id, vendor, town, tier, mrr, renews_at, state, since FROM subscriptions
		WHERE vendor ILIKE $1 ORDER BY vendor LIMIT $2`, pattern, limit)
	if err != nil {
		return out, err
	}
	defer subRows.Close()
	for subRows.Next() {
		var sub domain.Subscription
		if err := subRows.Scan(&sub.ID, &sub.VendorID, &sub.Vendor, &sub.Town, &sub.Tier,
			&sub.MRR, &sub.RenewsAt, &sub.State, &sub.Since); err != nil {
			return out, err
		}
		out.Subscriptions = append(out.Subscriptions, sub)
	}
	return out, subRows.Err()
}

// ---- admin notifications ----

func (s *Store) MarkAdminNotificationsRead(ctx context.Context, adminID string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO admin_notification_reads (admin_id, last_read_at) VALUES ($1, now())
		ON CONFLICT (admin_id) DO UPDATE SET last_read_at = EXCLUDED.last_read_at`, adminID)
	return err
}

// pluralSuffix is a tiny English-only helper for the notification copy
// below — "1 vendor" vs "2 vendors".
func pluralSuffix(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// AdminNotifications derives up to 3 buckets — pending-approval vendors,
// flagged content, bad-review-flagged vendors — live from current data
// rather than from stored discrete rows, so they can never drift out of
// sync with the underlying state (matches how CountAdminKPIBase already
// works). Each bucket is marked unread if its latest-activity timestamp is
// after adminID's last read (see MarkAdminNotificationsRead).
func (s *Store) AdminNotifications(ctx context.Context, adminID string) ([]domain.AdminNotification, error) {
	var lastRead time.Time
	if err := s.Pool.QueryRow(ctx,
		`SELECT COALESCE((SELECT last_read_at FROM admin_notification_reads WHERE admin_id=$1), '1970-01-01')`, adminID,
	).Scan(&lastRead); err != nil {
		return nil, err
	}

	out := []domain.AdminNotification{}

	var pendingCount int
	var pendingLatest time.Time
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*), COALESCE(max(submitted_at), '1970-01-01') FROM vendors WHERE status='pending'`,
	).Scan(&pendingCount, &pendingLatest); err != nil {
		return nil, err
	}
	if pendingCount > 0 {
		out = append(out, domain.AdminNotification{
			ID: "pending-vendors", Icon: "UserCheck", Tone: "sun",
			Title:  fmt.Sprintf("%d vendor%s awaiting approval", pendingCount, pluralSuffix(pendingCount)),
			Sub:    "Review new listings in the approval queue.",
			Time:   pendingLatest.Format("Jan 2, 3:04 PM"),
			Unread: pendingLatest.After(lastRead),
		})
	}

	var flagCount int
	var flagLatest time.Time
	if err := s.Pool.QueryRow(ctx,
		`SELECT count(*), COALESCE(max(created_at), '1970-01-01') FROM feed_items WHERE status='flag'`,
	).Scan(&flagCount, &flagLatest); err != nil {
		return nil, err
	}
	if flagCount > 0 {
		out = append(out, domain.AdminNotification{
			ID: "flagged-content", Icon: "Flag", Tone: "accent",
			Title:  fmt.Sprintf("%d flagged content item%s", flagCount, pluralSuffix(flagCount)),
			Sub:    "Moderate flagged posts in the content monitor.",
			Time:   flagLatest.Format("Jan 2, 3:04 PM"),
			Unread: flagLatest.After(lastRead),
		})
	}

	var badVendorCount int
	var badLatest time.Time
	if err := s.Pool.QueryRow(ctx, `
		SELECT count(*), COALESCE(max(r.max_created), '1970-01-01') FROM (
			SELECT vendor_id, max(created_at) AS max_created
			FROM reviews
			WHERE vendor_id IS NOT NULL AND rating <= 2
			GROUP BY vendor_id
			HAVING count(*) >= $1
		) r`, BadReviewFlagThreshold,
	).Scan(&badVendorCount, &badLatest); err != nil {
		return nil, err
	}
	if badVendorCount > 0 {
		out = append(out, domain.AdminNotification{
			ID: "bad-review-vendors", Icon: "ShieldAlert", Tone: "muted",
			Title:  fmt.Sprintf("%d vendor%s flagged for bad reviews", badVendorCount, pluralSuffix(badVendorCount)),
			Sub:    "Check red-flagged pins on geo-analytics.",
			Time:   badLatest.Format("Jan 2, 3:04 PM"),
			Unread: badLatest.After(lastRead),
		})
	}

	return out, nil
}

// ---- misc ----

// Wipe truncates all data tables (used by cmd/seed for idempotent reloads).
func (s *Store) Wipe(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `
		TRUNCATE users, offers, claims, trips, groups, feed_items,
			regions, subscriptions, audio_guides, vibe_checkins, checkins,
			vendor_notifications, vendor_blobs, reviews, vendor_invites, admin_audit_log,
			admin_notification_reads`)
	return err
}

func (s *Store) CountAdminKPIBase(ctx context.Context) (pendingVendors int, liveFeed int, err error) {
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM vendors WHERE status='pending'`).Scan(&pendingVendors); err != nil {
		return
	}
	err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM feed_items`).Scan(&liveFeed)
	return
}

// ---- push notification tokens ----

func (s *Store) UpsertUserPushToken(ctx context.Context, token domain.UserPushToken) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO user_push_tokens (id, user_id, device_token, platform, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (user_id, device_token) DO UPDATE SET
			platform = EXCLUDED.platform, updated_at = now()`,
		token.ID, token.UserID, token.DeviceToken, token.Platform,
	)
	return err
}

func (s *Store) DeleteUserPushToken(ctx context.Context, userID, deviceToken string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM user_push_tokens WHERE user_id = $1 AND device_token = $2`, userID, deviceToken)
	return err
}

func (s *Store) UserPushTokens(ctx context.Context, userID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT device_token FROM user_push_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []string
	for rows.Next() {
		var tok string
		if err := rows.Scan(&tok); err == nil {
			tokens = append(tokens, tok)
		}
	}
	return tokens, nil
}

// ---- payment methods & transactions ----

func (s *Store) CreatePaymentMethod(ctx context.Context, pm domain.PaymentMethod) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO payment_methods (id, user_id, brand, last4, exp_month, exp_year, holder_name, is_default, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())`,
		pm.ID, pm.UserID, pm.Brand, pm.Last4, pm.ExpMonth, pm.ExpYear, pm.HolderName, pm.IsDefault,
	)
	return err
}

func (s *Store) PaymentMethodsByUser(ctx context.Context, userID string) ([]domain.PaymentMethod, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, user_id, brand, last4, exp_month, exp_year, holder_name, is_default, created_at
		FROM payment_methods WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var methods []domain.PaymentMethod
	for rows.Next() {
		var pm domain.PaymentMethod
		if err := rows.Scan(&pm.ID, &pm.UserID, &pm.Brand, &pm.Last4, &pm.ExpMonth, &pm.ExpYear, &pm.HolderName, &pm.IsDefault, &pm.CreatedAt); err == nil {
			methods = append(methods, pm)
		}
	}
	return methods, nil
}

func (s *Store) DeletePaymentMethod(ctx context.Context, userID, id string) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM payment_methods WHERE id = $1 AND user_id = $2`, id, userID)
	return err
}

func (s *Store) SetDefaultPaymentMethod(ctx context.Context, userID, id string) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE payment_methods SET is_default = false WHERE user_id = $1`, userID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE payment_methods SET is_default = true WHERE id = $1 AND user_id = $2`, id, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) CreatePaymentTransaction(ctx context.Context, tx domain.PaymentTransaction) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO payment_transactions (id, user_id, vendor_id, amount_cents, currency, status, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())`,
		tx.ID, tx.UserID, tx.VendorID, tx.AmountCents, tx.Currency, tx.Status, tx.Description,
	)
	return err
}

func (s *Store) GetVendorSponsorship(ctx context.Context, vendorID string) (domain.VendorSponsorship, error) {
	var v domain.VendorSponsorship
	err := s.Pool.QueryRow(ctx, `
		SELECT vendor_id, reward_description, reward_limit, is_active, status, created_at
		FROM vendor_sponsorships WHERE vendor_id = $1`, vendorID).
		Scan(&v.VendorID, &v.RewardDescription, &v.RewardLimit, &v.IsActive, &v.Status, &v.CreatedAt)
	if err == pgx.ErrNoRows {
		return domain.VendorSponsorship{}, ErrNotFound
	}
	return v, err
}

func (s *Store) UpsertVendorSponsorship(ctx context.Context, v domain.VendorSponsorship) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendor_sponsorships (vendor_id, reward_description, reward_limit, is_active, status, created_at)
		VALUES ($1, $2, $3, $4, 'pending', now())
		ON CONFLICT (vendor_id) DO UPDATE SET
			status = CASE WHEN vendor_sponsorships.reward_description != EXCLUDED.reward_description THEN 'pending' ELSE vendor_sponsorships.status END,
			reward_description = EXCLUDED.reward_description,
			reward_limit = EXCLUDED.reward_limit,
			is_active = EXCLUDED.is_active`,
		v.VendorID, v.RewardDescription, v.RewardLimit, v.IsActive)
	return err
}

func (s *Store) CreateQuestClaim(ctx context.Context, claim domain.QuestClaim) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO quest_claims (id, traveller_id, vendor_id, date, claimed_at)
		VALUES ($1, $2, $3, $4, now())`,
		claim.ID, claim.TravellerID, claim.VendorID, claim.Date)
	return err
}

func (s *Store) GetQuestClaimsCount(ctx context.Context, vendorID, date string) (int, error) {
	var count int
	err := s.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM quest_claims WHERE vendor_id = $1 AND date = $2`,
		vendorID, date).Scan(&count)
	return count, err
}

func (s *Store) ListVendorSponsorshipsByStatus(ctx context.Context, status string) ([]domain.VendorSponsorship, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT vendor_id, reward_description, reward_limit, is_active, status, created_at
		FROM vendor_sponsorships WHERE status = $1 ORDER BY created_at DESC`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []domain.VendorSponsorship
	for rows.Next() {
		var v domain.VendorSponsorship
		if err := rows.Scan(&v.VendorID, &v.RewardDescription, &v.RewardLimit, &v.IsActive, &v.Status, &v.CreatedAt); err != nil {
			return nil, err
		}
		res = append(res, v)
	}
	return res, rows.Err()
}

func (s *Store) UpdateVendorSponsorshipStatus(ctx context.Context, vendorID, status string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendor_sponsorships SET status = $1 WHERE vendor_id = $2`, status, vendorID)
	return err
}

func (s *Store) GetQuestSchedule(ctx context.Context, date string) (domain.QuestSchedule, error) {
	var q domain.QuestSchedule
	err := s.Pool.QueryRow(ctx, `
		SELECT d.date, d.vendor_id, d.goal, v.name, vs.reward_description
		FROM daily_quests d
		JOIN vendors v ON v.id = d.vendor_id
		JOIN vendor_sponsorships vs ON vs.vendor_id = d.vendor_id
		WHERE d.date = $1`, date).
		Scan(&q.Date, &q.VendorID, &q.Goal, &q.SponsorName, &q.RewardDescription)
	if err == pgx.ErrNoRows {
		return domain.QuestSchedule{}, ErrNotFound
	}
	return q, err
}

func (s *Store) SetQuestSchedule(ctx context.Context, q domain.QuestSchedule) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO daily_quests (date, vendor_id, goal, created_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (date) DO UPDATE SET vendor_id = EXCLUDED.vendor_id, goal = EXCLUDED.goal`,
		q.Date, q.VendorID, q.Goal)
	return err
}

func (s *Store) ListQuestSchedules(ctx context.Context) ([]domain.QuestSchedule, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT d.date, d.vendor_id, d.goal, v.name, vs.reward_description,
		       (SELECT COUNT(*) FROM quest_completions qc WHERE qc.date = d.date),
		       (SELECT COUNT(*) FROM quest_claims cl WHERE cl.date = d.date)
		FROM daily_quests d
		JOIN vendors v ON v.id = d.vendor_id
		JOIN vendor_sponsorships vs ON vs.vendor_id = d.vendor_id
		ORDER BY d.date DESC LIMIT 30`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []domain.QuestSchedule
	for rows.Next() {
		var q domain.QuestSchedule
		if err := rows.Scan(&q.Date, &q.VendorID, &q.Goal, &q.SponsorName, &q.RewardDescription, &q.Completions, &q.Redemptions); err != nil {
			return nil, err
		}
		res = append(res, q)
	}
	return res, rows.Err()
}

func (s *Store) PlatformAnalytics(ctx context.Context) (domain.PlatformAnalytics, error) {
	var a domain.PlatformAnalytics
	err := s.Pool.QueryRow(ctx, `
		SELECT 
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM vendors),
			(SELECT COUNT(*) FROM claims),
			(SELECT COUNT(*) FROM reviews)
	`).Scan(&a.TotalUsers, &a.TotalVendors, &a.TotalClaims, &a.TotalReviews)
	return a, err
}

func (s *Store) GetRateLimitRPM(ctx context.Context) (int, error) {
	var val string
	err := s.Pool.QueryRow(ctx, `SELECT value FROM platform_settings WHERE key = 'rate_limit_rpm'`).Scan(&val)
	if err != nil {
		return 200, err
	}
	var limit int
	if err := json.Unmarshal([]byte(val), &limit); err != nil {
		return 200, err
	}
	return limit, nil
}

func (s *Store) SetRateLimitRPM(ctx context.Context, limit int) error {
	b, _ := json.Marshal(limit)
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO platform_settings (key, value) VALUES ('rate_limit_rpm', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, b)
	return err
}

func (s *Store) GetGuideRateLimitRPM(ctx context.Context) (int, error) {
	var val string
	err := s.Pool.QueryRow(ctx, `SELECT value FROM platform_settings WHERE key = 'guide_rate_limit_rpm'`).Scan(&val)
	if err == pgx.ErrNoRows {
		return 15, nil
	}
	if err != nil {
		return 15, err
	}
	var limit int
	if err := json.Unmarshal([]byte(val), &limit); err != nil {
		return 15, err
	}
	return limit, nil
}

func (s *Store) SetGuideRateLimitRPM(ctx context.Context, limit int) error {
	b, _ := json.Marshal(limit)
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO platform_settings (key, value) VALUES ('guide_rate_limit_rpm', $1)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, b)
	return err
}

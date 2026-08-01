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

const vendorCols = `id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng`

func scanVendor(row pgx.Row) (domain.Vendor, error) {
	var v domain.Vendor
	var flags []byte
	err := row.Scan(&v.ID, &v.Name, &v.Category, &v.Town, &v.Country, &v.RegionID, &v.Owner, &v.Status,
		&v.Priority, &v.Docs, &v.Photos, &v.WifiScore, &flags, &v.Note, &v.Tier,
		&v.Profile, &v.Menu, &v.SubmittedAt, &v.Lat, &v.Lng)
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
		INSERT INTO vendors (id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name, category=EXCLUDED.category, town=EXCLUDED.town, country=EXCLUDED.country,
			region_id=EXCLUDED.region_id, owner=EXCLUDED.owner, status=EXCLUDED.status,
			priority=EXCLUDED.priority, docs=EXCLUDED.docs, photos=EXCLUDED.photos,
			wifi_score=EXCLUDED.wifi_score, flags=EXCLUDED.flags, note=EXCLUDED.note,
			tier=EXCLUDED.tier, profile=EXCLUDED.profile, menu=EXCLUDED.menu,
			submitted_at=EXCLUDED.submitted_at, lat=EXCLUDED.lat, lng=EXCLUDED.lng`,
		v.ID, v.Name, v.Category, v.Town, v.Country, v.RegionID, v.Owner, v.Status, v.Priority,
		v.Docs, v.Photos, v.WifiScore, jsonb(v.Flags), v.Note, v.Tier,
		orEmptyObj(v.Profile), orEmptyObj(v.Menu), v.SubmittedAt, v.Lat, v.Lng)
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
	return "id, name, category, town, country, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng"
}

func scanVendorWithBadCount(row pgx.Row) (domain.Vendor, int, error) {
	var v domain.Vendor
	var flags []byte
	var badCount int
	err := row.Scan(&v.ID, &v.Name, &v.Category, &v.Town, &v.Country, &v.RegionID, &v.Owner, &v.Status,
		&v.Priority, &v.Docs, &v.Photos, &v.WifiScore, &flags, &v.Note, &v.Tier,
		&v.Profile, &v.Menu, &v.SubmittedAt, &v.Lat, &v.Lng, &badCount)
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

const mediaCols = `id, owner_type, owner_id, kind, role, object_key, url, thumb_key, content_type, size, width, height, uploaded_by, status, sort, created_at`

func scanMedia(row pgx.Row) (domain.Media, error) {
	var m domain.Media
	err := row.Scan(&m.ID, &m.OwnerType, &m.OwnerID, &m.Kind, &m.Role, &m.ObjectKey, &m.URL, &m.ThumbKey,
		&m.ContentType, &m.Size, &m.Width, &m.Height, &m.UploadedBy, &m.Status, &m.Sort, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, ErrNotFound
	}
	return m, err
}

func (s *Store) InsertMedia(ctx context.Context, m domain.Media) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO media (`+mediaCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		m.ID, m.OwnerType, m.OwnerID, m.Kind, m.Role, m.ObjectKey, m.URL, m.ThumbKey,
		m.ContentType, m.Size, m.Width, m.Height, m.UploadedBy, m.Status, m.Sort, m.CreatedAt)
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
		`SELECT p.id, p.name, p.vendor_id, p.data FROM places p WHERE `+placeVisibleWhere+` ORDER BY p.id`)
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

func (s *Store) PlaceByID(ctx context.Context, id string) (domain.Place, error) {
	var p domain.Place
	err := s.Pool.QueryRow(ctx,
		`SELECT p.id, p.name, p.vendor_id, p.data FROM places p WHERE p.id=$1 AND `+placeVisibleWhere, id).
		Scan(&p.ID, &p.Name, &p.VendorID, &p.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
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

// ReviewPhotos batch-loads the attached photo URLs for a set of reviews,
// keyed by review id — one query for a whole reviews list (no N+1).
func (s *Store) ReviewPhotos(ctx context.Context, reviewIDs []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(reviewIDs) == 0 {
		return out, nil
	}
	rows, err := s.Pool.Query(ctx,
		`SELECT owner_id, url FROM media
		 WHERE owner_type='review' AND role='review' AND status='ready' AND owner_id = ANY($1)
		 ORDER BY created_at ASC`, reviewIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ownerID, url string
		if err := rows.Scan(&ownerID, &url); err != nil {
			return nil, err
		}
		out[ownerID] = append(out[ownerID], url)
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

func (s *Store) AddCheckin(ctx context.Context, placeID, travellerID string) error {
	_, err := s.Pool.Exec(ctx,
		`INSERT INTO checkins (place_id, traveller_id) VALUES ($1,$2)`, placeID, travellerID)
	return err
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
		TRUNCATE users, vendors, offers, places, claims, trips, groups, feed_items,
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

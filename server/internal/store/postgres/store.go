// Package postgres is the pgx-backed store used by the API and seed command.
package postgres

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

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
		u.ID, u.Email, u.PasswordHash, u.Role, u.Name, u.Country, u.TravelStyle,
		jsonb(u.Interests), u.VendorID, u.Extra, u.CreatedAt, vendorRole)
	return err
}

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	var interests []byte
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.Name, &u.Country,
		&u.TravelStyle, &interests, &u.VendorID, &u.Extra, &u.CreatedAt,
		&u.TOTPSecret, &u.TOTPPendingSecret, &u.TOTPEnabled, &u.TokenVersion,
		&u.ShareLocation, &u.Lat, &u.Lng, &u.LocationUpdatedAt,
		&u.Phone, &u.VendorRole, &u.NotifyRedemptions, &u.NotifyReviews, &u.NotifyPayouts, &u.NotifyGrowthTips)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	if err != nil {
		return u, err
	}
	json.Unmarshal(interests, &u.Interests)
	return u, nil
}

const userCols = `id, email, password_hash, role, name, country, travel_style, interests, vendor_id, extra, created_at, totp_secret, totp_pending_secret, totp_enabled, token_version, share_location, lat, lng, location_updated_at, phone, vendor_role, notify_redemptions, notify_reviews, notify_payouts, notify_growth_tips`

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE lower(email)=lower($1)`, email))
}

func (s *Store) UserByID(ctx context.Context, id string) (domain.User, error) {
	return scanUser(s.Pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id=$1`, id))
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

func (s *Store) UpdateUserProfile(ctx context.Context, id, name, country, style string, interests []string) error {
	_, err := s.Pool.Exec(ctx,
		`UPDATE users SET name=$2, country=$3, travel_style=$4, interests=$5 WHERE id=$1`,
		id, name, country, style, jsonb(interests))
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
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	TravelStyle string  `json:"travelStyle"`
	DistanceKm  float64 `json:"distanceKm"`
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
		if err := rows.Scan(&t.ID, &t.Name, &t.TravelStyle, &t.DistanceKm); err != nil {
			return nil, err
		}
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

// RemoveVendorTeamMember detaches a staff member from the vendor account
// (clears vendor_id/role rather than deleting the account outright, so
// their login persists but loses all vendor access) — scoped to vendorID
// and vendor_role='staff' so an owner can't be removed this way.
func (s *Store) RemoveVendorTeamMember(ctx context.Context, vendorID, userID string) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE users SET vendor_id=NULL, vendor_role='' WHERE id=$1 AND vendor_id=$2 AND vendor_role=$3`,
		userID, vendorID, domain.VendorRoleStaff)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CreateVendorInvite stores a freshly generated invite code.
func (s *Store) CreateVendorInvite(ctx context.Context, inv domain.VendorInvite) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendor_invites (id, vendor_id, code, created_by, created_at, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		inv.ID, inv.VendorID, inv.Code, inv.CreatedBy, inv.CreatedAt, inv.ExpiresAt)
	return err
}

func scanVendorInvite(row pgx.Row) (domain.VendorInvite, error) {
	var inv domain.VendorInvite
	err := row.Scan(&inv.ID, &inv.VendorID, &inv.Code, &inv.CreatedBy, &inv.CreatedAt,
		&inv.ExpiresAt, &inv.AcceptedAt, &inv.AcceptedUserID, &inv.Revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return inv, ErrNotFound
	}
	return inv, err
}

const vendorInviteCols = `id, vendor_id, code, created_by, created_at, expires_at, accepted_at, accepted_user_id, revoked`

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

const vendorCols = `id, name, category, town, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng`

func scanVendor(row pgx.Row) (domain.Vendor, error) {
	var v domain.Vendor
	var flags []byte
	err := row.Scan(&v.ID, &v.Name, &v.Category, &v.Town, &v.RegionID, &v.Owner, &v.Status,
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
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vendors (id, name, category, town, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name, category=EXCLUDED.category, town=EXCLUDED.town,
			region_id=EXCLUDED.region_id, owner=EXCLUDED.owner, status=EXCLUDED.status,
			priority=EXCLUDED.priority, docs=EXCLUDED.docs, photos=EXCLUDED.photos,
			wifi_score=EXCLUDED.wifi_score, flags=EXCLUDED.flags, note=EXCLUDED.note,
			tier=EXCLUDED.tier, profile=EXCLUDED.profile, menu=EXCLUDED.menu,
			submitted_at=EXCLUDED.submitted_at, lat=EXCLUDED.lat, lng=EXCLUDED.lng`,
		v.ID, v.Name, v.Category, v.Town, v.RegionID, v.Owner, v.Status, v.Priority,
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
	return "id, name, category, town, region_id, owner, status, priority, docs, photos, wifi_score, flags, note, tier, profile, menu, submitted_at, lat, lng"
}

func scanVendorWithBadCount(row pgx.Row) (domain.Vendor, int, error) {
	var v domain.Vendor
	var flags []byte
	var badCount int
	err := row.Scan(&v.ID, &v.Name, &v.Category, &v.Town, &v.RegionID, &v.Owner, &v.Status,
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

func (s *Store) UpdateVendorMenu(ctx context.Context, id string, menu json.RawMessage) error {
	_, err := s.Pool.Exec(ctx, `UPDATE vendors SET menu=$2 WHERE id=$1`, id, menu)
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

const offerCols = `id, vendor_id, title, type, value, status, start_at, end_at, lim, claimed, redeemed`

func scanOffer(row pgx.Row) (domain.Offer, error) {
	var o domain.Offer
	err := row.Scan(&o.ID, &o.VendorID, &o.Title, &o.Type, &o.Value, &o.Status,
		&o.StartAt, &o.EndAt, &o.Limit, &o.Claimed, &o.Redeemed)
	if errors.Is(err, pgx.ErrNoRows) {
		return o, ErrNotFound
	}
	return o, err
}

func (s *Store) UpsertOffer(ctx context.Context, o domain.Offer) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO offers (id, vendor_id, title, type, value, status, start_at, end_at, lim, claimed, redeemed)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO UPDATE SET
			title=EXCLUDED.title, type=EXCLUDED.type, value=EXCLUDED.value,
			status=EXCLUDED.status, start_at=EXCLUDED.start_at, end_at=EXCLUDED.end_at,
			lim=EXCLUDED.lim, claimed=EXCLUDED.claimed, redeemed=EXCLUDED.redeemed`,
		o.ID, o.VendorID, o.Title, o.Type, o.Value, o.Status, o.StartAt, o.EndAt,
		o.Limit, o.Claimed, o.Redeemed)
	return err
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
	return out, rows.Err()
}

func (s *Store) OfferByID(ctx context.Context, id string) (domain.Offer, error) {
	return scanOffer(s.Pool.QueryRow(ctx, `SELECT `+offerCols+` FROM offers WHERE id=$1`, id))
}

func (s *Store) SetOfferStatus(ctx context.Context, id, vendorID string, status domain.OfferStatus) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE offers SET status=$3 WHERE id=$1 AND vendor_id=$2`, id, vendorID, status)
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

func (s *Store) Places(ctx context.Context) ([]domain.Place, error) {
	rows, err := s.Pool.Query(ctx, `SELECT id, name, vendor_id, data FROM places ORDER BY id`)
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
		`SELECT id, name, vendor_id, data FROM places WHERE id=$1`, id).
		Scan(&p.ID, &p.Name, &p.VendorID, &p.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// ---- claims ----

// claimCols joins traveller display fields for vendor-facing views.
const claimCols = `c.id, c.code, c.place_id, c.offer_label, c.offer_value, c.vendor_id, c.traveller_id, c.status, c.claimed_at, c.redeemed_at,
	COALESCE(u.name,''), COALESCE(u.country,''), COALESCE(u.travel_style,'')`
const claimFrom = ` FROM claims c LEFT JOIN users u ON u.id = c.traveller_id `

func scanClaim(row pgx.Row) (domain.Claim, error) {
	var c domain.Claim
	err := row.Scan(&c.ID, &c.Code, &c.PlaceID, &c.OfferLabel, &c.OfferValue,
		&c.VendorID, &c.TravellerID, &c.Status, &c.ClaimedAt, &c.RedeemedAt,
		&c.TravellerName, &c.TravellerCountry, &c.TravellerStyle)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

func (s *Store) CreateClaim(ctx context.Context, c domain.Claim) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO claims (id, code, place_id, offer_label, offer_value, vendor_id, traveller_id, status, claimed_at, redeemed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		c.ID, c.Code, c.PlaceID, c.OfferLabel, c.OfferValue, c.VendorID, c.TravellerID,
		c.Status, c.ClaimedAt, c.RedeemedAt)
	return err
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

// ---- vibes / checkins ----

func (s *Store) AddVibeVote(ctx context.Context, placeID, vibeID, travellerID string) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO vibe_votes (place_id, vibe_id, traveller_id) VALUES ($1,$2,$3)
		ON CONFLICT DO NOTHING`, placeID, vibeID, travellerID)
	return err
}

func (s *Store) VibeCounts(ctx context.Context, placeID string) (map[string]int, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT vibe_id, count(*) FROM vibe_votes WHERE place_id=$1 GROUP BY vibe_id`, placeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var id string
		var n int
		if err := rows.Scan(&id, &n); err != nil {
			return nil, err
		}
		out[id] = n
	}
	return out, rows.Err()
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

// ---- misc ----

// Wipe truncates all data tables (used by cmd/seed for idempotent reloads).
func (s *Store) Wipe(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `
		TRUNCATE users, vendors, offers, places, claims, trips, groups, feed_items,
			regions, subscriptions, audio_guides, vibe_votes, checkins,
			vendor_notifications, vendor_blobs, reviews, vendor_invites`)
	return err
}

func (s *Store) CountAdminKPIBase(ctx context.Context) (pendingVendors int, liveFeed int, err error) {
	if err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM vendors WHERE status='pending'`).Scan(&pendingVendors); err != nil {
		return
	}
	err = s.Pool.QueryRow(ctx, `SELECT count(*) FROM feed_items`).Scan(&liveFeed)
	return
}

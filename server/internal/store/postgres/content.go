package postgres

// Travel content (migration 049): posts written in the Command Center by
// admins and publishers, read by travellers once published.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/spotto/server/internal/domain"
)

// ErrContentState means the post isn't in a status that allows the change
// (e.g. publishing a post that's already live).
var ErrContentState = errors.New("post can't be changed in its current state")

// contentSelect is the shared SELECT; list views leave the body out.
func contentSelect(withBody bool) string {
	body := `''`
	if withBody {
		body = `c.body`
	}
	return `SELECT c.id, c.type, c.slug, c.title, c.summary, ` + body + `, c.cover_path, c.audio_path,
		c.audio_seconds, c.chapters, c.place_ids, c.days, c.tags, c.region_id, COALESCE(rg.name, ''),
		rg.lat, rg.lng, c.language, c.author_id, COALESCE(au.name, ''), c.status, c.review_note,
		c.reviewed_by, c.submitted_at, c.published_at, c.created_at, c.updated_at, c.views, c.plays
		FROM content_items c
		LEFT JOIN regions rg ON rg.id = c.region_id
		LEFT JOIN users au ON au.id = c.author_id`
}

func scanContent(row pgx.Row) (domain.ContentItem, error) {
	var c domain.ContentItem
	var chapters, placeIDs, days []byte
	err := row.Scan(&c.ID, &c.Type, &c.Slug, &c.Title, &c.Summary, &c.Body, &c.CoverURL, &c.AudioURL,
		&c.AudioSeconds, &chapters, &placeIDs, &days, &c.Tags, &c.RegionID, &c.RegionName,
		&c.RegionLat, &c.RegionLng, &c.Language, &c.AuthorID, &c.AuthorName, &c.Status, &c.ReviewNote,
		&c.ReviewedBy, &c.SubmittedAt, &c.PublishedAt, &c.CreatedAt, &c.UpdatedAt, &c.Views, &c.Plays)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, ErrNotFound
	}
	if err != nil {
		return c, err
	}
	_ = json.Unmarshal(chapters, &c.Chapters)
	_ = json.Unmarshal(placeIDs, &c.PlaceIDs)
	_ = json.Unmarshal(days, &c.Days)
	if c.Chapters == nil {
		c.Chapters = []domain.ContentChapter{}
	}
	if c.PlaceIDs == nil {
		c.PlaceIDs = []string{}
	}
	if c.Days == nil {
		c.Days = []domain.ItineraryDay{}
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	return c, nil
}

func contentJSON(c domain.ContentItem) (chapters, placeIDs, days []byte) {
	chapters, _ = json.Marshal(nonNil(c.Chapters))
	placeIDs, _ = json.Marshal(nonNil(c.PlaceIDs))
	days, _ = json.Marshal(nonNil(c.Days))
	return
}

func nonNil[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// CreateContent inserts a new post (always a draft).
func (s *Store) CreateContent(ctx context.Context, c domain.ContentItem) error {
	chapters, placeIDs, days := contentJSON(c)
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO content_items (id, type, slug, title, summary, body, audio_seconds, chapters,
			place_ids, days, tags, region_id, language, author_id, status, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,'draft',now(),now())`,
		c.ID, c.Type, c.Slug, c.Title, c.Summary, c.Body, c.AudioSeconds, chapters, placeIDs, days,
		nonNil(c.Tags), c.RegionID, c.Language, c.AuthorID)
	return err
}

// UpdateContent saves a post's editable fields. The slug only follows the
// title until the post is first published, so shared links keep working.
func (s *Store) UpdateContent(ctx context.Context, c domain.ContentItem) error {
	chapters, placeIDs, days := contentJSON(c)
	tag, err := s.Pool.Exec(ctx, `
		UPDATE content_items SET type=$2, slug=CASE WHEN published_at IS NULL THEN $3 ELSE slug END,
			title=$4, summary=$5, body=$6, audio_seconds=$7, chapters=$8, place_ids=$9, days=$10,
			tags=$11, region_id=$12, language=$13, updated_at=now()
		WHERE id=$1`,
		c.ID, c.Type, c.Slug, c.Title, c.Summary, c.Body, c.AudioSeconds, chapters, placeIDs, days,
		nonNil(c.Tags), c.RegionID, c.Language)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// SetContentMedia records an uploaded cover image or audio file ("cover"
// or "audio"), or clears it with an empty path.
func (s *Store) SetContentMedia(ctx context.Context, id, role, path string) error {
	col := map[string]string{"cover": "cover_path", "audio": "audio_path"}[role]
	if col == "" {
		return fmt.Errorf("content media role %q", role)
	}
	_, err := s.Pool.Exec(ctx, `UPDATE content_items SET `+col+`=$2, updated_at=now() WHERE id=$1`, id, path)
	return err
}

func (s *Store) ContentByID(ctx context.Context, id string) (domain.ContentItem, error) {
	return scanContent(s.Pool.QueryRow(ctx, contentSelect(true)+` WHERE c.id=$1`, id))
}

// PublishedContent is a live post by id or slug.
func (s *Store) PublishedContent(ctx context.Context, key string) (domain.ContentItem, error) {
	return scanContent(s.Pool.QueryRow(ctx,
		contentSelect(true)+` WHERE (c.id=$1 OR c.slug=$1) AND c.status='published'`, key))
}

type ContentFilter struct {
	AuthorID  string // only this author's posts
	Status    string
	Type      string
	RegionID  string
	Tag       string
	Q         string // title or summary contains
	Published bool   // only live posts, newest first
	Limit     int
	Offset    int
}

func (f ContentFilter) where() (string, []any) {
	var conds []string
	var args []any
	// Each ? in cond is the one argument v.
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}
	if f.Published {
		conds = append(conds, `c.status='published'`)
	}
	if f.AuthorID != "" {
		add(`c.author_id=?`, f.AuthorID)
	}
	if f.Status != "" {
		add(`c.status=?`, f.Status)
	}
	if f.Type != "" {
		add(`c.type=?`, f.Type)
	}
	if f.RegionID != "" {
		add(`c.region_id=?`, f.RegionID)
	}
	if f.Tag != "" {
		add(`? = ANY(c.tags)`, f.Tag)
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		add(`(c.title ILIKE '%' || ? || '%' OR c.summary ILIKE '%' || ? || '%')`, q)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ListContent lists posts without their bodies. Live posts come newest
// published first; the Command Center's list is most recently edited first.
func (s *Store) ListContent(ctx context.Context, f ContentFilter) ([]domain.ContentItem, error) {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 500
	}
	where, args := f.where()
	order := ` ORDER BY c.updated_at DESC`
	if f.Published {
		order = ` ORDER BY c.published_at DESC, c.id`
	}
	args = append(args, f.Limit, f.Offset)
	rows, err := s.Pool.Query(ctx, contentSelect(false)+where+order+
		fmt.Sprintf(` LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ContentItem{}
	for rows.Next() {
		c, err := scanContent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ContentStatusCounts counts posts by status (one author's, or everyone's
// when authorID is empty), for the Command Center's tabs.
func (s *Store) ContentStatusCounts(ctx context.Context, authorID string) (map[string]int, error) {
	rows, err := s.Pool.Query(ctx,
		`SELECT status, count(*)::int FROM content_items WHERE $1 = '' OR author_id = $1 GROUP BY status`, authorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, err
		}
		out[st] = n
	}
	return out, rows.Err()
}

// TransitionContent moves a post to status `to` if it's currently in one of
// `from`, and returns it. Submitting stamps submitted_at; publishing stamps
// published_at (kept from the first time) and the reviewer; rejecting
// stores the note and the reviewer. The note is cleared otherwise.
func (s *Store) TransitionContent(ctx context.Context, id string, from []string, to, actorID, note string) (domain.ContentItem, error) {
	tag, err := s.Pool.Exec(ctx, `
		UPDATE content_items SET status=$3, review_note=$5,
			submitted_at = CASE WHEN $3='in_review' THEN now() ELSE submitted_at END,
			published_at = CASE WHEN $3='published' THEN COALESCE(published_at, now()) ELSE published_at END,
			reviewed_by = CASE WHEN $3 IN ('published','rejected') THEN $4 ELSE reviewed_by END,
			updated_at = now()
		WHERE id=$1 AND status = ANY($2)`, id, from, to, actorID, note)
	if err != nil {
		return domain.ContentItem{}, err
	}
	if tag.RowsAffected() == 0 {
		if _, err := s.ContentByID(ctx, id); err != nil {
			return domain.ContentItem{}, err
		}
		return domain.ContentItem{}, ErrContentState
	}
	return s.ContentByID(ctx, id)
}

func (s *Store) DeleteContent(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM content_items WHERE id=$1`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CountContentInReview is how many posts wait for an admin.
func (s *Store) CountContentInReview(ctx context.Context) (int, error) {
	var n int
	err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM content_items WHERE status='in_review'`).Scan(&n)
	return n, err
}

// CountContentView / CountContentPlay add one view or play to a live post.
func (s *Store) CountContentView(ctx context.Context, id string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE content_items SET views=views+1 WHERE id=$1 AND status='published'`, id)
	return err
}

func (s *Store) CountContentPlay(ctx context.Context, id string) error {
	tag, err := s.Pool.Exec(ctx,
		`UPDATE content_items SET plays=plays+1 WHERE id=$1 AND status='published' AND type='audio'`, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

const contentPlaceCols = `p.id, p.name, COALESCE(NULLIF(p.data->>'type', ''), p.data->>'categoryId', ''),
	COALESCE((p.data->>'lat')::float8, 0), COALESCE((p.data->>'lng')::float8, 0),
	COALESCE(NULLIF(p.data->>'coverThumb', ''), p.data->>'cover', '')`

func scanContentPlaces(rows pgx.Rows) ([]domain.ContentPlace, error) {
	defer rows.Close()
	out := []domain.ContentPlace{}
	for rows.Next() {
		var p domain.ContentPlace
		if err := rows.Scan(&p.ID, &p.Name, &p.Category, &p.Lat, &p.Lng, &p.Cover); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ContentPlaces resolves linked place ids to visible places, in the order
// given; ids that no longer resolve are dropped.
func (s *Store) ContentPlaces(ctx context.Context, ids []string) ([]domain.ContentPlace, error) {
	if len(ids) == 0 {
		return []domain.ContentPlace{}, nil
	}
	rows, err := s.Pool.Query(ctx, `SELECT `+contentPlaceCols+` FROM places p
		JOIN unnest($1::text[]) WITH ORDINALITY AS want(id, n) ON want.id = p.id
		WHERE `+placeVisibleWhere+` ORDER BY want.n`, ids)
	if err != nil {
		return nil, err
	}
	return scanContentPlaces(rows)
}

// SearchPlacesByName finds visible places whose name contains q, for the
// Command Center's place picker.
func (s *Store) SearchPlacesByName(ctx context.Context, q string, limit int) ([]domain.ContentPlace, error) {
	rows, err := s.Pool.Query(ctx, `SELECT `+contentPlaceCols+` FROM places p
		WHERE p.name ILIKE '%' || $1 || '%' AND `+placeVisibleWhere+`
		ORDER BY (lower(p.name) = lower($1)) DESC, length(p.name), p.name LIMIT $2`, q, limit)
	if err != nil {
		return nil, err
	}
	return scanContentPlaces(rows)
}

package httpapi

// Travel content (audio guides and audiobooks, blog posts, travel guides,
// itineraries and tips). Admins and publishers write it in the Command
// Center under /studio; a publisher's post goes live only once an admin
// publishes it (/admin/content); travellers and the website read live posts
// from /content without signing in.

import (
	"errors"
	"log"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// RequirePublisher lets admins and accounts an admin made a publisher
// through. Like RequireSuperAdmin it reads the flag from the database, so
// turning it off takes effect on the next request.
func (s *Server) RequirePublisher(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := auth.FromContext(r.Context())
		if c == nil {
			writeErr(w, http.StatusForbidden, "publishers only")
			return
		}
		if c.Role != domain.RoleAdmin {
			u, err := s.Store.UserByID(r.Context(), c.Subject)
			if err != nil || !u.Publisher {
				writeErr(w, http.StatusForbidden, "publishers only; ask an admin to make you a publisher")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// authorCanEdit: a publisher can change their post while it's a draft,
// was sent back, or was taken down — not while it waits for review or is
// live. Admins can edit any post.
func authorCanEdit(status string) bool {
	return status == domain.ContentDraft || status == domain.ContentRejected || status == domain.ContentArchived
}

// studioItem loads a post for the caller: admins see every post, publishers
// only their own (anyone else's reads as not found).
func (s *Server) studioItem(w http.ResponseWriter, r *http.Request) (domain.ContentItem, *auth.Claims, bool) {
	c := auth.FromContext(r.Context())
	item, err := s.Store.ContentByID(r.Context(), chi.URLParam(r, "id"))
	if err == nil && c.Role != domain.RoleAdmin && (item.AuthorID == nil || *item.AuthorID != c.Subject) {
		err = postgres.ErrNotFound
	}
	if err != nil {
		storeErr(w, err)
		return item, c, false
	}
	return item, c, true
}

// contentInput is what the editor sends on create and save.
type contentInput struct {
	Type         string                  `json:"type"`
	Title        string                  `json:"title"`
	Summary      string                  `json:"summary"`
	Body         string                  `json:"body"`
	AudioSeconds int                     `json:"audioSeconds"`
	Chapters     []domain.ContentChapter `json:"chapters"`
	PlaceIDs     []string                `json:"placeIds"`
	Days         []domain.ItineraryDay   `json:"days"`
	Tags         []string                `json:"tags"`
	RegionID     string                  `json:"regionId"`
	Language     string                  `json:"language"`
}

var clockTime = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// apply validates in and copies it onto item, or returns a message for the
// editor.
func (s *Server) applyContentInput(r *http.Request, item *domain.ContentItem, in contentInput) string {
	if !slices.Contains(domain.ContentTypes, in.Type) {
		return "Choose what kind of post this is."
	}
	title := sanitizeText(in.Title, 0)
	if title == "" {
		return "Give the post a title."
	}
	if len([]rune(title)) > 160 {
		return "Keep the title under 160 characters."
	}
	summary := sanitizeText(in.Summary, 0)
	if len([]rune(summary)) > 400 {
		return "Keep the summary under 400 characters."
	}
	if len(in.Body) > 100_000 {
		return "The text is too long (100,000 characters at most)."
	}
	if in.AudioSeconds < 0 || in.AudioSeconds > 24*3600 {
		return "The audio length doesn't look right."
	}
	if len(in.Chapters) > 100 {
		return "Up to 100 chapters."
	}
	chapters := make([]domain.ContentChapter, 0, len(in.Chapters))
	for _, ch := range in.Chapters {
		ch.Title = sanitizeText(ch.Title, 120)
		if ch.Title == "" {
			return "Every chapter needs a title."
		}
		if ch.StartSec < 0 || (in.AudioSeconds > 0 && ch.StartSec > in.AudioSeconds) {
			return "A chapter starts outside the recording."
		}
		chapters = append(chapters, ch)
	}
	slices.SortStableFunc(chapters, func(a, b domain.ContentChapter) int { return a.StartSec - b.StartSec })

	placeIDs := []string{}
	for _, id := range in.PlaceIDs {
		if id = strings.TrimSpace(id); id != "" && !slices.Contains(placeIDs, id) {
			placeIDs = append(placeIDs, id)
		}
	}
	if len(placeIDs) > 50 {
		return "Link up to 50 places."
	}

	if len(in.Days) > 30 {
		return "Itineraries can be up to 30 days."
	}
	days := make([]domain.ItineraryDay, 0, len(in.Days))
	for i, d := range in.Days {
		d.Title = sanitizeText(d.Title, 120)
		if len(d.Stops) > 30 {
			return "Up to 30 stops a day."
		}
		stops := make([]domain.ItineraryStop, 0, len(d.Stops))
		for _, st := range d.Stops {
			st.PlaceID = strings.TrimSpace(st.PlaceID)
			st.Name = sanitizeText(st.Name, 120)
			st.Note = sanitizeText(st.Note, 500)
			st.Time = strings.TrimSpace(st.Time)
			if st.Name == "" {
				return "Day " + strconv.Itoa(i+1) + ": every stop needs a name."
			}
			if st.Time != "" && !clockTime.MatchString(st.Time) {
				return "Day " + strconv.Itoa(i+1) + ": write times as 09:30."
			}
			if (st.Lat == nil) != (st.Lng == nil) ||
				(st.Lat != nil && (*st.Lat < -90 || *st.Lat > 90 || *st.Lng < -180 || *st.Lng > 180)) {
				return "Day " + strconv.Itoa(i+1) + ": a stop's location doesn't look right."
			}
			stops = append(stops, st)
		}
		d.Stops = stops
		days = append(days, d)
	}

	tags := []string{}
	for _, t := range in.Tags {
		t = strings.ToLower(sanitizeText(strings.TrimPrefix(strings.TrimSpace(t), "#"), 30))
		if t != "" && !slices.Contains(tags, t) {
			tags = append(tags, t)
		}
	}
	if len(tags) > 10 {
		return "Up to 10 tags."
	}

	var regionID *string
	if id := strings.TrimSpace(in.RegionID); id != "" {
		regions, err := s.Store.Regions(r.Context())
		if err != nil {
			return "Couldn't check the town. Please try again."
		}
		if !slices.ContainsFunc(regions, func(rg domain.Region) bool { return rg.ID == id }) {
			return "That town isn't in the list."
		}
		regionID = &id
	}
	lang := strings.ToLower(strings.TrimSpace(in.Language))
	if lang == "" {
		lang = "en"
	}
	if len(lang) > 8 {
		return "The language doesn't look right."
	}

	item.Type, item.Title, item.Summary = in.Type, title, summary
	item.Body = unsignMediaPaths(strings.TrimSpace(in.Body))
	item.AudioSeconds, item.Chapters, item.PlaceIDs, item.Days = in.AudioSeconds, chapters, placeIDs, days
	item.Tags, item.RegionID, item.Language = tags, regionID, lang
	item.Slug = contentSlug(title, item.ID)
	return ""
}

// contentSlug is the post's URL name: the title in lowercase words plus the
// end of its id, which keeps it unique.
func contentSlug(title, id string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
		if b.Len() >= 60 {
			break
		}
	}
	slug := strings.Trim(b.String(), "-")
	suffix := id[strings.LastIndexByte(id, '_')+1:]
	if len(suffix) > 6 {
		suffix = suffix[len(suffix)-6:]
	}
	if slug == "" {
		return suffix
	}
	return slug + "-" + suffix
}

// contentReadyMsg is why a post can't be submitted or published yet, or "".
func contentReadyMsg(c domain.ContentItem) string {
	switch {
	case c.Title == "":
		return "Give the post a title first."
	case c.Type == domain.ContentAudio && c.AudioURL == "":
		return "Upload the audio file first."
	case (c.Type == domain.ContentBlog || c.Type == domain.ContentGuide) && c.Body == "":
		return "Write the post first."
	case c.Type == domain.ContentTip && c.Body == "" && c.Summary == "":
		return "Write the tip first."
	case c.Type == domain.ContentItinerary && !slices.ContainsFunc(c.Days, func(d domain.ItineraryDay) bool { return len(d.Stops) > 0 }):
		return "Add at least one stop to the itinerary first."
	}
	return ""
}

// rawMediaPath matches a "/media/raw/<key>" path in Markdown, with any view
// token a client copied along with it.
var rawMediaPath = regexp.MustCompile(`/media/raw/[A-Za-z0-9._/-]+(\?token=[A-Za-z0-9._-]*)?`)

// unsignMediaPaths drops view tokens from media links in a body before it's
// stored; signContent adds fresh ones when it's served.
func unsignMediaPaths(body string) string {
	return rawMediaPath.ReplaceAllStringFunc(body, func(m string) string {
		base, _, _ := strings.Cut(m, "?")
		return base
	})
}

// signContent signs the post's media links for the response.
func (s *Server) signContent(c domain.ContentItem) domain.ContentItem {
	c.CoverURL = s.signMediaPath(c.CoverURL)
	c.AudioURL = s.signMediaPath(c.AudioURL)
	if c.Body != "" {
		c.Body = rawMediaPath.ReplaceAllStringFunc(c.Body, s.signMediaPath)
	}
	for i := range c.Places {
		c.Places[i].Cover = s.signMediaPath(c.Places[i].Cover)
	}
	return c
}

// ---- studio: writing posts (admins and publishers) ----

func (s *Server) handleStudioContentList(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	q := r.URL.Query()
	f := postgres.ContentFilter{
		Status: q.Get("status"), Type: q.Get("type"), RegionID: q.Get("region"), Q: q.Get("q"),
	}
	if c.Role == domain.RoleAdmin {
		f.AuthorID = q.Get("author")
	} else {
		f.AuthorID = c.Subject
	}
	items, err := s.Store.ListContent(r.Context(), f)
	if err != nil {
		log.Printf("studio content list: %v", err)
		storeErr(w, err)
		return
	}
	countAuthor := ""
	if c.Role != domain.RoleAdmin {
		countAuthor = c.Subject
	}
	counts, err := s.Store.ContentStatusCounts(r.Context(), countAuthor)
	if err != nil {
		storeErr(w, err)
		return
	}
	for i := range items {
		items[i] = s.signContent(items[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "counts": counts})
}

func (s *Server) handleStudioContentCreate(w http.ResponseWriter, r *http.Request) {
	var in contentInput
	if !readJSON(w, r, &in) {
		return
	}
	c := auth.FromContext(r.Context())
	item := domain.ContentItem{ID: newID("cnt"), AuthorID: &c.Subject}
	if msg := s.applyContentInput(r, &item, in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.Store.CreateContent(r.Context(), item); err != nil {
		log.Printf("studio content create: %v", err)
		storeErr(w, err)
		return
	}
	created, err := s.Store.ContentByID(r.Context(), item.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s.signContent(created))
}

func (s *Server) handleStudioContentGet(w http.ResponseWriter, r *http.Request) {
	item, _, ok := s.studioItem(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.signContent(item))
}

func (s *Server) handleStudioContentUpdate(w http.ResponseWriter, r *http.Request) {
	item, c, ok := s.studioItem(w, r)
	if !ok {
		return
	}
	if c.Role != domain.RoleAdmin && !authorCanEdit(item.Status) {
		writeErr(w, http.StatusConflict, "This post is waiting for review or live, so it can't be changed now.")
		return
	}
	var in contentInput
	if !readJSON(w, r, &in) {
		return
	}
	if msg := s.applyContentInput(r, &item, in); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.Store.UpdateContent(r.Context(), item); err != nil {
		log.Printf("studio content update id=%s: %v", item.ID, err)
		storeErr(w, err)
		return
	}
	saved, err := s.Store.ContentByID(r.Context(), item.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s.signContent(saved))
}

func (s *Server) handleStudioContentDelete(w http.ResponseWriter, r *http.Request) {
	item, c, ok := s.studioItem(w, r)
	if !ok {
		return
	}
	if c.Role != domain.RoleAdmin && !authorCanEdit(item.Status) {
		writeErr(w, http.StatusConflict, "This post is waiting for review or live, so it can't be deleted now.")
		return
	}
	files, err := s.Store.MediaByOwner(r.Context(), "content", item.ID)
	if err != nil {
		storeErr(w, err)
		return
	}
	for _, m := range files {
		if s.Media != nil {
			_ = s.Media.Delete(r.Context(), m.ObjectKey)
			if m.ThumbKey != "" {
				_ = s.Media.Delete(r.Context(), m.ThumbKey)
			}
		}
		_ = s.Store.DeleteMedia(r.Context(), m.ID)
	}
	if err := s.Store.DeleteContent(r.Context(), item.ID); err != nil {
		storeErr(w, err)
		return
	}
	if c.Role == domain.RoleAdmin {
		s.logAdminAction(r.Context(), "content.delete", "content", item.ID, map[string]any{"title": item.Title})
	}
	writeJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// handleStudioContentSubmit sends a post to the admins for review.
func (s *Server) handleStudioContentSubmit(w http.ResponseWriter, r *http.Request) {
	item, c, ok := s.studioItem(w, r)
	if !ok {
		return
	}
	if msg := contentReadyMsg(item); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	updated, err := s.Store.TransitionContent(r.Context(), item.ID,
		[]string{domain.ContentDraft, domain.ContentRejected, domain.ContentArchived}, domain.ContentInReview, c.Subject, "")
	if !s.contentTransitionOK(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, s.signContent(updated))
}

func (s *Server) contentTransitionOK(w http.ResponseWriter, err error) bool {
	if errors.Is(err, postgres.ErrContentState) {
		writeErr(w, http.StatusConflict, "This post has changed since you opened it. Reload and try again.")
		return false
	}
	if err != nil {
		storeErr(w, err)
		return false
	}
	return true
}

// handleStudioRegions lists the towns a post can be about.
func (s *Server) handleStudioRegions(w http.ResponseWriter, r *http.Request) {
	regions, err := s.Store.Regions(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	type town struct {
		ID   string  `json:"id"`
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	}
	out := make([]town, 0, len(regions))
	for _, rg := range regions {
		out = append(out, town{rg.ID, rg.Name, rg.Lat, rg.Lng})
	}
	slices.SortFunc(out, func(a, b town) int { return strings.Compare(a.Name, b.Name) })
	writeJSON(w, http.StatusOK, out)
}

// handleStudioPlaces finds places to link (?q=, by name) or resolves the
// ones already linked (?ids=a,b).
func (s *Server) handleStudioPlaces(w http.ResponseWriter, r *http.Request) {
	var places []domain.ContentPlace
	var err error
	if ids := strings.TrimSpace(r.URL.Query().Get("ids")); ids != "" {
		list := strings.Split(ids, ",")
		if len(list) > 100 {
			list = list[:100]
		}
		places, err = s.Store.ContentPlaces(r.Context(), list)
	} else {
		q := strings.TrimSpace(r.URL.Query().Get("q"))
		if len([]rune(q)) < 2 {
			writeJSON(w, http.StatusOK, []domain.ContentPlace{})
			return
		}
		places, err = s.Store.SearchPlacesByName(r.Context(), sanitizeText(q, 80), 20)
	}
	if err != nil {
		storeErr(w, err)
		return
	}
	for i := range places {
		places[i].Cover = s.signMediaPath(places[i].Cover)
	}
	writeJSON(w, http.StatusOK, places)
}

// ---- admin: reviewing posts ----

func (s *Server) handleAdminContentPublish(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	item, err := s.Store.ContentByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		storeErr(w, err)
		return
	}
	if msg := contentReadyMsg(item); msg != "" {
		writeErr(w, http.StatusBadRequest, msg)
		return
	}
	// An admin's own post can go live straight away; someone else's once
	// they've submitted it (or again after it was taken down).
	from := []string{domain.ContentInReview, domain.ContentArchived}
	if item.AuthorID != nil && *item.AuthorID == c.Subject {
		from = append(from, domain.ContentDraft, domain.ContentRejected)
	} else if item.Status == domain.ContentDraft || item.Status == domain.ContentRejected {
		writeErr(w, http.StatusConflict, "The author hasn't sent this post for review yet.")
		return
	}
	updated, err := s.Store.TransitionContent(r.Context(), item.ID, from, domain.ContentPublished, c.Subject, "")
	if !s.contentTransitionOK(w, err) {
		return
	}
	s.logAdminAction(r.Context(), "content.publish", "content", item.ID, map[string]any{"title": item.Title, "type": item.Type})
	writeJSON(w, http.StatusOK, s.signContent(updated))
}

func (s *Server) handleAdminContentReject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Note string `json:"note"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	note := sanitizeText(req.Note, 1000)
	if note == "" {
		writeErr(w, http.StatusBadRequest, "Tell the author what to change.")
		return
	}
	c := auth.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	updated, err := s.Store.TransitionContent(r.Context(), id, []string{domain.ContentInReview}, domain.ContentRejected, c.Subject, note)
	if !s.contentTransitionOK(w, err) {
		return
	}
	s.logAdminAction(r.Context(), "content.reject", "content", id, map[string]any{"title": updated.Title, "note": note})
	writeJSON(w, http.StatusOK, s.signContent(updated))
}

func (s *Server) handleAdminContentUnpublish(w http.ResponseWriter, r *http.Request) {
	c := auth.FromContext(r.Context())
	id := chi.URLParam(r, "id")
	updated, err := s.Store.TransitionContent(r.Context(), id, []string{domain.ContentPublished}, domain.ContentArchived, c.Subject, "")
	if !s.contentTransitionOK(w, err) {
		return
	}
	s.logAdminAction(r.Context(), "content.unpublish", "content", id, map[string]any{"title": updated.Title})
	writeJSON(w, http.StatusOK, s.signContent(updated))
}

func (s *Server) handleAdminContentReviewCount(w http.ResponseWriter, r *http.Request) {
	n, err := s.Store.CountContentInReview(r.Context())
	if err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": n})
}

// handleAdminSetUserPublisher lets an account write content, or stops it.
// Any admin can; it's recorded in the audit log.
func (s *Server) handleAdminSetUserPublisher(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Publisher bool   `json:"publisher"`
		Reason    string `json:"reason"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	reason := sanitizeText(req.Reason, 500)
	if reason == "" {
		writeErr(w, http.StatusBadRequest, "Add a short reason.")
		return
	}
	id := chi.URLParam(r, "id")
	u, err := s.Store.UserByID(r.Context(), id)
	if err != nil {
		storeErr(w, err)
		return
	}
	if err := s.Store.SetUserPublisher(r.Context(), id, req.Publisher); err != nil {
		storeErr(w, err)
		return
	}
	s.logAdminAction(r.Context(), "user.publisher", "user", id, map[string]any{
		"email": u.Email, "publisher": req.Publisher, "reason": reason,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"publisher": req.Publisher})
}

// ---- public: reading live posts ----

// publicContent hides the review details from travellers.
func publicContent(c domain.ContentItem) domain.ContentItem {
	c.ReviewNote, c.ReviewedBy, c.SubmittedAt, c.AuthorID = "", nil, nil, nil
	return c
}

// publicCORS lets any website read live posts: they're public and the
// requests carry no credentials, so the site's address needn't be listed in
// CORS_ORIGINS.
func publicCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleContentList(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	offset, _ := strconv.Atoi(q.Get("offset"))
	if offset < 0 {
		offset = 0
	}
	items, err := s.Store.ListContent(r.Context(), postgres.ContentFilter{
		Published: true, Type: q.Get("type"), RegionID: q.Get("region"), Tag: strings.ToLower(q.Get("tag")),
		Q: sanitizeText(q.Get("q"), 80), Limit: limit + 1, Offset: offset,
	})
	if err != nil {
		log.Printf("content list: %v", err)
		storeErr(w, err)
		return
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	for i := range items {
		items[i] = publicContent(s.signContent(items[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "hasMore": hasMore})
}

// handleContentDetail is one live post by id or slug, with its linked
// places. Each read counts as a view.
func (s *Server) handleContentDetail(w http.ResponseWriter, r *http.Request) {
	item, err := s.Store.PublishedContent(r.Context(), chi.URLParam(r, "key"))
	if err != nil {
		storeErr(w, err)
		return
	}
	ids := slices.Clone(item.PlaceIDs)
	for _, d := range item.Days {
		for _, st := range d.Stops {
			if st.PlaceID != "" && !slices.Contains(ids, st.PlaceID) {
				ids = append(ids, st.PlaceID)
			}
		}
	}
	if item.Places, err = s.Store.ContentPlaces(r.Context(), ids); err != nil {
		storeErr(w, err)
		return
	}
	if err := s.Store.CountContentView(r.Context(), item.ID); err != nil {
		log.Printf("content view id=%s: %v", item.ID, err)
	}
	writeJSON(w, http.StatusOK, publicContent(s.signContent(item)))
}

// handleContentPlay counts a listen of a live audio post.
func (s *Server) handleContentPlay(w http.ResponseWriter, r *http.Request) {
	if err := s.Store.CountContentPlay(r.Context(), chi.URLParam(r, "id")); err != nil {
		storeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

package seed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

const demoPassword = "spotto123"

func ptr[T any](v T) *T { return &v }

// Run wipes all tables and reloads the demo dataset. Idempotent.
func Run(ctx context.Context, s *postgres.Store) error {
	if err := s.Wipe(ctx); err != nil {
		return fmt.Errorf("wipe: %w", err)
	}
	now := time.Now()
	hash, err := auth.HashPassword(demoPassword)
	if err != nil {
		return err
	}
	// ── Vendor Categories ────────────────────────────────────────────────────────
	for _, c := range InitialCategories {
		if err := s.CreateVendorCategory(ctx, c); err != nil {
			return fmt.Errorf("create category %s: %w", c.ID, err)
		}
	}

	// Update domain memory cache so validation passes during the rest of the seed run
	cats, err := s.GetVendorCategories(ctx)
	if err != nil {
		return fmt.Errorf("get categories: %w", err)
	}
	domain.InitVendorCategories(cats)

	// ── Vendors ──────────────────────────────────────────────────────────
	driftwoodID := "ven_driftwood"
	if err := s.UpsertVendor(ctx, domain.Vendor{
		ID: driftwoodID, Name: "Driftwood Coffee & Co-Work", Category: "cafe",
		Town: "Weligama", RegionID: ptr("weligama"), Owner: "Tharindu Perera",
		Status: domain.VendorApproved, Priority: "normal", Docs: true, Photos: 9,
		WifiScore: ptr(9.2), Flags: []string{}, Tier: "Premium",
		Profile: json.RawMessage(driftwoodProfile), Menu: json.RawMessage(driftwoodMenu),
		SubmittedAt: now.AddDate(0, -6, 0),
		Lat:         5.9731, Lng: 80.4280,
	}); err != nil {
		return fmt.Errorf("driftwood vendor: %w", err)
	}

	// VENDOR_QUEUE from admin-data.jsx — pending approvals.
	queue := []struct {
		id, name, cat, town, owner, note, priority, region string
		agoMin                                             int
		docs                                               bool
		photos                                             int
		wifi                                               *float64
		flags                                              []string
		lat, lng                                           float64
	}{
		{"V-2041", "Ahangama Yoga Studio", "Yoga / Wellness", "Ahangama", "Nimali Fernando",
			"Sunrise rooftop shala, 200hr certified instructors. Requesting Premium.", "high", "south", 12, true, 6, ptr(7.4), []string{}, 5.9730, 80.3597},
		{"V-2040", "Hiriketiya Poke Bar", "Restaurant", "Dickwella", "Sea & Co Ltd",
			"Beachfront poke & smoothies. Shares plot with registered surf shop.", "normal", "south", 38, true, 4, ptr(6.1), []string{"Duplicate address?"}, 5.9667, 80.7000},
		{"V-2039", "Ella Rock Trail Guides", "Tours / Activity", "Ella", "Kasun Bandara",
			"Guided sunrise hikes. License upload pending.", "normal", "hill", 60, false, 2, nil, []string{"Missing business license"}, 6.8667, 81.0466},
		{"V-2038", "Fort Print & Coffee", "Coworking Café", "Galle Fort", "Dilan Jayawardene",
			"Heritage-building café with fibre internet. Strong nomad fit.", "high", "south", 120, true, 9, ptr(9.0), []string{}, 6.0269, 80.2168},
		{"V-2037", "Arugam Board Co.", "Surf School", "Arugam Bay", "Wave Theory",
			"Board rental + lessons on the east coast point break.", "normal", "east", 180, true, 5, ptr(5.2), []string{}, 6.8404, 81.8360},
		{"V-2036", "Kandy Spice Kitchen", "Restaurant", "Kandy", "Heritage Foods",
			"Family curry house near the temple.", "low", "hill", 300, true, 3, ptr(6.8), []string{"Low photo quality"}, 7.2906, 80.6337},
	}
	for _, q := range queue {
		if err := s.UpsertVendor(ctx, domain.Vendor{
			ID: q.id, Name: q.name, Category: q.cat, Town: q.town, RegionID: ptr(q.region),
			Owner: q.owner, Status: domain.VendorPending, Priority: q.priority,
			Docs: q.docs, Photos: q.photos, WifiScore: q.wifi, Flags: q.flags,
			Note: q.note, Tier: "Free",
			SubmittedAt: now.Add(-time.Duration(q.agoMin) * time.Minute),
			Lat:         q.lat, Lng: q.lng,
		}); err != nil {
			return fmt.Errorf("queue vendor %s: %w", q.id, err)
		}
	}

	// ── Users ────────────────────────────────────────────────────────────
	type demoUser struct {
		id, email, name, country, style string
		role                            domain.Role
		vendorID                        *string
		interests                       []string
	}
	users := []demoUser{
		{"usr_lena", "lena@spotto.dev", "Lena M.", "Germany", "nomad", domain.RoleTraveller, nil, []string{"coffee", "coworking", "surf"}},
		{"usr_tom", "tom@spotto.dev", "Tom R.", "UK", "surf", domain.RoleTraveller, nil, []string{"surf", "beach"}},
		{"usr_sara", "sara@spotto.dev", "Sara P.", "Spain", "nomad", domain.RoleTraveller, nil, []string{"coffee", "coworking"}},
		{"usr_marco", "marco@spotto.dev", "Marco", "Italy", "surf", domain.RoleTraveller, nil, []string{"surf"}},
		{"usr_aiko", "aiko@spotto.dev", "Aiko", "Japan", "nomad", domain.RoleTraveller, nil, []string{"coffee"}},
		{"usr_sam", "sam@spotto.dev", "Sam", "Australia", "backpacker", domain.RoleTraveller, nil, []string{"beach"}},
		{"usr_chloe", "chloe@spotto.dev", "Chloé", "France", "luxury", domain.RoleTraveller, nil, []string{"food"}},
		{"usr_vendor", "vendor@driftwood.dev", "Tharindu Perera", "Sri Lanka", "", domain.RoleVendor, &driftwoodID, nil},
		{"usr_admin", "admin@spotto.dev", "Spotto Admin", "Sri Lanka", "", domain.RoleAdmin, nil, nil},
	}
	for _, u := range users {
		var style []string
		if u.style != "" {
			style = []string{u.style}
		}
		if err := s.CreateUser(ctx, domain.User{
			ID: u.id, Email: u.email, PasswordHash: hash, Role: u.role, Name: u.name,
			Country: u.country, TravelStyle: style, Interests: u.interests,
			VendorID: u.vendorID, CreatedAt: now.AddDate(0, -1, 0),
		}); err != nil {
			return fmt.Errorf("user %s: %w", u.email, err)
		}
	}

	// ── Places ───────────────────────────────────────────────────────────
	places := []domain.Place{
		{ID: "driftwood", Name: "Driftwood Coffee & Co-Work", VendorID: &driftwoodID, Data: json.RawMessage(placeDriftwoodData)},
		{ID: "baysurf", Name: "Weligama Bay Surf School", Data: json.RawMessage(placeBaysurfData)},
		{ID: "hangtime", Name: "Hangtime Rooftop & Sunset Bar", Data: json.RawMessage(placeHangtimeData)},
		{ID: "claystudio", Name: "Clay & Co Pottery Studio", Data: json.RawMessage(placeClaystudioData)},
		{ID: "batik", Name: "Sea Salt Batik Atelier", Data: json.RawMessage(placeBatikData)},
		// Wider-Weligama-area trip-planner stops — no vendor account, not
		// curated Spotto businesses, just points of interest for the trip
		// planner catalog (see data.go's comment above these blobs).
		{ID: "bayvilla", Name: "Bay Villa Boutique Stay", Data: json.RawMessage(placeBayvillaData)},
		{ID: "palmhostel", Name: "Palm & Board Hostel", Data: json.RawMessage(placePalmhostelData)},
		{ID: "welibeach", Name: "Weligama Beach Point", Data: json.RawMessage(placeWelibeachData)},
		{ID: "mirissa", Name: "Mirissa Beach", Data: json.RawMessage(placeMirissaData)},
		{ID: "coconuthill", Name: "Coconut Tree Hill", Data: json.RawMessage(placeCoconuthillData)},
		{ID: "kottu", Name: "Smoky Kottu Shack", Data: json.RawMessage(placeKottuData)},
		{ID: "reefdine", Name: "Reef & Vine Dining", Data: json.RawMessage(placeReefdineData)},
		{ID: "secretroot", Name: "Secret Root Spa", Data: json.RawMessage(placeSecretrootData)},
		{ID: "taprobane", Name: "Taprobane Island Viewpoint", Data: json.RawMessage(placeTaprobaneData)},
		{ID: "oldlight", Name: "Old Light Gallery Bar", Data: json.RawMessage(placeOldlightData)},
		{ID: "morningmkt", Name: "Weligama Morning Market", Data: json.RawMessage(placeMorningmktData)},
	}
	for _, p := range places {
		if err := s.UpsertPlace(ctx, p); err != nil {
			return fmt.Errorf("place %s: %w", p.ID, err)
		}
	}

	// ── Offers (OFFERS from vendor-data.jsx) ─────────────────────────────
	offers := []domain.Offer{
		{ID: "o1", VendorID: driftwoodID, Title: "Free flat white with any day pass", Type: "Freebie", Value: "Free coffee",
			Status: domain.OfferLive, StartAt: "Today 07:00", EndAt: "Today 17:00", Claimed: 34, Limit: 50, Redeemed: 21},
		{ID: "o2", VendorID: driftwoodID, Title: "20% off all single-origin pour-overs", Type: "Percentage", Value: "20% off",
			Status: domain.OfferLive, StartAt: "Mon", EndAt: "Sun", Claimed: 88, Limit: 200, Redeemed: 61},
		{ID: "o3", VendorID: driftwoodID, Title: "Buy 5 day-passes, get 1 free", Type: "Bundle", Value: "6-for-5",
			Status: domain.OfferEnding, StartAt: "This week", EndAt: "Today 23:59", Claimed: 47, Limit: 50, Redeemed: 39},
		{ID: "o4", VendorID: driftwoodID, Title: "Rainy-day bundle: coffee + covered desk", Type: "Bundle", Value: "2-for-1",
			Status: domain.OfferDraft, StartAt: "—", EndAt: "—", Claimed: 0, Limit: 40, Redeemed: 0},
		{ID: "o5", VendorID: driftwoodID, Title: "Full-moon night: free ginger tea", Type: "Freebie", Value: "Free tea",
			Status: domain.OfferEnded, StartAt: "Last Fri", EndAt: "Last Sat", Claimed: 50, Limit: 50, Redeemed: 44},
	}
	for _, o := range offers {
		if err := s.UpsertOffer(ctx, o); err != nil {
			return fmt.Errorf("offer %s: %w", o.ID, err)
		}
	}

	// ── Claims: VALID_CODES (unredeemed) + REDEMPTIONS (redeemed) ────────
	claims := []domain.Claim{
		{ID: "clm_drif", Code: "SPOTTO-DRIF", PlaceID: "driftwood", OfferLabel: "Free flat white with any day pass",
			OfferValue: "Free coffee", VendorID: &driftwoodID, TravellerID: "usr_lena",
			Status: domain.ClaimClaimed, ClaimedAt: now.Add(-30 * time.Minute)},
		{ID: "clm_x4k9", Code: "SPOTTO-X4K9", PlaceID: "driftwood", OfferLabel: "20% off all single-origin pour-overs",
			OfferValue: "20% off", VendorID: &driftwoodID, TravellerID: "usr_tom",
			Status: domain.ClaimClaimed, ClaimedAt: now.Add(-2 * time.Hour)},
		{ID: "clm_88q2", Code: "SPOTTO-88Q2", PlaceID: "driftwood", OfferLabel: "Buy 5 day-passes, get 1 free",
			OfferValue: "6-for-5", VendorID: &driftwoodID, TravellerID: "usr_sara",
			Status: domain.ClaimClaimed, ClaimedAt: now.Add(-3 * time.Hour)},
		{ID: "clm_9kq2", Code: "SPOTTO-9KQ2", PlaceID: "driftwood", OfferLabel: "20% off pour-over",
			OfferValue: "20% off", VendorID: &driftwoodID, TravellerID: "usr_marco",
			Status: domain.ClaimRedeemed, ClaimedAt: now.Add(-1 * time.Hour), RedeemedAt: ptr(now.Add(-18 * time.Minute))},
		{ID: "clm_7hx1", Code: "SPOTTO-7HX1", PlaceID: "driftwood", OfferLabel: "Free flat white",
			OfferValue: "Free coffee", VendorID: &driftwoodID, TravellerID: "usr_aiko",
			Status: domain.ClaimRedeemed, ClaimedAt: now.Add(-2 * time.Hour), RedeemedAt: ptr(now.Add(-41 * time.Minute))},
		{ID: "clm_2bvt", Code: "SPOTTO-2BVT", PlaceID: "driftwood", OfferLabel: "Day-pass bundle",
			OfferValue: "6-for-5", VendorID: &driftwoodID, TravellerID: "usr_sam",
			Status: domain.ClaimRedeemed, ClaimedAt: now.Add(-3 * time.Hour), RedeemedAt: ptr(now.Add(-1 * time.Hour))},
		{ID: "clm_mr48", Code: "SPOTTO-MR48", PlaceID: "driftwood", OfferLabel: "20% off pour-over",
			OfferValue: "20% off", VendorID: &driftwoodID, TravellerID: "usr_chloe",
			Status: domain.ClaimRedeemed, ClaimedAt: now.Add(-4 * time.Hour), RedeemedAt: ptr(now.Add(-2 * time.Hour))},
	}
	for _, c := range claims {
		if err := s.CreateClaim(ctx, c); err != nil {
			return fmt.Errorf("claim %s: %w", c.Code, err)
		}
	}

	// ── Group for Lena (groupSeed from places.dart) ──────────────────────
	// Members must be real user IDs (not the old mock traveler names) so
	// GET /me/group can resolve them to actual profiles.
	if err := s.UpsertGroup(ctx, domain.Group{
		OwnerID: "usr_lena", Name: "Weligama Crew", Code: "WLG-4K2",
		MemberIDs: []string{"usr_tom", "usr_sara", "usr_marco"},
	}); err != nil {
		return fmt.Errorf("group: %w", err)
	}

	// ── Feed seed (FEED_SEED from admin-data.jsx) ────────────────────────
	feed := []struct {
		vendor, town, typ, text, status, reason string
		agoMin                                  int
	}{
		{"Ella Gin Bar", "Ella", "Event", "Live acoustic night · free entry", "review", "New vendor · first post", 4},
		{"Mirissa Whale Co", "Mirissa", "Offer", "Dawn whale-watching · 15% off groups", "ok", "", 3},
		{"Lucky Tuk Tours", "Galle", "Offer", "CHEAPEST TOURS!!! 90% OFF CALL NOW", "flag", "Spammy formatting · unverifiable discount", 2},
		{"Hangtime Rooftop", "Weligama", "Event", "Full-moon DJ set · Saturday 9PM", "ok", "", 1},
		{"Driftwood Coffee", "Weligama", "Offer", "Free flat white with any day pass", "ok", "", 0},
	}
	for _, f := range feed {
		if _, err := s.InsertFeedItem(ctx, domain.FeedItem{
			Vendor: f.vendor, Town: f.town, Type: f.typ, Text: f.text,
			Status: domain.FeedStatus(f.status), Reason: f.reason,
			CreatedAt: now.Add(-time.Duration(f.agoMin) * time.Minute),
		}); err != nil {
			return fmt.Errorf("feed item: %w", err)
		}
	}

	// ── Regions (REGIONS from admin-data.jsx) ────────────────────────────
	regions := []domain.Region{
		{ID: "colombo", Name: "Colombo", X: 25, Y: 56, Lat: 6.9271, Lng: 79.8612, Travelers: 88, Vendors: 96},
		{ID: "negombo", Name: "Negombo", X: 23, Y: 49, Lat: 7.2083, Lng: 79.8358, Travelers: 54, Vendors: 60},
		{ID: "kandy", Name: "Kandy", X: 50, Y: 47, Lat: 7.2906, Lng: 80.6337, Travelers: 71, Vendors: 64},
		{ID: "sigiriya", Name: "Sigiriya", X: 54, Y: 34, Lat: 7.9570, Lng: 80.7603, Travelers: 66, Vendors: 28},
		{ID: "ella", Name: "Ella", X: 61, Y: 60, Lat: 6.8667, Lng: 81.0466, Travelers: 82, Vendors: 41},
		{ID: "nuwara", Name: "Nuwara Eliya", X: 54, Y: 56, Lat: 6.9497, Lng: 80.7891, Travelers: 58, Vendors: 39},
		{ID: "galle", Name: "Galle", X: 31, Y: 82, Lat: 6.0535, Lng: 80.2210, Travelers: 90, Vendors: 78},
		{ID: "weligama", Name: "Weligama", X: 39, Y: 86, Lat: 5.9724, Lng: 80.4292, Travelers: 95, Vendors: 70},
		{ID: "mirissa", Name: "Mirissa", X: 44, Y: 88, Lat: 5.9483, Lng: 80.4589, Travelers: 84, Vendors: 52},
		{ID: "arugam", Name: "Arugam Bay", X: 80, Y: 61, Lat: 6.8404, Lng: 81.8360, Travelers: 76, Vendors: 22},
		{ID: "trinco", Name: "Trincomalee", X: 70, Y: 27, Lat: 8.5874, Lng: 81.2152, Travelers: 61, Vendors: 18},
		{ID: "jaffna", Name: "Jaffna", X: 45, Y: 8, Lat: 9.6615, Lng: 80.0255, Travelers: 44, Vendors: 12},
	}
	for _, r := range regions {
		if err := s.UpsertRegion(ctx, r); err != nil {
			return fmt.Errorf("region %s: %w", r.ID, err)
		}
	}

	// ── Subscriptions (SUBS; "in N days"/"overdue" become timestamps) ────
	subs := []struct {
		id, vendor, town, tier, state, since string
		mrr                                  int
		renewDays                            *int
		vendorID                             *string
	}{
		{"V-1001", "Driftwood Coffee & Co-Work", "Weligama", "Premium", "soon", "Jan 2024", 49, ptr(4), &driftwoodID},
		{"V-1002", "Hangtime Rooftop & Bar", "Weligama", "Premium", "active", "Mar 2024", 49, ptr(21), nil},
		{"V-1003", "Weligama Bay Surf School", "Weligama", "Premium", "active", "Feb 2024", 49, ptr(12), nil},
		{"V-1004", "Mirissa Whale Co.", "Mirissa", "Free", "trial", "Aug 2024", 0, nil, nil},
		{"V-1005", "Ella Gin Bar", "Ella", "Free", "free", "Sep 2024", 0, nil, nil},
		{"V-1006", "Fort Print & Coffee", "Galle", "Premium", "soon", "Apr 2024", 49, ptr(2), nil},
		{"V-1007", "Unawatuna Dive Center", "Unawatuna", "Premium", "overdue", "Nov 2023", 49, ptr(-3), nil},
		{"V-1008", "Kandy Spice Kitchen", "Kandy", "Free", "free", "Jul 2024", 0, nil, nil},
		{"V-1009", "Arugam Board Co.", "Arugam Bay", "Free", "trial", "Oct 2024", 0, nil, nil},
		{"V-1010", "Colombo Roasters", "Colombo", "Premium", "active", "Dec 2023", 49, ptr(30), nil},
	}
	for _, sub := range subs {
		var renews *time.Time
		if sub.renewDays != nil {
			renews = ptr(now.AddDate(0, 0, *sub.renewDays))
		}
		if err := s.UpsertSubscription(ctx, domain.Subscription{
			ID: sub.id, VendorID: sub.vendorID, Vendor: sub.vendor, Town: sub.town,
			Tier: sub.tier, MRR: sub.mrr, RenewsAt: renews, State: sub.state, Since: sub.since,
		}); err != nil {
			return fmt.Errorf("subscription %s: %w", sub.id, err)
		}
	}

	// ── Audio guides (AUDIO_GUIDES from admin-data.jsx) ──────────────────
	guides := []domain.AudioGuide{
		{ID: "AG-01", Site: "Galle Fort Ramparts", Region: "Galle", Lat: 6.0269, Lng: 80.2168, Lang: "EN",
			Published: true, Plays: 3120,
			Depths:   json.RawMessage(`["Quick","Standard","In-depth"]`),
			Duration: json.RawMessage(`{"Quick":"4m","Standard":"11m","In-depth":"28m"}`)},
		{ID: "AG-02", Site: "Sigiriya Rock Fortress", Region: "Sigiriya", Lat: 7.9570, Lng: 80.7603, Lang: "EN",
			Published: true, Plays: 2410,
			Depths:   json.RawMessage(`["Quick","Standard"]`),
			Duration: json.RawMessage(`{"Quick":"5m","Standard":"14m"}`)},
		{ID: "AG-03", Site: "Temple of the Tooth", Region: "Kandy", Lat: 7.2936, Lng: 80.6413, Lang: "EN",
			Published: false, Plays: 0,
			Depths:   json.RawMessage(`["Standard","In-depth"]`),
			Duration: json.RawMessage(`{"Standard":"12m","In-depth":"31m"}`)},
		{ID: "AG-04", Site: "Dutch Hospital Precinct", Region: "Colombo", Lat: 6.9330, Lng: 79.8430, Lang: "EN",
			Published: true, Plays: 880,
			Depths:   json.RawMessage(`["Quick"]`),
			Duration: json.RawMessage(`{"Quick":"3m"}`)},
	}
	for _, g := range guides {
		if err := s.UpsertAudioGuide(ctx, g); err != nil {
			return fmt.Errorf("audio guide %s: %w", g.ID, err)
		}
	}

	// ── Vendor dashboard/growth/reports blobs + notifications ────────────
	blobs := map[string]string{
		"dashboard": driftwoodDashboard,
		"growth":    driftwoodGrowth,
		"reports":   driftwoodReports,
	}
	for kind, data := range blobs {
		if err := s.UpsertVendorBlob(ctx, driftwoodID, kind, json.RawMessage(data)); err != nil {
			return fmt.Errorf("blob %s: %w", kind, err)
		}
	}

	notifs := []domain.VendorNotification{
		{ID: "n1", Icon: "Ticket", Tone: "teal", Title: "New redemption", Sub: "Marco redeemed 20% off pour-over", Time: "18 min ago", Unread: true},
		{ID: "n2", Icon: "TrendingUp", Tone: "sun", Title: "Weekly report ready", Sub: "Redemptions up 18% vs last week", Time: "2 hr ago", Unread: true},
		{ID: "n3", Icon: "Star", Tone: "coral", Title: "Offer ending soon", Sub: "Buy 5 day-passes ends today 23:59", Time: "5 hr ago", Unread: false},
	}
	for _, n := range notifs {
		if err := s.UpsertVendorNotification(ctx, driftwoodID, n); err != nil {
			return fmt.Errorf("notification %s: %w", n.ID, err)
		}
	}

	// A minimal place for the "Hiriketiya Poke Bar" queue vendor (V-2040) so
	// it can carry demo bad reviews — enough for the admin map's red-flag
	// threshold, showing off the pinpoint-a-problem-vendor flow distinctly
	// from the healthy, approved Driftwood vendor.
	hiriketiyaVendorID := "V-2040"
	if err := s.UpsertPlace(ctx, domain.Place{
		ID: "hiriketiya", Name: "Hiriketiya Poke Bar", VendorID: &hiriketiyaVendorID,
		Data: json.RawMessage(`{"type":"Restaurant","icon":"UtensilsCrossed","color":"coral","x":50,"y":50,"lat":5.9667,"lng":80.7000,"rating":2.1,"reviews":5,"distance":"1.2 km","priceLevel":"$$","hours":"11:00 AM – 9:00 PM","openNow":true,"bestTimes":["afternoon"],"tags":["Poke","Smoothies"],"blurb":"Beachfront poke & smoothie bar.","seaView":true,"photo":"surf"}`),
	}); err != nil {
		return fmt.Errorf("hiriketiya place: %w", err)
	}
	badReviews := []struct {
		id, traveller, text string
		rating              int
	}{
		{"rev1", "usr_tom", "Waited 45 minutes and the poke bowl was warm, not chilled. Wouldn't return.", 1},
		{"rev2", "usr_sara", "Overcharged us compared to the menu board. Asked twice, never fixed.", 1},
		{"rev3", "usr_marco", "Smoothie tasted off, mild food poisoning after. Avoid.", 1},
		{"rev4", "usr_aiko", "Rude staff when we asked about allergens.", 2},
		{"rev5", "usr_sam", "Nice location but the fish wasn't fresh at all.", 2},
	}
	for i, br := range badReviews {
		if err := s.UpsertReview(ctx, domain.Review{
			ID: br.id, PlaceID: "hiriketiya", VendorID: &hiriketiyaVendorID,
			TravellerID: br.traveller, Rating: br.rating, Text: br.text,
			CreatedAt: now.Add(-time.Duration(i+1) * 24 * time.Hour),
		}); err != nil {
			return fmt.Errorf("bad review %s: %w", br.id, err)
		}
	}

	return nil
}

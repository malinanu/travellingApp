// Package gendata generates a large, varied, randomized dataset for manual
// testing. Unlike internal/seed (a small, fixed, hand-written demo used for
// a stable "lena@spotto.dev" login), every value here — names, emails,
// business types, coordinates, review text, offer numbers — is generated
// via gofakeit or weighted random choice, so no two runs produce the same
// content. It wipes the database first, same as seed.Run, so each run is
// one complete, internally-consistent randomized dataset.
package gendata

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"github.com/brianvoe/gofakeit/v7"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// TestPassword is shared by every generated account so a human can log in
// immediately after a run without querying the database for a hash.
const TestPassword = "Testing123!"

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O/1/I

// newID returns a random 12-char hex id with a prefix — a package-local
// copy of internal/http's helper (kept local rather than imported, so this
// data-generation package doesn't depend on the HTTP layer).
func newID(prefix string) string {
	b := make([]byte, 6)
	rand.Read(b)
	return fmt.Sprintf("%s_%x", prefix, b)
}

// uniqueEmail tags a fake address with a short random suffix — gofakeit's
// own randomness alone doesn't guarantee no collisions across the ~60
// accounts a run creates, and users.email has a UNIQUE constraint, so a
// rare collision would otherwise abort the whole run.
func uniqueEmail() string {
	local, domain, ok := strings.Cut(gofakeit.Email(), "@")
	if !ok {
		return newID("user") + "@spotto.test"
	}
	b := make([]byte, 3)
	rand.Read(b)
	return fmt.Sprintf("%s+%x@%s", local, b, domain)
}

func newCode(prefix string, n int) string {
	b := make([]byte, n)
	rand.Read(b)
	out := make([]byte, n)
	for i, v := range b {
		out[i] = codeAlphabet[int(v)%len(codeAlphabet)]
	}
	return prefix + string(out)
}

func ptr[T any](v T) *T { return &v }

var (
	travelStyles = []string{"nomad", "surf", "backpacker", "luxury"}
	interestPool = []string{"coffee", "coworking", "nightlife", "surf", "yoga", "food", "beach", "music"}

	poiTypes = []string{"Beach", "Viewpoint", "Local Market", "Street Food", "Hotel", "Hostel", "Restaurant", "Bar & Gallery"}

	placeIcons  = []string{"Coffee", "Waves", "Martini", "Palette", "Brush", "UtensilsCrossed", "BedDouble", "Backpack", "Flower2", "Mountain", "ShoppingBasket", "Wine"}
	placeColors = []string{"teal", "coral", "sun", "clay", "plum"}
	placeTags   = []string{"Fast Wi-Fi", "Power outlets", "Sea view", "Board rental", "Beginner friendly", "Live music", "Indoor", "Local artisan", "Hands-on", "Ocean view", "Quiet zone", "Air-con", "Vegan options"}

	menuItemNames = []string{
		"Coconut roti & curd bowl", "Jackfruit curry & rice", "Single-origin pour-over",
		"Arrack sour", "Hot butter cuttlefish", "Grilled catch of the day",
		"King coconut", "Isso vade plate", "Prawn kottu", "Ginger beer float",
		"Dhal & string hoppers", "Passionfruit smoothie",
	}

	reviewAspects    = []string{"the wifi", "the food", "the service", "the atmosphere", "the value for money", "the location", "the staff", "the coffee", "the view"}
	positivePhrases  = []string{"was fantastic", "exceeded expectations", "made the visit", "was a highlight", "was better than I hoped", "kept us coming back"}
	negativePhrases  = []string{"was disappointing", "needs work", "let the place down", "wasn't what we expected", "was frustrating"}
	neutralPhrases   = []string{"was okay", "was fine, nothing special", "did the job"}
	closingPositive  = []string{"Would definitely come back.", "Highly recommend.", "One of our favorite stops.", "Ask for the local's menu."}
	closingNegative  = []string{"Not sure I'd return.", "Might give it another chance.", "Expected more for the price."}
	vendorNoteFlags  = []string{"Missing business license", "Low photo quality", "Duplicate address?", ""}
	offerTypes       = []string{"Percentage", "Amount", "Freebie", "Bundle"}
	audioDepths      = []string{"Quick", "Standard", "In-depth"}
	audioDurationsBy = map[string]string{"Quick": "4m", "Standard": "12m", "In-depth": "28m"}

	// Curated content-template pools. gofakeit.Comment() was tried for all
	// of these initially and produces nonsensical, grammatically-broken
	// business-jargon text ("does this growth scale", "lean into the their
	// woman") when spliced into place/offer/feed content — a small but
	// readable pool of real phrases beats a larger pool of gibberish.
	blurbTemplates = []string{
		"A cozy neighborhood spot with a loyal local following and a laid-back vibe.",
		"Tucked just off the main road — perfect for a relaxed afternoon.",
		"A firm favorite with regulars; worth booking ahead on weekends.",
		"Fresh, simple, and consistently good — exactly what you want after a day outdoors.",
		"Family-run and full of character, with a menu that shifts with the season.",
		"Small but mighty — the kind of place locals recommend to friends.",
		"Right in the heart of town, easy to find and always buzzing.",
		"A quiet retreat with friendly staff and a genuinely warm welcome.",
	}
	dealLabelTemplates = []string{
		"Free item with any purchase over Rs 1000",
		"20% off for first-time visitors",
		"Happy hour discount, 4–6 PM daily",
		"Buy 2, get 1 free on selected items",
		"Free upgrade with any booking this week",
		"10% off for same-day bookings",
	}
	menuDescTemplates = []string{
		"A house favorite, made fresh to order.",
		"Locally sourced and simply prepared.",
		"A generous portion, best shared.",
		"Light, refreshing, and a good way to start.",
		"Rich and warming — a local classic.",
	}
	offerLeadPhrases   = []string{"Limited-time deal", "Weekend special", "New customer offer", "Flash discount", "Loyalty reward", "Off-peak special"}
	notificationTitles = []string{
		"New redemption", "Offer expiring soon", "New review posted", "Weekly summary ready", "Profile view spike",
	}
	notificationSubs = []string{
		"A traveller just redeemed one of your offers.",
		"One of your offers ends in the next 24 hours.",
		"A traveller left feedback on your listing.",
		"Your weekly performance report is ready to view.",
		"Your profile is getting more attention than usual.",
	}
	boostEventNames = []string{
		"Weekend Live Session", "Sunset Social", "Grand Reopening", "Seasonal Tasting Night", "Community Meetup",
	}
)

// Summary reports what was generated, including a handful of ready-to-use
// login credentials so a human can start testing immediately.
type Summary struct {
	Travellers    int
	Vendors       int
	Places        int
	Offers        int
	Claims        int
	Reviews       int
	Groups        int
	Subscriptions int
	AudioGuides   int
	SampleLogins  []string
}

func Run(ctx context.Context, store *postgres.Store) (Summary, error) {
	gofakeit.Seed(0) // 0 = time-based, so every run differs

	if err := store.Wipe(ctx); err != nil {
		return Summary{}, fmt.Errorf("wipe: %w", err)
	}

	passwordHash, err := auth.HashPassword(TestPassword)
	if err != nil {
		return Summary{}, fmt.Errorf("hash test password: %w", err)
	}

	now := time.Now()
	var sum Summary

	regions := realRegions() // real Sri Lankan towns — factual geography, not dummy content
	for _, r := range regions {
		if err := store.UpsertRegion(ctx, r); err != nil {
			return sum, fmt.Errorf("region %s: %w", r.ID, err)
		}
	}

	travellers, err := genTravellers(ctx, store, passwordHash, 40, now)
	if err != nil {
		return sum, err
	}
	sum.Travellers = len(travellers)

	admin, err := genAdmin(ctx, store, passwordHash, now)
	if err != nil {
		return sum, err
	}

	vendors, vendorUsers, places, err := genVendorsAndPlaces(ctx, store, passwordHash, 18, regions, now)
	if err != nil {
		return sum, err
	}
	sum.Vendors = len(vendors)
	sum.Places = len(places)

	offers, err := genOffers(ctx, store, vendors)
	if err != nil {
		return sum, err
	}
	sum.Offers = len(offers)

	claims, err := genClaims(ctx, store, travellers, offers, 60, now)
	if err != nil {
		return sum, err
	}
	sum.Claims = len(claims)

	reviews, err := genReviews(ctx, store, travellers, places, 250, now)
	if err != nil {
		return sum, err
	}
	sum.Reviews = len(reviews)

	groups, err := genGroups(ctx, store, travellers, 5)
	if err != nil {
		return sum, err
	}
	sum.Groups = groups

	subs, err := genSubscriptions(ctx, store, vendors, now)
	if err != nil {
		return sum, err
	}
	sum.Subscriptions = subs

	guides, err := genAudioGuides(ctx, store, regions, 8)
	if err != nil {
		return sum, err
	}
	sum.AudioGuides = guides

	if err := genVendorBlobs(ctx, store, vendors, now); err != nil {
		return sum, err
	}

	if err := syncRegionCounts(ctx, store, regions, len(travellers), vendors); err != nil {
		return sum, err
	}

	sum.SampleLogins = buildSampleLogins(admin, vendorUsers, travellers)
	return sum, nil
}

func realRegions() []domain.Region {
	return []domain.Region{
		{ID: "colombo", Name: "Colombo", X: 25, Y: 56, Lat: 6.9271, Lng: 79.8612},
		{ID: "negombo", Name: "Negombo", X: 23, Y: 49, Lat: 7.2083, Lng: 79.8358},
		{ID: "kandy", Name: "Kandy", X: 50, Y: 47, Lat: 7.2906, Lng: 80.6337},
		{ID: "sigiriya", Name: "Sigiriya", X: 54, Y: 34, Lat: 7.9570, Lng: 80.7603},
		{ID: "ella", Name: "Ella", X: 61, Y: 60, Lat: 6.8667, Lng: 81.0466},
		{ID: "nuwara", Name: "Nuwara Eliya", X: 54, Y: 56, Lat: 6.9497, Lng: 80.7891},
		{ID: "galle", Name: "Galle", X: 31, Y: 82, Lat: 6.0535, Lng: 80.2210},
		{ID: "weligama", Name: "Weligama", X: 39, Y: 86, Lat: 5.9724, Lng: 80.4292},
		{ID: "mirissa", Name: "Mirissa", X: 44, Y: 88, Lat: 5.9483, Lng: 80.4589},
		{ID: "arugam", Name: "Arugam Bay", X: 80, Y: 61, Lat: 6.8404, Lng: 81.8360},
		{ID: "trinco", Name: "Trincomalee", X: 70, Y: 27, Lat: 8.5874, Lng: 81.2152},
		{ID: "jaffna", Name: "Jaffna", X: 45, Y: 8, Lat: 9.6615, Lng: 80.0255},
	}
}

func jitter(v float64, spreadDeg float64) float64 {
	return v + (gofakeit.Float64Range(-spreadDeg, spreadDeg))
}

func capFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

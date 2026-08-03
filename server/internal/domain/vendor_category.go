package domain

import "strings"

// VendorSegment groups vendor categories into the stakeholder segments from
// the investor business plan (Hospitality, Activity & experience,
// Independent local providers, Event promoters, Brand partners, Future
// partners). SegmentFuture is deliberately never onboardable today — those
// are affiliate-style partners (accommodation platforms, flight partners,
// insurance, transfers, booking platforms), not a vendor type a business
// can sign up as yet.
type VendorSegment string

const (
	SegmentHospitality VendorSegment = "hospitality"
	SegmentActivity    VendorSegment = "activity"
	SegmentIndependent VendorSegment = "independent"
	SegmentEvents      VendorSegment = "events"
	SegmentBrand       VendorSegment = "brand"
	SegmentFuture      VendorSegment = "future"
)

// VendorCategory is one selectable business type within a segment. Icon is a
// Lucide icon name (string, not a component — this package has no UI
// dependency; web/vendor and the Flutter app each resolve it locally).
// RelevantVibeDimensions is the subset of "productive"/"loud"/"crowded"
// that's meaningful to ask/score for this category (e.g. "productive"
// doesn't mean anything for a beach bar) — see RelevantVibeDimensions.
type VendorCategory struct {
	ID                     string
	Label                  string
	Segment                VendorSegment
	Icon                   string
	RelevantVibeDimensions []string
}

// VendorCategories is the single canonical taxonomy — the source of truth
// mirrored by web/packages/shared/src/vendor-categories.ts (web/vendor,
// web/admin) and app/lib/data/vendor_categories.dart (the Flutter app).
// Replaces what used to be 5 independently hardcoded, drifting lists.
var VendorCategories = []VendorCategory{
	// Hospitality vendors — cafes, restaurants, bars, hotels, beach clubs.
	{ID: "cafe", Label: "Café", Segment: SegmentHospitality, Icon: "Coffee", RelevantVibeDimensions: []string{"productive", "loud", "crowded"}},
	{ID: "coworking-cafe", Label: "Coworking Café", Segment: SegmentHospitality, Icon: "Laptop", RelevantVibeDimensions: []string{"productive", "loud", "crowded"}},
	{ID: "restaurant", Label: "Restaurant", Segment: SegmentHospitality, Icon: "UtensilsCrossed", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "street-food", Label: "Street Food", Segment: SegmentHospitality, Icon: "Soup", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "bakery", Label: "Bakery", Segment: SegmentHospitality, Icon: "Cookie", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "bar-pub", Label: "Bar & Pub", Segment: SegmentHospitality, Icon: "Martini", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "club", Label: "Club / Nightlife", Segment: SegmentHospitality, Icon: "Music", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "hotel", Label: "Hotel", Segment: SegmentHospitality, Icon: "Building2", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "hostel", Label: "Hostel / Guesthouse", Segment: SegmentHospitality, Icon: "BedSingle", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "resort", Label: "Resort", Segment: SegmentHospitality, Icon: "Palmtree", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "beach-club", Label: "Beach Club", Segment: SegmentHospitality, Icon: "Waves", RelevantVibeDimensions: []string{"loud", "crowded"}},

	// Activity and experience providers — surf/dive schools, yoga, tours.
	{ID: "surf-school", Label: "Surf School", Segment: SegmentActivity, Icon: "Waves", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "dive-school", Label: "Diving / Dive Shop", Segment: SegmentActivity, Icon: "Anchor", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "yoga-studio", Label: "Yoga Studio", Segment: SegmentActivity, Icon: "Flower2", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "spa-wellness", Label: "Spa & Wellness", Segment: SegmentActivity, Icon: "Sparkles", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "cooking-class", Label: "Cooking Class", Segment: SegmentActivity, Icon: "ChefHat", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "pottery-workshop", Label: "Pottery Workshop", Segment: SegmentActivity, Icon: "Hammer", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "batik-atelier", Label: "Batik / Textile Atelier", Segment: SegmentActivity, Icon: "Shirt", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "spice-trail", Label: "Spice Trail", Segment: SegmentActivity, Icon: "Leaf", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "safari-wildlife", Label: "Safari & Wildlife", Segment: SegmentActivity, Icon: "PawPrint", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "boat-trip", Label: "Boat Trip", Segment: SegmentActivity, Icon: "Ship", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "city-tour", Label: "City / Cultural Tour", Segment: SegmentActivity, Icon: "Compass", RelevantVibeDimensions: []string{"crowded"}},

	// Independent local providers — guides, artisans, tailors, solo hosts.
	{ID: "tour-guide", Label: "Tour / Walking Guide", Segment: SegmentIndependent, Icon: "Compass", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "artisan-shop", Label: "Local Artisan / Craft Shop", Segment: SegmentIndependent, Icon: "Palette", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "tailor-clothing", Label: "Tailor / Local Clothing", Segment: SegmentIndependent, Icon: "Shirt", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "experience-host", Label: "Independent Experience Host", Segment: SegmentIndependent, Icon: "Backpack", RelevantVibeDimensions: []string{"crowded"}},

	// Event promoters — festivals, parties, competitions, community events.
	{ID: "festival-event", Label: "Festival", Segment: SegmentEvents, Icon: "PartyPopper", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "party-promoter", Label: "Party Promotion", Segment: SegmentEvents, Icon: "PartyPopper", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "surf-competition", Label: "Surf Competition", Segment: SegmentEvents, Icon: "Waves", RelevantVibeDimensions: []string{"loud", "crowded"}},
	{ID: "community-charity-event", Label: "Community / Charity Event", Segment: SegmentEvents, Icon: "HeartHandshake", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "music-night", Label: "Music Night", Segment: SegmentEvents, Icon: "Music", RelevantVibeDimensions: []string{"loud", "crowded"}},

	// Brand partners — sponsors, not physical venues, so vibe is irrelevant.
	{ID: "beverage-brand", Label: "Beverage Brand", Segment: SegmentBrand, Icon: "GlassWater"},
	{ID: "alcohol-brand", Label: "Beer & Alcohol Brand", Segment: SegmentBrand, Icon: "Wine"},
	{ID: "food-brand", Label: "Food Brand", Segment: SegmentBrand, Icon: "UtensilsCrossed"},
	{ID: "lifestyle-brand", Label: "Lifestyle Brand", Segment: SegmentBrand, Icon: "Sparkles"},
	{ID: "tourism-company", Label: "Tourism Company", Segment: SegmentBrand, Icon: "Plane"},

	// Future partners (accommodation/flight/insurance/transfer/booking
	// platforms) deliberately have no categories — affiliate-only per the
	// business plan, not an active onboarding type. See OnboardableCategories.
}

// legacyCategoryAliases maps every free-text category string the app has
// ever written (across the 5 pre-taxonomy hardcoded lists) onto a canonical
// VendorCategories id. Used by CategoryByID for backward-compatible label
// lookups on rows that predate migration 024, and by that migration itself
// to normalize existing data. "craft"/"shop" have no reliable automatic
// mapping — they're given a best-effort default and should be spot-checked
// by an admin post-migration (see migration 024's comment).
var legacyCategoryAliases = map[string]string{
	"restaurant":       "restaurant",
	"Restaurant":       "restaurant",
	"cafe":             "cafe",
	"Cafe":             "cafe",
	"Café":             "cafe",
	"Coworking Café":   "coworking-cafe",
	"bar":              "bar-pub",
	"Bar & Sunset":     "bar-pub",
	"Bar":              "bar-pub",
	"surf":             "surf-school",
	"Surf & Water":     "surf-school",
	"Surf School":      "surf-school",
	"Yoga / Wellness":  "yoga-studio",
	"Spa & Wellness":   "spa-wellness",
	"Hostel":           "hostel",
	"Hotel":            "hotel",
	"Pottery Workshop": "pottery-workshop",
	"Batik Atelier":    "batik-atelier",
	"Tours / Activity": "city-tour",
	"Street Food":      "street-food",
	"Bakery":           "bakery",
	"Dive Shop":        "dive-school",
	"craft":            "pottery-workshop", // best-effort default, flagged for review
	"Craft & Studio":   "pottery-workshop", // best-effort default, flagged for review
	"shop":             "artisan-shop",     // best-effort default, flagged for review
	"Shop":             "artisan-shop",     // best-effort default, flagged for review
}

func canonicalCategoryByID(id string) (VendorCategory, bool) {
	norm := strings.ToLower(strings.TrimSpace(id))
	for _, c := range VendorCategories {
		if strings.ToLower(c.ID) == norm {
			return c, true
		}
	}
	return VendorCategory{}, false
}

// CategoryByID resolves a category id to its canonical VendorCategory,
// falling back through legacyCategoryAliases (case-insensitive) so vendor
// rows written before migration 024 still resolve to a real category for
// display purposes.
func CategoryByID(id string) (VendorCategory, bool) {
	if c, ok := canonicalCategoryByID(id); ok {
		return c, true
	}
	for alias, canonical := range legacyCategoryAliases {
		if strings.EqualFold(alias, id) {
			return canonicalCategoryByID(canonical)
		}
	}
	return VendorCategory{}, false
}

// ValidCategoryID reports whether id is a real, currently onboardable
// category — i.e. an exact canonical id (not a legacy alias) outside
// SegmentFuture. Used to validate new writes (vendor category changes,
// admin edits, registration) so category stops being unconstrained free
// text going forward.
func ValidCategoryID(id string) bool {
	c, ok := canonicalCategoryByID(id)
	return ok && c.Segment != SegmentFuture
}

// OnboardableCategories returns every category outside SegmentFuture — what
// a vendor registration/profile picker or admin dropdown should offer.
func OnboardableCategories() []VendorCategory {
	out := make([]VendorCategory, 0, len(VendorCategories))
	for _, c := range VendorCategories {
		if c.Segment != SegmentFuture {
			out = append(out, c)
		}
	}
	return out
}

// RelevantVibeDimensions returns which of "productive"/"loud"/"crowded" are
// meaningful to ask/score for a given vendor category (e.g. "productive"
// doesn't mean anything for a beach bar). Falls back to all 3 for an
// unknown or blank category id — the same graceful-fallback convention
// vendor.go's categoryLabel uses for unrecognized categories.
func RelevantVibeDimensions(categoryID string) []string {
	if c, ok := CategoryByID(categoryID); ok && len(c.RelevantVibeDimensions) > 0 {
		return c.RelevantVibeDimensions
	}
	return []string{"productive", "loud", "crowded"}
}

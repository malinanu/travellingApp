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
	SegmentFoodAndDrink VendorSegment = "food_and_drink"
	SegmentThingsToDo   VendorSegment = "things_to_do"
	SegmentShopping     VendorSegment = "shopping"
	SegmentServices     VendorSegment = "services"
	SegmentBrand        VendorSegment = "brand"
	SegmentFuture       VendorSegment = "future"
)

// PinColorForSegment maps a category's segment to a Place.color pin-tone
// key. Mirrors app/lib/core/theme/palette.dart's pinTones keys exactly —
// keep both in sync if either changes.
func PinColorForSegment(seg VendorSegment) string {
	switch seg {
	case SegmentFoodAndDrink:
		return "coral"
	case SegmentThingsToDo:
		return "sun"
	case SegmentShopping:
		return "plum"
	case SegmentServices:
		return "clay"
	default: // SegmentBrand, SegmentFuture, unknown
		return "teal"
	}
}

// VendorCategory is one selectable business type within a segment. Icon is a
// Lucide icon name (string, not a component — this package has no UI
// dependency; web/vendor and the Flutter app each resolve it locally).
// RelevantVibeDimensions is the subset of "productive"/"loud"/"crowded"
// that's meaningful to ask/score for this category (e.g. "productive"
// doesn't mean anything for a beach bar) — see RelevantVibeDimensions.
type VendorCategory struct {
	ID                     string        `json:"id"`
	Label                  string        `json:"label"`
	Segment                VendorSegment `json:"segment"`
	Icon                   string        `json:"icon"`
	RelevantVibeDimensions []string      `json:"relevantVibeDimensions"`
	// BookingType is which BookingType (see booking.go) this category's
	// vendors can take real bookings for — "" means not bookable in this
	// phase (walk-in businesses, brand partners, and Events — Events isn't
	// a built feature yet; ticket booking can reuse this same system once
	// it ships). See BookingTypeByCategory/IsBookableCategory.
	BookingType BookingType `json:"bookingType"`
}

// VendorCategories is the single canonical taxonomy — the source of truth
// mirrored by web/packages/shared/src/vendor-categories.ts (web/vendor,
// web/admin) and app/lib/data/vendor_categories.dart (the Flutter app).
// Replaces what used to be 5 independently hardcoded, drifting lists.
var VendorCategories []VendorCategory

func InitVendorCategories(categories []VendorCategory) {
	VendorCategories = categories
}

func AddVendorCategory(category VendorCategory) {
	for i, c := range VendorCategories {
		if c.ID == category.ID {
			VendorCategories[i] = category
			return
		}
	}
	VendorCategories = append(VendorCategories, category)
}

func RemoveVendorCategory(id string) {
	for i, c := range VendorCategories {
		if c.ID == id {
			VendorCategories = append(VendorCategories[:i], VendorCategories[i+1:]...)
			return
		}
	}
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

// BookingTypeByCategory returns which BookingType a category's vendors can
// take real bookings for, or "" if the category isn't bookable at all
// (unknown/blank id, or a category with no BookingType set — e.g. brand
// partners, event categories, or walk-in-only categories like bakeries).
func BookingTypeByCategory(categoryID string) BookingType {
	if c, ok := CategoryByID(categoryID); ok {
		return c.BookingType
	}
	return ""
}

// IsBookableCategory reports whether categoryID's vendors can take real
// bookings through the platform.
func IsBookableCategory(categoryID string) bool {
	return BookingTypeByCategory(categoryID) != ""
}

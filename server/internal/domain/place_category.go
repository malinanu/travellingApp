package domain

import "strings"

// CategoryForPlaceData classifies a place's stored JSON (its Google-derived
// "type" and "tags") into a canonical vendor-category id. Shared by the
// importer and by the API, which uses it to fill in categoryId for places
// stored before categoryId was written. Always returns a category id.
func CategoryForPlaceData(data map[string]any) string {
	t, _ := data["type"].(string)
	tagSet := map[string]bool{}
	if raw, ok := data["tags"].([]any); ok {
		for _, v := range raw {
			if s, ok := v.(string); ok {
				tagSet[strings.ToLower(s)] = true
			}
		}
	}

	hasAny := func(tags ...string) bool {
		for _, tag := range tags {
			if tagSet[tag] {
				return true
			}
		}
		return false
	}

	// 1. Specific matches based on tags
	if hasAny("hotel", "resort hotel", "lodging", "guest house", "bed and breakfast", "hostel", "private guest room") {
		return "hotels"
	}
	if hasAny("park", "wildlife park", "national park", "nature preserve", "botanical garden", "city park") {
		return "parks"
	}
	if hasAny("gym", "fitness center", "yoga studio", "health", "sports club", "sports complex", "sports school") {
		return "gyms"
	}
	if hasAny("museum", "history museum") {
		return "museums"
	}
	if hasAny("art gallery") {
		return "art"
	}
	if hasAny("live music venue") {
		return "live-music"
	}
	if hasAny("night club", "karaoke") {
		return "nightlife"
	}
	if hasAny("car dealer", "truck dealer") {
		return "car-dealers"
	}
	if hasAny("car rental") {
		return "car-rental"
	}
	if hasAny("parking", "park and ride") {
		return "parking"
	}
	if hasAny("sporting goods store") {
		return "sporting-goods"
	}
	if hasAny("grocery store", "food store", "supermarket") {
		return "groceries"
	}
	if hasAny("bakery", "cake shop", "pastry shop", "dessert shop", "ice cream shop", "confectionery") {
		return "coffee"
	}
	if hasAny("spa", "massage", "massage spa") {
		return "beauty-salons"
	}
	if hasAny("bar", "bar and grill", "cocktail bar", "wine bar", "lounge bar", "pub", "gastropub", "hookah bar") {
		return "bars"
	}
	if hasAny("cafe", "coffee shop", "tea house", "cat cafe", "coffee roastery", "coffee stand") {
		return "coffee"
	}
	if hasAny("food delivery", "meal delivery", "pizza delivery") {
		return "delivery"
	}
	if hasAny("meal takeaway", "takeout", "sandwich shop", "juice shop") {
		return "takeout"
	}
	if hasAny("restaurant", "breakfast restaurant", "seafood restaurant", "asian restaurant", "pizza restaurant", "asian fusion restaurant", "buffet restaurant", "family restaurant", "fusion restaurant", "italian restaurant", "sri lankan restaurant", "cuban restaurant", "fine dining restaurant", "tapas restaurant", "korean restaurant", "vegetarian restaurant", "brunch restaurant", "american restaurant", "european restaurant", "bistro", "indian restaurant", "chinese noodle restaurant", "chinese restaurant", "mediterranean restaurant", "western restaurant", "mexican restaurant", "vegan restaurant", "japanese restaurant", "dessert restaurant", "dumpling restaurant") {
		return "restaurants"
	}
	if hasAny("store", "market", "liquor store", "tea store", "shopping center", "apparel") {
		return "convenience-stores"
	}

	// 2. Fallbacks by type
	switch t {
	case "cafe":
		return "coffee"
	case "food":
		if hasAny("bar") {
			return "bars"
		}
		return "restaurants"
	case "landmark", "temple", "attraction", "cultural landmark", "castle", "bridge", "church", "place of worship", "buddhist temple", "hindu temple":
		return "attractions"
	case "wildlife", "park":
		return "parks"
	case "beach", "marina", "water park", "natural feature":
		return "attractions"
	case "lodging":
		return "hotels"
	}

	// 3. Very generic tags
	if hasAny("tourist attraction", "historical landmark", "scenic spot", "tour agency", "travel agency") {
		return "attractions"
	}

	// Ensure we only return a valid predefined category, fallback to attractions.
	return "attractions"
}

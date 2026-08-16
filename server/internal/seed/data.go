// Package seed loads the demo dataset ported from the prototype's data files
// (app/lib/data/mock/places.dart, Prototype/design/vendor-data.jsx,
// Prototype/design/admin-data.jsx). Relative-time strings become timestamps
// offset from now.
package seed

import "github.com/spotto/server/internal/domain"

var InitialCategories = []domain.VendorCategory{
	// Food & Drink
	{ID: "restaurants", Label: "Restaurants", Segment: domain.SegmentFoodAndDrink, Icon: "UtensilsCrossed", RelevantVibeDimensions: []string{"loud", "crowded"}, BookingType: domain.BookingTypeTable},
	{ID: "takeout", Label: "Takeout", Segment: domain.SegmentFoodAndDrink, Icon: "ShoppingBag", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "bars", Label: "Bars", Segment: domain.SegmentFoodAndDrink, Icon: "Martini", RelevantVibeDimensions: []string{"loud", "crowded"}, BookingType: domain.BookingTypeTable},
	{ID: "delivery", Label: "Delivery", Segment: domain.SegmentFoodAndDrink, Icon: "Truck", RelevantVibeDimensions: []string{}},
	{ID: "coffee", Label: "Coffee", Segment: domain.SegmentFoodAndDrink, Icon: "Coffee", RelevantVibeDimensions: []string{"productive", "loud", "crowded"}, BookingType: domain.BookingTypeTable},

	// Things to do
	{ID: "parks", Label: "Parks", Segment: domain.SegmentThingsToDo, Icon: "TreePine", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "gyms", Label: "Gyms", Segment: domain.SegmentThingsToDo, Icon: "Dumbbell", RelevantVibeDimensions: []string{"loud", "crowded"}, BookingType: domain.BookingTypeSession},
	{ID: "art", Label: "Art", Segment: domain.SegmentThingsToDo, Icon: "Palette", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "attractions", Label: "Attractions", Segment: domain.SegmentThingsToDo, Icon: "Ticket", RelevantVibeDimensions: []string{"crowded"}, BookingType: domain.BookingTypeSession},
	{ID: "nightlife", Label: "Nightlife", Segment: domain.SegmentThingsToDo, Icon: "Music", RelevantVibeDimensions: []string{"loud", "crowded"}, BookingType: domain.BookingTypeTable},
	{ID: "live-music", Label: "Live music", Segment: domain.SegmentThingsToDo, Icon: "Mic", RelevantVibeDimensions: []string{"loud", "crowded"}, BookingType: domain.BookingTypeTable},
	{ID: "movies", Label: "Movies", Segment: domain.SegmentThingsToDo, Icon: "Film", RelevantVibeDimensions: []string{"crowded"}, BookingType: domain.BookingTypeSession},
	{ID: "museums", Label: "Museums", Segment: domain.SegmentThingsToDo, Icon: "Landmark", RelevantVibeDimensions: []string{"crowded"}, BookingType: domain.BookingTypeSession},
	{ID: "libraries", Label: "Libraries", Segment: domain.SegmentThingsToDo, Icon: "Library", RelevantVibeDimensions: []string{"productive", "crowded"}},

	// Shopping
	{ID: "groceries", Label: "Groceries", Segment: domain.SegmentShopping, Icon: "ShoppingCart", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "beauty-supplies", Label: "Beauty supplies", Segment: domain.SegmentShopping, Icon: "Sparkles", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "car-dealers", Label: "Car dealers", Segment: domain.SegmentShopping, Icon: "Car", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "home-garden", Label: "Home & garden", Segment: domain.SegmentShopping, Icon: "Flower2", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "apparel", Label: "Apparel", Segment: domain.SegmentShopping, Icon: "Shirt", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "shopping-centers", Label: "Shopping centers", Segment: domain.SegmentShopping, Icon: "Store", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "electronics", Label: "Electronics", Segment: domain.SegmentShopping, Icon: "Laptop", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "sporting-goods", Label: "Sporting goods", Segment: domain.SegmentShopping, Icon: "Bike", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "convenience-stores", Label: "Convenience stores", Segment: domain.SegmentShopping, Icon: "Store", RelevantVibeDimensions: []string{"crowded"}},

	// Services
	{ID: "hotels", Label: "Hotels", Segment: domain.SegmentServices, Icon: "Building2", RelevantVibeDimensions: []string{"crowded"}, BookingType: domain.BookingTypeStay},
	{ID: "atms", Label: "ATMs", Segment: domain.SegmentServices, Icon: "Banknote", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "beauty-salons", Label: "Beauty salons", Segment: domain.SegmentServices, Icon: "Scissors", RelevantVibeDimensions: []string{"crowded"}, BookingType: domain.BookingTypeSession},
	{ID: "car-rental", Label: "Car rental", Segment: domain.SegmentServices, Icon: "Key", RelevantVibeDimensions: []string{"crowded"}, BookingType: domain.BookingTypeRental},
	{ID: "car-wash", Label: "Car wash", Segment: domain.SegmentServices, Icon: "Droplets", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "dry-cleaning", Label: "Dry cleaning", Segment: domain.SegmentServices, Icon: "Shirt", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "charging-stations", Label: "Charging stations", Segment: domain.SegmentServices, Icon: "Zap", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "gas", Label: "Gas", Segment: domain.SegmentServices, Icon: "Fuel", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "hospitals-clinics", Label: "Hospitals & clinics", Segment: domain.SegmentServices, Icon: "Stethoscope", RelevantVibeDimensions: []string{"crowded"}, BookingType: domain.BookingTypeSession},
	{ID: "mail-shipping", Label: "Mail & shipping", Segment: domain.SegmentServices, Icon: "Mail", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "parking", Label: "Parking", Segment: domain.SegmentServices, Icon: "SquareParking", RelevantVibeDimensions: []string{"crowded"}},
	{ID: "pharmacies", Label: "Pharmacies", Segment: domain.SegmentServices, Icon: "Pill", RelevantVibeDimensions: []string{"crowded"}},

	// Brand
	{ID: "beverage-brand", Label: "Beverage Brand", Segment: domain.SegmentBrand, Icon: "GlassWater"},
	{ID: "alcohol-brand", Label: "Beer & Alcohol Brand", Segment: domain.SegmentBrand, Icon: "Wine"},
	{ID: "food-brand", Label: "Food Brand", Segment: domain.SegmentBrand, Icon: "UtensilsCrossed"},
	{ID: "lifestyle-brand", Label: "Lifestyle Brand", Segment: domain.SegmentBrand, Icon: "Sparkles"},
	{ID: "tourism-company", Label: "Tourism Company", Segment: domain.SegmentBrand, Icon: "Plane"},
}

var Regions = []domain.Region{
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
	{ID: "anuradhapura", Name: "Anuradhapura", X: 48, Y: 30, Lat: 8.3300, Lng: 80.4100, Travelers: 40, Vendors: 20},
	{ID: "polonnaruwa", Name: "Polonnaruwa", X: 60, Y: 35, Lat: 7.9400, Lng: 81.0000, Travelers: 35, Vendors: 15},
	{ID: "dambulla", Name: "Dambulla", X: 52, Y: 36, Lat: 7.8600, Lng: 80.6500, Travelers: 50, Vendors: 25},
	{ID: "hikkaduwa", Name: "Hikkaduwa", X: 28, Y: 78, Lat: 6.1400, Lng: 80.1000, Travelers: 75, Vendors: 60},
	{ID: "unawatuna", Name: "Unawatuna", X: 32, Y: 83, Lat: 6.0100, Lng: 80.2500, Travelers: 85, Vendors: 70},
	{ID: "ahangama", Name: "Ahangama", X: 36, Y: 85, Lat: 5.9700, Lng: 80.3600, Travelers: 80, Vendors: 55},
	{ID: "dickwella", Name: "Dickwella", X: 52, Y: 87, Lat: 5.9600, Lng: 80.7000, Travelers: 70, Vendors: 40},
	{ID: "tangalle", Name: "Tangalle", X: 56, Y: 86, Lat: 6.0200, Lng: 80.7900, Travelers: 60, Vendors: 35},
	{ID: "yala", Name: "Yala", X: 70, Y: 75, Lat: 6.3700, Lng: 81.4500, Travelers: 55, Vendors: 25},
	{ID: "kalpitiya", Name: "Kalpitiya", X: 20, Y: 35, Lat: 8.2300, Lng: 79.7700, Travelers: 45, Vendors: 20},
	{ID: "passikudah", Name: "Passikudah", X: 75, Y: 40, Lat: 7.9200, Lng: 81.5600, Travelers: 35, Vendors: 15},
	{ID: "batticaloa", Name: "Batticaloa", X: 78, Y: 45, Lat: 7.7400, Lng: 81.6900, Travelers: 30, Vendors: 10},
	{ID: "ampara", Name: "Ampara", X: 78, Y: 47, Lat: 7.2978, Lng: 81.6790, Travelers: 15, Vendors: 5},
	{ID: "badulla", Name: "Badulla", X: 61, Y: 60, Lat: 6.8999, Lng: 81.0571, Travelers: 20, Vendors: 10},
	{ID: "gampaha", Name: "Gampaha", X: 27, Y: 53, Lat: 7.0873, Lng: 79.9997, Travelers: 35, Vendors: 40},
	{ID: "hambantota", Name: "Hambantota", X: 63, Y: 80, Lat: 6.1427, Lng: 81.1212, Travelers: 25, Vendors: 15},
	{ID: "kalutara", Name: "Kalutara", X: 26, Y: 68, Lat: 6.5854, Lng: 79.9607, Travelers: 40, Vendors: 25},
	{ID: "kegalle", Name: "Kegalle", X: 40, Y: 48, Lat: 7.2513, Lng: 80.3464, Travelers: 15, Vendors: 10},
	{ID: "kilinochchi", Name: "Kilinochchi", X: 47, Y: 15, Lat: 9.3803, Lng: 80.3770, Travelers: 5, Vendors: 2},
	{ID: "kurunegala", Name: "Kurunegala", X: 40, Y: 42, Lat: 7.4863, Lng: 80.3629, Travelers: 20, Vendors: 15},
	{ID: "mannar", Name: "Mannar", X: 28, Y: 20, Lat: 8.9800, Lng: 79.9000, Travelers: 10, Vendors: 5},
	{ID: "matale", Name: "Matale", X: 50, Y: 42, Lat: 7.4675, Lng: 80.6234, Travelers: 25, Vendors: 15},
	{ID: "matara", Name: "Matara", X: 47, Y: 88, Lat: 5.9485, Lng: 80.5353, Travelers: 65, Vendors: 45},
	{ID: "monaragala", Name: "Monaragala", X: 68, Y: 60, Lat: 6.8700, Lng: 81.3500, Travelers: 15, Vendors: 5},
	{ID: "mullaitivu", Name: "Mullaitivu", X: 58, Y: 18, Lat: 9.2500, Lng: 80.8167, Travelers: 5, Vendors: 2},
	{ID: "puttalam", Name: "Puttalam", X: 25, Y: 28, Lat: 8.0408, Lng: 79.8265, Travelers: 15, Vendors: 10},
	{ID: "ratnapura", Name: "Ratnapura", X: 42, Y: 64, Lat: 6.6932, Lng: 80.3995, Travelers: 30, Vendors: 15},
	{ID: "vavuniya", Name: "Vavuniya", X: 48, Y: 25, Lat: 8.7542, Lng: 80.4982, Travelers: 10, Vendors: 5},
}

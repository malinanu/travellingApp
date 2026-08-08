CREATE TABLE vendor_categories (
    id TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    segment TEXT NOT NULL,
    icon TEXT NOT NULL,
    relevant_vibe_dimensions TEXT[],
    booking_type TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Insert existing categories from vendors as placeholders
INSERT INTO vendor_categories (id, label, segment, icon, relevant_vibe_dimensions, booking_type)
SELECT DISTINCT category, category, 'General', 'storefront', '{}', NULL
FROM vendors
WHERE category IS NOT NULL AND category != ''
ON CONFLICT (id) DO NOTHING;

-- Update the inserted placeholders with real metadata from domain model
UPDATE vendor_categories as vc
SET label = v.label,
    segment = v.segment,
    icon = v.icon
FROM (VALUES
    ('cafe', 'Café', 'hospitality', 'Coffee'),
    ('coworking-cafe', 'Coworking Café', 'hospitality', 'Laptop'),
    ('restaurant', 'Restaurant', 'hospitality', 'UtensilsCrossed'),
    ('street-food', 'Street Food', 'hospitality', 'Soup'),
    ('bakery', 'Bakery', 'hospitality', 'Cookie'),
    ('bar-pub', 'Bar & Pub', 'hospitality', 'Martini'),
    ('club', 'Club / Nightlife', 'hospitality', 'Music'),
    ('hotel', 'Hotel', 'hospitality', 'Building2'),
    ('hostel', 'Hostel / Guesthouse', 'hospitality', 'BedSingle'),
    ('resort', 'Resort', 'hospitality', 'Palmtree'),
    ('beach-club', 'Beach Club', 'hospitality', 'Waves'),
    ('surf-school', 'Surf School', 'activity', 'Waves'),
    ('dive-school', 'Diving / Dive Shop', 'activity', 'Anchor'),
    ('yoga-studio', 'Yoga Studio', 'activity', 'Flower2'),
    ('spa-wellness', 'Spa & Wellness', 'activity', 'Sparkles'),
    ('cooking-class', 'Cooking Class', 'activity', 'ChefHat'),
    ('pottery-workshop', 'Pottery Workshop', 'activity', 'Hammer'),
    ('batik-atelier', 'Batik / Textile Atelier', 'activity', 'Shirt'),
    ('spice-trail', 'Spice Trail', 'activity', 'Leaf'),
    ('safari-wildlife', 'Safari & Wildlife', 'activity', 'PawPrint'),
    ('boat-trip', 'Boat Trip', 'activity', 'Ship'),
    ('city-tour', 'City / Cultural Tour', 'activity', 'Compass'),
    ('surf-rental', 'Surfboard Rental', 'activity', 'Waves'),
    ('bike-rental', 'Bicycle Rental', 'activity', 'Bike'),
    ('scooter-rental', 'Scooter Rental', 'activity', 'Bike'),
    ('kayak-rental', 'Kayak / Canoe Rental', 'activity', 'Anchor'),
    ('tour-guide', 'Tour / Walking Guide', 'independent', 'Compass'),
    ('artisan-shop', 'Local Artisan / Craft Shop', 'independent', 'Palette'),
    ('tailor-clothing', 'Tailor / Local Clothing', 'independent', 'Shirt'),
    ('experience-host', 'Independent Experience Host', 'independent', 'Backpack'),
    ('festival-event', 'Festival', 'events', 'PartyPopper'),
    ('party-promoter', 'Party Promotion', 'events', 'PartyPopper'),
    ('surf-competition', 'Surf Competition', 'events', 'Waves'),
    ('community-charity-event', 'Community / Charity Event', 'events', 'HeartHandshake'),
    ('music-night', 'Music Night', 'events', 'Music'),
    ('beverage-brand', 'Beverage Brand', 'brand', 'GlassWater'),
    ('alcohol-brand', 'Beer & Alcohol Brand', 'brand', 'Wine'),
    ('food-brand', 'Food Brand', 'brand', 'UtensilsCrossed'),
    ('lifestyle-brand', 'Lifestyle Brand', 'brand', 'Sparkles'),
    ('tourism-company', 'Tourism Company', 'brand', 'Plane')
) AS v(id, label, segment, icon)
WHERE vc.id = v.id;


-- Foreign key to vendor_categories in vendors table
ALTER TABLE vendors
ADD CONSTRAINT fk_vendors_category
FOREIGN KEY (category) REFERENCES vendor_categories(id)
ON DELETE RESTRICT; -- Don't allow deleting a category if it's in use

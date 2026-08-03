-- Normalizes vendors.category from the 5 divergent free-text vocabularies
-- that existed before the canonical taxonomy (see
-- server/internal/domain/vendor_category.go) onto its stable ids. Mirrors
-- that file's legacyCategoryAliases map exactly — keep both in sync if
-- either changes.
--
-- "craft" and "shop" have no reliable automatic mapping to the richer new
-- taxonomy (they were generic catch-all buckets) — they get a best-effort
-- default below and should be spot-checked by an admin afterwards via the
-- admin vendor directory's category filter.
UPDATE vendors SET category = 'cafe' WHERE category IN ('cafe', 'Cafe', 'Café');
UPDATE vendors SET category = 'coworking-cafe' WHERE category = 'Coworking Café';
UPDATE vendors SET category = 'restaurant' WHERE category IN ('restaurant', 'Restaurant');
UPDATE vendors SET category = 'bar-pub' WHERE category IN ('bar', 'Bar & Sunset', 'Bar');
UPDATE vendors SET category = 'surf-school' WHERE category IN ('surf', 'Surf & Water', 'Surf School');
UPDATE vendors SET category = 'yoga-studio' WHERE category = 'Yoga / Wellness';
UPDATE vendors SET category = 'spa-wellness' WHERE category = 'Spa & Wellness';
UPDATE vendors SET category = 'hostel' WHERE category = 'Hostel';
UPDATE vendors SET category = 'hotel' WHERE category = 'Hotel';
UPDATE vendors SET category = 'pottery-workshop' WHERE category IN ('Pottery Workshop', 'craft', 'Craft & Studio');
UPDATE vendors SET category = 'batik-atelier' WHERE category = 'Batik Atelier';
UPDATE vendors SET category = 'city-tour' WHERE category = 'Tours / Activity';
UPDATE vendors SET category = 'street-food' WHERE category = 'Street Food';
UPDATE vendors SET category = 'bakery' WHERE category = 'Bakery';
UPDATE vendors SET category = 'dive-school' WHERE category = 'Dive Shop';
UPDATE vendors SET category = 'artisan-shop' WHERE category IN ('shop', 'Shop');

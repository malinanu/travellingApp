-- Differentiate category icons that previously collided with an unrelated
-- category (see web/packages/shared/src/vendor-categories.ts for the
-- taxonomy source of truth these mirror).
UPDATE vendor_categories SET icon = 'Umbrella' WHERE id = 'beach-club';
UPDATE vendor_categories SET icon = 'Medal' WHERE id = 'surf-competition';
UPDATE vendor_categories SET icon = 'Brush' WHERE id = 'batik-atelier';
UPDATE vendor_categories SET icon = 'Ruler' WHERE id = 'tailor-clothing';
UPDATE vendor_categories SET icon = 'Sailboat' WHERE id = 'kayak-rental';

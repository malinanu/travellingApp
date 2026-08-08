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

-- Foreign key to vendor_categories in vendors table
ALTER TABLE vendors
ADD CONSTRAINT fk_vendors_category
FOREIGN KEY (category) REFERENCES vendor_categories(id)
ON DELETE RESTRICT; -- Don't allow deleting a category if it's in use

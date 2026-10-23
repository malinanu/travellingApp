-- The admin "Content monitor" was a prototype feed that only ever showed
-- rows made by cmd/gendata; nothing in the app wrote to it. It's gone, and
-- so is its table. Real moderation lives on the vendor's Reviews/Offers tabs.
DROP TABLE IF EXISTS feed_items;

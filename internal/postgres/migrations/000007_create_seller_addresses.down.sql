DROP INDEX IF EXISTS idx_deliveries_origin_address;
ALTER TABLE deliveries DROP COLUMN IF EXISTS origin_address_id;

DROP INDEX IF EXISTS idx_seller_addresses_ownership;
DROP INDEX IF EXISTS idx_seller_addresses_one_default;
DROP INDEX IF EXISTS idx_seller_addresses_seller;
DROP TABLE IF EXISTS seller_addresses;

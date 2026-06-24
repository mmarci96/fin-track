-- FULL structural wipe — drops everything including users, merchants, categories
-- and currencies. This DELETES real user rows and learned reference data, so it
-- is gated behind `make db-nuke` (dev-only RUNTIME_ENV guard). Use only when the
-- schema of the reference tables themselves changes; for routine dev resets use
-- sql/drop.sql (`make reset`), which preserves identities.
DROP TABLE IF EXISTS receipt_images CASCADE;
DROP TABLE IF EXISTS product_categories CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS receipts CASCADE;
DROP TABLE IF EXISTS currencies CASCADE;
DROP TABLE IF EXISTS merchants CASCADE;
DROP TABLE IF EXISTS users CASCADE;

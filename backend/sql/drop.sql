-- Dev-only soft reset: clears scanned/transactional data so the parsing pipeline
-- can be re-tested from a clean slate, while PRESERVING identities and reference
-- data (users, merchants, categories, currencies). This is what `make reset`
-- runs. For a full structural wipe (e.g. when changing the schema of users /
-- merchants / etc.) use `make db-nuke` (sql/nuke.sql).
--
-- Order matters: drop children before parents. receipt_images.receipt_id is
-- ON DELETE SET NULL, but we drop the table outright here since it is recreated
-- by schema.sql.
DROP TABLE IF EXISTS receipt_images CASCADE;
DROP TABLE IF EXISTS product_categories CASCADE;
DROP TABLE IF EXISTS products CASCADE;
DROP TABLE IF EXISTS receipts CASCADE;

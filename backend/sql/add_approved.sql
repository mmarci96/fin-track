-- Idempotent migration: adds receipt_images.approved_at to an existing database
-- without dropping data. Safe to run more than once. (schema.sql already
-- includes this for fresh databases.)
--
-- approved_at is set when a human approves a capture's clean transcript as
-- ground truth; NULL means not yet approved.

ALTER TABLE receipt_images
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP;

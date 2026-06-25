-- Idempotent migration: adds merchants.verified and marks the two real
-- merchants verified, without touching the legacy auto-created junk rows. Safe
-- to run more than once. (schema.sql already includes this for fresh databases.)
--
-- Only verified merchants are matched during parsing, so unverified junk rows
-- (garbled OCR headers that the old self-poisoning path created) stop being
-- treated as "known".

ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS verified BOOLEAN NOT NULL DEFAULT false;

UPDATE merchants SET verified = true
    WHERE normalized_name IN (
        'ALDI MAGYARORSZAG ELELMISZER BT',
        'ROSSMANN MAGYARORSZAG KFT'
    );

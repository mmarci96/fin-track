-- Idempotent migration: adds the recognition-flywheel structures to an existing
-- database without dropping data. Safe to run more than once. (schema.sql already
-- includes these for fresh databases.)
--
--   * receipt_images.clean_parse_json — the structured result of re-parsing the
--     human-corrected transcript (clean_text); the approved subset is the
--     labeled ground-truth corpus.
--   * merchant_aliases — garbled raw-OCR header variants (normalized) that map to
--     a canonical merchant, learned on approval and fed back into the parser.

ALTER TABLE receipt_images
    ADD COLUMN IF NOT EXISTS clean_parse_json JSONB;

CREATE TABLE IF NOT EXISTS merchant_aliases (
    id SERIAL PRIMARY KEY,
    merchant_id INTEGER NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    normalized_alias TEXT NOT NULL UNIQUE,
    source_image_id INTEGER REFERENCES receipt_images(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

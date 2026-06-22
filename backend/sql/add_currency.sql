-- Idempotent migration: adds the currencies table and receipts.currency_id to
-- an existing database without dropping data. Safe to run more than once.
-- (schema.sql already includes these for fresh databases.)

CREATE TABLE IF NOT EXISTS currencies (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

INSERT INTO currencies (code) VALUES ('HUF'), ('EUR')
    ON CONFLICT (code) DO NOTHING;

ALTER TABLE receipts
    ADD COLUMN IF NOT EXISTS currency_id INTEGER NOT NULL
        REFERENCES currencies(id) DEFAULT 1;

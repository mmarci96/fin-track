-- fin-track schema. Idempotent: safe to re-run on an existing database without
-- destroying data. Every CREATE is IF NOT EXISTS and every seed is ON CONFLICT
-- DO NOTHING, so `make start` against a populated volume is a no-op rather than
-- an error. Destructive resets are explicit and dev-only (see drop.sql /
-- nuke.sql and the Makefile guards).

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Default user (id = 1) used for unauthenticated uploads in dev. Real users are
-- created out of band (auth-service create-user maps to users.id). The explicit
-- id + ON CONFLICT keeps this seed from duplicating or clobbering real rows.
INSERT INTO users (id, name) VALUES (1, 'default')
    ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS merchants (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    -- accent-free, uppercase, punctuation-free key for de-duplication; must match
    -- receipt.NormalizeName so OCR variants collapse onto the canonical row.
    normalized_name TEXT NOT NULL UNIQUE
);

INSERT INTO merchants (name, normalized_name) VALUES
    ('ROSSMANN MAGYARORSZAG KFT', 'ROSSMANN MAGYARORSZAG KFT'),
    ('ALDI MAGYARORSZAG ELELMISZER Bt.', 'ALDI MAGYARORSZAG ELELMISZER BT')
    ON CONFLICT (normalized_name) DO NOTHING;

-- Supported currencies. HUF is inserted first so it gets id = 1, which the
-- receipts.currency_id default below relies on. Amounts are stored as integers
-- in each currency's minor unit (HUF has 0 decimals, EUR has 2).
CREATE TABLE IF NOT EXISTS currencies (
    id SERIAL PRIMARY KEY,
    code TEXT NOT NULL UNIQUE
);

INSERT INTO currencies (code) VALUES ('HUF'), ('EUR')
    ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS receipts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) DEFAULT 1,
    merchant_id INTEGER NOT NULL REFERENCES merchants(id),
    currency_id INTEGER NOT NULL REFERENCES currencies(id) DEFAULT 1,
    total_amount INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    receipt_id INTEGER NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    price INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

INSERT INTO categories (name) VALUES
    ('Food'),
    ('Healthy'),
    ('Fastfood'),
    ('Clothing'),
    ('Housekeeping'),
    ('Drugs'),
    ('Alcohol'),
    ('Tobacco'),
    ('Electronics'),
    ('Entertainment'),
    ('Going out'),
    ('Transportation'),
    ('Travel'),
    ('Gifts'),
    ('Beauty'),
    ('Sports'),
    ('Education'),
    ('Home'),
    ('Other')
    ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS product_categories (
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

-- receipt_images backs the developer "debug" upload path: the original image is
-- saved to disk (stored_path, under IMAGE_STORE_DIR) so it can be viewed next to
-- its transcript on the UI. It also accumulates the raw OCR text, the parser
-- output (parse_json) and, later, a human-corrected transcript (clean_text) —
-- the labeled dataset the recognition flywheel learns from. receipt_id is
-- nullable / SET NULL on delete so captured training data outlives the receipt
-- (and a rejected scan can still be saved with no receipt at all).
CREATE TABLE IF NOT EXISTS receipt_images (
    id SERIAL PRIMARY KEY,
    receipt_id INTEGER REFERENCES receipts(id) ON DELETE SET NULL,
    user_id INTEGER NOT NULL REFERENCES users(id),
    stored_path TEXT NOT NULL,
    original_name TEXT,
    content_type TEXT,
    ocr_text TEXT NOT NULL DEFAULT '',
    clean_text TEXT,
    parse_json JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS receipt_images_user_id_idx ON receipt_images (user_id);

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

-- The explicit id=1 seed above does NOT advance the SERIAL sequence, so the
-- first auto-id INSERT (a real user) would otherwise collide on id=1. Bump the
-- sequence past the current max. Idempotent and safe to re-run.
SELECT setval(pg_get_serial_sequence('users', 'id'), GREATEST((SELECT MAX(id) FROM users), 1));

CREATE TABLE IF NOT EXISTS merchants (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    -- accent-free, uppercase, punctuation-free key for de-duplication; must match
    -- receipt.NormalizeName so OCR variants collapse onto the canonical row.
    normalized_name TEXT NOT NULL UNIQUE,
    -- Only verified merchants are matched against during parsing. Human-curated
    -- (seeded / created via the merchant API); the UNKNOWN sentinel and any
    -- legacy auto-created rows stay false so they can't poison detection. See
    -- sql/add_verified_merchant.sql.
    verified BOOLEAN NOT NULL DEFAULT false
);

-- Idempotent for databases created before verified existed.
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS verified BOOLEAN NOT NULL DEFAULT false;

INSERT INTO merchants (name, normalized_name, verified) VALUES
    ('ROSSMANN MAGYARORSZAG KFT', 'ROSSMANN MAGYARORSZAG KFT', true),
    ('ALDI MAGYARORSZAG ELELMISZER Bt.', 'ALDI MAGYARORSZAG ELELMISZER BT', true)
    ON CONFLICT (normalized_name) DO UPDATE SET verified = true;

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
    -- Structured result of re-parsing clean_text (the ground-truth parse). Filled
    -- in when the corrected transcript is saved; the approved subset is the
    -- flywheel's labeled corpus. See sql/add_flywheel.sql for the migration.
    clean_parse_json JSONB,
    -- Set when a human approves the clean transcript as ground truth; NULL means
    -- not yet approved. Kept as a timestamp (not a bare bool) so the approval is
    -- self-auditing. See sql/add_approved.sql for the standalone migration.
    approved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Idempotent for databases created before these columns existed (schema.sql runs
-- on every boot; the CREATE TABLE above only fires for a fresh database).
ALTER TABLE receipt_images ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP;
ALTER TABLE receipt_images ADD COLUMN IF NOT EXISTS clean_parse_json JSONB;

CREATE INDEX IF NOT EXISTS receipt_images_user_id_idx ON receipt_images (user_id);

-- merchant_aliases is the merchant arm of the recognition flywheel: a garbled
-- raw-OCR header variant (normalized) that approving a clean transcript taught us
-- maps to a canonical merchant. The parser matches headers against these too, so
-- long legal-name headers that score below the canonical-name threshold start
-- resolving. normalized_alias is UNIQUE so re-learning re-points rather than
-- duplicates.
CREATE TABLE IF NOT EXISTS merchant_aliases (
    id SERIAL PRIMARY KEY,
    merchant_id INTEGER NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    normalized_alias TEXT NOT NULL UNIQUE,
    source_image_id INTEGER REFERENCES receipt_images(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

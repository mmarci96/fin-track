CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Default user (id = 1) used for unauthenticated uploads.
INSERT INTO users (name) VALUES ('default');

CREATE TABLE merchants (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    -- accent-free, uppercase, punctuation-free key for de-duplication; must match
    -- receipt.NormalizeName so OCR variants collapse onto the canonical row.
    normalized_name TEXT NOT NULL UNIQUE
);

INSERT INTO merchants (name, normalized_name) VALUES
    ('ROSSMANN MAGYARORSZAG KFT', 'ROSSMANN MAGYARORSZAG KFT'),
    ('ALDI MAGYARORSZAG ELELMISZER Bt.', 'ALDI MAGYARORSZAG ELELMISZER BT');

CREATE TABLE receipts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) DEFAULT 1,
    merchant_id INTEGER NOT NULL REFERENCES merchants(id),
    total_amount INTEGER NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    receipt_id INTEGER NOT NULL REFERENCES receipts(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    price INTEGER NOT NULL
);

CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE product_categories (
    product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

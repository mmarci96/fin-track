-- auth service schema. Loaded as the database owner (auth_uat / auth_dev) so
-- the tables belong to the service role.
--
-- A handful of users only. Passwords are never stored in plaintext: the auth
-- service hashes them with bcrypt and writes the hash here (see `createuser`).
-- `app_user_id` is the fin-track users.id injected downstream as X-User-ID, so
-- the auth identity is decoupled from the application's own user row.
CREATE TABLE IF NOT EXISTS auth_users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    app_user_id   INTEGER NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

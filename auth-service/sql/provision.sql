-- provision.sql — idempotent provisioning of the auth service's own roles and
-- databases inside the SHARED postgres container. The auth service is a
-- cross-service controller, so its credentials live in dedicated databases
-- (`auth_uat`, `auth_dev`) fully decoupled from the fin-track app schema.
--
-- Run as the cluster superuser against any existing database, e.g.:
--   psql -U fin-track -d fin-track-db -f sql/provision.sql

-- Roles ---------------------------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_uat') THEN
        CREATE ROLE auth_uat LOGIN PASSWORD 'auth_uat_password';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_dev') THEN
        CREATE ROLE auth_dev LOGIN PASSWORD 'auth_dev_password';
    END IF;
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'auth_prod') THEN
        CREATE ROLE auth_prod LOGIN PASSWORD 'auth_prod_password';
    END IF;
END
$$;

-- Databases -----------------------------------------------------------------
SELECT 'CREATE DATABASE auth_uat OWNER auth_uat'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'auth_uat')\gexec

SELECT 'CREATE DATABASE auth_dev OWNER auth_dev'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'auth_dev')\gexec

SELECT 'CREATE DATABASE auth_prod OWNER auth_prod'
    WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'auth_prod')\gexec


-- Isolation -----------------------------------------------------------------
REVOKE CONNECT ON DATABASE auth_uat FROM PUBLIC;
GRANT  CONNECT ON DATABASE auth_uat TO auth_uat;

REVOKE CONNECT ON DATABASE auth_dev FROM PUBLIC;
GRANT  CONNECT ON DATABASE auth_dev TO auth_dev;

REVOKE CONNECT ON DATABASE auth_prod FROM PUBLIC;
GRANT  CONNECT ON DATABASE auth_prod TO auth_prod;

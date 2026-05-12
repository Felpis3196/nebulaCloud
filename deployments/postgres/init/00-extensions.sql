-- ============================================================================
-- NebulaCloud — Postgres bootstrap
-- Runs once on first cluster initialisation. Creates required extensions.
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

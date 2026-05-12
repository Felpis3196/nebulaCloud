-- =============================================================================
-- NebulaCloud — initial schema (DOWN)
-- =============================================================================
BEGIN;

DROP TABLE IF EXISTS audit_logs       CASCADE;
DROP TABLE IF EXISTS service_events   CASCADE;
DROP TABLE IF EXISTS domains          CASCADE;
DROP TABLE IF EXISTS builds           CASCADE;
DROP TABLE IF EXISTS deployments      CASCADE;
DROP TABLE IF EXISTS env_vars         CASCADE;
DROP TABLE IF EXISTS services         CASCADE;
DROP TABLE IF EXISTS projects         CASCADE;
DROP TABLE IF EXISTS memberships      CASCADE;
DROP TABLE IF EXISTS organizations    CASCADE;
DROP TABLE IF EXISTS sessions         CASCADE;
DROP TABLE IF EXISTS oauth_accounts   CASCADE;
DROP TABLE IF EXISTS users            CASCADE;

DROP FUNCTION IF EXISTS nebula_set_updated_at();

COMMIT;

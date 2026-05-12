-- =============================================================================
-- NebulaCloud — initial schema
-- =============================================================================
-- This migration is intentionally exhaustive: it defines every table required
-- by the MVP roadmap (Phases 1–8). Empty tables impose no cost and let later
-- phases focus purely on application code.
-- =============================================================================

BEGIN;

-- Required for gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "citext";

-- ----------------------------------------------------------------------------
-- Identity
-- ----------------------------------------------------------------------------
CREATE TABLE users (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  email           CITEXT      NOT NULL UNIQUE,
  password_hash   TEXT        NOT NULL,
  display_name    TEXT,
  avatar_url      TEXT,
  is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
  is_admin        BOOLEAN     NOT NULL DEFAULT FALSE,
  email_verified  BOOLEAN     NOT NULL DEFAULT FALSE,
  mfa_secret      TEXT,                 -- TOTP secret (nullable)
  mfa_enabled     BOOLEAN     NOT NULL DEFAULT FALSE,
  last_login_at   TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE oauth_accounts (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider            TEXT        NOT NULL,                       -- 'github', etc.
  provider_user_id    TEXT        NOT NULL,
  provider_login      TEXT,
  access_token_enc    BYTEA,                                      -- AES-GCM
  refresh_token_enc   BYTEA,
  scopes              TEXT[]      NOT NULL DEFAULT '{}',
  installation_id     BIGINT,                                     -- GitHub App install
  metadata            JSONB       NOT NULL DEFAULT '{}'::jsonb,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (provider, provider_user_id)
);

CREATE TABLE sessions (
  id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  refresh_token_hash  BYTEA       NOT NULL UNIQUE,                -- sha256(token)
  ip                  INET,
  user_agent          TEXT,
  expires_at          TIMESTAMPTZ NOT NULL,
  revoked_at          TIMESTAMPTZ,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX sessions_user_idx ON sessions(user_id);
CREATE INDEX sessions_expires_idx ON sessions(expires_at);

-- ----------------------------------------------------------------------------
-- Organisations and membership
-- ----------------------------------------------------------------------------
CREATE TABLE organizations (
  id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  slug        TEXT        NOT NULL UNIQUE,
  name        TEXT        NOT NULL,
  owner_id    UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  plan        TEXT        NOT NULL DEFAULT 'free',
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE memberships (
  organization_id  UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role             TEXT NOT NULL CHECK (role IN ('admin', 'developer', 'viewer')),
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (organization_id, user_id)
);
CREATE INDEX memberships_user_idx ON memberships(user_id);

-- ----------------------------------------------------------------------------
-- Projects, services, env vars
-- ----------------------------------------------------------------------------
CREATE TABLE projects (
  id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  organization_id          UUID        NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  slug                     TEXT        NOT NULL,
  name                     TEXT        NOT NULL,
  description              TEXT,
  repo_url                 TEXT,
  default_branch           TEXT        NOT NULL DEFAULT 'main',
  github_installation_id   BIGINT,                            -- ties to oauth_accounts
  created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (organization_id, slug)
);

CREATE TABLE services (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  project_id      UUID        NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  slug            TEXT        NOT NULL,
  name            TEXT        NOT NULL,
  type            TEXT        NOT NULL CHECK (type IN ('web', 'worker', 'cron', 'static')),
  status          TEXT        NOT NULL DEFAULT 'idle'
                  CHECK (status IN ('idle', 'building', 'deploying', 'running', 'failed', 'stopped')),
  build_config    JSONB       NOT NULL DEFAULT '{}'::jsonb, -- builder, dockerfile_path, env, etc.
  runtime_config  JSONB       NOT NULL DEFAULT '{}'::jsonb, -- cmd, port, healthcheck, replicas
  current_image   TEXT,                                      -- registry/image:tag of running version
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (project_id, slug)
);

CREATE TABLE env_vars (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id      UUID        NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  key             TEXT        NOT NULL,
  value_enc       BYTEA       NOT NULL,                          -- AES-GCM
  is_secret       BOOLEAN     NOT NULL DEFAULT TRUE,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (service_id, key)
);

-- ----------------------------------------------------------------------------
-- Deployments and builds
-- ----------------------------------------------------------------------------
CREATE TABLE deployments (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id      UUID        NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  triggered_by    UUID        REFERENCES users(id) ON DELETE SET NULL,
  trigger         TEXT        NOT NULL DEFAULT 'manual'
                  CHECK (trigger IN ('manual', 'webhook', 'rollback', 'retry')),
  commit_sha      TEXT,
  commit_message  TEXT,
  ref             TEXT,                                      -- e.g. refs/heads/main
  image_ref       TEXT,                                      -- registry/image:tag
  status          TEXT        NOT NULL DEFAULT 'queued'
                  CHECK (status IN ('queued', 'building', 'pushing', 'deploying', 'running', 'failed', 'canceled', 'rolled_back')),
  error_message   TEXT,
  metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at      TIMESTAMPTZ,
  finished_at     TIMESTAMPTZ,
  rolled_back_to  UUID        REFERENCES deployments(id) ON DELETE SET NULL
);
CREATE INDEX deployments_service_created_idx
  ON deployments (service_id, created_at DESC);

CREATE TABLE builds (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  deployment_id   UUID        NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
  worker_id       TEXT,
  status          TEXT        NOT NULL DEFAULT 'queued'
                  CHECK (status IN ('queued', 'cloning', 'detecting', 'building', 'pushing', 'success', 'failed', 'canceled')),
  detected_stack  TEXT,                                       -- 'node', 'python', 'go', 'dotnet', 'docker', ...
  builder_image   TEXT,
  log_object_key  TEXT,                                       -- where the build log was archived
  exit_code       INTEGER,
  error_message   TEXT,
  duration_ms     INTEGER,
  metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
  started_at      TIMESTAMPTZ,
  finished_at     TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX builds_deployment_idx ON builds(deployment_id);

-- ----------------------------------------------------------------------------
-- Domains and SSL
-- ----------------------------------------------------------------------------
CREATE TABLE domains (
  id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
  service_id      UUID        NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  hostname        TEXT        NOT NULL UNIQUE,
  is_primary      BOOLEAN     NOT NULL DEFAULT FALSE,
  ssl_status      TEXT        NOT NULL DEFAULT 'pending'
                  CHECK (ssl_status IN ('pending', 'issued', 'failed', 'disabled')),
  verification_token TEXT,
  verified_at     TIMESTAMPTZ,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX domains_service_idx ON domains(service_id);

-- ----------------------------------------------------------------------------
-- Service runtime events (status timeline)
-- ----------------------------------------------------------------------------
CREATE TABLE service_events (
  id          BIGSERIAL   PRIMARY KEY,
  service_id  UUID        NOT NULL REFERENCES services(id) ON DELETE CASCADE,
  type        TEXT        NOT NULL,                          -- 'started', 'stopped', 'restarted', 'unhealthy', ...
  payload     JSONB       NOT NULL DEFAULT '{}'::jsonb,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX service_events_service_created_idx
  ON service_events (service_id, created_at DESC);

-- ----------------------------------------------------------------------------
-- Audit log
-- ----------------------------------------------------------------------------
CREATE TABLE audit_logs (
  id              BIGSERIAL   PRIMARY KEY,
  actor_id        UUID        REFERENCES users(id) ON DELETE SET NULL,
  organization_id UUID        REFERENCES organizations(id) ON DELETE SET NULL,
  correlation_id  TEXT,
  action          TEXT        NOT NULL,                      -- 'user.login', 'project.create', ...
  target_type     TEXT,                                       -- 'project', 'service', ...
  target_id       UUID,
  ip              INET,
  user_agent      TEXT,
  metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
  created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX audit_logs_org_created_idx
  ON audit_logs (organization_id, created_at DESC);

-- ----------------------------------------------------------------------------
-- updated_at triggers
-- ----------------------------------------------------------------------------
CREATE OR REPLACE FUNCTION nebula_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DO $$
DECLARE
  tbl TEXT;
BEGIN
  FOR tbl IN
    SELECT table_name FROM information_schema.columns
    WHERE column_name = 'updated_at'
      AND table_schema = current_schema()
  LOOP
    EXECUTE format(
      'CREATE TRIGGER %I_set_updated_at BEFORE UPDATE ON %I
         FOR EACH ROW EXECUTE FUNCTION nebula_set_updated_at();',
      tbl, tbl
    );
  END LOOP;
END$$;

COMMIT;

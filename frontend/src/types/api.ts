/**
 * NebulaCloud — API DTO mirrors.
 *
 * These types intentionally mirror the Go DTOs used by the HTTP handlers.
 * Whenever the backend contract evolves (Phase 2+), keep this file in sync.
 */

export type Role = "admin" | "developer" | "viewer";

export interface Organization {
  id: string;
  slug: string;
  name: string;
  plan: string;
  created_at: string;
  updated_at: string;
}

export interface ApiEnvelope<T> {
  data: T;
  meta?: Record<string, unknown>;
}

export interface ApiErrorBody {
  error: {
    kind:
      | "validation"
      | "unauthorized"
      | "forbidden"
      | "not_found"
      | "conflict"
      | "rate_limited"
      | "unavailable"
      | "internal";
    code?: string;
    message: string;
    details?: Record<string, string>;
  };
}

// ----------------------------------------------------------------------------
// Identity
// ----------------------------------------------------------------------------

export interface User {
  id: string;
  email: string;
  display_name?: string;
  avatar_url?: string;
  is_admin: boolean;
  email_verified: boolean;
  mfa_enabled: boolean;
  last_login_at?: string;
  created_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  access_expiry: string;
  refresh_expiry: string;
  token_type: "Bearer";
  user: User;
}

// ----------------------------------------------------------------------------
// Projects / Services (Phase 2 contract — used today by mock data)
// ----------------------------------------------------------------------------

export type ServiceType = "web" | "worker" | "cron" | "static";
export type ServiceStatus =
  | "idle"
  | "building"
  | "deploying"
  | "running"
  | "failed"
  | "stopped";

export interface Project {
  id: string;
  organization_id: string;
  slug: string;
  name: string;
  description?: string;
  repo_url?: string;
  default_branch: string;
  /** GitHub App installation id when linked (webhook matching). */
  github_installation_id?: number;
  services_count: number;
  created_at: string;
  updated_at: string;
}

export interface Service {
  id: string;
  project_id: string;
  slug: string;
  name: string;
  type: ServiceType;
  status: ServiceStatus;
  current_image?: string;
  url?: string;
  region?: string;
  replicas: number;
  created_at: string;
  updated_at: string;
}

export interface EnvVar {
  id: string;
  service_id: string;
  key: string;
  is_secret: boolean;
  preview?: string; // last 4 chars for display
  updated_at: string;
}

// ----------------------------------------------------------------------------
// Deployments
// ----------------------------------------------------------------------------

export type DeploymentStatus =
  | "queued"
  | "building"
  | "pushing"
  | "deploying"
  | "running"
  | "failed"
  | "canceled"
  | "rolled_back";

export type DeploymentTrigger = "manual" | "webhook" | "rollback" | "retry";

export interface Deployment {
  id: string;
  service_id: string;
  service_name: string;
  project_id: string;
  project_name: string;
  trigger: DeploymentTrigger;
  status: DeploymentStatus;
  commit_sha?: string;
  commit_message?: string;
  ref?: string;
  image_ref?: string;
  error_message?: string;
  route_host?: string;
  listen_port?: number;
  duration_ms?: number;
  triggered_by?: { id: string; email: string };
  created_at: string;
  started_at?: string;
  finished_at?: string;
}

// ----------------------------------------------------------------------------
// Logs / Metrics
// ----------------------------------------------------------------------------

export type LogLevel = "debug" | "info" | "warn" | "error";

export interface LogLine {
  ts: string; // ISO timestamp
  level: LogLevel;
  service: string;
  message: string;
  correlation_id?: string;
}

/** Build/deploy log line from Redis history or WebSocket stream. */
export interface BuildLogLine {
  deployment_id?: string;
  level?: string;
  message: string;
  ts?: string;
  source?: string;
}

export interface MetricSeries {
  name: string;
  unit: string;
  points: { ts: number; value: number }[];
}

// ----------------------------------------------------------------------------
// Domains
// ----------------------------------------------------------------------------

export type SslStatus = "pending" | "issued" | "failed" | "disabled";

export interface Domain {
  id: string;
  service_id: string;
  service_name: string;
  hostname: string;
  is_primary: boolean;
  ssl_status: SslStatus;
  verified_at?: string;
  created_at: string;
}

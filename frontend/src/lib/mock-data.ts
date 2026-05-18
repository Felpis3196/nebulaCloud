/**
 * Mock fixtures for demos and unit-style stories. Prefer live hooks (`use-*`) in
 * dashboard routes — several global pages no longer import this file.
 */

import type {
  Deployment,
  DeploymentStatus,
  Domain,
  EnvVar,
  LogLevel,
  LogLine,
  MetricSeries,
  Project,
  Service,
} from "@/types/api";

const ORG_ID = "8c1d9b7e-5c4a-4ed1-9b14-79a4d2b5f3aa";

// ----------------------------------------------------------------------------
// Projects + services
// ----------------------------------------------------------------------------

export const MOCK_PROJECTS: Project[] = [
  {
    id: "p_payments",
    organization_id: ORG_ID,
    slug: "payments",
    name: "Payments",
    description: "Stripe-style billing API for the marketplace.",
    repo_url: "https://github.com/acme/payments-api",
    default_branch: "main",
    services_count: 3,
    created_at: daysAgo(42),
    updated_at: minutesAgo(8),
  },
  {
    id: "p_storefront",
    organization_id: ORG_ID,
    slug: "storefront",
    name: "Storefront",
    description: "Customer-facing Next.js app and image worker.",
    repo_url: "https://github.com/acme/storefront",
    default_branch: "main",
    services_count: 2,
    created_at: daysAgo(120),
    updated_at: minutesAgo(40),
  },
  {
    id: "p_analytics",
    organization_id: ORG_ID,
    slug: "analytics",
    name: "Analytics ETL",
    description: "Nightly ETL into the warehouse plus realtime metrics.",
    repo_url: "https://github.com/acme/analytics-etl",
    default_branch: "main",
    services_count: 4,
    created_at: daysAgo(76),
    updated_at: hoursAgo(3),
  },
];

export const MOCK_SERVICES: Service[] = [
  service("svc_payments_api", "p_payments", "api", "API", "web", "running", "v8a3c2f0", 3),
  service("svc_payments_worker", "p_payments", "worker", "Worker", "worker", "running", "v8a3c2f0", 1),
  service("svc_payments_cron", "p_payments", "cron", "Reconciler", "cron", "idle", "vd13bc02", 1),
  service("svc_storefront_web", "p_storefront", "web", "Web", "web", "running", "v3e7c1a9", 4),
  service("svc_storefront_img", "p_storefront", "img-worker", "Image worker", "worker", "running", "v3e7c1a9", 2),
  service("svc_analytics_api", "p_analytics", "api", "API", "web", "running", "v0c9b21e", 2),
  service("svc_analytics_etl", "p_analytics", "etl", "Nightly ETL", "cron", "idle", "v0c9b21e", 1),
  service("svc_analytics_stream", "p_analytics", "stream", "Stream", "worker", "building", undefined, 2),
  service("svc_analytics_dash", "p_analytics", "dashboard", "Dashboard", "web", "running", "v0c9b21e", 1),
];

// ----------------------------------------------------------------------------
// Deployments
// ----------------------------------------------------------------------------

const DEPLOY_STATUSES: DeploymentStatus[] = [
  "running",
  "running",
  "running",
  "building",
  "deploying",
  "failed",
  "running",
  "queued",
  "running",
  "running",
  "rolled_back",
  "running",
];

export const MOCK_DEPLOYMENTS: Deployment[] = DEPLOY_STATUSES.map((status, i) => {
  const svc = MOCK_SERVICES[i % MOCK_SERVICES.length]!;
  const proj = MOCK_PROJECTS.find((p) => p.id === svc.project_id)!;
  const created = new Date(Date.now() - i * 9 * 60 * 1000 - Math.random() * 60000);
  const duration = status === "running" ? 28000 + i * 1100 : status === "failed" ? 12000 : 18000;
  return {
    id: `dep_${i + 1}`,
    service_id: svc.id,
    service_name: svc.name,
    project_id: proj.id,
    project_name: proj.name,
    trigger: i % 5 === 0 ? "manual" : "webhook",
    status,
    commit_sha: ["8a3c2f0d3", "b71efc9aa", "d13bc02fe", "3e7c1a981", "0c9b21e7a"][i % 5],
    commit_message: [
      "fix(api): handle stripe webhook idempotency",
      "feat(checkout): add SCA challenge flow",
      "chore: bump go-redis to v9.7",
      "refactor(image-worker): batch uploads",
      "feat: add invoice PDF rendering",
    ][i % 5],
    ref: "refs/heads/main",
    image_ref: `registry.nebula.localhost:5000/${proj.slug}/${svc.slug}:8a3c2f0`,
    duration_ms: duration,
    triggered_by: { id: "u_owner", email: "ada@acme.test" },
    created_at: created.toISOString(),
    started_at: created.toISOString(),
    finished_at:
      status === "running" || status === "building" || status === "deploying" || status === "queued"
        ? undefined
        : new Date(created.getTime() + duration).toISOString(),
  };
});

// ----------------------------------------------------------------------------
// Logs
// ----------------------------------------------------------------------------

const LOG_TEMPLATES: { level: LogLevel; message: string }[] = [
  { level: "info", message: "GET /api/orders/831 200 18ms" },
  { level: "info", message: "POST /api/checkouts 201 142ms" },
  { level: "warn", message: "stripe webhook retry  attempt=2 reason=signature_mismatch" },
  { level: "info", message: "background worker tick  queue=invoices items=12" },
  { level: "debug", message: "cache hit  key=user:9281 ttl=120s" },
  { level: "error", message: "redis dial error  addr=redis:6379 attempt=3" },
  { level: "info", message: "build step 4/9 cached" },
  { level: "info", message: "running healthcheck on 0.0.0.0:8080/healthz" },
];

export const MOCK_LOGS: LogLine[] = Array.from({ length: 240 }).map((_, i) => {
  const tpl = LOG_TEMPLATES[i % LOG_TEMPLATES.length]!;
  const svc = MOCK_SERVICES[i % MOCK_SERVICES.length]!;
  return {
    ts: new Date(Date.now() - i * 1300).toISOString(),
    level: tpl.level,
    service: svc.name,
    message: tpl.message,
    correlation_id: ["01J9X2", "01J9X3", "01J9X4"][i % 3] + Math.floor(Math.random() * 9999).toString(36),
  };
});

// ----------------------------------------------------------------------------
// Metrics
// ----------------------------------------------------------------------------

export function makeSeries(name: string, base: number, jitter: number, points = 60): MetricSeries {
  const now = Date.now();
  return {
    name,
    unit: "",
    points: Array.from({ length: points }).map((_, i) => {
      const ts = now - (points - 1 - i) * 60_000;
      const wave = Math.sin(i / 6) * (jitter / 4);
      const noise = (Math.random() - 0.5) * jitter;
      return { ts, value: Math.max(0, base + wave + noise) };
    }),
  };
}

// ----------------------------------------------------------------------------
// Env vars
// ----------------------------------------------------------------------------

export const MOCK_ENV_VARS: EnvVar[] = [
  envvar("NODE_ENV", "production", false),
  envvar("DATABASE_URL", "postgres://... 1d2f", true, "1d2f"),
  envvar("REDIS_URL", "redis://... a07c", true, "a07c"),
  envvar("STRIPE_SECRET_KEY", "sk_live_... 0021", true, "0021"),
  envvar("PORT", "8080", false),
  envvar("LOG_LEVEL", "info", false),
];

// ----------------------------------------------------------------------------
// Domains
// ----------------------------------------------------------------------------

export const MOCK_DOMAINS: Domain[] = [
  {
    id: "d1",
    service_id: "svc_storefront_web",
    service_name: "Web",
    hostname: "shop.acme.test",
    is_primary: true,
    ssl_status: "issued",
    verified_at: hoursAgo(48),
    created_at: daysAgo(40),
  },
  {
    id: "d2",
    service_id: "svc_payments_api",
    service_name: "API",
    hostname: "api.acme.test",
    is_primary: true,
    ssl_status: "issued",
    verified_at: hoursAgo(96),
    created_at: daysAgo(80),
  },
  {
    id: "d3",
    service_id: "svc_analytics_dash",
    service_name: "Dashboard",
    hostname: "stats.acme.test",
    is_primary: false,
    ssl_status: "pending",
    created_at: hoursAgo(2),
  },
];

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

function service(
  id: string,
  projectId: string,
  slug: string,
  name: string,
  type: Service["type"],
  status: Service["status"],
  image: string | undefined,
  replicas: number,
): Service {
  return {
    id,
    project_id: projectId,
    slug,
    name,
    type,
    status,
    current_image: image,
    url: type === "web" ? `https://${slug}.${projectId.replace(/^p_/, "")}.nebula.app` : undefined,
    region: "us-east-1",
    replicas,
    created_at: daysAgo(30),
    updated_at: minutesAgo(2),
  };
}

function envvar(key: string, _value: string, secret: boolean, preview?: string): EnvVar {
  return {
    id: `ev_${key.toLowerCase()}`,
    service_id: "svc_payments_api",
    key,
    is_secret: secret,
    preview,
    updated_at: daysAgo(2),
  };
}

function daysAgo(n: number) {
  return new Date(Date.now() - n * 86_400_000).toISOString();
}
function hoursAgo(n: number) {
  return new Date(Date.now() - n * 3_600_000).toISOString();
}
function minutesAgo(n: number) {
  return new Date(Date.now() - n * 60_000).toISOString();
}

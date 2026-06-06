# NebulaCloud — local backend boot

This guide validates **Part A** of the backend plan: a healthy Docker stack, migrations, and the phases 0–5 deploy pipeline.

## Prerequisites

- Docker 24+ with Compose v2
- On Windows: Docker Desktop with WSL2 (workers need `/var/run/docker.sock`)
- Host entries (`make hosts`):

```text
127.0.0.1  api.nebula.localhost app.nebula.localhost grafana.nebula.localhost
           traefik.nebula.localhost registry.nebula.localhost
```

## Required secrets

Copy `.env.example` to `.env` only if you need overrides. Compose loads `.env.example` by default.

| Variable | Rule |
|----------|------|
| `NEBULA_JWT_SECRET` | Strong random value in production |
| `NEBULA_PASSWORD_PEPPER` | Strong random value in production |
| `NEBULA_SECRETS_KEY` | Base64 of **exactly 32 bytes** (`openssl rand -base64 32`) |

Optional GitHub:

- `NEBULA_GITHUB_APP_WEBHOOK_SECRET` — HMAC for push webhooks
- `NEBULA_GITHUB_APP_CLIENT_ID` + `NEBULA_GITHUB_OAUTH_REDIRECT_URL` — OAuth start/callback

## Start the stack

From the **repo root** (not `backend/`):

```bash
docker compose up -d --build
# or: make up

make ps          # wait until postgres, redis, api, frontend are healthy
```

`docker-compose.override.yml` is merged automatically (frontend + API dev).

Development enables auto-migrations and the web terminal on the API (override file):

- `NEBULA_AUTO_MIGRATE=true`
- `NEBULA_TERMINAL_ENABLED=true` (requires Docker socket on `api`)

Manual migrations (production or if auto-migrate is off):

```bash
make migrate-up
make migrate-down   # rolls back one version
```

## Smoke checks

```bash
curl -s http://api.nebula.localhost/healthz
curl -s http://api.nebula.localhost/readyz
```

End-to-end (phases 0–5):

1. `POST /api/v1/auth/register` and `POST /api/v1/auth/login`
2. Create organization and project
3. `PATCH` project with `repo_url`; optional GitHub installation
4. Create service and `POST .../deployments` (or GitHub push webhook)
5. Confirm build-worker → registry → runtime-agent → container with label `nebula_service=<uuid>`
6. App URL: `http://<service>.<project>.nebula.localhost`

## Phases 6–9 endpoints

| Feature | Endpoint |
|---------|----------|
| Log stream (WS) | `GET /api/v1/services/{id}/logs/stream?deployment_id=<uuid>&token=<jwt>` |
| Custom domains | `GET/POST/DELETE /api/v1/services/{id}/domains` |
| Verify domain | `POST /api/v1/services/{id}/domains/{domainId}/verify` |
| Web terminal (WS) | `GET /api/v1/services/{id}/terminal?token=<jwt>` (developer+, `NEBULA_TERMINAL_ENABLED=true`) |
| Metrics | `GET /api/v1/services/{id}/metrics` (cAdvisor + `container_label_nebula_service`) |

## Worker health

```bash
docker compose exec build-worker /app/build-worker healthcheck
docker compose exec runtime-agent /app/runtime-agent healthcheck
```

## Known dev issues

- Traefik + Docker provider on Windows can be flaky — check `docker compose logs traefik`
- Runtime network defaults to `nebula_platform` (matches Traefik Docker provider)
- Promtail maps `nebula_service` label → Loki `service` label for log queries
- **cAdvisor on Docker Desktop (WSL2):** logs about `machine-id`, crio/mesos/podman are normal. Success line: `Registration of the docker container factory successfully`. Errors like `layerdb/mounts/.../mount-id` for container `18b2eb2d...` are usually cAdvisor monitoring **itself** — safe to ignore. After changing the image, run `docker compose up -d --force-recreate cadvisor`.

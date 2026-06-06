# NebulaCloud — Deploy Flow

This document describes the end-to-end lifecycle of a single deployment,
from "user clicks Deploy" to "container is serving traffic".

## States

A deployment moves through the following states (see `deployments.status`):

```
queued → building → pushing → deploying → running
                                       ↘ failed
                                       ↘ canceled
running → rolled_back  (when a newer deployment supersedes and a rollback is requested)
```

`builds.status` runs in parallel with the build leg:

```
queued → cloning → detecting → building → pushing → success
                                                  ↘ failed
                                                  ↘ canceled
```

## Sequence

```mermaid
sequenceDiagram
    participant U as User
    participant FE as Dashboard
    participant API
    participant DB as Postgres
    participant Q as Redis (asynq)
    participant BW as Build Worker
    participant REG as Registry
    participant RA as Runtime Agent
    participant TR as Traefik

    U->>FE: click Deploy on a service
    FE->>API: POST /services/:id/deploy
    API->>DB: insert deployment(queued)
    API->>Q: enqueue build.run
    API-->>FE: 202 deployment_id

    BW->>Q: pull build.run
    BW->>DB: build(cloning)
    BW->>BW: git clone {repo}@{sha}
    BW->>DB: build(detecting)
    BW->>BW: detect stack (Dockerfile? buildpacks?)
    BW->>DB: build(building)
    BW->>BW: pack build OR docker build
    BW-->>FE: WS build.progress (via API multiplexer)
    BW->>DB: build(pushing)
    BW->>REG: docker push
    BW->>DB: build(success), deployment(deploying), set image_ref
    BW->>Q: enqueue deploy.run

    RA->>Q: pull deploy.run
    RA->>RA: docker pull image_ref
    RA->>RA: docker run stable name + Traefik labels on nebula_platform
    RA->>RA: wait container HTTP on listen_port
    RA->>TR: write nebula-*.yml + verify router (file provider)
    RA->>RA: remove orphan containers for service
    RA->>DB: deployment(running), service.current_image=image_ref

    Note over RA,TR: Failure path<br/>RA marks deployment(failed)<br/>removes route file if written
```

### MVP runtime (local compose)

- Workloads join **`nebula_platform`** (Traefik DNS to `nebula-svc-*` container names).
- **Routing (dev):** `runtime-agent` writes **`deployments/traefik/dynamic/nebula-*.yml`** and verifies via Traefik API (`NEBULA_TRAEFIK_API_URL`). Docker provider is unreliable on Docker Desktop; file routes are primary.
- **One container per service**: stable name `nebula-svc-<prefix>`; previous container with same name is removed before `docker run`; orphan labeled containers are pruned after success.
- **Listen port:** `build-worker` detects `EXPOSE` / stack default, persists `runtime_config.listen_port`, passes it in `deploy.run`.
- **Not yet implemented:** automated rollback container swap, full health-gated traffic switch with keep-old-on-failure.

## Rollback

```mermaid
sequenceDiagram
    participant U as User
    participant API
    participant DB
    participant Q
    participant RA as Runtime Agent

    U->>API: POST /deployments/:id/rollback
    API->>DB: validate target.image_ref present
    API->>DB: insert deployment(trigger=rollback, image_ref=target.image_ref)
    API->>Q: enqueue deploy.run (skip build leg)
    Q->>RA: deploy.run
    RA->>RA: docker run target image, replace
    RA->>DB: deployment(running), source.rolled_back_to=current.id
```

Rollback skips the build pipeline entirely; we re-use the already-pushed
image. This is what makes rollbacks fast and safe.

## Failure handling

- **Build failure**: deployment goes to `failed`. The previous running
  container is untouched.
- **Push failure**: same as build failure.
- **Container start / healthcheck failure**: agent stops the new container,
  restores the previous one, marks deployment `failed`.
- **Crash loop after rollout**: the agent emits `service.unhealthy` events.
  An auto-restart policy is applied (with exponential backoff and a cap).

## Webhooks (Phase 3+)

Pushes to a connected branch arrive on the GitHub webhook receiver
(`POST /api/v1/webhooks/github`). The handler:

1. Verifies the HMAC signature using `NEBULA_GITHUB_APP_WEBHOOK_SECRET` when configured.
2. Resolves projects by normalized `repo_url` and `default_branch` (from the push `ref`).
   When the payload includes `installation.id` and the project row has
   `github_installation_id` set, only projects for that installation match
   (avoids accidental deploys from fork URLs).
3. Inserts a deployment with `trigger='webhook'` and dispatches build.run for each service in matching projects.

Idempotency: GitHub may redeliver. We dedupe on `(service_id, commit_sha,
trigger)` within a 60s window.

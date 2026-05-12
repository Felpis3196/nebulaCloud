# NebulaCloud

> Self-hosted PaaS in the spirit of Railway, Render, Heroku, and Vercel — built as an enterprise-grade reference architecture.

NebulaCloud lets developers connect a Git repository, auto-detect the stack, build an OCI image, and roll out a running container behind an HTTPS endpoint with realtime logs, metrics, and a web terminal — all on infrastructure you control.

## Status

This repository is being built incrementally following the roadmap in [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

| Phase | Theme                              | Status |
| ----- | ---------------------------------- | ------ |
| 0     | Foundation (infra + platform core) | done   |
| 1     | Identity (auth, RBAC, audit)       | done   |
| 1.5   | Frontend Phase 1 (dashboard MVP)   | done   |
| 2     | Projects, services, env vars       | done   |
| 3     | GitHub App + webhooks              | in progress (prep)   |
| 4     | Build pipeline (Buildpacks)        | todo   |
| 5     | Runtime agent + Traefik            | todo   |
| 6     | Logs / metrics streaming           | todo   |
| 7     | Custom domains + ACME              | todo   |
| 8     | Web terminal                       | todo   |
| 9     | Hardening, tests, polish           | todo   |

## High-level architecture

```
                 ┌─────────────────────────────┐
                 │  Next.js 15 dashboard (TS)  │
                 └──────────────┬──────────────┘
                                │  HTTPS / WSS
                       ┌────────▼────────┐
                       │     Traefik     │  TLS, routing, ACME
                       └────────┬────────┘
                                │
                       ┌────────▼─────────┐
                       │  API Gateway     │  JWT, RBAC, rate-limit
                       │  (Go modular     │
                       │   monolith)      │
                       └────────┬─────────┘
            ┌──────────────┬────┴────┬──────────────┐
            ▼              ▼         ▼              ▼
     ┌────────────┐  ┌──────────┐ ┌────────┐  ┌──────────────┐
     │ Postgres   │  │  Redis   │ │  Loki  │  │  Prometheus  │
     └────────────┘  └────┬─────┘ └────────┘  └──────────────┘
                          │ asynq queue
              ┌───────────┴───────────┐
              ▼                       ▼
       ┌─────────────┐         ┌──────────────┐
       │ Build       │         │ Runtime      │
       │ Worker(s)   │         │ Agent        │
       │ Buildpacks  │ push    │ Docker SDK   │
       │             │────────▶│ + Traefik    │
       └─────────────┘ images  └──────────────┘
              │                       │
              ▼                       ▼
       ┌─────────────┐         ┌──────────────┐
       │ OCI         │         │ User         │
       │ Registry    │         │ containers   │
       └─────────────┘         └──────────────┘
```

See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) for full details, sequence diagrams, and rationale.

## Stack

| Layer          | Tech                                                         |
| -------------- | ------------------------------------------------------------ |
| Backend        | Go 1.23, chi, pgx/v5, redis/go-redis, asynq                  |
| Frontend       | Next.js 15, TypeScript, Tailwind, shadcn/ui                  |
| Database       | PostgreSQL 16                                                |
| Cache / Queue  | Redis 7                                                      |
| Containers     | Docker, Cloud Native Buildpacks                              |
| Reverse proxy  | Traefik v3                                                   |
| Observability  | Prometheus, Grafana, Loki, Promtail, OpenTelemetry SDK       |
| CI             | GitHub Actions                                               |
| Local infra    | Docker Compose                                               |
| Future         | Kubernetes (orchestrator interface designed in)              |

## Local setup

### Prerequisites

- Docker 24+ and Docker Compose v2
- Go 1.23+ (for native dev outside containers)
- Node.js 20+ (for the dashboard, when implemented)
- `make`

### First run

```bash
git clone <this repo> nebula-cloud
cd nebula-cloud

cp .env.example .env
# edit .env — at minimum rotate NEBULA_JWT_SECRET, NEBULA_SECRETS_KEY,
# NEBULA_PASSWORD_PEPPER and POSTGRES_PASSWORD.

cp frontend/.env.example frontend/.env.local
# defaults work out of the box; override NEXT_PUBLIC_API_URL if you
# point Traefik at a different host.

# Add local hostnames so Traefik routing works
make hosts          # prints the line, copy it into your hosts file

make up             # boots Postgres, Redis, Traefik, Prom, Grafana, Loki, Promtail, Registry, API, Frontend
make ps             # verify everything is healthy

curl http://api.nebula.localhost/healthz
# {"status":"ok",...}

# Dashboard: http://app.nebula.localhost
```

### Running the frontend on its own

The dashboard is a stock Next.js 15 app, so you can also run it natively:

```bash
cd frontend
npm install
npm run dev          # http://localhost:3000
```

It expects the API at `NEXT_PUBLIC_API_URL` (default `http://api.nebula.localhost`).
Useful scripts:

| Command            | What it does                                    |
| ------------------ | ----------------------------------------------- |
| `npm run dev`      | Hot-reloading dev server                        |
| `npm run build`    | Production build (`.next/`)                     |
| `npm run start`    | Serve a production build                        |
| `npm run lint`     | `next lint` (ESLint + Next rules)               |
| `npm run typecheck`| `tsc --noEmit` against the strict config        |
| `npm run format`   | Prettier (with the `prettier-plugin-tailwindcss`) |

### Useful URLs (after `make up`)

| Service       | URL                                       |
| ------------- | ----------------------------------------- |
| API           | http://api.nebula.localhost               |
| Dashboard     | http://app.nebula.localhost               |
| Traefik UI    | http://traefik.nebula.localhost:8080      |
| Grafana       | http://grafana.nebula.localhost           |
| Prometheus    | http://localhost:9090                     |
| Loki          | http://localhost:3100                     |
| Registry      | http://registry.nebula.localhost:5000     |

### Common commands

```bash
make up           # start the full stack
make ps           # status
make logs SVC=api # tail one service
make psql         # interactive psql
make test         # run Go tests
make build        # build all backend binaries
make down         # stop the stack
make nuke         # stop AND drop volumes (destructive)
```

See `make help` for the full list.

## Repository layout

```
nebula-cloud/
├── backend/              # Go control plane, build worker, runtime agent
├── frontend/             # Next.js dashboard
├── deployments/          # Infra config (traefik, prom, grafana, loki, ...)
├── docs/                 # Architecture, deploy flow, security notes
├── docker-compose.yml    # Base stack
├── docker-compose.dev.yml# Dev overrides (hot reload, exposed ports)
└── Makefile
```

The backend follows a **modular monolith** split by bounded context (DDD-lite), with each module exposing `domain → application → infrastructure → interfaces` layers. See [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md).

## Security

Production checklist (live as the platform grows): JWT with rotated `kid`, Argon2id passwords, encrypted env vars (AES-256-GCM), refresh-token rotation, audit log, RBAC, container isolation flags, rate limiting, CORS allow-list, security headers (HSTS, CSP, X-Frame-Options). See [`docs/SECURITY.md`](docs/SECURITY.md) when populated.

## Contributing

This is a portfolio-grade project — PRs and issues with concrete improvements (architectural critique, security review, hardening, tests) are welcome.

## License

MIT

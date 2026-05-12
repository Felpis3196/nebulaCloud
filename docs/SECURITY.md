# NebulaCloud — Security

This document is the living security baseline for the platform. It covers
authentication, secrets handling, transport, container isolation, audit,
and the threat-model assumptions we are *not* yet defending against.

## Threat model summary

| Trust boundary                   | Defenses                                            |
| -------------------------------- | --------------------------------------------------- |
| Public internet → Traefik         | TLS 1.2+, HSTS, ACME, strict SNI                    |
| Traefik → API gateway             | Internal network, request-id propagation            |
| API → Postgres / Redis            | Internal network, password auth, planned mTLS       |
| API → Build Worker (queue)        | Redis ACLs, signed payloads (planned)               |
| Build Worker → Registry           | Bearer tokens, registry write scope only            |
| Runtime Agent → Docker engine     | Local socket, no network exposure                   |
| User container → other containers | Per-service network, read-only rootfs, no-new-privs |

Out of scope for the MVP: tenant-level network isolation between
organisations, sandboxed build (gVisor / Firecracker), HSM-backed signing
keys.

## Authentication

- Passwords hashed with **Argon2id** (`time=3, memory=64 MiB, threads=4`).
  A server-side `NEBULA_PASSWORD_PEPPER` is mixed into the input so a
  database leak alone does not enable offline attacks.
- Access tokens: short-lived JWTs (15 min) with HS256 signature. The
  signing key is `NEBULA_JWT_SECRET`. Migration to RS256 with a JWK set
  is straightforward (we already store a `kid` claim).
- Refresh tokens: 256-bit opaque random values, sha256-hashed in
  `sessions.refresh_token_hash`. Rotated on every use; reuse of a
  rotated token is treated as compromise and revokes the entire session.
- OAuth GitHub flow uses PKCE; installation tokens are encrypted at rest.

## Secrets at rest

User-supplied secrets (env vars, OAuth tokens, GitHub App tokens, etc.)
are encrypted with **AES-256-GCM** using `NEBULA_SECRETS_KEY` (32-byte
base64). Layout: `nonce(12) || ciphertext+tag`.

The `secrets.Sealer` interface lets us swap the in-process key for a
KMS-backed implementation (AWS KMS, GCP KMS, Vault Transit) without
touching any callsite.

## Transport

- TLS 1.2+ everywhere on the public edge (Traefik).
- HSTS (`max-age=63072000; includeSubDomains; preload`) by default.
- CSP: `default-src 'self'`. Tightened per-page by the dashboard.
- `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`,
  `Referrer-Policy: strict-origin-when-cross-origin`.
- `Permissions-Policy` denies geolocation / microphone / camera by default.

## Authorization (RBAC)

Three roles per organisation:

| Role       | Capabilities                                                |
| ---------- | ----------------------------------------------------------- |
| admin      | Manage members, billing, projects, deployments, secrets     |
| developer  | Manage projects, deployments, env vars; cannot manage org   |
| viewer     | Read-only access to projects, deployments, logs, metrics    |

Middleware uses `auth.Role.AtLeast()` to gate routes. Every privileged
mutation writes an `audit_logs` entry with the actor, action, target,
correlation id, and request metadata.

## Container isolation (Phase 5+)

User containers are launched with:

- Non-root user (uid >= 1000) — buildpacks already enforce this.
- `--security-opt no-new-privileges`.
- `--read-only` rootfs with explicit `--tmpfs /tmp /run` mounts.
- `--cap-drop ALL` and a minimal `--cap-add` allow-list (none by default).
- CPU + memory + pid limits per service tier.
- A dedicated Docker network per organisation (Phase 7 enhancement).

## Rate limiting

Token-bucket per-IP and per-user, backed by Redis. Defaults:
`NEBULA_RATE_LIMIT_RPS=20`, `NEBULA_RATE_LIMIT_BURST=40`. Sensitive
endpoints (login, refresh, register, password reset) get a stricter
bucket configured in their handlers.

## Audit

Every action that mutates state, every login attempt (success or
failure), and every admin change to RBAC writes a row to `audit_logs`
with `correlation_id` so an end-to-end trace can be reconstructed.

## Reporting a vulnerability

For now: open a private GitHub Security Advisory on the repository.
A formal `security@` mailbox will be set up before any production
deployment.

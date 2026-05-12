# Contributing to NebulaCloud

Thanks for your interest. This is a portfolio-grade reference platform —
contributions that improve clarity, security, or test coverage are very
welcome.

## Local setup

```bash
git clone <repo> nebula-cloud
cd nebula-cloud
cp .env.example .env
make hosts                 # add the printed line to /etc/hosts
make up                    # bring up the full stack
curl http://api.nebula.localhost/healthz
```

Useful targets:

- `make tidy` — go mod tidy
- `make test` — run unit tests with race detector
- `make build` — build all backend binaries
- `make logs SVC=api` — tail logs for one service

## Code layout

See [`docs/ARCHITECTURE.md`](ARCHITECTURE.md) for the layered module
layout. New code should respect the layering rules:

- `domain` is the inner ring — pure business types and ports.
- `application` orchestrates `domain` via ports.
- `infrastructure` adapts to outer systems (databases, APIs, queues).
- `interfaces` adapts to delivery mechanisms (HTTP, WS, CLI).

## Style

- `gofmt` + `goimports` are mandatory (CI fails otherwise).
- Run `make lint` before pushing — `golangci-lint` is configured in
  `backend/.golangci.yml`.
- Public types and packages **must** carry doc comments per `revive`.
- Errors returned from public functions should be `*platform/errors.Error`
  values (or wrap one) so the HTTP layer can translate them to status codes.
- Logs use `log/slog` via `platform/logger`; never use `fmt.Println` or
  `log.Printf` outside of binary entrypoints.

## Tests

- Prefer unit tests at the application layer (mock the repos).
- Integration tests live in `backend/test/integration/` and rely on the
  Compose stack; gate them behind a `+build integration` tag.
- For HTTP handlers, prefer `httptest.NewServer` + table-driven tests.

## Commit messages

Conventional Commits are encouraged:

```
feat(identity): add refresh-token rotation
fix(httpx): preserve correlation id on panic
docs(deploy-flow): clarify rollback semantics
```

## Pull requests

- Keep PRs scoped to a single topic.
- Include a screenshot or curl trace for any user-visible change.
- Reference the related phase in `ARCHITECTURE.md` ("Phase 4 — build worker").
- Update tests and docs in the same PR.

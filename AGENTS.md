# AGENTS.md

Landlord is a multi-tenant identity/authorization service for applications,
built on AWS Cognito (user pools) and DynamoDB. It ships as one Go binary that
is both a CLI (`landlord tenant ...`, `login`, `refresh`, ...) and a server
(`landlord server`) exposing a REST API and the Landlord Console web app.

## Layout

- `main.go`, `cmd/` — cobra CLI and server entry points.
- `api/` — OpenAPI spec and mustache templates; `make gen` regenerates the Go
  client (`pkg/`) and gin server stubs. Do not hand-edit generated code.
- `internal/appenv` — env-var driven `Settings` (`LANDLORD_*`), see README.
- `internal/appcore` — `Clients` bundle (AWS config, Cognito, DynamoDB, settings).
- `internal/authx` — Cognito logic: user pools, pool *clients*, password and
  refresh auth, MFA, invites, tenants.
- `internal/db`, `internal/multi` — DynamoDB persistence and domain types.
- `internal/rest` — REST handlers (`handle_*.go`), `App` interface in `call_impl.go`.
- `internal/webapp` — Console (gin + templates). `internal/perms` — OPA/rego policy.
- `internal/jwx` — JWT validation. `internal/mocks` — mockery output.
- `internal/apptest` — LocalStack/AWS-emulator test setup helpers.

## Build and test

- `make setup` installs tools (goimports, mockery, staticcheck, ...).
- `make build` runs `gen`, mockery, goimports, tests (`AWS_PROFILE=localstack`),
  staticcheck, vet, `go mod tidy`. Set `LANDLORD_SKIP_TESTS=1` to skip tests.
- `go test ./...` runs unit tests; integration-style tests skip themselves when
  LocalStack is not running (`apptest.InitLocalStack` returns false).
- `make itest` runs tests per directory without caching.
- There is no real AWS account for development. Test against LocalStack or the
  AWS Emulator (see TODO.md), never production.

## Conventions

- Tests use testify suites and the mockery mocks in `internal/mocks`
  (regenerate with `mockery` after changing the `AWSCognitoIdentityProvider` interface).
- REST handlers get config through `a.Settings()`; CLI commands through `cs.Settings()`.
- Errors surfaced to HTTP callers use `internal/apperr`.
- Run `gofmt`/`goimports` before committing; CI runs `make setup && make rel`.

## User pool client selection

A Cognito user pool may hold several app clients (e.g. Landlord's plus an MCP
client). `authx.GetUserPoolClient(ctx, cogIdp, poolID, preferred)` resolves
which one Landlord uses: a pool with exactly one client uses it; with several,
`preferred` (the client name or ID, from `LANDLORD_DEFAULT_USER_POOL_CLIENT`
via `Settings().UserPoolClient()`) must match exactly one, else an error. Always
pass the setting through; never call a "singleton" lookup.

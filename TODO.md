# TODO

## Done
- [x] Honor `LANDLORD_DEFAULT_USER_POOL_CLIENT` so pools with multiple app
      clients work (login, refresh, password flows, CLI).

## Next
- [ ] Deploy the fix and set `LANDLORD_DEFAULT_USER_POOL_CLIENT` to
      `Prod-landlord-default-user-pool-client` (or `ghut4vs4ub4ri7tnp70idb53d`).
- [ ] Integration tests against the AWS Emulator (no AWS account / LocalStack
      license available): verify it implements `cognito-idp` ListUserPoolClients,
      DescribeUserPoolClient, CreateUserPoolClient, InitiateAuth, plus DynamoDB
      and Lambda (a 501 `NotImplemented` means unsupported). Add a multi-client
      integration test in `internal/rest`/`internal/apptest`.
- [ ] Pass `preferred` client per tenant (store a client ID on the tenant) rather
      than one global setting.
- [ ] Upgrade deps: `go get -u ./... && go mod tidy` (aws-sdk-go-v2, gin, opa,
      jwx, ...); review OPA and jwx major-version changes. Go toolchain: go.mod
      says 1.26.0, CI 1.26.1, Dockerfile uses unpinned `golang:alpine`; pin them.
- [ ] CI: add `go test ./...` / `go vet` / `staticcheck` as explicit steps (today
      tests only run inside `make rel`); bump actions (checkout, setup-go).
- [ ] Replace deprecated `::set-output` in ci.yaml; `golang.org/x/lint/golint`
      is deprecated in `make setup`.
- [ ] Remove committed artifacts/noise: stray `.env` files, `session.json`,
      built binaries are ignored but check `tmp/`.
- [ ] Paginate `ListUserPoolClients` (only first page is read today).

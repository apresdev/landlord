# TODO

## Done
- [x] Honor `LANDLORD_DEFAULT_USER_POOL_CLIENT` so pools with multiple app
      clients work (login, refresh, password flows, CLI).

## Next
- [ ] Deploy the fix and set `LANDLORD_DEFAULT_USER_POOL_CLIENT` to
      `Prod-landlord-default-user-pool-client` (or `ghut4vs4ub4ri7tnp70idb53d`).
- [x] Integration tests run against the AWS Emulator (`scripts/aws-emulator`),
      including a multi-client pool test. Fixed helper/pagination bugs found on the way.
- [ ] Emulator gap: `InitiateAuth` succeeds without `SECRET_HASH` for a client that
      has a secret (real Cognito rejects it). Don't rely on the emulator to catch that.
- [x] LocalStack support removed; CI runs the tests against the AWS Emulator.
- [ ] Pass `preferred` client per tenant (store a client ID on the tenant) rather
      than one global setting.
- [x] Upgrade deps and Go (1.27.4); go.mod, CI and Dockerfile pinned.
- [ ] CI: add `go test ./...` / `go vet` / `staticcheck` as explicit steps (today
      tests only run inside `make rel`); bump actions (checkout, setup-go).
- [ ] Replace deprecated `::set-output` in ci.yaml; `golang.org/x/lint/golint`
      is deprecated in `make setup`.
- [ ] Remove committed artifacts/noise: stray `.env` files, `session.json`,
      built binaries are ignored but check `tmp/`.
- [ ] Paginate `ListUserPoolClients` (only first page is read today).
- [x] Fixed `getUserPoolDescriptions` looping forever with more than 10 user pools.
- [ ] The emulator install uses the unpinned `LTS/latest` package (CI and scripts/aws-emulator); a new release can change behaviour. Pin a version if one becomes available.

# Separate cached reads from SQL execution budgets

## 1. Problem

Cached wpgx reads start `timeout` before DCache. A slow cache miss can exhaust
that context before SQL starts. A local delayed-cache experiment reproduces
this independently of a slow database.

## 2. Behavior

- Preserve `timeout` as the overall operation budget for existing queries.
- Add optional `sql_timeout` for cached `:one` and `:many` queries. Start it inside
  the database callback; it remains bounded by the operation/caller deadline.
- Preserve cache keys, TTLs, invalidation, return values and public method signatures.

## 3. Decisions

Use an opt-in annotation, not a global change to documented timeout semantics.
Consumers can opt in with a 1s operation budget and a 250ms SQL budget. This does not change Redis configuration or singleflight error handling.

## 4. Implementation

Parse/validate the annotation, render its context inside both cached read
callbacks, document it, and exercise generated output. Consumer adoption remains explicit and separate from this generator change.

Normalize ANTLR 4.13.1 exit workarounds during SQLite generation: move the
dummy goto before its label instead of leaving it after return. Keep the
grammar, ANTLR version and parser tables unchanged. This fixes 115 existing
full-vet findings; regeneration retains the fix.

## 5. Verification

Run option tests, generator golden/compatibility tests, pinned Go 1.26.8 build,
formatting and CI-equivalent tests/vet. A consuming service additionally verifies slow cache misses, SQL deadline
enforcement and parent cancellation against its runtime dependency versions. No production access is needed for those tests.

## 6. Authorization and checklist

User authorized isolating cache/SQL budgets and creating PRs on 2026-10-23.
No merge or release is included.

- [x] Regression tests fail before implementation.
- [x] Implement opt-in annotation and regenerate intended fixtures.
- [x] Verify legacy generated behavior and runtime consumer.
- [ ] Review, run publication checks and create linked PRs.

## 7. Results

- Parser regression failed before implementation, then passed.
- Go 1.26.8: `make build CGO_ENABLED=0 COMMIT_HASH=v2.4.0-dev` and
  `CGO_ENABLED=0 go build ./...` passed; all nested generated fixtures built.
- `go run ./scripts/test-local -- go test -count=1 -tags=examples -timeout=20m ./...`
  passed across 34 test-bearing packages with command-scoped external proxies
  and localhost exclusions. An earlier direct-network attempt stalled fetching
  a WASM plugin; its diagnostic stack identified the external HTTP read.
- `make fmt` completed; unrelated existing formatting changes in three parser
  adapters were restored. Changed Go files are formatted with Go 1.26.8.
- Full `CGO_ENABLED=0 go vet ./...` passed after fixing the 115 baseline
  unreachable-code findings through the ANTLR generation pipeline. The
  normalization regression runs both exit paths, checks idempotence and
  validates the result with `go run` and `go vet`. The generated parser diff
  contains exactly 115 goto moves and no other changes.
- Pinned bookstore commit `2a7f7e396451821a8fe3054eb265037b830af68d`:
  `make sqlc`, `make sqlc-verify`, and unchanged generated diff passed;
  `go test -race -count=1 -p 1 -timeout=10m ./pkg/usecases` passed (6.842s).
- No new dependency, schema, cache-key or existing-method signature changes.

## 8. Remaining work

Consumer adoption, merges and deployment remain separate. The user requested
fixing the existing full-vet baseline rather than accepting an exception.

# feat: pass query timeouts to wpgx as database timeouts

Local record. The complete design, decisions and risks are in the primary
record, Stumble/wpgx `docs/changelogs/2026-11-20-db-timeout-after-acquire.md`.

## 1. Background and Current State

- Every wicked query requires `-- -- timeout: T`. Generated methods start
  `context.WithTimeout(ctx, T)` at the top, so T also covers the dcache lookup
  and the wait for a pooled connection (`queryCode.tmpl`, `copyfromCopy.tmpl`).
- The opt-in Each companion (v2.5.0) runs the visitor under that context and
  checks it after each row and after closing.
- Fixtures under `internal/endtoend/testdata` compile against wpgx v0.3.1; the
  `wicked` CI job regenerates Stumble/bookstore at a pinned commit and requires
  no diff, then runs its runtime contracts.

## 2. Problem Model and End-to-End Behavior

- B1 — Every generated method (`:one`, `:many`, `:exec`, `:execrows`,
  `:execresult`, `:copyfrom`, Each) passes T to
  `wpgx.WithDBTimeout(ctx, "<unique label>", T)` instead of starting its own
  deadline. wpgx starts T once the statement has a connection (B1/B2 of the
  primary record). Annotation syntax, validation and method signatures are
  unchanged.
- B2 — dcache receives the caller's context (plus the timeout value), so cache
  lookups and fills are bounded by the Redis client's timeouts, not by T.
- B3 — Each visitors receive the statement context from wpgx rows (falling
  back to the method context; `qctx` is reassigned, so no new local name can
  collide with a SQL parameter). Each reports through `wpgx.StatementErr`: an
  ended statement context, a visitor's error that is that context's deadline,
  and, after closing the rows, a database timeout that expired while reading or
  draining buffered rows all match `wpgx.ErrDBTimeout` (as the previous Each
  failed when its query context expired). Other visitor and row errors stay
  primary; the caller's context is checked last.
- B4 — Generated files import `github.com/stumble/wpgx`; regenerated code
  needs a wpgx release with `WithDBTimeout` (compile-time check).
- F1 — Missing timeouts still fail generation.
- F2 — A SQL parameter named `wpgx` is generated as `wpgx_` (reserved), so it
  cannot shadow the package the method bodies now call.
- B2 precondition: the Redis-client bound holds with dcache v0.3.0+ and Redis
  single flight off (default); see R6 of the primary record.
- Compatibility: regenerating changes generated code by design (owner chose
  the new semantics as the default of this minor release). Old generated code
  keeps working with new wpgx.

## 3. Research, Findings, and Architecture Decision

See the primary record (D1–D5, R1–R5). Local decisions:

- D6 — Keep the template variable `qctx` so the diff stays small and
  readable; it now carries only the timeout value.
- D7 — Each discovers the statement context through an anonymous
  `interface{ Context() context.Context }` assertion, so fake queriers in
  tests keep working and no new exported API is needed in generated code.

## 4. Implementation Design

- `internal/codegen/wicked/templates/wpgx/queryCode.tmpl`: replace the five
  `context.WithTimeout` blocks and the Each body.
- `internal/codegen/wicked/templates/wpgx/copyfromCopy.tmpl`: same for
  `:copyfrom`.
- `internal/codegen/wicked/imports.go`: import wpgx in query and copyfrom files.
- Regenerate `internal/endtoend/testdata/wicked_*` and bump the testdata
  module's wpgx; update the Each runtime contract tests; add a PostgreSQL
  runtime check for the timeout boundary next to the existing optional Each
  check.
- `.github/workflows/wicked.yml`: point the bookstore job at the bookstore
  commit regenerated with this change.
- `GUIDE.md`: document the timeout semantics.

### Serial Implementation Checklist

- [x] Templates and imports.
- [x] Regenerate fixtures; bump the testdata wpgx; update/add runtime tests.
- [x] Guide.
- [x] Bookstore pin (commit 89b3fbc, published with this branch).

## 5. Verification and E2E Design

E2E Required: no.

Commands (repository root, GOMAXPROCS=2, GOWORK=off unless noted):

- `go build -p=1 -ldflags "-X github.com/sqlc-dev/sqlc/internal/info.Version=v2.4.0-dev-wicked-fork" -o ../sqlc-new ./cmd/sqlc` (as CI's `make build COMMIT_HASH=v2.4.0-dev`) and a default-version build for fixtures.
- `go test -p=1 -count=1 ./internal/codegen/wicked ./internal/cmd`.
- `go test -p=1 -count=1 -run 'TestReplay/base/wicked' ./internal/endtoend`.
- In `internal/endtoend/testdata`: `go build ./...` (`make build-endtoend`) and
  `go test ./wicked_compat/go`, plus the PostgreSQL checks with
  `SQLC_EACH_TEST_DSN`.
- Bookstore: `make sqlc`, `make sqlc-verify`, `git diff --exit-code -- pkg/repos`,
  `go test -race -count=1 -p 1 ./pkg/usecases`.

## 6. Human Decisions and Interaction

See the primary record, section 6.

## 7. Outcome and Evidence

- Result: B1–B4 and F1–F2 as specified; generated code requires wpgx with
  `WithDBTimeout`/`StatementErr` (wpgx v0.6.0 in the testdata module; the
  reviewed content pinned the same code as
  `v0.5.2-0.20261120053742-4b29cac2136d`).
- Deviations (reviewed): Each reassigns `qctx` instead of declaring `vctx`
  (R1-05); `wpgx` is a reserved identifier; Each reports through
  `wpgx.StatementErr` (R2-02). The testdata module was already not tidy at the
  base, so only `go get` was applied.
- Verification on the final content (`GOMAXPROCS=2`, `-p 1`):
  - Builds: `go build -p=1 -ldflags "-X github.com/sqlc-dev/sqlc/internal/info.Version=v2.4.0-dev-wicked-fork" -o ../sqlc-new ./cmd/sqlc` and the default-label `../sqlc-fixture`; fixtures regenerated with the latter.
  - `GOWORK=off go test -p=1 -count=1 ./internal/codegen/wicked ./internal/cmd`: pass.
  - `GOWORK=off go test -p=1 -count=1 -run 'TestReplay/base/' ./internal/endtoend` with this repository's `sqlc-gen-json` on PATH: pass (a stale `~/go/bin/sqlc-gen-json` from another build fails `process_plugin_sqlc_gen_json` independently of this change); `-run 'TestReplay/base/wicked'`: pass.
  - `internal/endtoend/testdata` with `GOWORK=off CGO_ENABLED=0`: `go build ./...` pass; `SQLC_EACH_TEST_DSN=postgres://…@127.0.0.1:55491/sqlc_test go test -race -count=1 ./wicked_compat/go`: pass, including `TestGeneratedTimeoutPostgres` (5 subtests) and `TestGeneratedEachPostgres`.
  - Mutation checks: the v2.5.0 generated Each fixture fails the timeout test; reverting the two `StatementErr` lines fails both new Each regressions.
  - Wicked CI bookstore job reproduced on a clean clone of bookstore 89b3fbc: regenerate, `sqlc diff`, no diff, build.
- Cross-agent review: primary record section 7 (Codex session
  `5e2002b6-dea5-4c53-96c9-cde8cdd6a84b`, rounds 1–5, final ACCEPT of
  5291d2071 with wpgx 4b29cac and bookstore 89b3fbc). Later changes here are
  record-only.
- Not run locally: the full CI suite (`go test --tags=examples ./...` needs
  MySQL and PostgreSQL services); CI runs it.

- Release follow-up (2026-11-22 US Pacific): this change merged as
  `be5161e1b` and is released as v2.6.0. The testdata module pins wpgx v0.6.0
  (merged Stumble/wpgx `3673ea0`, same tree as the reviewed head) and
  `.github/workflows/wicked.yml` checks out the merged Stumble/bookstore#3
  commit. `CGO_ENABLED=0 go build ./...` and
  `go test -race -count=1 ./wicked_compat/go` with `SQLC_EACH_TEST_DSN`
  (including `TestGeneratedTimeoutPostgres`) pass on the testdata module.

## 8. Remaining Work

- None in this repository. Consumers adopt the timeout semantics by
  regenerating with v2.6.0 and requiring wpgx v0.6.0.

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
  collide with a SQL parameter). An ended statement context is reported with
  `wpgx.ContextErr`: `ctx.Err()`, which also matches `wpgx.ErrDBTimeout` when
  the database timeout ended it. After closing the rows only the caller's
  context is checked.
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

- [ ] Templates and imports.
- [ ] Regenerate fixtures; bump the testdata wpgx; update/add runtime tests.
- [ ] Guide.
- [ ] Bookstore pin after the bookstore branch is published.

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

Pending.

## 8. Remaining Work

Pending.

# feat: generate opt-in Each companions for many reads

## 1. Background and Current State

Wicked `:many` methods collect every row before returning. Downstream Go
services with variable-width records consequently maintain handwritten SQL
iteration to enforce request policies. A row-count limit alone cannot bound
the result size. Six existing downstream reads motivate the companion API;
consumer migration remains a separate follow-up after a generator release.

Current source is sqlc main e0d9cae. The wicked backend owns comment options,
row types, collectors, caching and Dump/Load. Wpgx already exposes pgx.Rows.
Top-level SELECT classification alone misses data-changing CTEs. Current
wicked fixtures do not characterize `:many` output.

Eight inspected consumers use wpgx, from v0.3.4 to v0.5.0. Source-level
compatibility includes method types and optional generated interfaces, not
only ordinary calls. Production inspection is unnecessary for this change.

## 2. Problem Model and End-to-End Behavior

- B1 Existing SQL without opt-in generates identical output on the same
  compiler/version baseline. Existing methods, constructors, Querier,
  cache/invalidation, timeout, replicas and Dump/Load retain their contracts.
- B2 `-- -- each: true` on a read-only `:many` adds XxxEach on Queries and,
  when allowed, ReadOnlyQueries. Existing slice methods remain independent.
- B3 The synchronous visitor receives the query context and a fresh Row and
  returns `(more bool, err error)`. True continues; false completes normal
  consumption; error aborts consumption and remains the primary error.
  Filtering means omitting output and continuing. Each retains no result slice.
- B4 Each uses the same connection/transaction, SQL, arguments, timeout and
  CountIntent policy, directly accessing the DB without result-cache fill.
- B5 Normal stop and callback error drain/close by default. Cleanup/source
  errors surface when there is no earlier primary error. Explicitly cancelling
  a query-specific caller context opts into network abort and may invalidate
  the physical connection/transaction. Panic propagates after cleanup.
- B6 The callback is never called for an empty selection; it must respect the
  query context and cannot reenter the same active transaction connection.
- F1 Nil visitors fail before SQL. Invalid option values, unsupported commands,
  mutations/data-changing CTEs, missing safety facts and companion-name
  collisions fail generation. No new policy applies to legacy queries.
- F2 Callback, scan, source, deadline and close errors do not become success.
  Normal stop preserves a separate channel so a late drain error remains visible.

Single-row receive protection remains explicit SQL. Policies, row costing,
domain errors and all-or-nothing application collectors stay in their clients.
Downstream migration must preserve each reader's existing abort/drain policy
and any outer deadline shared with subsequent queries.

## 3. Research, Findings, and Architecture Decision

D1 Owner approved the reviewed Each design and requested implementation on
2026-11-11 (US Pacific), then explicitly requested continued local progress
while arranging a project-specific Stumble token. The reviewed API, cleanup
algorithm and compatibility obligations are the implementation contract.

D2 Generate a companion with `visit func(context.Context, Row) (more bool,
err error)`, returning error after cleanup. Do not turn the old collector into
an adapter or add variadic parameters to existing methods. Typed options remain
a credible alternative for a narrower full-result-or-error guard, but alter
exact method types. Bare Seq2/cursors have additional terminal-error/lifetime
contracts and are not generated in this iteration.

D3 The feature is wicked-only and opt-in. Reuse wpgx as-is; no runtime release,
SQL rewriting, migrations or budget primitive. If emit_interface is enabled,
EachQuerier is separate from the legacy Querier method set.

D4 Reject syntactically visible mutations anywhere in an opted-in SELECT AST.
This does not prove absence of volatile-function side effects; opt-in follows
existing read/replica declarations and DB permissions. Legacy classification
is intentionally not redefined.

Independent Codex/Claude design review accepted the same synthesis. The author
challenged error-only stop sentinels because they can mask late drain errors;
Claude accepted the separate normal-stop channel. AI-related benefits are
reasoned from API shape, not measured model success rates.

R1 Cancellation can destroy a connection; callback work holds the connection
and consumes the deadline. R2 Cache-bypassing companions may increase DB load.
R3 Standard []byte ownership differs from custom borrowing codecs. R4 Bool
false means stop, not skip; generated docs/examples must be explicit.
R5 A new generator release is needed before adopting the annotation; existing
binaries fail on unknown options. R6 Exact downstream upgrades remain separate
from this generator change, with generator publication/release boundaries.

References: prior scratch proposal and AI-facing synthesis; pgx Rows and
ForEachRow documentation; Go iter documentation; wicked option/query/interface
code and dcache GetWithTtl. The primary implementation record lives here.

## 4. Implementation Design

The approved public signature and loop/cleanup design map to these locations.

- `protos/plugin/codegen.proto` and generated Go protocol code gain an optional
  wicked per-query fact for syntactically visible mutations. The shim obtains
  it through the existing recursive AST walker; other backend metadata remains
  absent and legacy generation ignores the new fact.
- Wicked option parsing accepts literal true/false for `each`. Query building
  validates command/safety facts only when enabled. Validation checks generated
  companion methods/helper/interface names against existing declarations.
- The wpgx query template adds an independent visitor loop. It reuses existing
  Row/Params and Scan representations, checks context before/after visits,
  closes rows before cancelling its own context, and retains primary errors.
  Visitor identifiers must avoid scalar/multi-parameter name collisions.
- The interface template adds EachQuerier only when companions exist; legacy
  Querier stays byte-identical. Docs describe direct DB path, callback lifetime,
  error/stop meanings and cooperative cancellation.
- Characterization fixtures precede template edits. Behavioral tests execute
  actual generated methods with controllable Rows to exercise Close-only errors,
  callback failures and context/panic paths; a scoped real-PostgreSQL check
  verifies transaction reuse and explicit cancellation if available.
- First publication is a branch/PR against Stumble/sqlc using owner-provided
  project credentials. Owner retains merge/release authority. Consumer code
  migration follows publication/release separately, respecting existing downstream feature boundaries.

### Serial Implementation Checklist

- [x] Characterize existing many generation and save an immutable baseline binary.
- [x] Add each option, AST safety metadata and focused rejection tests.
- [x] Generate companion methods/docs and separate optional interface.
- [x] Prove generated runtime behavior and scoped integration semantics.
- [x] Compare non-opt-in fixture/consumer output against baseline; inspect goldens.
- [ ] Update GUIDE and complete author verification, independent Claude review.
- [ ] With verified project credentials, publish branch/PR and monitor checks.

## 5. Verification and E2E Design

E2E Required: no — no full Alva local-stack E2E was requested. Generator tests,
compiled generated code and scoped PG checks establish the affected contract.
No production DB or credentials are needed for those tests. Builds/tests run
serially with reduced parallelism on the shared VM.

Focused commands (from repository root, GOMAXPROCS=2, GOWORK=off):

- `go build -p=2 -o ../sqlc-before ./cmd/sqlc` before product changes.
- `go test -p=2 -count=1 ./internal/codegen/wicked ./internal/cmd`.
- `buf generate --path protos/plugin/codegen.proto` using installed tooling;
  inspect generated diff and restrict changes to the protocol source of truth.
- `go build -p=2 -o ../sqlc-after ./cmd/sqlc`.
- From `internal/endtoend/testdata`, run `go test -p=2 -count=1 ./wicked_compat/go ./wicked_many/go ./wicked_each/go`. Runtime contracts live in the existing wicked_compat CI suite and import the generated Each fixture, so existing CI executes them.
- Run both binaries on isolated copies of the non-opt-in fixtures and real
  alva-backend/synthdb input at recorded revisions; require byte equality.
  Normalize only the version header if the two version strings differ.
- Broaden focused compiler/AST/protocol checks only for changes in those paths.
- `git diff --check`; read all modified golden/generated files.

B1 is proved by fixture and real-consumer regeneration, including cached many,
mutation/invalidate, scalar and interface cases. B2/B4 require generation and
compile checks for Queries/ReadOnlyQueries and no replica exposure when disabled.
B3/B5/B6/F2 require generated-method behavior tests for ordering, complete/early
stop, error precedence, source/Close errors, empty/nil/cancel/deadline/panic,
retention and lifetime. F1 requires invalid option/command/safety/collision
negative cases. Cache+Each tests must show no cache lookup/fill by the companion
while the old collector retains cache behavior.

## 6. Human Decisions and Interaction

The owner approved the current design and requested progress, including
continuing local work before the temporary Stumble token is configured. This
plan records the previously reviewed callback signature, control flow, resource
semantics and compatibility matrix, plus routine source-local implementation
choices. No new product behavior is inferred from token availability.

Credentials are project-scoped. The existing alva-bot01 identity has pull-only
access to Stumble/sqlc and wpgx. The requested temporary fine-grained credential
is limited to sqlc branch/PR work and CI reads. Its contents must never appear
in chat, logs, git or review artifacts.

## 7. Outcome and Evidence

Baseline `../sqlc-before` built from e0d9cae. `GOMAXPROCS=2 GOWORK=off go test -p=2 -count=1 ./internal/cmd -run '^TestWickedManyCompatibility$'` passed before product-template changes, for both interface modes. The initial generated 443-line golden was inspected. It was subsequently
expanded using the same immutable baseline binary to include cached struct
parameters and a legacy data-modifying CTE. The expanded compatibility test
passes with interfaces both enabled and disabled.

Focused `go test -p=2 -count=1` passed for internal/cmd,
internal/codegen/wicked, internal/compiler, internal/sql/astutils and
internal/plugin. Feature tests first failed on the legacy compiler's unknown
`each` option, then passed after implementation.

The generated runtime suite passed in the testdata module. It exercises actual
emitted methods, including Close-only failures after normal stop, primary-error
precedence, panic, context/deadline, scalar/struct parameters, retained rows,
replicas, CountIntent and cache bypass. It is imported by the pre-existing
wicked_compat CI entrypoint, avoiding a separate unexecuted test suite.

Eight isolated, immutable downstream input snapshots regenerated with both
binaries produced 1,031 Go files with zero differences, including names and
contents. Original working trees were not modified. Per-consumer HEADs and
hashes are retained in the task's local compatibility results.

Runtime checks passed with wpgx v0.3.1 (the fixture's current dependency),
v0.3.4, v0.4.8, v0.4.9 and v0.5.0. A disposable postgres:16-alpine instance
was bound only to localhost, limited to one CPU/512 MiB. The real PG test
proved same-transaction reuse after normal stop and ordinary visitor error,
primary-error preservation by Each on explicit abort, and pool recovery with a
replacement connection. Optional integration is enabled with SQLC_EACH_TEST_DSN.
Older wpgx transaction wrappers can overwrite an error during failed rollback;
that pre-existing runtime behavior is not changed by Each.

The full generated-fixture module compiled. A subsequent CLI build invocation
was initially issued from that fixture directory, where ./cmd/sqlc does not
exist; the root build is being rerun from the correct repository directory.
Root `GOMAXPROCS=2 GOWORK=off go build -p=2 ./...` and the current CLI build passed. `buf lint` passed (the existing DEFAULT-category deprecation warning remains). Generated fixture `go test -race -p=2 -count=1 ./wicked_compat/go ./wicked_many/go ./wicked_each/go` passed. The repository has no lint-fix target; no unrelated lint/configuration changes were introduced. Independent code review remains pending.

## 8. Remaining Work

Complete independent code review and PR publication. Downstream migration follows an approved generator release.

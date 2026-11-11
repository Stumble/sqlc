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
- [x] Update GUIDE and complete author verification, independent Claude review.
- [x] With verified project credentials, publish branch/PR and monitor checks.

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
access to Stumble/sqlc and wpgx. This task uses the owner-provided temporary credential
only for sqlc branch/PR work and CI reads. Its contents must never appear
in chat, logs, git or review artifacts.

## 7. Outcome and Evidence

The completed product content was verified at `49ac2682cf881acaf52c66fa3df4eedec2617607`,
against `e0d9cae63e18e46c50e6bba7e17f6f1341f36df4`. Each is an independent,
opt-in generated consumer; no wpgx or library dependency changes were necessary.
CI follow-up C1 updates the Go patch version in go.mod and existing workflows.
GUIDE and generated godoc describe the API and resource contract.

All final local checks passed, serially with `GOTOOLCHAIN=go1.26.11 GOWORK=off GOMAXPROCS=2`:

- Repository root: `go build -p=2 ./...` and
  `go test -p=2 -count=1 ./internal/cmd ./internal/codegen/wicked ./internal/compiler ./internal/sql/astutils ./internal/plugin`.
- `internal/endtoend/testdata`: `go build -p=2 ./...` and
  `go test -race -p=2 -count=1 ./wicked_compat/go ./wicked_many/go ./wicked_each/go`.
  The latter used SQLC_EACH_TEST_DSN for an owned disposable PostgreSQL instance,
  including wpgx v0.3.1. No production/shared database was used.
- Isolated runtime modules: `go test -mod=mod -p=2 -count=1 ./...` with the
  same PG integration enabled, for wpgx v0.3.4, v0.4.8, v0.4.9 and v0.5.0.
- Clean candidate regeneration of wicked_each, `buf lint`, `git diff --check`,
  and `git diff --exit-code` after generation. Buf's existing DEFAULT-category
  deprecation warning is unrelated and remains unchanged.

The baseline binary was built from e0d9cae before template changes. Legacy
many goldens, including cached struct parameters and a data-modifying CTE,
were generated by that immutable baseline. Compatibility tests pass with and
without interfaces. Final candidate binary provenance is `vcs.revision=49ac268`,
`vcs.modified=false`, SHA256
`06fc5b654399d68bfcbbf7309162d5444634644da0f5a8d0d914ae27a5b09765`.
Both binaries report the same version header, so no normalization was needed.

Eight immutable SQL/config input snapshots generated 1,031 Go files with
identical names and contents under both binaries; original worktrees were not
modified. This also proves cache/invalidate/timeout and Dump/Load output remains
unchanged in those consumers.

| Consumer | Input revision | Generated Go files | Differences |
|---|---|---:|---:|
| synthdb | e9c563190e | 12 | 0 |
| alfs | 0bce08940b | 30 | 0 |
| alva-backend | 8c1e7b545d | 605 | 0 |
| jagent | 0d9c077e78 | 9 | 0 |
| worldline | 7e1f8b563f | 168 | 0 |
| trex | 8fcf54971c | 36 | 0 |
| connectors | 463b3ec43f | 15 | 0 |
| llm-data | ddc23400cd | 156 | 0 |

| ID | Approved intent and implementation | Evidence | Status |
|---|---|---|---|
| B1 | Preserve legacy methods, interfaces and output | Baseline goldens, interface on/off, 1,031-file comparison | DONE |
| B2 | Generate opted-in companions and replica exposure | Compiled fixture, generation/routing tests | DONE |
| B3 | Typed synchronous consumer, fresh row, no collector | Executed order/retention/scalar/struct tests | DONE |
| B4 | Reuse routing, context, CountIntent; bypass cache | Replica, effective-context, poisoned-cache and intent tests | DONE |
| B5 | Drain before own cancel; preserve primary/late errors | Fake Rows error/panic/order cases and real PG transaction/pool checks | DONE |
| B6 | Empty selection and callback lifetime | Empty/deadline/context-lifetime tests; explicit API docs | DONE |
| F1 | Reject nil visitors and unsafe/incompatible opt-ins | Nil visitor side-effect checks; option/command/fact/name negative tests | DONE |
| F2 | Preserve source/scan/callback/close/context failures | Executed failure matrix and PG abort check | DONE |
| D1-D4 | Approved companion API, separate interface, no runtime release, syntactic safety | Templates, proto/shim facts, full independent review | DONE |
| R1-R4 | Cancellation, connection lifetime, cache cost, codec ownership and bool meaning | GUIDE/godoc and scoped runtime tests | DONE |
| R5-R6 | Release before downstream adoption | Documented staged adoption; no consumer change in this PR | DONE |

The PG check used a localhost-only postgres:16-alpine container capped at one
CPU and 512 MiB. Each test creates/drops a unique owned schema, proves reuse
of the same transaction after stop/ordinary visitor error, and proves connection
replacement after explicit abort. Older wpgx transaction wrappers can overwrite
callback errors during failed rollback; Each's own primary error is asserted
separately, without changing that pre-existing runtime behavior.

No full Alva service-stack E2E was requested. Full repository CI is left to
GitHub; local scoped integration and generated-runtime tests cover this contract.

### Review and dispositions

Both agents previously explicitly accepted the complete net diff at **49ac2682cf881acaf52c66fa3df4eedec2617607**:
Codex main `baa9cfcd-d88f-424e-a8a3-6e095ee3d37b` and Claude reviewer
`aee3fbde-9a50-4c30-94fe-ea3d7d1e4de6` (native conversation
`af103b26-bc43-4187-80b6-a68a4e6613c8`). Same VM `env-woddv6j0`, owner and
unchanged treeParentId were verified, including the parent's children listing
with no further pages. The original creation request was
`604f3b3c-fd56-4c95-bd32-77d60702dcc7`. Latest final review delivery request was
`d4ae62ca-dc77-4d25-91b5-cedfb1e4795e`; acceptance comes from the actual report,
not the delivery receipt. The a119f55 verification completed 2026-11-11 10:22:30 PST;
C1 verification and renewed Claude acceptance subsequently completed the same
US Pacific morning, recorded in the security-patch reports below. There are no unresolved
actionable findings. Evidence-only checklist/sections 7-8 updates do not change
the accepted product content.

- Initial M1-M2: generated runtime tests now run through the existing wicked_compat
  CI entrypoint; local checks use explicit packages and an owned PG container.
- M3-M4: baseline fixtures cover cached struct params and modifying CTEs;
  fail-closed safety facts reject mutations, unknown CTE statements and SELECT INTO.
- M5: deterministic visitor naming handles SQL parameters and full Go type identifiers.
- S1-S7: cache bypass, cleanup/error ordering, version/consumer matrices, locking
  reads and resource/stop semantics are tested or documented; no new query import.
- Author A1-A2, found after first acceptance of 5420baf: fixed custom/generic
  type shadowing and helper/query-constant declaration collisions, added focused
  regressions, reran final checks and obtained renewed full-diff acceptance.
- Optional final O1-O4: tied comparison evidence to a clean binary revision,
  removed stale wording, added SELECT INTO fact coverage and isolated PG schema.
- Claude independently built clean a119f55, matched both fixture goldens,
  regenerated synthdb/alva-backend (617 files) without differences, ran focused
  generator/runtime tests, and checked the complete diff. Other version/PG/consumer
  checks were inspected from the author's evidence rather than claimed as rerun.

Local review artifacts are retained in the dedicated task scratch directory.
Their hashes preserve provenance; summaries and both acceptances are recorded
above so the record does not depend on those files remaining accessible.

| Artifact | SHA256 |
|---|---|
| reviewer-creation.json | `9bd5056157d1fb2ba58e55bc1931ed76158687a3f8f71925f7798bea809d7acb` |
| implementation-review-initial.md | `17fdc100e3a7273a4f102c5134647e021ad86aafb4415d951f9c25611a4ca2c4` |
| implementation-code-review.md | `e13a90a1939b328b6bde93bafede2077c3b847d346199e450719796c01916f63` |
| implementation-code-review-final.md | `adaf6e5a09ef32f9065aa88bc6e8e8146c603a475ef651ccdb5f49d27693bf5b` |
| author-final-review.md | `40f4ea5af13545d576e344c2e7564e758511885a1120759cc474d7b781082253` |
| implementation-final-verification.md | `edf1778f3cbbeab7f002ae9ccf19bb23a56ace8f6296a9911faa59317ee85efb` |
| consumer-compat/results-a119f55.json | `ce752d46c70cac6bb3b691769e1edb5f29af6017ab3bdf6ca512277d47f72a1a` |
| runtime-matrix/results.json | `151e539eb962343739b2905e6a7d43e3b274a20aed1382a3f058345f63b33980` |

## 8. Remaining Work

Published ready-for-review PR [#24](https://github.com/Stumble/sqlc/pull/24)
on branch `yumin-xia/many-each-hooks`. Current-head CI and feedback are monitored
on the PR; their live status is authoritative. Initial publication succeeded
2026-11-11 10:26 PST using the task-specific owner credential, without changing
the shared GitHub identity.
The owner retains merge and release authority. Downstream migration of the six
handwritten readers follows an approved generator release and must preserve
per-reader abort/drain policies and SQL single-value guards. Existing feed-search
work is untouched. No runtime release or database migration is required.


CI follow-up C1 (2026-11-11 US Pacific): current-head Windows cross-build
failed while downloading grpc from proxy.golang.org (HTTP/2 INTERNAL_ERROR),
which cancelled the other matrix builds. The vulnerability job independently
reported GO-2026-6859 in the existing Go 1.26.10 standard library, fixed in
1.26.11. The official [Go release history](https://go.dev/doc/devel/release#go1.26.11)
records that security patch on 2026-11-10. Update the root build/release minimum
and every existing CI Go pin to 1.26.11, keeping the same Go minor version.
No library dependency or generated API changes. This necessary CI/security
follow-up passed fresh generation/build/runtime checks and renewed complete
cross-agent review at 49ac268. Earlier acceptances remain historical.
`GOTOOLCHAIN=go1.26.11 GOWORK=off GOMAXPROCS=2 go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...`
passed with 0 reachable vulnerabilities. The scanner also lists 2 imported-package
and 2 required-module vulnerabilities not called by this code; none were
suppressed. The new clean binary again generated all 1,031 consumer files
without differences. Go 1.26.11 root/focused/fixture builds, race and real PG
checks also passed. Claude independently rebuilt exact 49ac268 using Go1.26.11,
matched fixture goldens, regenerated 605 alva-backend files without differences
and ran focused generator/runtime checks. No actionable finding remains; the
current-head CI matrix and vulnerability scan rerun after pushing. Initial-head
repository-wide tests and bookstore compatibility/runtime CI had passed.

C1 final review evidence (same verified Codex parent/Claude child and VM):

| Artifact | SHA256 |
|---|---|
| implementation-code-review-security.md | `9a3e118a7136af827aa5b0c9613513c5d8b0c99040fbaa029b396b04ae69e336` |
| author-security-review.md | `05e49acfd1040ea520479787d48e2b5657bac6338fb303ce9cdbe54ea1c0291d` |
| security-patch-verification.md | `4b03f270323de3f56f2d449226f91f82f7808951380cdab39e9bb1ef9b19d279` |
| security-patch-local-checks.log | `ef26ba4e279161c4a7d0d281d0a2bbaeda250cb47504b63a2adacb0f52b0913e` |
| consumer-compat/results.json | `eecb640bbadfdb0704656bc29f194af57fe92f67bd194d28f5998f60ca814b73` |

C2 publication blocker (2026-11-11 US Pacific): GitHub rejected the push of
the reviewed C1 follow-up because the temporary PAT lacks `workflow` scope.
The published PR remains at 18ed706; the security patch and its review/evidence
are retained locally. Repository push/admin role does not confer a missing PAT
scope. The PR body records this distinction. At that point workflow permission was required before publication; this
blocker is now resolved as recorded below.
No permission bypass or release/merge was attempted.

The owned disposable PG container was removed and its absence verified after
all checks. The completed dedicated Claude reviewer was closed by its creating
parent after preserving both acceptances and report hashes. Closed state was
verified with unchanged native ID/treeParentId, live=false and retained parent
children listing. It had no cron or continuing responsibility. Review history
can be reopened if later CI requires another product fix.

- implementation-review-completion.json: SHA256 `de2f4be5c48df3e0aa8cfe0daac6286deb28fe18fd437116a2c3e9dc82cabdc0`.

- implementation-review-close.json: SHA256 `5e4550142cbddb59778cdfdc14a7e3fc664120356f4d0a207480a83d1f7376d0`.

- implementation-review-closed-state.json: SHA256 `ae205f0fc61be5fe5d93f844aa6bf0b46cbe470ea66e037a696e62efaf329234`.

- implementation-review-closed-tree.json: SHA256 `7b1f154eee28e7603a7736faaade4de9d65f6f69224610efc881e030a049a8ce`.

- postgres-cleanup.json: SHA256 `f63b430740a027b5afb8b008b5f5704e2b0239f8f83e2c88d8b7476c5c4652c3`.


Owner-requested final reconfirmation (2026-11-11 US Pacific): the owner added
workflow permission. An actual push of 1416a5b, including the workflow changes,
succeeded; PR #24's head was verified through GitHub. C2 is resolved. Builds and
checks now run on the new published head; live PR status remains authoritative.

At the owner's explicit request, the creating parent reopened the same tracked
Claude reviewer session on the same VM, preserving native conversation
`af103b26-bc43-4187-80b6-a68a4e6613c8`, child ID and treeParentId. The reviewer
reverified its identity and the parent's complete children listing. Both Codex
and Claude **explicitly endorse the design and accept the complete net diff
`e0d9cae63e18e46c50e6bba7e17f6f1341f36df4..1416a5bde7a109d76de00c7d1dcda219d0221c3a`**.
There is no unresolved actionable finding. Later changes in this final evidence
commit are confined to changelog sections 7-8.

Claude's new report distinguishes full source audit across the implementation
rounds, independently executed fixture/generator/runtime/consumer checks, and
inspection of the author's PG/race/runtime-version/1,031-file evidence. For this
reconfirmation it proved non-document content matches the verified 49ac268
byte for byte, reread key implementation/design choices and audited all new
evidence prose. It did not claim to rerun all tests or remote CI. The endorsed
design preserves exact legacy signatures with a typed Each companion, keeps
policy in the visitor, separates normal stop from error, centralizes cleanup,
bypasses cache for request-specific consumption and retains syntactic safety
and existing routing/transaction constraints. Its limitations remain explicit.

Optional O1 is intentionally unchanged, independently accepted by Claude:
Each and the slice method share the SQL operation label and CountIntent policy.
Close-based wpgx latency includes callback work, and explicit cancellation may
appear as a context-canceled DB error under that label. Downstream adoption
should account for this in metrics/alerts. Separate mode labels remain a possible
future observability choice; no correctness/compatibility change is required.

The follow-up delivery request was `594bc92b-e9d7-49e1-bd51-030beb61d352`,
with factual-status/disposition request `6bc7631a-1e53-4bd7-bd5d-033b6387ff40`.
The actual revised report, rather than delivery, supplies the acceptance.
Both result summaries/hashes are preserved here before parent-managed reviewer
completion. The dedicated reviewer has no continuing watch/cron responsibility.

| Final reconfirmation artifact | SHA256 |
|---|---|
| owner-reconfirmation-main.md | `a7c9bf99385e2b7a8388bc39274a13f2f0d68001bd45ff296e8b4b8b7d9e70e2` |
| owner-reconfirmation-review.md | `a7f40e374a2dda69957874f596ed66e7100d7d6361e167d06a3c50c77a8dca22` |
| owner-reconfirmation-reopened-state.json | `26168ce4a6e463f0f2ca6e81c310e310b71788a422d8bae969ef4fd2497402cb` |
| owner-reconfirmation-tree.json | `d6f65be59db2f0ba30bb28cfcbafbfd629a193a9202a8e76cfb5bdbaf73cca04` |
| owner-reconfirmation-delivery.json | `090aaf878111fcd283d2dbfe19e7a29eb1cbf75cb83fe6fc4559e9e62df090a3` |

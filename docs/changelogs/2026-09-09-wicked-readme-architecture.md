# wicked-sqlc README architecture

## 1. Current State

The README still presents upstream sqlc's homepage with a short fork preface.
`main` now contains the validated compiler changes and dedicated wicked backend,
but the homepage does not explain that architecture or how to select it.

## 2. Intended Behavior

- B1: Introduce wicked-sqlc in concise, natural English.
- B2: Explain compiler ownership, the codegen contract, backend responsibilities,
  and the single-binary installation model.
- B3: Show the existing build, configuration, and query-comment workflow; link to
  the guide and working example for details.
- B4: Describe regular rebases onto upstream releases without tying the
  architecture to a specific upstream version.
- B5: Preserve the original upstream README at the end, including its badges,
  links, and sponsors.
- F1: Do not imply that a separate plugin, custom option protocol, or proven
  read-only analysis exists.

## 3. Decisions and Risks

- D1: Lead with a fork-specific introduction while crediting and linking upstream.
  Following user review, preserve the original upstream homepage below it rather
  than replacing it.
- D2: Describe the implementation on `main`, not the completed migration's branch
  mechanics. Preserve the guide and historical migration record unchanged.
- R1: Architecture prose can overstate implementation guarantees. Check each
  claim against the compiler, request shim/proto, dispatcher, and backend.

## 4. Scope and Checklist

The user's request authorizes a README rewrite and a new PR against `main`.
No architecture or runtime changes are proposed.

- [x] Inspect the merged implementation and existing documentation.
- [x] Rewrite `README.md` and add this change record.
- [x] Verify examples, links, whitespace, and scope; review the complete diff.
- [x] Apply the user's version-neutral wording and upstream-preservation feedback;
  verify exact preservation and re-review the complete PR diff.

Publication follows the normal secret-scan, commit, PR, and CI/review monitoring
workflow, with live results recorded on the PR.

## 5. Verification Plan

E2E Required: no. Only Markdown changes; no source, templates, configuration,
dependencies, or generated output changes.

- Inspect `internal/compiler/compile.go`, `internal/compiler/wicked.go`,
  `internal/cmd/shim.go`, `protos/plugin/codegen.proto`,
  `internal/cmd/generate.go`, and `internal/codegen/wicked` for architecture claims.
- Generate and diff a temporary fixture using the README's exact YAML/query
  blocks and a minimal books schema with the verified main-equivalent binary.
- Check README relative links and fenced code blocks; run `git diff --check`.
- Compare the appended upstream section byte-for-byte with the upstream README
  inherited by `main`. Preserve its existing whitespace and check the net diff
  against `origin/main` for newly introduced whitespace issues.
- Review `git diff origin/main` and confirm only README and this record changed.
- No `make lint-fix` target exists; lint is unavailable. Do not substitute another
  linter. Full local runtime tests are unnecessary for this docs-only change;
  monitor the existing GitHub workflows after publication.

## 6. Human Decisions

User requested succinct, native English and a PR. Keep the implementation,
generated API, and installation entrypoints unchanged. No additional decision
or migration is required.

In follow-up review, the user requested version-neutral architecture wording that
states the regular upstream-rebase policy, and the original upstream README at
the end as acknowledgment of upstream's work. This supersedes the initial
replacement-only layout.

## 7. Outcome and Evidence

The initial revision was a 357-word introduction covering the architecture and a
small working example. The follow-up keeps that concise introduction, removes
the specific upstream version, and restores the original upstream README beneath
it. Only the README and this record change.

| Item | Evidence | Status |
| --- | --- | --- |
| B1, D1 | Concise fork-specific homepage with upstream attribution | DONE |
| B2, F1, R1 | Checked compiler, shim/proto, in-process handler, and backend | DONE |
| B3 | Exact README YAML/query fixture generates and passes `sqlc diff` | DONE |
| D2 | Guide and historical migration record unchanged | DONE |
| B4 | Version-neutral wording states regular upstream-release rebases | DONE |
| B5 | Appended upstream README matches the inherited original byte-for-byte | DONE |

Initial revision checks (revalidated as applicable after user feedback below):

- In `/home/forge/worktrees/sqlc-wicked-readme`, `git diff --check` passed;
  `make -n install` confirmed the existing installation command without changing
  the installed binary. No `make lint-fix` target exists.
- `node /tmp/sqlc-readme-validation.1zZzaq/verify.mjs` passed: all relative links
  exist, code fences are balanced, and the YAML/SQL fixture exactly matches the
  README blocks.
- In `/tmp/sqlc-readme-validation.1zZzaq`, both
  `/home/forge/worktrees/sqlc-wicked-on-upstream/bin/sqlc generate` and
  `/home/forge/worktrees/sqlc-wicked-on-upstream/bin/sqlc diff` exited 0.
  Inspected generated `GetBook`, its 500 ms timeout, 10-minute cache TTL, replica
  method, and load/dump helpers. The fixture adds only a minimal books schema.
- The binary is the previously verified `v2.4.0-dev-wicked-fork` build from
  `41bb4c1581bcd0eb86a968f480594603f7953602`; a non-Markdown diff against the merged
  base `e40892b969f75131c219b5adbda4dc570e406e08` is empty. No runtime retest or new
  binary build is claimed for this documentation change.
- Reviewed the complete diff for scope, architecture accuracy, example validity,
  compatibility, operations, and prose. No unresolved findings. No migrations,
  dependency changes, or API changes; full local E2E is not required.

Follow-up verification after user feedback:

- `node /tmp/sqlc-readme-validation.1zZzaq/verify.mjs` passed again, now also
  checking version-neutral wording and exact equality of the appended section
  with `git show a95e91d70ad9e1181253c333a1cfdd75ae4b95a5:README.md`.
  This SHA identifies the inherited document for the preservation check, not an
  ongoing version constraint on the fork.
- Re-ran the fixture's `sqlc generate` and `sqlc diff` commands above; both
  exited 0. The YAML and SQL examples remain unchanged.
- `git diff --check origin/main` passed. Upstream's original trailing spaces
  are deliberately preserved; the net PR diff introduces no whitespace errors.
- Re-reviewed the entire PR diff against `origin/main`, including the restored
  upstream text, updated decisions, and unchanged implementation boundary.
  No unresolved findings. The follow-up is an additive commit on PR #18.

## 8. Remaining Work

None for the documentation change. Publication and current-head CI/review status
are tracked on the PR; no runtime follow-up is introduced.

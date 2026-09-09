# Refresh the guide and prepare wicked-sqlc v2.4.0

## 1. Current State

The migration and README PRs are merged on `main` at
`fff88434a16a113e915e15ef85c3136d96c8e78c`; its Go and wicked workflows passed.
The latest published release is v2.3.4. The guide still points to that old tag and
the migration branch, and says CGO is mandatory.

Prior releases have no binary attachments. The current workflows do not publish
on tag or release events; `scripts/release.go` has disabled Equinox publishing.
No new distribution infrastructure is needed for a source release.

## 2. Intended Behavior

- B1: Document current-main builds and reproducible release-tag builds without
  tying the architecture to one upstream release or a migration branch.
- B2: Explain optional CGO, version output, runtime dependencies, and the existing
  comment-based backend options accurately.
- B3: Prepare v2.4.0 notes with upgrade instructions and honest validation scope.
- F1: Do not publish an unmerged preparation branch or imply an unpublished tag
  exists. Do not reset checkouts, switch downstream binaries, or deploy services.

## 3. Decisions and Risks

- D1: Change only GUIDE.md, release notes, and this record; preserve the README
  and its original upstream section.
- D2: Use v2.4.0 as the next minor release, following the existing wicked version
  series and the migration's v2.4.0-dev test label. Do not move old tags.
- D3: Keep the existing tag + GitHub Release/source distribution model. No new
  release workflow, registry publishing, or binary-attachment contract.
- D4: Submit the preparation PR and wait for human merge before creating the
  release tag on the exact verified `main` commit. Publication is a separate
  post-merge step, not automatic PR merging.
- R1: A release label must match the CLI and generated headers in both build modes.
- R2: Existing generated snapshots use a development label; compare them with
  release-candidate output while ignoring only the version-comment line.

## 4. Scope and Checklist

- [x] Inspect merged code, guide, current releases, and all publishing triggers.
- [x] Refresh the install/architecture/options sections and prepare release notes.
- [x] Verify CGO/non-CGO installs into isolated directories and version output.
- [x] Run release-version regression tests and bookstore generation parity.
- [x] Review the complete diff and prepare the PR handoff.

Secret scanning, publication, and current-head CI/review monitoring follow the
normal push workflow; their live results are recorded on the PR.

## 5. Verification Plan

E2E Required: no for this docs-only preparation PR. The product tree is unchanged
from the already tested migration; PR CI will rerun the existing runtime suites.

- `git diff --check` and a non-documentation diff against `origin/main`.
- Build with `make install COMMIT_HASH=v2.4.0` and its `CGO_ENABLED=0` variant,
  setting `GOBIN` to separate temporary directories so the user's binary is not
  replaced. Check both binaries report `v2.4.0-wicked-fork`.
- `GOMAXPROCS=2 go test -p 2 -count=1 ./scripts ./internal/cmd -run 'Test(ReleaseVersionMatchesGeneratedCode|Wicked)'`.
- Clone bookstore `main` in a temporary directory; generate with each candidate
  binary, compare tracked Go output ignoring only SQLC version comments, and run
  `sqlc diff`. Check generated headers report the release-candidate version.
- Check new documentation links and installation snippets against the Makefile.
- No `make lint-fix` target exists in sqlc; lint is unavailable, without a
  substitute command. No full local Alva service-stack tests are claimed.

## 6. Human Decisions

The user approved updating GUIDE and preparing a formal release. The announced
target is v2.4.0. The human still merges the preparation PR; there is no approval
to bypass the PR workflow or modify downstream deployments.

## 7. Outcome and Evidence

The guide now matches the merged architecture and installation model. Release
notes are ready in `docs/releases/v2.4.0.md`; no v2.4.0 tag or release has been
created. Changes remain limited to three Markdown files.

| Item | Evidence | Status |
| --- | --- | --- |
| B1, B2 | Guide removes old tag/branch directions and documents both build modes | DONE |
| B3 | Upgrade notes include compatibility evidence and known limits | DONE |
| D1, F1 | README/product unchanged; only temporary installations/clones used | DONE |
| D2, D3 | v2.4.0 unused; existing source-release convention inspected | DONE |
| D4 | Human merge and exact-main publication remain explicit prerequisites | DONE |
| R1 | Both CLI builds and generated headers report v2.4.0-wicked-fork | DONE |
| R2 | Both generators match all 17 bookstore Go files except version comments | DONE |

Fresh checks from `/home/forge/worktrees/sqlc-wicked-release`:

- `GOMAXPROCS=2 GOBIN=/tmp/sqlc-v2.4-release.jAOb4e/cgo make install COMMIT_HASH=v2.4.0`
  and `GOMAXPROCS=2 GOBIN=/tmp/sqlc-v2.4-release.jAOb4e/nocgo make install CGO_ENABLED=0 COMMIT_HASH=v2.4.0`
  both passed. Each isolated binary's `version` command reports the expected
  `v2.4.0-wicked-fork`; no installed user binary was replaced.
- `GOMAXPROCS=2 go test -p 2 -count=1 ./scripts ./internal/cmd -run 'Test(ReleaseVersionMatchesGeneratedCode|Wicked)'`
  passed: scripts 8.680s, cmd 0.109s, including an actual release-binary/header test.
- `node /tmp/sqlc-v2.4-release.jAOb4e/verify.mjs` passed: both binaries generate and
  pass `sqlc diff`, all 17 outputs match bookstore after removing only the version
  comment, every generated header has the candidate version, and no untracked
  generated files appear. Documentation links/fences and unchanged README pass.
- The isolated bookstore clone is merged `main` at
  `444685d1b6fe870143f9e8aef190410e9a30570b`. From that clone,
  `GOMAXPROCS=2 go test -p 2 -count=1 ./pkg/repos/...` passed compilation of all
  five generated packages; these packages contain no test files.
- `git diff --check` passed. The complete diff was reviewed against `origin/main`
  for scope, architecture, compatibility, test claims, operations, and prose.
  No unresolved findings. No product, workflow, dependency, or migration changes.

The new tag-clone example cannot run before publication; it is prepared release
text, not evidence that the tag already exists. Post-merge verification remains
required below. No full local runtime or Alva service-stack rerun is claimed.

## 8. Remaining Work

Human merge of the preparation PR, then verify current `main`, ensure v2.4.0 is
still unused, create the tag and source release using `docs/releases/v2.4.0.md`,
and verify the published tag, notes, and fresh tag-clone installation. Do not
claim the formal release is complete before those post-merge checks pass.

# Update Go and dependencies flagged by govulncheck

Yumin, 2026-11-04: fix the dependencies before publishing release binaries.

## 1. Current State

`vuln_check` (govulncheck) fails on `main` (`dec50bf`) with 14 reachable
vulnerabilities published after the last green run (2026-09-09):

- Go standard library 1.26.8: crypto/tls, crypto/x509, net/http, net/textproto,
  html/template (fixed in 1.26.9; GO-2026-6809 in 1.26.10).
- golang.org/x/net v0.55.0 (fixed in v0.60.0), golang.org/x/text v0.39.0
  (fixed in v0.41.0), google.golang.org/grpc v1.82.1 (fixed in v1.83.1).

## 2. Intended Behavior

- B1 govulncheck reports no vulnerability that sqlc's code calls.
- B2 Generated code and CLI behavior do not change.

## 3. Decisions and Risks

- D1 Smallest fixed versions: Go 1.26.10, x/net v0.60.0 (requires x/text
  v0.42.0), grpc v1.83.1; `go mod tidy` moves x/sync, x/sys, cel.dev/expr and
  genproto as those require.
- D2 Every workflow that pins 1.26.8 moves to 1.26.10 with go.mod.
- R1 v2.4.0's source keeps the old dependencies; binaries built from that tag
  carry them. A patch release is needed for fixed binaries.

## 4. Changes

- `go.mod`, `go.sum`.
- `.github/workflows/{build,ci,ci-kotlin,ci-python,ci-typescript}.yml`.

## 5. Verification

- `govulncheck ./...` with Go 1.26.10: no called vulnerabilities (was 14).
- `go build ./...` passes. Locally `make test` fails only in MySQL containers
  under parallel load and in a stale `sqlc-gen-json` on the machine; the same
  tests pass alone. CI runs the full suite.

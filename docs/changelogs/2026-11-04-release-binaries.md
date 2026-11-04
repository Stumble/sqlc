# Release binaries with checksums and an install script

Tracking: alva-ai/mono-meta#1115. Yumin, 2026-11-04: developers should not
have to clone this repository and run `make` to get the right sqlc; publish
binaries for several platforms with an install script that checks the SHA-256.

## 1. Current State

- Releases carry no binaries. The only way to get a binary that reports its
  version is `make build` / `make install` in a clone: `internal/info.Version`
  is `v1.31.1` in the source, and the Makefile sets `<tag>-wicked-fork`
  through `-ldflags`.
- Builds without the Makefile (`go install ./cmd/sqlc`, as ggsrc/ggsrv-layout's
  CI did) write `sqlc v1.31.1` into generated headers.
- The PostgreSQL engine uses pg_query_go with CGO and its WebAssembly build
  (wasilibs/go-pgquery) without CGO. Measured on v2.4.0: identical output on
  `wicked_compat`, ggsrv-layout's repositories and Stumble/bookstore; ~60–80 ms
  with CGO, ~2.2 s without (WebAssembly start-up).

## 2. Intended Behavior

- B1 A tag push (or a manual run with an existing tag) builds `sqlc` for
  linux/amd64, linux/arm64, darwin/amd64, darwin/arm64 and windows/amd64 and
  attaches `sqlc_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows), each with
  `sqlc` and `LICENSE`, plus `checksums.txt` (SHA-256) to the tag's release.
- B2 Each binary reports `<tag>-wicked-fork`, like `make build` from the tag.
- B3 Every binary runs `sqlc version`, `sqlc compile` and `sqlc generate` on its
  own platform before anything is published; after publishing,
  `scripts/install.sh` installs the release on Linux and macOS runners.
- B4 Linux binaries are static (CGO, musl) and run on any distribution; macOS
  binaries use CGO and run on macOS 12 or later; the Windows binary has no CGO.
- B5 `scripts/install.sh [version]` picks the platform, downloads the archive,
  checks it against `SQLC_SHA256` or the release's `checksums.txt`, and installs
  it into `INSTALL_DIR` (default `$(go env GOPATH)/bin`, else `~/.local/bin`).
- B6 Build provenance is attested (`gh attestation verify`).
- F1 A wrong checksum, a missing archive or a tag that is not `vX.Y.Z[-pre]`
  stops without installing or publishing.

## 3. Decisions and Risks

- D1 CGO where it is available (Linux via a static musl build in Alpine, macOS
  on native runners): 30× faster runs than the WebAssembly parser, same output.
- D2 Native runners for every platform, so each binary is executed where it
  will run. Windows is cross-built on Linux and run on a Windows runner.
- D3 The build steps live in the workflow, not the Makefile, so a manual run
  can build tags older than this change (v2.4.0); the version flag matches the
  Makefile's.
- D4 Go: exactly the tag's go.mod version (`go 1.26.10` for v2.4.1). The first
  v2.4.1 run used `1.26`, which setup-go resolved to a cached 1.26.7 on
  `macos-15-intel`; go.mod refused it.
- R1 GitHub runner labels (`macos-15-intel`, `ubuntu-24.04-arm`) are checked
  only when the workflow first runs.

## 4. Changes

- `.github/workflows/release.yml` (new): B1–B4, B6.
- `scripts/install.sh` (new): B5, F1.
- `GUIDE.md`, `README.md`: install from a release.

## 5. Verification

Local, v2.4.0 source, with the workflow's Linux build command:

- `golang:1.26-alpine`, CGO, `-extldflags -static`: `statically linked`;
  `sqlc version` = `v2.4.0-wicked-fork`; `compile` and `generate` on
  `wicked_compat` pass. The binary runs on Debian 11, Ubuntu 20.04 and Alpine
  3.20 and generates ggsrv-layout's repositories identically to a CGO
  `make build` (except the version header, which the old build got wrong).
- Cross builds with `CGO_ENABLED=0` succeed for all five platforms.
- `scripts/install.sh` against a local copy of the release layout, on Ubuntu
  20.04 (dash) and Alpine 3.20 (busybox): latest-release lookup, explicit
  version, pinned `SQLC_SHA256`, wrong checksum (refused), unknown version
  (404, nothing installed).
- ShellCheck (`install.sh`) and actionlint (`release.yml`): clean.

## 6. Human Decisions

- Yumin, 2026-11-04 (request above).
- Yumin, 2026-11-04: govulncheck failed on `main`; fix the dependencies first
  (#22) and publish the first binaries as v2.4.1 rather than attaching
  binaries built from v2.4.0's vulnerable dependencies.

## 7. Outcome

- [ ] Merged.
- [ ] v2.4.1 released by a tag push, with binaries.

## 8. Remaining Work

- None.

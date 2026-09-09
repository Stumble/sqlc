# wicked-sqlc

A PostgreSQL-focused [sqlc](https://github.com/sqlc-dev/sqlc) fork that generates
type-safe Go code for [wpgx](https://github.com/Stumble/wpgx) and
[dcache](https://github.com/Stumble/dcache). Write SQL; generate query methods with
timeouts, optional caching, cache invalidation, replica access, and load/dump
helpers for tests.

## Architecture

We regularly rebase wicked-sqlc onto upstream sqlc releases. Our changes fall into
two layers: compiler changes in the fork and a dedicated Go backend.

1. **Compiler** — Parses SQL, resolves schema dependencies, and infers SQL types
   and parameter nullability. In wpgx mode, the first schema file owns the primary
   model; the remaining files supply dependencies and are analyzed first.
2. **Codegen contract** — sqlc's `GenerateRequest` carries the catalog, typed
   queries, and comments. A small `WickedMetadata` extension adds the primary
   schema source, its relation, and top-level SELECT classification. Options such
   as `timeout`, `cache`, and `invalidate` travel as ordinary query comments, not
   custom compiler parameters.
3. **Wicked backend** — [internal/codegen/wicked](internal/codegen/wicked) maps SQL
   types to Go and renders the templates. It parses comment options and generates
   cache keys, timeout handling, invalidation hooks, replica APIs, and test helpers.
   The generated code uses wpgx and dcache at runtime.

The backend runs in-process, bundled in the `sqlc` binary. There is no separate
plugin to install. `sql_package: wpgx` selects it; other Go configurations use
upstream's standard Go backend.

Compiler fixes stay in this fork until validated with the full wicked stack, then
can be proposed upstream independently. Generation-specific conventions stay in
the wicked backend.

## Use it

Build the current `main` branch with the Go version specified in [go.mod](go.mod):

```sh
git clone https://github.com/Stumble/sqlc.git
cd sqlc
make install
```

Select the backend in `sqlc.yaml`:

```yaml
version: '2'
sql:
  - engine: postgresql
    schema: books/schema.sql
    queries: books/query.sql
    gen:
      go:
        sql_package: wpgx
        package: books
        out: books
```

Every query requires a timeout. Caching is opt-in:

```sql
-- name: GetBook :one
-- -- timeout: 500ms
-- -- cache: 10m
SELECT * FROM books WHERE id = @id;
```

Run `sqlc generate` to generate code, or `sqlc diff` to check that it is up to date.
Existing wicked projects keep the same configuration and commands.

See the [guide](GUIDE.md) for schema conventions, query options, and runtime setup,
or [bookstore](https://github.com/Stumble/bookstore) for a working example.
[Upstream documentation](https://docs.sqlc.dev) covers standard sqlc features.

---

The original upstream README is preserved below.

# sqlc: A SQL Compiler

![go](https://github.com/sqlc-dev/sqlc/workflows/go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/sqlc-dev/sqlc)](https://goreportcard.com/report/github.com/sqlc-dev/sqlc)

sqlc generates **type-safe code** from SQL. Here's how it works:

1. You write queries in SQL.
1. You run sqlc to generate code with type-safe interfaces to those queries.
1. You write application code that calls the generated code.

Check out [an interactive example](https://play.sqlc.dev/) to see it in action, and the [introductory blog post](https://conroy.org/introducing-sqlc) for the motivation behind sqlc.

## Overview

- [Documentation](https://docs.sqlc.dev)
- [Installation](https://docs.sqlc.dev/en/latest/overview/install.html)
- [Playground](https://play.sqlc.dev)
- [Website](https://sqlc.dev)
- [Downloads](https://downloads.sqlc.dev/)
- [Community](https://discord.gg/EcXzGe5SEs)

## Supported languages

- [sqlc-gen-go](https://github.com/sqlc-dev/sqlc-gen-go)
- [sqlc-gen-kotlin](https://github.com/sqlc-dev/sqlc-gen-kotlin)
- [sqlc-gen-python](https://github.com/sqlc-dev/sqlc-gen-python)
- [sqlc-gen-typescript](https://github.com/sqlc-dev/sqlc-gen-typescript)

Additional languages can be added via [plugins](https://docs.sqlc.dev/en/latest/reference/language-support.html#community-language-support).

## Sponsors

Development is possible thanks to our sponsors. If you would like to support sqlc,
please consider [sponsoring on GitHub](https://github.com/sponsors/kyleconroy).

<p align="center">
  <a href="https://riza.io?utm_source=sqlc+readme"><img width=400 src="https://sqlc.dev/sponsors/riza-readme.png" alt="Riza.io"></a>
</p>

<p align="center">
  <a href="https://coder.com?utm_source=sqlc+readme"><img width=200 src="https://sqlc.dev/sponsors/coder-readme.png" alt="Coder.com" /></a>
  <a href="https://mint.fun?utm_source=sqlc+readme"><img width=200 src="https://sqlc.dev/sponsors/mint-readme.png" alt="Mint.fun" /></a>
  <a href="https://mux.com?utm_source=sqlc+readme"><img width=200 src="https://sqlc.dev/sponsors/mux-readme.png" alt="Mux.com" /></a>
</p>

<p align="center">
  <a href="https://github.com/Cyberax">Cyberax</a> - 
  <a href="https://github.com/NaNuNaNu">NaNuNaNu</a> - 
  <a href="https://github.com/Stumble">Stumble</a> - 
  <a href="https://github.com/WestfalNamur">WestfalNamur</a> - 
  <a href="https://github.com/alecthomas">alecthomas</a> - 
  <a href="https://github.com/cameronnewman">cameronnewman</a> - 
  <a href="https://github.com/danielbprice">danielbprice</a> - 
  <a href="https://github.com/davherrmann">davherrmann</a> - 
  <a href="https://github.com/dvob">dvob</a> - 
  <a href="https://github.com/gilcrest">gilcrest</a> - 
  <a href="https://github.com/gzuidhof">gzuidhof</a> - 
  <a href="https://github.com/jeffreylo">jeffreylo</a> - 
  <a href="https://github.com/mmcloughlin">mmcloughlin</a> - 
  <a href="https://github.com/ryohei1216">ryohei1216</a> - 
  <a href="https://github.com/sgielen">sgielen</a>
</p>

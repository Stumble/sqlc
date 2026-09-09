# wicked-sqlc

A PostgreSQL-focused [sqlc](https://github.com/sqlc-dev/sqlc) fork that generates
type-safe Go code for [wpgx](https://github.com/Stumble/wpgx) and
[dcache](https://github.com/Stumble/dcache). Write SQL; generate query methods with
timeouts, optional caching, cache invalidation, replica access, and load/dump
helpers for tests.

## Architecture

wicked-sqlc builds on upstream sqlc v1.31.1, with two layers of customization:
compiler changes in the fork and a dedicated Go backend.

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

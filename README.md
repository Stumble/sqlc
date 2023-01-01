# A wicked fork

## Opinionated choices
1. Use pgx/v5 types, if NULL-able.

## Opinionated fixes (changes)
1. Duplicated model for partitioned table:
   Only one model, which is defined by the first table creation statement of the first
   schema, will be generated into the model file.
2. Need to preserve camel-styled names:
   Well we cannot. Tokens were lower-cased in pg parser. To generate good-looking camel style
   variable names for golang, if you are not using the recommended snake case in SQL, you will
   need to use the `rename` feature. However, currently the rename option is not exposed to plugins.
   Plus that there is no global rename option, which is not convenient.
3. Not really doing type-checking on everything:
   Although using type cast can help to generate correctly typed code, but we found that not
   all SQL code are type-checked correctly. We might need to implement a new type check pass.

## Cherry-picked fixes

### Rename
https://github.com/kyleconroy/sqlc/pull/2001

### XXX
https://github.com/kyleconroy/sqlc/pull/1996

# sqlc: A SQL Compiler

![go](https://github.com/kyleconroy/sqlc/workflows/go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyleconroy/sqlc)](https://goreportcard.com/report/github.com/kyleconroy/sqlc)

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

## Sponsors

sqlc development is funded by our [generous
sponsors](https://github.com/sponsors/kyleconroy), including the following
companies:

- [Context](https://context.app)
- [ngrok](https://ngrok.com)
- [RStudio](https://www.rstudio.com/)
- [時雨堂](https://shiguredo.jp/)

If you use sqlc at your company, please consider [becoming a
sponsor](https://github.com/sponsors/kyleconroy) today.

Sponsors receive priority support via the sqlc Slack organization.

## Acknowledgements

sqlc was inspired by [PugSQL](https://pugsql.org/) and
[HugSQL](https://www.hugsql.org/).

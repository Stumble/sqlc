# A wicked fork

## Opinionated fixes (changes)
1. a set of rules mapping pg types to go types.
2. always emit JSON tag.
3. TODO: Unified place for all types defined in the query. 
   Usecase: eliminate duplicated ENUM values, in every generated model file.
4. Since NULL is not “equal to” NULL, (The null value represents an unknown
   value, and it is not known whether two unknown values are equal), You should never pass
   a nil pointer to the function argument in a select query where condition.
   Also, unlike needle, cache for query with parameters having pointer fields is not supported.
5. cache key uniqueness: cache key for a query is consisted by
   `packageName + methodName + "joining arguments in string format with ","`.
   The uniqueness of package names are checked for one configuration file.
6. Please define only *ONE* table per schema.sql file.
   Only one model, which is defined by the only table creation statement of the first
   schema, will be generated into the model file. Internally, to allow user to put
   `create materialized view` into schema.sql, we reversed the order of parsing schema files.
7. If you need to preserve camel-styled names, use rename option in configuration file.
   There is no way for us to do it automatically, because tokens were lower-cased in pg parser. 
   It is recommended to snake case in SQL.
8. Not really doing type-checking on everything:
   Although using type cast can help to generate correctly typed code, but we found that not
   all SQL code are type-checked correctly. We might need to implement a new type check pass.
9. Schema.sql will be copied into db.go file as `var Schema`. User need to be careful with using
   those schema. type/function declaration: does not support `IF NOT EXISTS`, so they should only
   be executed once. `Create [materialized] view` can only be executed after dependency tables 
   have been created.

## TODOs
1. Batch support for wpgx.

## Cherry-picked fixes
+ TBD: https://github.com/kyleconroy/sqlc/pull/2001

# sqlc: A SQL Compiler

![go](https://github.com/kyleconroy/sqlc/workflows/go/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/kyleconroy/sqlc)](https://goreportcard.com/report/github.com/kyleconroy/sqlc)

sqlc generates **type-safe code** from SQL. Here's how it works:

1. You write queries in SQL.
1. You run sqlc to generate code with type-safe interfaces to those queries.
1. You write application code that calls the generated code.

Check out [an interactive example](https://play.sqlc.dev/) to see it in action.

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

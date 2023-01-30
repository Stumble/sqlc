# Intro

Here we present a combo of using `wicked-sqlc + wpgx + cache` that can:

+ Generate fully type-safe idiomatic Go code with built-in
  + memory-redis cache layer with compression and singleflight protection.
  + telemetry: Prometheus and OpenTelemetry (WIP).
  + auto-generated Load and Dump function for easy golden testing.
+ Envvar configuration template for Redis and PostgreSQL.
+ A Testsuite for writing golden tests that can
  + import data from a test file into tables to setup background data
  + dump table to a file in JSON format
  + compare current DB data with a golden file

NOTE: this combo is for PostgreSQL, if you are using MySQL, you can checkout this project:
[Needle](https://github.com/Stumble/needle). It provides the same set of functionalities
as this combo.

# Sqlc (this wicked fork)

Although using sqlc might increase the productivity, as you no longer need to manually write the
boilerplate codes while having cache and telemetries out of the box,
it is **NOT** our goal.

Instead, by adopting to this restricted form, we hope to:

+ Make it extremely easy to see all possible ways to query DB. By explicitly listing all of them
  in the query.sql file, DBAs can examine query patterns and design indexes wisely. In the future,
  we might even be able to find out possible slow queries in compile time.
+ Force you to think twice before creating a new query. Some business logics can share the same
  query, which means higher cache hit ratio. Sometimes when there are multiple ways to implement a
  usecase, choose the one that can reuse existing indexes.

Sometimes, you might find sqlc too *restricted*, and cannot hold the eager to
write a function that builds the
SQL dynamically based on conditions, **don't**  do it, unless it is a must, which is hardly true.
In the end of the day, the so-called backend development is more or less about
building a data-intensive software, where the most common bottleneck, is that fragile database,
which is very costly to scale.

From another perspective, the time will either be spent on (1) later, when the business grew and
the bottleneck was reached, diagnosing the problem and refactoring your database codes, while
your customers are disappointed, or (2) before the product is launched, writing queries.

## Install

```bash
# cgo must be enabled because: https://github.com/pganalyze/pg_query_go
git clone https://github.com/Stumble/sqlc.git
cd sqlc/
make install
sqlc version
# you shall see: v*****-wicked-fork
```

## Getting started

It is recommended to read [Sqlc doc](https://docs.sqlc.dev/en/stable/) to get some
general ideas of how to use sqlc. In the following example, we will pay more
attention to things that are different to official sqlc.

In this tutorial, we will build a online bookstore, with unit tests, to demonstrate how to use this combo.
The project can be found here: [bookstore](https://github.com/Stumble/bookstore).

### Project structure

After `go mod init`, we created a `sqlc.yaml` file that manages the code generation, under `pkg/repos/`.
This will be the root directory for our data access layer.

First, let's start with building a table that stores book information.

```bash
.
├── go.mod
└── pkg
    └── repos
        ├── books
        │   ├── query.sql
        │   └── schema.sql
        └── sqlc.yaml
```

Initially, the YAML configuration file looks like this:

```yaml
version: "2"
sql:
- schema: "books/schema.sql"
  queries: "books/query.sql"
  engine: "postgresql"
  gen:
    go:
      sql_package: "wpgx"
      package: "books"
      out: "books"
```

It configures sqlc to generate Go code for `books` table based on the schema and queries SQL file,
under `books/` directory, relatively to `sqlc.yaml` file.
The only thing different from the official sqlc is the `sql_package` option. This wicked fork will
use `wpgx` package as the SQL driver, so you have to set `sql_package` to this value.

### Schema

A schema file is 1-to-1 mapped to a logical table. That is, you need to write 1 schema file for
each **logical** table in DB. To be more clear:

+ 1 schema file for 1 normal physical table.
+ For **Declarative Partitioning**, the table declaration and all its partitions can be, and should
  be placed into one schema file, as they are logically one table.
+ For **(Materialized) View**, one schema file per view is required.

You can and you should list all the **constraints and indexes** in the schema file. In the future,
we might have some static analyze tool to check for slow queries. Also, listing them here will
make code viewers' lives much easier.

Different from the official sqlc, for each schema section in the sqlc.yaml file,
only the **first** schema file in the array will be considered as the source of generating Go struct.
For example, if the config is `- schema: ["t1.sql", "t2.sql"]`,
forked sqlc will only generate a Go struct for
the first (and the only) table definition in `t1.sql`. If there are two table creation statements,
sqlc will error out.
Schema files after the first one are used as references for column types.

Now let's look into `books/schema.sql` file.

```SQL
CREATE TYPE category AS ENUM (
    'computer_science',
    'philosophy',
    'comic'
);

CREATE TABLE IF NOT EXISTS books (
   id            BIGSERIAL           GENERATED ALWAYS AS IDENTITY,
   name          VARCHAR(255)        NOT NULL,
   description   VARCHAR(255)        NOT NULL,
   metadata      JSON,
   category      category            NOT NULL,
   price         DECIMAL(10,2)       NOT NULL,
   created_at    TIMESTAMP           NOT NULL DEFAULT NOW(),
   updated_at    TIMESTAMP           NOT NULL DEFAULT NOW(),
   CONSTRAINT books_id_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS book_name_idx ON books (name);
CREATE INDEX IF NOT EXISTS book_category_id_idx ON books (category, id);
```

We defined a table called books, using id as primary key, with two indexes.
There are two interesting columns:

+ Column `category` is of type `book_category`. Sqlc will generate new type `BookCategory` in `models.go`
  file, with `Scan` and `Value` methods to allow it to be used by the pgx driver.
  Unlike tables, all enum types will be generated in the model file, if the schema file is referenced.
+ Column `price` will be of type `pgtype.Numeric`, which is defined in `github.com/jackc/pgx/v5/pgtype`.
  This is because that there is no native type in GO to represent a decimal number.

The generated `models.go` file would contain a struct that represents a *row* of the table.

```go
type Book struct {
  ID          int64          `json:"id"`
  Name        string         `json:"name"`
  Description string         `json:"description"`
  Metadata    []byte         `json:"metadata"`
  Category    BookCategory   `json:"category"`
  Price       pgtype.Numeric `json:"price"`
  CreatedAt   time.Time      `json:"created_at"`
  UpdatedAt   time.Time      `json:"updated_at"`
}
```

Then, let's create another table for storing users.

```sql
CREATE TABLE IF NOT EXISTS users (
   id          INT          GENERATED ALWAYS AS IDENTITY,
   name        VARCHAR(255) NOT NULL,
   metadata    JSON,
   image       TEXT         NOT NULL,
   created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
   CONSTRAINT users_id_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS users_created_at_idx
    ON Users (CreatedAt);
CREATE UNIQUE INDEX IF NOT EXISTS users_lower_name_idx
    ON Users ((lower(Name))) INCLUDE (ID);
```

#### Reference other schema

When the schema file (e.g., creating a view),
or the queries (e.g., joining other tables) in the
`query.sql` file referenced other tables, you must list those dependencies in the schema section.
The order of tables in the array must be a topological sort of the dependency graph.
Another way to say it: it is just like C headers, but you list them reversely.

For example, when creating a table of orders that looks like:

```sql
CREATE TABLE IF NOT EXISTS orders (
   id         INT       GENERATED ALWAYS AS IDENTITY,
   user_id    INT       references users(ID) ON DELETE SET NULL,
   book_id    INT       references books(ID) ON DELETE SET NULL,
   price      BIGINT    NOT NULL,
   created_at TIMESTAMP NOT NULL DEFAULT NOW(),
   is_deleted BOOLEAN   NOT NULL,
   CONSTRAINT orders_id_pkey PRIMARY KEY (id)
);
```

If we add a query that joins books and users with the order table, for example,

```sql
-- name: GetOrderByID :one
SELECT
  orders.ID,
  orders.user_id,
  orders.book_id,
  orders.created_at,
  users.name AS user_name,
  users.image AS user_thumbnail,
  books.name AS book_name,
  books.price As book_price,
  books.metadata As book_metadata
FROM
  orders
  INNER JOIN books ON orders.book_id = books.id
  INNER JOIN users ON orders.user_id = users.id
WHERE
  orders.is_deleted = FALSE;
```

we must list the schema file of books and users after orders table in the configuration file.

```yaml
- schema: ["orders/schema.sql", "books/schema.sql", "users/schema.sql"]
  queries: "orders/query.sql"
  ...
```

Otherwise, sqlc will complain

```text
orders/query.sql:1:1: relation "books" does not exist
orders/query.sql:45:1: relation "users" does not exist
```

Another example is the `revenues` table schema. It is a materialized view

```sql
CREATE MATERIALIZED VIEW IF NOT EXISTS by_book_revenues AS
  SELECT
    books.id,
    books.name,
    books.category,
    books.price,
    books.created_at,
    sum(orders.price) AS total,
    sum(
      CASE WHEN
        (orders.created_at > now() - interval '30 day')
      THEN orders.price ELSE 0 END
    ) AS last30d
  FROM
    books
    LEFT JOIN orders ON books.id = orders.book_id
  GROUP BY
      books.id;
```

Because this table is depending on both orders and books, in the schema file we must list them after
the revenue table.

```yaml
- schema: ["revenues/schema.sql", "orders/schema.sql", "books/schema.sql"]
```

Lastly, each schema file will be saved into a string named `Schema`, defined in the `models.go`.
They are made there to be convenient for you to setup DB for unit tests.
Also, it is a good practice to always include the `IF NOT EXISTS` clause when creating tables and indexes.

### Query

`query.sql` file is where your define all the possible ways to access to the table. Each table
must have 1 query file.
Queries can access all the table columns as long as their tables are listed in the schema section in
the configuration. We have seen an example, `GetOrderByID`, where the query joins other tables.

Here is an example of listing all books of a category, with using id
as the cursor for pagination.

```sql
-- name: ListByCategory :many
SELECT *
FROM
  books
WHERE
  category = @category AND id > @after
ORDER BY
  id
LIMIT @first;
```

This wicked forked sqlc adds two abilities to query: cache and invalidate.

Both of them are added by extending sqlc to allow passing additional options per each query.
Originally, you can only specify name and the type of result in the comments before SQL.
The new feature allows you to pass any options to codegen backend by adding comments starts with `-- --`.

For example, this will generate code that caches the result of all books for 10 minutes.

```sql
-- name: GetAllBooks :many
-- -- cache : 10m
SELECT * FROM books;
```

Btw, this syntax looks very similar as passing arguments to the underlying script in npm.

```bash
npm run server -- --port=8080 // invokes `run server script with --port=8080`
```

#### Cache

Cache accepts a [Go time.Duration format](https://pkg.go.dev/maze.io/x/duration#ParseDuration) as the
only argument, which specify how long the result will be cached, if a cache is configured
in the queries struct. If no cache is injected, caching is not possible and duration will be ignored.

The best practice is to cache frequently queried objects, especially

+ cache results that we know how to invalidate for longer time, in most cases, they are result of single
  rows. For example, a row of book information can be cached for a long time, because we know when the book
  information will be updated so that we can apply invalidate accordingly.
+ cache results that we do not know how to invalidate for a shorter time. For example, a list of top seller
  books, because it is hard for us to know if we should invalidate the cache of that list when we are updating
  information of some books, (unless you do some fancy bloom-filter stuff..).

#### Invalidate

When we mutate the state of table, we should proactively invalidate some cache values.

#### Best practices

+ Use `@arg_name` to explicitly name all the arguments for the query. If somehow not working, try to use
  `sqlc.arg()`, or `sqlc.narg()` if appropriate.
  It is highly recommended to read [this doc](https://docs.sqlc.dev/en/latest/howto/named_parameters.html).
+ DO NOT mix `$`, `@` and `sqlc.arg()/sqlc.narg()` in one SQL query. Each query should purely use one kind
  of parameter style.
+ Use `::type` postgreSQL type conversion to hint sqlc for arguments that their types are hard or
  impossible to be inferred.

#### Known issues

+ `from unnest(array1, arry2)` is not supported yet. Use `select unnest(array1), unnest(array1)` instead.
  Note, when the arrays are not all the same length then the shorter ones are padded with NULLs.
+ In some cases, you must put a space before the "@" symbol for named parameter,
  For example, a statement like `select ... where a=@a`
  cannot be correctly parsed by sqlc. You must change it to `select ... a = @a`.
  You shall notice this type of error after code generation, as you will see that some parameters are
  missing in the generated code and an incorrect SQL is used for query (still including @).
+ Enum type support is very limited. First, you cannot use copyfrom for when the column is
  an enum type. Also, when using enum type in any clause, e.g., `enum_col = ANY(@xxx::enum_type[])`, it won't work. You have to do `enum_col = ANY(@xxx::text[]::enum_type[])`, and
  unfortunately the parameters type will become string array, instead of exptected enum array.

#### Case study

##### Bulk insert and upsert

If data will not violate any constraints, you can just use copyfrom.
When a constraint fails, an error is throw, and none of data are copied (it is rolled back).

```sql
-- name: BulkInsert :copyfrom
INSERT INTO books (
   name, description, metadata, category, price
) VALUES (
  $1, $2, $3, $4, $5
);
```

But If you want to implement bulk upsert, the best practice is to use `unnest` function to pass each
column as an array. For example, the following query will generate a bulk upsert method.

```sql
-- name: UpsertUsers :exec
insert into users
  (name, metadata, image)
select
        unnest(@name::VARCHAR(255)[]),
        unnest(@metadata::JSON[]),
        unnest(@image::TEXT[])
on conflict ON CONSTRAINT users_lower_name_key do
update set
    metadata = excluded.metadata,
    image = excluded.image;
```

The generated Go code will look like:

```go
type UpsertUsersParams struct {
  Name     []string
  Metadata [][]byte
  Image    []string
}

func (q *Queries) UpsertUsers(ctx context.Context, arg UpsertUsersParams) error {
  _, err := q.db.WExec(ctx, "UpsertUsers", upsertUsers,
    arg.Name, arg.Metadata, arg.Image)
  // ...
}
```

##### Other bulk operations

When you have too many parameters in a query, it can become slow.
To operate on data in bulk, it is a good practice to use `select UNNEST(@array_arg)...` to build
an intermediate table, and then use that table.

For example, to select based on different conditions, you can:

```sql
-- name: ListOrdersByUserAndBook :many
SELECT * FROM orders
WHERE
  (user_id, book_id) IN (
  SELECT
    UNNEST(@user_id::int[]),
    UNNEST(@book_id::int[])
);
```

To update different rows to different values, you can:

```sql
-- name: BulkUpdate :exec
UPDATE orders
SET
  price=temp.price,
  book_id=temp.book_id
FROM
  (
    SELECT
      UNNEST(@id::int[]) as id,
      UNNEST(@price::bigint[]) as price,
      UNNEST(@book_id::int[]) as book_id
  ) AS temp
WHERE
  orders.id=temp.id;
```

##### Partial update

If you wish to write one SQL update statement that only update some columns,
based on the arguments at runtime,
you can use the following trick that use `sqlc.narg` to generate nullable parameters and use
`coalesce` function, so that a column is set to the new value, if not null, or unchanged.

However, please note this trick CANNOT handle this case: when a column is nullable, you cannot set it
to null, using this trick. Also, You MUST write some unit tests to check if the SQL would work as expected.

```sql
-- name: PartialUpdateByID :exec
UPDATE books
SET
  description = coalesce(sqlc.narg('description'), description),
  metadata = coalesce(sqlc.narg('meta'), metadata),
  price = coalesce(sqlc.narg('price'), price),
  updated_at = NOW()
WHERE
  id = sqlc.arg('id');
```

##### Refresh materialized view

Refresh statement is supported, you can just list it as a query.

```sql
-- name: Refresh :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY by_book_revenues;
```

### SQL Naming conventions

In short, for table and column names, always use 'snake_case'.
More details: [Naming Conventions](https://www.geeksforgeeks.org/postgresql-naming-conventions/)

Indexes should be named in the following way:

```text
{tablename}_{columnname(s)}_{suffix}
```

where the suffix is one of the following:

+ ``pkey`` for a Primary Key constraint;
+ ``key`` for a Unique constraint;
+ ``excl`` for an Exclusion constraint;
+ ``idx`` for any other kind of index;
+ ``fkey`` for a Foreign key;
+ ``check`` for a Check constraint;

If the name is too long, (max length is 63), try to use shorter names for column names.

Table Partitions should be named as

```text
{{tablename}}_{{partition_name}}
```

where the partition name should represent how the table is partitioned.
For example:

```sql
CREATE TABLE measurement (
    city_id         int not null,
    logdate         date not null,
    peaktemp        int,
    unitsales       int
) PARTITION BY RANGE (logdate);

CREATE TABLE measurement_y2006m02 PARTITION OF measurement
    FOR VALUES FROM ('2006-02-01') TO ('2006-03-01');
```

#### Work with legacy project and CamelCase-style names

If you are working with a legacy codebase that its DB does not follow the above
naming convention, for example, used CamelCase style for column names, there are
some caveats you must pay attention to.

First, please note that, in PostgreSQL, identifiers (including column names) that are **not double-quoted** are folded to lowercase, while
column names that were created with double-quotes and thereby retained uppercase letters
(and/or other syntax violations) have to be double-quoted for the rest of their life.

Here's an example.

```sql
CREATE TABLE IF NOT EXISTS test (
   id           INT       GENERATED ALWAYS AS IDENTITY,
   CamelCase    INT,
   snake_case   INT,
   CONSTRAINT test_id_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS test2 (
   id            INT       GENERATED ALWAYS AS IDENTITY,
   "CamelCase"   INT,
   snake_case    INT,
   CONSTRAINT test2_id_pkey PRIMARY KEY (id)
);
```

The column `CamelCase` in table `test` were not created with double-quotes, so internally, the name
was actually stored in the lower-cased string. But `test2.CamelCase` did, so the name is stored in its
original camcal-case style. See below logs from psql.

```psql
# \d test
                            Table "public.test"
   Column   |  Type   | Collation | Nullable |           Default
------------+---------+-----------+----------+------------------------------
 id         | integer |           | not null | generated always as identity
 camelcase  | integer |           |          |
 snake_case | integer |           |          |

# \d test2
                            Table "public.test2"
   Column   |  Type   | Collation | Nullable |           Default
------------+---------+-----------+----------+------------------------------
 id         | integer |           | not null | generated always as identity
 CamelCase  | integer |           |          |
 snake_case | integer |           |          |
```

Differences of accessing these two tables:

```sql
-- This is okay!, all identifiers will be lowered-cased if not quoted.
insert into test (
  CaMelCASe, snake_case)
values (
  1, 2);

-- NOT okay!
-- ERROR:  column "camelcase" of relation "test2" does not exist
-- LINE 2:   CamelCase, snake_case)
insert into test2 (
  CamelCase, snake_case)
values (
  1, 2);

-- The right way to work with table test2.
insert into test2 (
  "CamelCase", snake_case)
values (
  1, 2);

-- Another example of quoting identifiers.
select t2."CamelCase" from test2 as t2;
```

Unfortunately, sqlc can not check for errors if you forgot to quote identifiers correctly, for now.
So you need to be very careful if the column names were actually stored in CamelCase.

Second, if you want to preserve the CamelCase name in go, use rename in the `sqlc.yaml` configuration,
for example,

```yaml
version: "2"
overrides:
  go:
    rename:
      createdat: "CreatedAt"
      updatedat: "UpdatedAt"
sql:
  ....
```

# DCache

[DCache](https://github.com/Stumble/dcache) is the core of protecting the database.

# WPgx

[WPgx](https://github.com/Stumble/wpgx) stands for 'wrapped-Pgx'. It simply wraps the common
query and execute functions of pgx driver to add prometheus and open telemetry tracer.

In addition to original pgx functions, we added a `PostExec(fn PostExecFunc)` to both
normal connection type `WConn` and transaction type `WTx`. The `fn` will be executed after
the 'transaction' is successfully committed. A common usecase is to run cache invalidation
functions post execution.

The code of wpgx is very simple, the best way to understand it is to read its source codes.

## Telemetry

### Prometheus

+ {appName}_wpgx_conn_pool{name="max_conns/total_conns/...."}: connection pool gauges.
+ {appName}_wpgx_request_total{name="$queryName"}: number of DB hits for each query.
+ {appName}_wpgx_latency_milliseconds{name="$queryName"}: histogram of SQL execution duration.

### Open Telemetry

TBD.

### Transaction

TBD.

### Testsuite

TBD.

# Unit testing

Most Unit tests follows this pattern:

1. Setup dependencies like DB, Redis and etc.. [X] 
2. Load background data into DB. [X] 
3. Run functions the test hopes to check.
4. Verify output of the function is expected.
5. Verify DB state is expected. [X] 

Steps with [X] mark indicates that we can use boilerplate function or code generated from
the `sqlc + wpgx` combo.

For example, to test a 'search book by names' usecase, the unit test may:

1. Setup a *Wpgx.pool that connects to the DB instance and pass it to the usecase.
2. Insert some book items into books table.
3. Run the search usecase function.
4. Expect the number of returned value to be N.
5. Verify that books book has not been changed at all, but the search_activity table does
   have a new entry.

[Here](https://github.com/Stumble/bookstore/blob/main/pkg/usecases/usecase_test.go) is the
example code that leverages auto-generated code to test the above usecase.

## Setup DB connection for the test

PgxTestSuite is a Testify.testsuite with some helper functions for easy-writing (1), (2) and (5).

First, your test suite needs to embed the WPgxTestSuite and initialize it with a wpgx.Config.
Like the below example, you can directly use the configuration of envvar. You can also
create the suite via `NewWPgxTestSuiteFromConfig`, if you hope to pass a Config.

One caveat: You must set POSTGRES_APPNAME envvar if you want to use the default
NewWPgxTestSuiteFromEnv,
because it is required. For example, you can do `export POSTGRES_APPNAME=xxxtests`.

```go
import (
  "github.com/stumble/wpgx/testsuite"
)

type myTestSuite struct {
  *testsuite.WPgxTestSuite
}

func newMyTestSuite() *myTestSuite {
 return &myTestSuite{
  WPgxTestSuite: testsuite.NewWPgxTestSuiteFromEnv("testDbName", []string{
   `CREATE TABLE IF NOT EXISTS books (
    // .....
    );`,
  }),
 }
}

func TestMyTestSuite(t *testing.T) {
 suite.Run(t, newMyTestSuite())
}
```

## Loader and Dumper

Note that testsuite defined two interfaces:

```go
type Loader interface {
  Load(data []byte) error
}
type Dumper interface {
  Dump() ([]byte, error)
}
```

They are **table-scope** loader and dumper that can load/dump table from/to bytes.
The wicked-fork sqlc will automatically generate load and dump functions for each table schema.

## Testsuite helpers

The testsuite provides these 3 helper functions:

```go
// load state into memory from file.
func (suite *WPgxTestSuite) LoadState(filename string, loader Loader);
// dump table state to file name via dumper.
func (suite *WPgxTestSuite) DumpState(filename string, dumper Dumper);
// dump table state via dumper to memory, and load testdata/xxx/yyy.${tableName}.golden
// into memory and then compare these two.
func (suite *WPgxTestSuite) Golden(tableName string, dumper Dumper);
```

### Setup background data from loader

## Known Issues

1. Cannot use auto-generated loader
   if the table has any `GENERATED ALWAYS AS IDENTITY` column.
   Unless required by business logics, it is recommended to use `GENERATED BY DEFAULT AS IDENTITY`.
2. Cannot use auto-generated loader if the table has a column of `pgtype.Numeric`, because
   currently `pgx/v5` does not provide any UnmarshalJSON for this type.

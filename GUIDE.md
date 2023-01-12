# Intro
Here we present a combo of using `wicked-sqlc + wpgx + dcache` that can:

+ Generate fully type-safe idiomatic Go code with built-in
  + memory-redis cache layer with compression and singleflight protection.
  + telemetry: Prometheus and open-telemetry (WIP).
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
It is recommended to read [Sqlc doc](https://docs.sqlc.dev/en/stable/) to get the
general idea of how to use sqlc. In the following example, we will pay more
attention to things that are different to official sqlc.

Now, we will build a online bookstore, with unit tests, to demonstrate how to use this combo.

### Project structure
After `go mod init`, we created a sqlc.yaml file that manages the code generation, under `pkg/repos/`. This will be the root directory for our data access layer. Also, let's start with building a table that stores book informations.

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
Initially, let's create a yaml configuration file looks like this:
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
under `books/` directory, relative to sqlc.yaml file.
The only thing different from the official sqlc is the `sql_package` option. This wicked fork will
use `wpgx` packge as the SQL driver, so you have to set `sql_packge` to this value.

### Schema
A schema file is 1-to-1 mapped to a logical table. That is, you need to write 1 schema file for
each **logical** table in DB. To be more clear:
+ 1 schema fiel for 1 normal physical table.
+ For **Declarative Partitioning**, the table declaration and all its partitions can be, and should
  be placed into one schema file, as they are logically one table.
+ For **(Materialized) View**, one schema file per view is required.

You can and you should list all the **constrants and indexes** in the schema file. In the future,
we might have some static analyze tool to check for slow queries. Also, listing them here will
make code viewer's lives much easier.

Different from the official sqlc, for each schema section in the sqlc.yaml file,
only the *first* schema file in the array will be considered as source of generating Go struct.
For example, if the config is `- schema: ["t1.sql", "t2.sql"]`,
forked sqlc will only generate a Go struct for
the first (and the only) table definition in `t1.sql`. If there are two table creation statements,
sqlc will error out.
Schema files after the first one are used as references for column types.

#### Reference other schema
If your first schema file (e.g., creating a view), or queries (e.g., joining other tables) in the
query.sql file referenced other tables, you must list those dependencies in the schema section.
The order of tables in the array must be a topological sort of the dependency graph.
Another way to say it: it is just like C headers, but you list them reversely.

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
   category      ItemCategory        NOT NULL,
   price         DECIMAL(10,2)       NOT NULL,
   created_at    TIMESTAMP           NOT NULL DEFAULT NOW(),
   updated_at    TIMESTAMP           NOT NULL DEFAULT NOW(),
   CONSTRAINT books_id_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS book_name_idx ON books (name);
CREATE INDEX IF NOT EXISTS book_category_id_idx ON books (category, id);
```
Pretty simple right?

### Query

## WPgx

### Testsuite

## DCache

## Naming conventions
In short, for table and column names, always use 'snake_case'. 
More details: [Naming Conventions](https://www.geeksforgeeks.org/postgresql-naming-conventions/) 

Indexes should be named in the following way:
```
{tablename}_{columnname(s)}_{suffix}
```
where the suffix is one of the following:
* ``pkey`` for a Primary Key constraint;
* ``key`` for a Unique constraint;
* ``excl`` for an Exclusion constraint;
* ``idx`` for any other kind of index;
* ``fkey`` for a Foreign key;
* ``check`` for a Check constraint;

If the name is too long, (max length is 63), try to use shorter names for columnnames.

Table Partitions should be named as
```
{{tablename}}_{{partition_name}}
```
where the partition name should represent how the is the the table being partitioned.
For example:
```
CREATE TABLE measurement (
    city_id         int not null,
    logdate         date not null,
    peaktemp        int,
    unitsales       int
) PARTITION BY RANGE (logdate);

CREATE TABLE measurement_y2006m02 PARTITION OF measurement
    FOR VALUES FROM ('2006-02-01') TO ('2006-03-01');
```

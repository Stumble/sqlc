package compiler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqlc-dev/sqlc/internal/config"
	"github.com/sqlc-dev/sqlc/internal/multierr"
	"github.com/sqlc-dev/sqlc/internal/opts"
)

func TestParameterContextInference(t *testing.T) {
	for _, tc := range []struct {
		name, condition string
	}{
		{"null_first", "sqlc.narg('id') IS NULL OR id = sqlc.narg('id')"},
		{"cast_after_null", "sqlc.narg('id') IS NULL OR id = sqlc.narg('id')::bigint"},
		{"comparison_first", "id = sqlc.narg('id') OR sqlc.narg('id') IS NULL"},
		{"cast_first", "sqlc.narg('id')::bigint IS NULL OR id = sqlc.narg('id')"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := compileParameterQuery(t, "SELECT * FROM things WHERE ("+tc.condition+") AND name = sqlc.arg('name');")
			if len(q.Params) != 2 {
				t.Fatalf("parameters = %d, want 2", len(q.Params))
			}
			id, name := q.Params[0], q.Params[1]
			if id.Number != 1 || id.Column.Name != "id" || strings.TrimPrefix(id.Column.DataType, "pg_catalog.") != "int8" || id.Column.NotNull {
				t.Errorf("id parameter: number=%d column=%+v; want nullable bigint parameter 1", id.Number, id.Column)
			}
			if name.Number != 2 || name.Column.Name != "name" || name.Column.DataType != "text" || !name.Column.NotNull {
				t.Errorf("name parameter: number=%d column=%+v; want non-null text parameter 2", name.Number, name.Column)
			}
		})
	}
}

func TestParameterContextPreservesInsertBinding(t *testing.T) {
	q := compileParameterQuery(t, `WITH previous AS (
    SELECT name FROM things WHERE id = $1
)
INSERT INTO things (id, name)
SELECT $1, @name::text
WHERE NOT EXISTS (SELECT * FROM previous WHERE previous.name = @name::text)
RETURNING *;`)
	if len(q.Params) != 2 {
		t.Fatalf("parameters = %d, want 2", len(q.Params))
	}
	id := q.Params[0]
	if id.Number != 1 || id.Column.Name != "id" || !id.Column.NotNull {
		t.Fatalf("lost INSERT target binding: %+v", id.Column)
	}
}

func TestParameterContextNullability(t *testing.T) {
	for _, tc := range []struct {
		name, query, dataType string
		notNull               bool
	}{
		{"id_comparison", "UPDATE things SET optional_id = @id WHERE id <> @id RETURNING *;", "int8", true},
		{"time_comparison", "UPDATE things SET ended_at = @now WHERE expires_at <= @now RETURNING *;", "timestamptz", true},
		{"nullable_assignment", "UPDATE things SET optional_id = @id RETURNING *;", "int8", false},
		{"explicit_nullable", "UPDATE things SET optional_id = sqlc.narg('id') WHERE id <> sqlc.narg('id') RETURNING *;", "int8", false},
		{"positional", "UPDATE things SET optional_id = $1 WHERE id <> $1 RETURNING *;", "int8", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			q := compileParameterQuery(t, tc.query)
			if len(q.Params) != 1 {
				t.Fatalf("parameters = %d, want 1", len(q.Params))
			}
			col := q.Params[0].Column
			if strings.TrimPrefix(col.DataType, "pg_catalog.") != tc.dataType || col.NotNull != tc.notNull {
				t.Fatalf("parameter = %+v, want %s not-null=%v", col, tc.dataType, tc.notNull)
			}
		})
	}
}

func TestParameterContextRepeatedRelation(t *testing.T) {
	q := compileParameterQuery(t, `WITH previous AS (
    SELECT name FROM things WHERE id = $1
)
SELECT * FROM things WHERE name IN (SELECT name FROM previous);`)
	if len(q.Params) != 1 || !q.Params[0].Column.NotNull || q.Params[0].Column.Name != "id" {
		t.Fatalf("parameter lost across repeated relation: %+v", q.Params)
	}
}

func TestParameterContextInsertSelectSameTable(t *testing.T) {
	q := compileParameterQuery(t, `INSERT INTO things (id, name, expires_at)
SELECT @id, name, expires_at FROM things WHERE name = @name RETURNING *;`)
	if len(q.Params) != 2 {
		t.Fatalf("parameters = %d, want 2", len(q.Params))
	}
	for i, want := range []string{"id", "name"} {
		if col := q.Params[i].Column; col.Name != want || !col.NotNull {
			t.Errorf("parameter %d = %+v, want non-null %s", i, col, want)
		}
	}
}

func TestParameterContextPreservesSelfJoinAmbiguity(t *testing.T) {
	c := parameterCompiler(t, `SELECT lhs.name FROM things lhs JOIN things rhs ON lhs.name = rhs.name WHERE id = $1;`)
	err := c.ParseQueries(c.conf.Queries, opts.Parser{})
	errs, ok := err.(*multierr.Error)
	if !ok || len(errs.Errs()) != 1 || !strings.Contains(errs.Errs()[0].Err.Error(), "ambiguous") {
		t.Fatalf("unqualified self-join parameter: %v, want ambiguity error", err)
	}
}

func compileParameterQuery(t *testing.T, query string) *Query {
	t.Helper()
	c := parameterCompiler(t, query)
	if err := c.ParseQueries(c.conf.Queries, opts.Parser{}); err != nil {
		t.Fatal(err)
	}
	return c.Result().Queries[0]
}

func parameterCompiler(t *testing.T, query string) *Compiler {
	t.Helper()
	dir := t.TempDir()
	schemaPath, queryPath := filepath.Join(dir, "schema.sql"), filepath.Join(dir, "query.sql")
	for path, contents := range map[string]string{
		schemaPath: "CREATE TABLE things (id bigint NOT NULL, name text NOT NULL, optional_id bigint, ended_at timestamptz, expires_at timestamptz NOT NULL);",
		queryPath:  "-- name: FindThings :many\n" + query,
	} {
		if err := os.WriteFile(path, []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	conf := config.SQL{Engine: config.EnginePostgreSQL, Schema: []string{schemaPath}, Queries: []string{queryPath}}
	c, err := NewCompiler(conf, config.Combine(config.Config{Version: "2"}, conf), opts.Parser{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	if err := c.ParseCatalog([]string{schemaPath}); err != nil {
		t.Fatal(err)
	}
	return c
}

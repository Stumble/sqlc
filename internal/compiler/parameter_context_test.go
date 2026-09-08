package compiler

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqlc-dev/sqlc/internal/config"
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

func compileParameterQuery(t *testing.T, query string) *Query {
	t.Helper()
	dir := t.TempDir()
	schemaPath, queryPath := filepath.Join(dir, "schema.sql"), filepath.Join(dir, "query.sql")
	for path, contents := range map[string]string{
		schemaPath: "CREATE TABLE things (id bigint NOT NULL, name text NOT NULL);",
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
	if err := c.ParseQueries([]string{queryPath}, opts.Parser{}); err != nil {
		t.Fatal(err)
	}
	return c.Result().Queries[0]
}

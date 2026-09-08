package compiler

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	goopts "github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/config"
	"github.com/sqlc-dev/sqlc/internal/opts"
)

func TestWickedSchemaOwnership(t *testing.T) {
	for _, tc := range []struct {
		name    string
		primary string
		want    string
		invalid bool
	}{
		{"table", "CREATE TABLE orders (id bigint, book_id bigint);", "orders", false},
		{"renamed_table", "CREATE TABLE initial (id bigint); ALTER TABLE initial RENAME TO renamed;", "renamed", false},
		{"materialized_view", "CREATE MATERIALIZED VIEW revenues AS SELECT id FROM books;", "revenues", false},
		{"partition", "CREATE TABLE events (id bigint) PARTITION BY RANGE (id); CREATE TABLE events_small PARTITION OF events FOR VALUES FROM (0) TO (100);", "events", false},
		{"two_layouts", "CREATE TABLE a (id bigint); CREATE TABLE b (id bigint);", "", true},
		{"no_layout", "CREATE TYPE category AS ENUM ('a');", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			primary, dependency := filepath.Join(dir, "primary.sql"), filepath.Join(dir, "books.sql")
			for path, sql := range map[string]string{primary: tc.primary, dependency: "CREATE TABLE books (id bigint NOT NULL);"} {
				if err := os.WriteFile(path, []byte(sql), 0600); err != nil {
					t.Fatal(err)
				}
			}
			conf := config.SQL{Engine: config.EnginePostgreSQL, Gen: config.SQLGen{Go: &goopts.Options{SqlPackage: "wpgx"}}}
			c, err := NewCompiler(conf, config.Combine(config.Config{Version: "2"}, conf), opts.Parser{})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { c.Close(context.Background()) })
			err = c.ParseCatalog([]string{primary, dependency})
			if tc.invalid {
				if err == nil {
					t.Fatal("expected invalid primary layout")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.wicked.PrimaryRelation.Name != tc.want || c.wicked.PrimaryRelation.Schema != "public" {
				t.Fatalf("primary = %+v, want public.%s", c.wicked.PrimaryRelation, tc.want)
			}
			if c.wicked.PrimarySchemaSQL != tc.primary || c.wicked.PrimarySchemaPath != primary {
				t.Fatal("primary schema source was changed or replaced by a dependency")
			}
		})
	}
}

func TestStandardSchemasKeepUpstreamSemantics(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "schema.sql")
	if err := os.WriteFile(file, []byte("CREATE TABLE a (id bigint); CREATE TABLE b (id bigint);"), 0600); err != nil {
		t.Fatal(err)
	}
	conf := config.SQL{Engine: config.EnginePostgreSQL}
	c, err := NewCompiler(conf, config.Combine(config.Config{Version: "2"}, conf), opts.Parser{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	if err := c.ParseCatalog([]string{file}); err != nil {
		t.Fatal(err)
	}
	if c.wicked != nil {
		t.Fatal("standard schema acquired wicked metadata")
	}
}

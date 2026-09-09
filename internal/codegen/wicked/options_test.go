package wicked

import (
	"context"
	"testing"
	"time"

	"github.com/sqlc-dev/sqlc/internal/plugin"
)

func TestLegacyOptionPrecedence(t *testing.T) {
	req := &plugin.GenerateRequest{
		Catalog:       &plugin.Catalog{DefaultSchema: "custom"},
		PluginOptions: []byte(`{"package":"books","sql_package":"wpgx","rename":{"id":"LocalID"},"overrides":[{"column":"books.id","go_type":"int32"}]}`),
		GlobalOptions: []byte(`{"rename":{"id":"GlobalID","metadata":"Meta"},"overrides":[{"column":"books.id","go_type":"int64"}]}`),
	}
	options, err := parseOptions(req)
	if err != nil {
		t.Fatal(err)
	}
	if options.Rename["id"] != "LocalID" || options.Rename["metadata"] != "Meta" {
		t.Fatalf("rename precedence: %+v", options.Rename)
	}
	col := &plugin.Column{Name: "id", Table: &plugin.Identifier{Schema: "custom", Name: "books"}, Type: &plugin.Identifier{Name: "int8"}, NotNull: true}
	if got := goType(req, options, col); got != "int64" {
		t.Fatalf("legacy global override precedence: %s", got)
	}
}

func TestQualifiedBuiltinOverride(t *testing.T) {
	req := &plugin.GenerateRequest{
		Catalog:       &plugin.Catalog{DefaultSchema: "public"},
		PluginOptions: []byte(`{"package":"books","sql_package":"wpgx","overrides":[{"db_type":"jsonb","nullable":true,"go_type":{"type":"byte","slice":true}}]}`),
	}
	options, err := parseOptions(req)
	if err != nil {
		t.Fatal(err)
	}
	col := &plugin.Column{Type: &plugin.Identifier{Schema: "pg_catalog", Name: "jsonb"}}
	if got := goType(req, options, col); got != "[]byte" {
		t.Fatalf("qualified builtin override: %s", got)
	}
}

func TestLegacyStructTagScope(t *testing.T) {
	req := &plugin.GenerateRequest{
		Catalog: &plugin.Catalog{DefaultSchema: "public"},
		PluginOptions: []byte(`{"package":"books","sql_package":"wpgx","overrides":[
			{"db_type":"pg_catalog.int8","go_struct_tag":"json:\"changed_id\""},
			{"column":"books.title","go_struct_tag":"json:\"display_title\""}
		]}`),
	}
	options, err := parseOptions(req)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ column, typ, want string }{
		{"id", "int8", "id"},
		{"title", "text", "display_title"},
	} {
		tags := map[string]string{"json": tc.column}
		col := &plugin.Column{Name: tc.column, Table: &plugin.Identifier{Schema: "public", Name: "books"}, Type: &plugin.Identifier{Schema: "pg_catalog", Name: tc.typ}, NotNull: true}
		addExtraGoStructTags(tags, req, options, col)
		if tags["json"] != tc.want {
			t.Errorf("%s JSON tag = %q, want %q", tc.column, tags["json"], tc.want)
		}
	}
}

func TestMissingGenerateRequest(t *testing.T) {
	if _, err := Generate(context.Background(), nil); err == nil {
		t.Fatal("expected missing request error")
	}
}

func TestQueryOptions(t *testing.T) {
	names := map[string]bool{"GetBook": true}
	opts, err := parseQueryOptions([]string{"documentation", " -- timeout: 2s", " -- cache: 1m", " -- cache: 3m"}, true, names)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Timeout != 2*time.Second || opts.Cache != 3*time.Minute || !opts.AllowReplica || !opts.CountIntent {
		t.Fatalf("options = %+v", opts)
	}
	opts, err = parseQueryOptions([]string{" -- timeout: 1s", " -- invalidate: [GetBook]"}, false, names)
	if err != nil {
		t.Fatal(err)
	}
	if opts.AllowReplica || opts.CountIntent || len(opts.Invalidates) != 1 || opts.Invalidates[0] != "GetBook" {
		t.Fatalf("mutation options = %+v", opts)
	}
}

func TestInvalidQueryOptions(t *testing.T) {
	for _, tc := range []struct {
		comment    string
		selectStmt bool
	}{
		{" -- typo: 1s", true},
		{" -- timeout: 0", true},
		{" -- timeout: 1us", true},
		{" -- cache: invalid", true},
		{" -- timeout", true},
		{" -- invalidate: [Missing]", false},
		{" -- invalidate: [GetBook]", true},
		{" -- allow_replica: true", false},
		{" -- count_intent: yes", true},
	} {
		t.Run(tc.comment, func(t *testing.T) {
			if _, err := parseQueryOptions([]string{tc.comment}, tc.selectStmt, map[string]bool{"GetBook": true}); err == nil {
				t.Fatal("expected an option error")
			}
		})
	}
}

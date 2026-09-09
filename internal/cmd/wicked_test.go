package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	goast "go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	goopts "github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/codegen/wicked"
	"github.com/sqlc-dev/sqlc/internal/compiler"
	"github.com/sqlc-dev/sqlc/internal/config"
	"github.com/sqlc-dev/sqlc/internal/opts"
	"github.com/sqlc-dev/sqlc/internal/plugin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestWickedGenerationContract(t *testing.T) {
	dir := writeWickedFixture(t, "-- -- timeout: 500ms\n-- -- cache: 1m\n")
	var stderr bytes.Buffer
	output, err := Generate(context.Background(), dir, "", &Options{Env: Env{NoRemote: true}, Stderr: &stderr})
	if err != nil {
		t.Fatalf("generate: %v: %s", err, &stderr)
	}
	methods := map[string]bool{}
	modelFields := map[string]string{}
	for name, source := range output {
		file, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			if fn, ok := decl.(*goast.FuncDecl); ok && fn.Recv != nil {
				star := fn.Recv.List[0].Type.(*goast.StarExpr)
				receiver := star.X.(*goast.Ident).Name
				methods[receiver+"."+fn.Name.Name] = true
			}
			if gen, ok := decl.(*goast.GenDecl); ok {
				for _, spec := range gen.Specs {
					ts, ok := spec.(*goast.TypeSpec)
					if !ok || ts.Name.Name != "Book" {
						continue
					}
					st := ts.Type.(*goast.StructType)
					for _, field := range st.Fields.List {
						if sel, ok := field.Type.(*goast.SelectorExpr); ok {
							modelFields[field.Names[0].Name] = sel.X.(*goast.Ident).Name + "." + sel.Sel.Name
						}
					}
				}
			}
		}
	}
	for _, method := range []string{"Queries.GetBook", "ReadOnlyQueries.GetBook", "Queries.CreateBook", "Queries.Load", "Queries.Dump", "Queries.WithTx", "Queries.UseReplica"} {
		if !methods[method] {
			t.Errorf("missing method %s", method)
		}
	}
	if methods["ReadOnlyQueries.CreateBook"] {
		t.Fatal("INSERT RETURNING was exposed on ReadOnlyQueries")
	}
	if modelFields["Metadata"] != "json.RawMessage" {
		t.Fatalf("metadata type = %q", modelFields["Metadata"])
	}
}

func TestWickedMissingTimeout(t *testing.T) {
	dir := writeWickedFixture(t, "")
	var stderr bytes.Buffer
	_, err := Generate(context.Background(), dir, "", &Options{Env: Env{NoRemote: true}, Stderr: &stderr})
	if err == nil || !strings.Contains(stderr.String(), "GetBook does not have a timeout") {
		t.Fatalf("error=%v stderr=%s", err, &stderr)
	}
}

func TestWickedProtocolFacts(t *testing.T) {
	dir := writeWickedFixture(t, "-- -- timeout: 500ms\n")
	conf := config.SQL{
		Engine: config.EnginePostgreSQL, Schema: []string{filepath.Join(dir, "schema.sql")}, Queries: []string{filepath.Join(dir, "query.sql")},
		Gen: config.SQLGen{Go: &goopts.Options{SqlPackage: "wpgx", Package: "books", Out: "books"}},
	}
	combined := config.Combine(config.Config{Version: "2"}, conf)
	c, err := compiler.NewCompiler(conf, combined, opts.Parser{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close(context.Background()) })
	if err := c.ParseCatalog(conf.Schema); err != nil {
		t.Fatal(err)
	}
	if err := c.ParseQueries(conf.Queries, opts.Parser{}); err != nil {
		t.Fatal(err)
	}
	req := codeGenRequest(c.Result(), combined)
	facts := req.GetWicked()
	if facts == nil || facts.PrimaryRelation.Name != "books" || !facts.QueryIsSelect["GetBook"] {
		t.Fatalf("facts=%+v", facts)
	}
	if v, present := facts.QueryIsSelect["CreateBook"]; !present || v {
		t.Fatal("INSERT RETURNING is not a SELECT")
	}
	blob, err := proto.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip plugin.GenerateRequest
	if err := proto.Unmarshal(blob, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(req, &roundtrip) {
		t.Fatal("metadata lost in protobuf transport")
	}
	req.PluginOptions, err = json.Marshal(conf.Gen.Go)
	if err != nil {
		t.Fatal(err)
	}
	before := proto.Clone(req)
	if _, err := wicked.Generate(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(before, req) {
		t.Fatal("generator modified its input request")
	}
	standard := proto.Clone(req).(*plugin.GenerateRequest)
	standard.BackendMetadata = nil
	blob, err = (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(standard)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(blob, &fields); err != nil {
		t.Fatal(err)
	}
	if _, found := fields["wicked"]; found {
		t.Fatal("standard request JSON contains wicked metadata")
	}
}

func writeWickedFixture(t *testing.T, options string) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"sqlc.yaml":  "version: '2'\nsql:\n- schema: schema.sql\n  queries: query.sql\n  engine: postgresql\n  gen:\n    go:\n      sql_package: wpgx\n      package: books\n      out: books\n",
		"schema.sql": "CREATE TABLE books (id bigint NOT NULL, metadata jsonb);",
		"query.sql":  "-- name: GetBook :one\n" + options + "SELECT * FROM books WHERE id = @id;\n\n-- name: CreateBook :one\n-- -- timeout: 500ms\nINSERT INTO books (id,metadata) VALUES (@id,@metadata) RETURNING *;\n",
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

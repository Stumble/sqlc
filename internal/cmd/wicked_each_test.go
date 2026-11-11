package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func generateEachFixture(t *testing.T, query, settings string) (map[string]string, string, error) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"schema.sql": "CREATE TABLE books (id bigint PRIMARY KEY, value bytea NOT NULL);",
		"query.sql":  query,
		"sqlc.yaml":  "version: '2'\nsql:\n- schema: schema.sql\n  queries: query.sql\n  engine: postgresql\n  gen:\n    go:\n      sql_package: wpgx\n      package: books\n      out: books\n" + settings,
	}
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var stderr bytes.Buffer
	output, err := Generate(context.Background(), dir, "", &Options{Env: Env{NoRemote: true}, Stderr: &stderr})
	return output, stderr.String(), err
}

func TestWickedEachGeneration(t *testing.T) {
	query := "-- name: ListBooks :many\n-- -- timeout: 1s\n-- -- each: true\nSELECT id, value FROM books WHERE id >= @visit ORDER BY id;\n"
	output, stderr, err := generateEachFixture(t, query, "      emit_interface: true\n")
	if err != nil {
		t.Fatalf("generate: %v: %s", err, stderr)
	}
	var source, iface string
	for name, value := range output {
		switch filepath.Base(name) {
		case "query.sql.go":
			source = value
		case "querier.go":
			iface = value
		}
	}
	for _, want := range []string{"func (q *Queries) ListBooksEach", "func (q *ReadOnlyQueries) ListBooksEach", "visit int64, visit1 func(context.Context, Book)"} {
		if !strings.Contains(source, want) {
			t.Errorf("missing %q", want)
		}
	}
	if !strings.Contains(iface, "type EachQuerier interface") {
		t.Fatal("missing separate EachQuerier")
	}
	start := strings.Index(iface, "type Querier interface")
	if start < 0 {
		t.Fatal("missing legacy Querier")
	}
	legacy, _, _ := strings.Cut(strings.TrimPrefix(iface[start:], "type Querier interface"), "\n}")
	if strings.Contains(legacy, "ListBooksEach") {
		t.Fatal("legacy Querier changed")
	}
}

func TestWickedEachValidation(t *testing.T) {
	for _, tc := range []struct{ name, query, want string }{
		{"exec", "-- name: ListBooks :exec\n-- -- timeout: 1s\n-- -- each: true\nSELECT * FROM books;", "each requires a read-only :many"},
		{"empty_value", "-- name: ListBooks :many\n-- -- timeout: 1s\n-- -- each:\nSELECT * FROM books;", "invalid each value"},
		{"one", "-- name: GetBook :one\n-- -- timeout: 1s\n-- -- each: true\nSELECT * FROM books WHERE id = 1;", "each requires a read-only :many"},
		{"mutation", "-- name: PutBook :many\n-- -- timeout: 1s\n-- -- each: true\nINSERT INTO books (id, value) VALUES (1, ''::bytea) RETURNING *;", "each requires a read-only :many"},
		{"cte", "-- name: PutBook :many\n-- -- timeout: 1s\n-- -- each: true\nWITH changed AS (DELETE FROM books RETURNING *) SELECT * FROM changed;", "each requires a read-only :many"},
		{"nested_cte", "-- name: PutBook :many\n-- -- timeout: 1s\n-- -- each: true\nWITH changed AS (INSERT INTO books (id, value) VALUES (1, ''::bytea) RETURNING *) SELECT * FROM (SELECT * FROM changed) AS nested;", "each requires a read-only :many"},
		{"value", "-- name: ListBooks :many\n-- -- timeout: 1s\n-- -- each: yes\nSELECT * FROM books;", "invalid each value"},
		{"helper_constant_collision", "-- name: ListBooks :many\n-- -- timeout: 1s\n-- -- each: true\nSELECT * FROM books;\n-- name: _ListBooksEach :many\n-- -- timeout: 1s\nSELECT * FROM books;", "Each companion name conflicts"},
		{"name_collision", "-- name: ListBooks :many\n-- -- timeout: 1s\n-- -- each: true\nSELECT * FROM books;\n-- name: ListBooksEach :many\n-- -- timeout: 1s\nSELECT * FROM books;", "Each companion name conflicts"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, err := generateEachFixture(t, tc.query, "")
			if err == nil || !strings.Contains(stderr, tc.want) {
				t.Fatalf("err=%v stderr=%s, want %s", err, stderr, tc.want)
			}
		})
	}
}

func TestWickedEachDisabled(t *testing.T) {
	output, stderr, err := generateEachFixture(t, "-- name: ListBooks :many\n-- -- timeout: 1s\n-- -- each: false\nSELECT * FROM books;", "      emit_interface: true\n")
	if err != nil {
		t.Fatalf("generate: %v: %s", err, stderr)
	}
	for _, source := range output {
		if strings.Contains(source, "ListBooksEach(") || strings.Contains(source, "type EachQuerier") {
			t.Fatal("disabled companion was emitted")
		}
	}
}

func TestWickedEachInterfaceNameCollision(t *testing.T) {
	query := "-- name: EachQuerier :many\n-- -- timeout: 1s\n-- -- each: true\nSELECT * FROM books;"
	_, stderr, err := generateEachFixture(t, query, "      emit_interface: true\n      emit_exported_queries: true\n")
	if err == nil || !strings.Contains(stderr, "Each companion name conflicts") {
		t.Fatalf("err=%v stderr=%s", err, stderr)
	}
}

func TestWickedEachModelNameCollision(t *testing.T) {
	query := "-- name: ListBooks :many\n-- -- timeout: 1s\n-- -- each: true\nSELECT * FROM books;"
	_, stderr, err := generateEachFixture(t, query, "      emit_interface: true\n      rename:\n        book: EachQuerier\n")
	if err == nil || !strings.Contains(stderr, "Each companion name conflicts with model") {
		t.Fatalf("err=%v stderr=%s", err, stderr)
	}
}

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The golden was generated before adding Each. Compare bytes so an opt-in
// feature cannot quietly refactor legacy collectors, caching or invalidation.
func TestWickedManyCompatibility(t *testing.T) {
	fixture := filepath.Join("..", "endtoend", "testdata", "wicked_many")
	for _, emitInterface := range []bool{true, false} {
		t.Run(map[bool]string{true: "interface", false: "no_interface"}[emitInterface], func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"sqlc.yaml", "schema.sql", "query.sql"} {
				data, err := os.ReadFile(filepath.Join(fixture, name))
				if err != nil {
					t.Fatal(err)
				}
				if name == "sqlc.yaml" && !emitInterface {
					data = bytes.Replace(data, []byte("emit_interface: true"), []byte("emit_interface: false"), 1)
				}
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			var stderr bytes.Buffer
			output, err := Generate(context.Background(), dir, "", &Options{Env: Env{NoRemote: true}, Stderr: &stderr})
			if err != nil {
				t.Fatalf("generate: %v: %s", err, &stderr)
			}
			entries, err := os.ReadDir(filepath.Join(fixture, "go"))
			if err != nil {
				t.Fatal(err)
			}
			wantCount := 0
			for _, entry := range entries {
				if !strings.HasSuffix(entry.Name(), ".go") || (!emitInterface && entry.Name() == "querier.go") {
					continue
				}
				wantCount++
				want, err := os.ReadFile(filepath.Join(fixture, "go", entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				name := filepath.Join(dir, "go", entry.Name())
				got, exists := output[name]
				if !exists || !bytes.Equal([]byte(got), want) {
					t.Errorf("legacy output changed: %s (exists=%v)", name, exists)
				}
			}
			if len(output) != wantCount {
				t.Errorf("files = %d, want %d", len(output), wantCount)
			}
		})
	}
}

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReleaseVersionMatchesGeneratedCode(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs the release binary")
	}
	dir := t.TempDir()
	binary := filepath.Join(dir, "sqlc")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	const version = "v2.4.0-version-test-wicked-fork"
	build := exec.Command("go", "build", "-ldflags", releaseLDFlags(version), "-o", binary, "./cmd/sqlc")
	build.Dir = ".."
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build release: %v\n%s", err, output)
	}
	output, err := exec.Command(binary, "version").CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != version {
		t.Fatalf("version: %v: %s", err, output)
	}
	for name, contents := range map[string]string{
		"sqlc.yaml":  "version: '2'\nsql:\n- schema: schema.sql\n  queries: query.sql\n  engine: postgresql\n  gen:\n    go:\n      sql_package: wpgx\n      package: records\n      out: generated\n",
		"schema.sql": "CREATE TABLE records (id bigint NOT NULL);",
		"query.sql":  "-- name: GetRecord :one\n-- -- timeout: 1s\nSELECT * FROM records WHERE id = @id;",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
	generate := exec.Command(binary, "generate")
	generate.Dir = dir
	if output, err := generate.CombinedOutput(); err != nil {
		t.Fatalf("generate: %v\n%s", err, output)
	}
	for _, name := range []string{"db.go", "models.go", "query.sql.go"} {
		code, err := os.ReadFile(filepath.Join(dir, "generated", name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(code, []byte("//   sqlc "+version+"\n")) {
			t.Errorf("%s does not report the release version", name)
		}
	}
}

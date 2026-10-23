package main

import (
	"bytes"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestNormalizeExitsPreservesBothControlFlows(t *testing.T) {
	source, err := format.Source([]byte(`package main
func rule(early bool) int {
 value := 11
 if early { value = 7; goto errorExit }
errorExit:
 return value
 goto errorExit // Trick to prevent compiler error if the label is not used
}
func noBranch() int {
errorExit:
 return 5
 goto errorExit // Trick to prevent compiler error if the label is not used
}
func main() {
 if rule(true) != 7 || rule(false) != 11 || noBranch() != 5 { panic("parser exit changed") }
}
`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := normalizeExits(source)
	if err != nil {
		t.Fatal(err)
	}
	again, err := normalizeExits(got)
	if err != nil || !bytes.Equal(got, again) {
		t.Fatalf("normalization is not idempotent: %v", err)
	}
	file := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(file, got, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"run", "vet"} {
		if out, err := exec.Command("go", operation, file).CombinedOutput(); err != nil {
			t.Fatalf("go %s: %v\n%s", operation, err, out)
		}
	}
}

func TestNormalizeExitsRejectsUnexpectedGeneratorShape(t *testing.T) {
	source := []byte("package p\nfunc f() {\notherExit:\nreturn\ngoto errorExit // Trick to prevent compiler error if the label is not used\n}\n")
	if _, err := normalizeExits(source); err == nil {
		t.Fatal("expected mismatched exits to fail rather than silently change generation")
	}
}

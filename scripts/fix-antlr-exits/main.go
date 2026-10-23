// Normalize ANTLR 4.13.1's unused-label workaround without unreachable code.
package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
)

func normalizeExits(source []byte) ([]byte, error) {
	marker := []byte("\tgoto errorExit // Trick to prevent compiler error if the label is not used\n")
	label := []byte("\nerrorExit:\n")
	count := bytes.Count(source, marker)
	if count == 0 {
		if bytes.Contains(source, []byte("Trick to prevent compiler error")) {
			return nil, fmt.Errorf("unexpected ANTLR exit workaround")
		}
		return source, nil
	}
	if count != bytes.Count(source, label) {
		return nil, fmt.Errorf("ANTLR exits: %d workarounds, %d labels", count, bytes.Count(source, label))
	}
	source = bytes.ReplaceAll(source, marker, nil)
	source = bytes.ReplaceAll(source, label, []byte("\n\tgoto errorExit\nerrorExit:\n"))
	return format.Source(source)
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: fix-antlr-exits parser.go")
	}
	path := os.Args[1]
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	normalized, err := normalizeExits(source)
	if err != nil {
		return err
	}
	return os.WriteFile(path, normalized, 0o644)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

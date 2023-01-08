package compiler

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kyleconroy/sqlc/internal/metadata"
	"github.com/kyleconroy/sqlc/internal/migrations"
	"github.com/kyleconroy/sqlc/internal/multierr"
	"github.com/kyleconroy/sqlc/internal/opts"
	"github.com/kyleconroy/sqlc/internal/sql/ast"
	"github.com/kyleconroy/sqlc/internal/sql/sqlerr"
	"github.com/kyleconroy/sqlc/internal/sql/sqlpath"
)

// TODO: Rename this interface Engine
type Parser interface {
	Parse(io.Reader) ([]ast.Statement, error)
	CommentSyntax() metadata.CommentSyntax
	IsReservedKeyword(string) bool
}

// end copypasta
func (c *Compiler) parseCatalog(schemas []string) error {
	files, err := sqlpath.Glob(schemas)
	if err != nil {
		return err
	}
	merr := multierr.New()
	// XXX(yumin): reverse the order of files to process dependencies first.
	orderReversedFiles := reversed(files)
	for i, filename := range orderReversedFiles {
		blob, err := os.ReadFile(filename)
		if err != nil {
			merr.Add(filename, "", 0, err)
			continue
		}
		contents := migrations.RemoveRollbackStatements(string(blob))
		stmts, err := c.parser.Parse(strings.NewReader(contents))
		if err != nil {
			merr.Add(filename, contents, 0, err)
			continue
		}
		tableDefined := false
		for _, stmt := range stmts {
			// XXX(yumin): generate table only when it's the originally the first table
			// creation of the first file in the first schema array.
			if err := c.catalog.Update(
				stmt, c, !tableDefined && i == len(orderReversedFiles)-1); err != nil {
				merr.Add(filename, contents, stmt.Pos(), err)
				continue
			}
			definingTable := c.catalog.IsCreatingNewTableLayout(stmt)
			if tableDefined && definingTable {
				merr.Add(filename, contents, stmt.Pos(),
					fmt.Errorf("only one table creation is allowed per schema.sql file"))
			}
			tableDefined = tableDefined || definingTable
		}
		// XXX(yumin): only the first schema file in the original order is added.
		if i == len(orderReversedFiles)-1 {
			c.catalog.AddRawSQL(contents)
		}
	}
	if len(merr.Errs()) > 0 {
		return merr
	}
	return nil
}

func (c *Compiler) parseQueries(o opts.Parser) (*Result, error) {
	var q []*Query
	merr := multierr.New()
	set := map[string]struct{}{}
	files, err := sqlpath.Glob(c.conf.Queries)
	if err != nil {
		return nil, err
	}
	for _, filename := range files {
		blob, err := os.ReadFile(filename)
		if err != nil {
			merr.Add(filename, "", 0, err)
			continue
		}
		src := string(blob)
		stmts, err := c.parser.Parse(strings.NewReader(src))
		if err != nil {
			merr.Add(filename, src, 0, err)
			continue
		}
		for _, stmt := range stmts {
			query, err := c.parseQuery(stmt.Raw, src, o)
			if err == ErrUnsupportedStatementType {
				continue
			}
			if err != nil {
				var e *sqlerr.Error
				loc := stmt.Raw.Pos()
				if errors.As(err, &e) && e.Location != 0 {
					loc = e.Location
				}
				merr.Add(filename, src, loc, err)
				continue
			}
			if query.Name != "" {
				if _, exists := set[query.Name]; exists {
					merr.Add(filename, src, stmt.Raw.Pos(), fmt.Errorf("duplicate query name: %s", query.Name))
					continue
				}
				set[query.Name] = struct{}{}
			}
			query.Filename = filepath.Base(filename)
			if query != nil {
				q = append(q, query)
			}
		}
	}
	if len(merr.Errs()) > 0 {
		return nil, merr
	}
	if len(q) == 0 {
		return nil, fmt.Errorf("no queries contained in paths %s", strings.Join(c.conf.Queries, ","))
	}
	return &Result{
		Catalog: c.catalog,
		Queries: q,
	}, nil
}

func reversed[V any](arr []V) []V {
	rv := make([]V, len(arr))
	for i := range arr {
		rv[len(arr)-1-i] = arr[i]
	}
	return rv
}

package compiler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sqlc-dev/sqlc/internal/migrations"
	"github.com/sqlc-dev/sqlc/internal/multierr"
	"github.com/sqlc-dev/sqlc/internal/opts"
	"github.com/sqlc-dev/sqlc/internal/rpc"
	"github.com/sqlc-dev/sqlc/internal/source"
	"github.com/sqlc-dev/sqlc/internal/sql/ast"
	"github.com/sqlc-dev/sqlc/internal/sql/sqlerr"
	"github.com/sqlc-dev/sqlc/internal/sql/sqlpath"
)

// TODO: Rename this interface Engine
type Parser interface {
	Parse(io.Reader) ([]ast.Statement, error)
	CommentSyntax() source.CommentSyntax
	IsReservedKeyword(string) bool
}

func (c *Compiler) parseCatalog(schemas []string) error {
	files, err := sqlpath.Glob(schemas)
	if err != nil {
		return err
	}
	if c.conf.IsWicked() {
		if c.databaseOnlyMode {
			return fmt.Errorf("wicked requires schema-based analysis to identify the primary model")
		}
		if len(files) == 0 {
			return fmt.Errorf("wicked requires a primary schema file")
		}
		c.wicked = &WickedSchema{PrimarySchemaPath: files[0]}
		slices.Reverse(files)
	}
	merr := multierr.New()
	for _, filename := range files {
		blob, err := os.ReadFile(filename)
		if err != nil {
			merr.Add(filename, "", 0, err)
			continue
		}
		contents := migrations.RemoveRollbackStatements(string(blob))
		if c.wicked != nil && filename == c.wicked.PrimarySchemaPath {
			c.wicked.PrimarySchemaSQL = contents
		}
		contents = migrations.RemovePsqlMetaCommands(contents)
		c.schema = append(c.schema, contents)

		// In database-only mode, we parse the schema to validate syntax
		// but don't update the catalog - the database will be the source of truth
		stmts, err := c.parser.Parse(strings.NewReader(contents))
		if err != nil {
			merr.Add(filename, contents, 0, err)
			continue
		}

		// Skip catalog updates in database-only mode
		if c.databaseOnlyMode {
			continue
		}

		layouts := 0
		for i := range stmts {
			if err := c.catalog.Update(stmts[i], c); err != nil {
				merr.Add(filename, contents, stmts[i].Pos(), err)
				continue
			}
			if c.wicked != nil {
				if rel := wickedLayout(stmts[i]); rel != nil {
					layouts++
					if layouts > 1 {
						merr.Add(filename, contents, stmts[i].Pos(), fmt.Errorf("only one table creation is allowed per schema.sql file"))
					}
					if filename == c.wicked.PrimarySchemaPath && layouts == 1 {
						table, err := c.catalog.GetTable(rel)
						if err != nil {
							merr.Add(filename, contents, stmts[i].Pos(), err)
							continue
						}
						// Track catalog identity through subsequent ALTER/RENAME.
						c.wicked.PrimaryRelation = table.Rel
					}
				}
			}
		}
	}
	if len(merr.Errs()) > 0 {
		return merr
	}
	if c.wicked != nil && c.wicked.PrimaryRelation == nil {
		return fmt.Errorf("wicked: primary schema %q has no supported table layout", c.wicked.PrimarySchemaPath)
	}
	if c.wicked != nil {
		identity := *c.wicked.PrimaryRelation
		if identity.Schema == "" {
			identity.Schema = c.catalog.DefaultSchema
		}
		c.wicked.PrimaryRelation = &identity
	}
	return nil
}

func (c *Compiler) parseQueries(o opts.Parser) (*Result, error) {
	ctx := context.Background()

	// In database-only mode, initialize the database connection before parsing queries
	if c.databaseOnlyMode && c.analyzer != nil {
		if err := c.analyzer.EnsureConn(ctx, c.schema); err != nil {
			return nil, fmt.Errorf("failed to initialize database connection: %w", err)
		}
	}

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
			if err != nil {
				var e *sqlerr.Error
				loc := stmt.Raw.Pos()
				if errors.As(err, &e) && e.Location != 0 {
					loc = e.Location
				}
				merr.Add(filename, src, loc, err)
				// If this rpc unauthenticated error bubbles up, then all future parsing/analysis will fail
				if errors.Is(err, rpc.ErrUnauthenticated) {
					return nil, merr
				}
				continue
			}
			if query == nil {
				continue
			}
			query.Metadata.Filename = filepath.Base(filename)
			queryName := query.Metadata.Name
			if queryName != "" {
				if _, exists := set[queryName]; exists {
					merr.Add(filename, src, stmt.Raw.Pos(), fmt.Errorf("duplicate query name: %s", queryName))
					continue
				}
				set[queryName] = struct{}{}
			}
			q = append(q, query)
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
		Wicked:  c.wicked,
	}, nil
}

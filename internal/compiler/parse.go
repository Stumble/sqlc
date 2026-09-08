package compiler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/sqlc-dev/sqlc/internal/debug"
	"github.com/sqlc-dev/sqlc/internal/metadata"
	"github.com/sqlc-dev/sqlc/internal/opts"
	"github.com/sqlc-dev/sqlc/internal/source"
	"github.com/sqlc-dev/sqlc/internal/sql/ast"
	"github.com/sqlc-dev/sqlc/internal/sql/astutils"
	"github.com/sqlc-dev/sqlc/internal/sql/validate"
)

func (c *Compiler) parseQuery(stmt ast.Node, src string, o opts.Parser) (*Query, error) {
	ctx := context.Background()

	if o.Debug.DumpAST {
		debug.Dump(stmt)
	}

	// validate sqlc-specific syntax
	if err := validate.SqlcFunctions(stmt); err != nil {
		return nil, err
	}

	// rewrite queries to remove sqlc.* functions

	raw, ok := stmt.(*ast.RawStmt)
	if !ok {
		return nil, errors.New("node is not a statement")
	}
	rawSQL, err := source.Pluck(src, raw.StmtLocation, raw.StmtLen)
	if err != nil {
		return nil, err
	}
	if rawSQL == "" {
		return nil, errors.New("missing semicolon at end of file")
	}

	name, cmd, err := metadata.ParseQueryNameAndType(rawSQL, metadata.CommentSyntax(c.parser.CommentSyntax()))
	if err != nil {
		return nil, err
	}

	if name == "" {
		return nil, nil
	}

	if err := validate.Cmd(raw.Stmt, name, cmd); err != nil {
		return nil, err
	}

	md := metadata.Metadata{
		Name: name,
		Cmd:  cmd,
	}

	// TODO eventually can use this for name and type/cmd parsing too
	cleanedComments, err := source.CleanedComments(rawSQL, c.parser.CommentSyntax())
	if err != nil {
		return nil, err
	}

	md.Params, md.Flags, md.RuleSkiplist, err = metadata.ParseCommentFlags(cleanedComments)
	if err != nil {
		return nil, err
	}

	var anlys *analysis
	if c.databaseOnlyMode && c.expander != nil {
		// In database-only mode, use the expander for star expansion
		// and rely entirely on the database analyzer for type resolution
		expandedQuery, err := c.expander.Expand(ctx, rawSQL)
		if err != nil {
			return nil, fmt.Errorf("star expansion failed: %w", err)
		}

		// Parse named parameters from the expanded query
		expandedStmts, err := c.parser.Parse(strings.NewReader(expandedQuery))
		if err != nil {
			return nil, fmt.Errorf("parsing expanded query failed: %w", err)
		}
		if len(expandedStmts) == 0 {
			return nil, errors.New("no statements in expanded query")
		}
		expandedRaw := expandedStmts[0].Raw

		// Use the analyzer to get type information from the database
		result, err := c.analyzer.Analyze(ctx, expandedRaw, expandedQuery, c.schema, nil)
		if err != nil {
			return nil, err
		}

		// Convert the analyzer result to the internal analysis format
		var cols []*Column
		for _, col := range result.Columns {
			cols = append(cols, convertColumn(col))
		}
		var params []Parameter
		for _, p := range result.Params {
			params = append(params, Parameter{
				Number: int(p.Number),
				Column: convertColumn(p.Column),
			})
		}

		// Determine the insert table if applicable
		var table *ast.TableName
		if insert, ok := expandedRaw.Stmt.(*ast.InsertStmt); ok {
			table, _ = ParseTableName(insert.Relation)
		}

		anlys = &analysis{
			Table:      table,
			Columns:    cols,
			Parameters: params,
			Query:      expandedQuery,
		}
	} else if c.analyzer != nil {
		inference, _ := c.inferQuery(raw, rawSQL)
		if inference == nil {
			inference = &analysis{}
		}
		if inference.Query == "" {
			inference.Query = rawSQL
		}

		result, err := c.analyzer.Analyze(ctx, raw, inference.Query, c.schema, inference.Named)
		if err != nil {
			return nil, err
		}

		// If the query uses star expansion, verify that it was edited. If not,
		// return an error.
		stars := astutils.Search(raw, func(node ast.Node) bool {
			_, ok := node.(*ast.A_Star)
			return ok
		})
		hasStars := len(stars.Items) > 0
		unchanged := inference.Query == rawSQL
		if unchanged && hasStars {
			return nil, fmt.Errorf("star expansion failed for query")
		}

		// FOOTGUN: combineAnalysis mutates inference
		anlys = combineAnalysis(inference, result)
	} else {
		anlys, err = c.analyzeQuery(raw, rawSQL)
		if err != nil {
			return nil, err
		}
	}

	expanded := anlys.Query

	// If the query string was edited, make sure the syntax is valid
	if expanded != rawSQL {
		if _, err := c.parser.Parse(strings.NewReader(expanded)); err != nil {
			return nil, fmt.Errorf("edited query syntax is invalid: %w", err)
		}
	}

	trimmed, comments, err := source.StripComments(expanded)
	if err != nil {
		return nil, err
	}

	md.Comments = comments

	return &Query{
		RawStmt:         raw,
		Metadata:        md,
		Params:          anlys.Parameters,
		Columns:         anlys.Columns,
		SQL:             trimmed,
		InsertIntoTable: anlys.Table,
	}, nil
}

func rangeVars(root ast.Node) []*ast.RangeVar {
	var vars []*ast.RangeVar
	find := astutils.VisitorFunc(func(node ast.Node) {
		switch n := node.(type) {
		case *ast.RangeVar:
			vars = append(vars, n)
		}
	})
	astutils.Walk(find, root)
	return vars
}

// scoreParamRefForTypeInference scores a parameter reference based on how good
// its context is for type inference. Higher scores indicate better contexts.
func scoreParamRefForTypeInference(ref paramRef) int {
	if ref.parent == nil {
		return 0 // No context
	}

	switch parent := ref.parent.(type) {
	case *ast.TypeCast:
		// Explicit type cast - excellent for type inference
		return 100

	case *ast.A_Expr:
		// Expression context - quality depends on the operator
		if parent.Name != nil && len(parent.Name.Items) > 0 {
			if nameStr, ok := parent.Name.Items[0].(*ast.String); ok {
				switch nameStr.Str {
				case "=", "==", "!=", "<>", "<", "<=", ">", ">=":
					// Comparison operations - very good for type inference
					return 100
				case "+", "-", "*", "/", "%":
					// Mathematical operations - good for type inference
					return 90
				case "||":
					// String concatenation - good for type inference
					return 90
				case "~~", "!~~", "~~*", "!~~*":
					// LIKE operations - good for type inference
					return 90
				case "IS", "IS NOT":
					// IS NULL/IS NOT NULL - poor for type inference
					return 0
				default:
					return 50
				}
			}
		}
		return 50 // Default for A_Expr without clear operator

	case *ast.BoolExpr:
		// Boolean expressions
		switch parent.Boolop {
		case ast.BoolExprTypeAnd, ast.BoolExprTypeOr:
			// Logical operations - still useful but lower priority
			return 60
		case ast.BoolExprTypeIsNull, ast.BoolExprTypeIsNotNull:
			// IS NULL/IS NOT NULL - poor for type inference
			return 20
		case ast.BoolExprTypeNot:
			// NOT operations - moderate for type inference
			return 50
		default:
			return 40
		}

	case *ast.BetweenExpr:
		// BETWEEN expressions - good for type inference
		return 75

	case *ast.FuncCall:
		// Function call context - depends on function, generally moderate
		// sqlc.narg() and similar functions have poor type inference context
		if parent.Funcname != nil && len(parent.Funcname.Items) > 0 {
			if nameStr, ok := parent.Funcname.Items[0].(*ast.String); ok {
				if nameStr.Str == "sqlc.narg" || nameStr.Str == "sqlc.arg" {
					// sqlc parameter functions in isolation - poor for type inference
					return 30
				}
			}
		}
		return 40

	case *ast.ResTarget:
		// Preserve the existing preference for comparisons over assignments.
		// A nullable assignment column does not by itself make every use of
		// the parameter nullable; explicit sqlc.narg still takes precedence.
		return 60

	case *ast.In:
		// IN expression - good for type inference
		return 70

	case *limitCount, *limitOffset:
		// LIMIT/OFFSET - known to be integer, good for type inference
		return 90

	default:
		// Unknown context - assign low score
		return 10
	}
}

func uniqueParamRefs(in []paramRef, dollar bool) []paramRef {
	positions := make(map[int]int, len(in))
	out := make([]paramRef, 0, len(in))
	for _, ref := range in {
		if ref.ref.Number == 0 {
			continue
		}
		if index, ok := positions[ref.ref.Number]; ok {
			if scoreParamRefForTypeInference(ref) > scoreParamRefForTypeInference(out[index]) {
				out[index] = ref
			}
		} else {
			positions[ref.ref.Number] = len(out)
			out = append(out, ref)
		}
	}
	if !dollar {
		next := 1
		for _, ref := range in {
			if ref.ref.Number != 0 {
				continue
			}
			for {
				if _, used := positions[next]; !used {
					break
				}
				next++
			}
			ref.ref.Number = next
			positions[next] = len(out)
			out = append(out, ref)
		}
	}
	return out
}

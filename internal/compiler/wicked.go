package compiler

import "github.com/sqlc-dev/sqlc/internal/sql/ast"

// WickedSchema records source ownership without adding generation policy to
// the shared SQL catalog. Dependency tables remain available for type analysis.
type WickedSchema struct {
	PrimarySchemaPath string
	PrimarySchemaSQL  string
	PrimaryRelation   *ast.TableName
}

// Model ownership and layout counting are separate legacy conventions. Tables
// with inherited columns (or columns added by ALTER) can own the main model
// even though the CREATE statement does not declare a new column layout.
func wickedRelation(stmt ast.Statement) *ast.TableName {
	if stmt.Raw == nil {
		return nil
	}
	switch n := stmt.Raw.Stmt.(type) {
	case *ast.CreateTableStmt:
		return n.Name
	case *ast.CreateTableAsStmt:
		if n.Into != nil && n.Into.Rel != nil {
			rel, err := ParseTableName(n.Into.Rel)
			if err == nil {
				return rel
			}
		}
	}
	return nil
}

// Physical partitions do not introduce another logical layout. Ordinary views
// were not marked GenerateModel by the legacy catalog and remain unsupported.
func wickedCreatesLayout(stmt ast.Statement) bool {
	if stmt.Raw == nil {
		return false
	}
	switch n := stmt.Raw.Stmt.(type) {
	case *ast.CreateTableStmt:
		return len(n.Cols) > 0
	case *ast.CreateTableAsStmt:
		return true
	}
	return false
}

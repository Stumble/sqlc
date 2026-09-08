package compiler

import "github.com/sqlc-dev/sqlc/internal/sql/ast"

// WickedSchema records source ownership without adding generation policy to
// the shared SQL catalog. Dependency tables remain available for type analysis.
type WickedSchema struct {
	PrimarySchemaPath string
	PrimarySchemaSQL  string
	PrimaryRelation   *ast.TableName
}

// Keep the established definition of a logical layout: CREATE TABLE with
// columns, or CREATE TABLE/MATERIALIZED VIEW AS. Physical partitions do not
// introduce another layout. Ordinary CREATE VIEW was not a supported primary
// model in the legacy fork.
func wickedLayout(stmt ast.Statement) *ast.TableName {
	if stmt.Raw == nil {
		return nil
	}
	switch n := stmt.Raw.Stmt.(type) {
	case *ast.CreateTableStmt:
		if len(n.Cols) > 0 {
			return n.Name
		}
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

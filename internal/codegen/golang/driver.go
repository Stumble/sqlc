package golang

type SQLDriver string

const (
	SQLPackagePGXV4    string = "pgx/v4"
	SQLPackagePGXV5    string = "pgx/v5"
	SQLPackageStandard string = "database/sql"

	// added by the wicked fork.
	SQLPackageWPGX     string = "wpgx"
)

const (
	SQLDriverPGXV4            SQLDriver = "github.com/jackc/pgx/v4"
	SQLDriverPGXV5                      = "github.com/jackc/pgx/v5"
	SQLDriverLibPQ                      = "github.com/lib/pq"
	SQLDriverGoSQLDriverMySQL           = "github.com/go-sql-driver/mysql"
	SQLDriverWPGX                       = "github.com/stumble/wpgx"
)

func parseDriver(sqlPackage string) SQLDriver {
	switch sqlPackage {
	case SQLPackagePGXV4:
		return SQLDriverPGXV4
	case SQLPackagePGXV5:
		return SQLDriverPGXV5
	case SQLPackageWPGX:
		return SQLDriverWPGX
	default:
		return SQLDriverLibPQ
	}
}

func (d SQLDriver) IsWPGX() bool {
	return d == SQLDriverWPGX
}

func (d SQLDriver) IsPGX() bool {
	return d == SQLDriverPGXV4 || d == SQLDriverPGXV5 || d == SQLDriverWPGX
}

func (d SQLDriver) IsGoSQLDriverMySQL() bool {
	return d == SQLDriverGoSQLDriverMySQL
}

func (d SQLDriver) Package() string {
	switch d {
	case SQLDriverPGXV4:
		return SQLPackagePGXV4
	case SQLDriverPGXV5:
		return SQLPackagePGXV5
	case SQLDriverWPGX:
		return SQLPackageWPGX
	default:
		return SQLPackageStandard
	}
}

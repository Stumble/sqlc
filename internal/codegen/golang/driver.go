package golang

type SQLDriver int

const (
	SQLPackagePGXV4    string = "pgx/v4"
	SQLPackagePGXV5    string = "pgx/v5"
	SQLPackageStandard string = "database/sql"

	// added by the wicked fork.
	SQLPackageWPGX     string = "wpgx"
)

const (
	SQLDriverPGXV4 SQLDriver = iota
	SQLDriverPGXV5
	SQLDriverLibPQ

	SQLDriverWPGX
)

const SQLDriverGoSQLDriverMySQL = "github.com/go-sql-driver/mysql"

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

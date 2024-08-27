package golang

import (
	"fmt"
	"strings"

	"github.com/sqlc-dev/sqlc/internal/metadata"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

type QueryValue struct {
	Emit        bool
	EmitPointer bool
	Name        string
	DBName      string // The name of the field in the database. Only set if Struct==nil.
	Struct      *Struct
	Typ         string
	SQLDriver   SQLDriver

	// Column is kept so late in the generation process around to differentiate
	// between mysql slices and pg arrays
	Column *plugin.Column
}

func (v QueryValue) EmitStruct() bool {
	return v.Emit
}

func (v QueryValue) IsStruct() bool {
	return v.Struct != nil
}

func (v QueryValue) IsPointer() bool {
	return v.EmitPointer && v.Struct != nil
}

func (v QueryValue) isEmpty() bool {
	return v.Typ == "" && v.Name == "" && v.Struct == nil
}

type Argument struct {
	Name string
	Type string
}

func (v QueryValue) Pair() string {
	var out []string
	for _, arg := range v.Pairs() {
		out = append(out, arg.Name+" "+arg.Type)
	}
	return strings.TrimRight(strings.Join(out, ","), ",")
}

// Return the argument name and type for query methods. Should only be used in
// the context of method arguments.
func (v QueryValue) Pairs() []Argument {
	if v.isEmpty() {
		return nil
	}
	if !v.EmitStruct() && v.IsStruct() {
		var out []Argument
		for _, f := range v.Struct.Fields {
			out = append(out, Argument{
				Name: toLowerCase(f.Name),
				Type: f.Type,
			})
		}
		return out
	}
	return []Argument{
		{
			Name: v.Name,
			Type: v.DefineType(),
		},
	}
}

func (v QueryValue) SlicePair() string {
	if v.isEmpty() {
		return ""
	}
	return v.Name + " []" + v.DefineType()
}

func (v QueryValue) Type() string {
	if v.Typ != "" {
		return v.Typ
	}
	if v.Struct != nil {
		return v.Struct.Name
	}
	panic("no type for QueryValue: " + v.Name)
}

func (v QueryValue) IsTypePointer() bool {
	return !v.isEmpty() && strings.HasPrefix(v.Type(), "*")
}

func (v *QueryValue) DefineType() string {
	t := v.Type()
	if v.IsPointer() {
		return "*" + t
	}
	return t
}

func (v *QueryValue) ReturnName() string {
	if v.IsPointer() {
		return "&" + v.Name
	}
	return v.Name
}

func (v QueryValue) UniqueFields() []Field {
	seen := map[string]struct{}{}
	fields := make([]Field, 0, len(v.Struct.Fields))

	for _, field := range v.Struct.Fields {
		if _, found := seen[field.Name]; found {
			continue
		}
		seen[field.Name] = struct{}{}
		fields = append(fields, field)
	}

	return fields
}

func (v QueryValue) Params() string {
	if v.isEmpty() {
		return ""
	}
	var out []string
	if v.Struct == nil {
		if !v.Column.IsSqlcSlice && strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array("+v.Name+")")
		} else {
			out = append(out, v.Name)
		}
	} else {
		for _, f := range v.Struct.Fields {
			if !f.HasSqlcSlice() && strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array("+v.VariableForField(f)+")")
			} else {
				out = append(out, v.VariableForField(f))
			}
		}
	}
	if len(out) <= 3 {
		return strings.TrimRight(strings.Join(out, ","), ",")
	}
	out = append(out, "")
	return strings.TrimRight(("\n" + strings.Join(out, ",\n")), ",\n")
}

func (v QueryValue) ColumnNames() []string {
	if v.Struct == nil {
		return []string{v.DBName}
	}
	names := make([]string, len(v.Struct.Fields))
	for i, f := range v.Struct.Fields {
		names[i] = f.DBName
	}
	return names
}

func (v QueryValue) ColumnNamesAsGoSlice() string {
	if v.Struct == nil {
		return fmt.Sprintf("[]string{%q}", v.DBName)
	}
	escapedNames := make([]string, len(v.Struct.Fields))
	for i, f := range v.Struct.Fields {
		escapedNames[i] = fmt.Sprintf("%q", f.DBName)
	}
	return "[]string{" + strings.Join(escapedNames, ", ") + "}"
}

// When true, we have to build the arguments to q.db.QueryContext in addition to
// munging the SQL
func (v QueryValue) HasSqlcSlices() bool {
	if v.Struct == nil {
		return v.Column != nil && v.Column.IsSqlcSlice
	}
	for _, v := range v.Struct.Fields {
		if v.Column.IsSqlcSlice {
			return true
		}
	}
	return false
}

func (v QueryValue) Scan() string {
	var out []string
	if v.Struct == nil {
		if strings.HasPrefix(v.Typ, "[]") && v.Typ != "[]byte" && !v.SQLDriver.IsPGX() {
			out = append(out, "pq.Array(&"+v.Name+")")
		} else {
			out = append(out, v.Name)
		}
	} else {
		for _, f := range v.Struct.Fields {

			// append any embedded fields
			if len(f.EmbedFields) > 0 {
				for _, embed := range f.EmbedFields {
					if strings.HasPrefix(embed.Type, "[]") && embed.Type != "[]byte" && !v.SQLDriver.IsPGX() {
						out = append(out, "pq.Array(&"+v.Name+"."+f.Name+"."+embed.Name+")")
					} else {
						out = append(out, "&"+v.Name+"."+f.Name+"."+embed.Name)
					}
				}
				continue
			}

			if strings.HasPrefix(f.Type, "[]") && f.Type != "[]byte" && !v.SQLDriver.IsPGX() {
				out = append(out, "pq.Array(&"+v.Name+"."+f.Name+")")
			} else {
				out = append(out, "&"+v.Name+"."+f.Name)
			}
		}
	}
	if len(out) <= 3 {
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

// Deprecated: This method does not respect the Emit field set on the
// QueryValue. It's used by the go-sql-driver-mysql/copyfromCopy.tmpl and should
// not be used other places.
func (v QueryValue) CopyFromMySQLFields() []Field {
	// fmt.Printf("%#v\n", v)
	if v.Struct != nil {
		return v.Struct.Fields
	}
	return []Field{
		{
			Name:   v.Name,
			DBName: v.DBName,
			Type:   v.Typ,
		},
	}
}

func (v QueryValue) VariableForField(f Field) string {
	if !v.IsStruct() {
		return v.Name
	}
	if !v.EmitStruct() {
		return toLowerCase(f.Name)
	}
	return v.Name + "." + f.Name
}

// CacheKeySprintf is used by WPgx only.
func (v QueryValue) CacheKeySprintf() string {
	if v.Struct == nil {
		panic(fmt.Errorf("trying to construct sprintf format for non-struct query arg: %+v", v))
	}
	format := make([]string, 0)
	args := make([]string, 0)
	for _, f := range v.Struct.Fields {
		format = append(format, "%+v")
		if strings.HasPrefix(f.Type, "*") {
			args = append(args, wrapPtrStr(v.Name+"."+f.Name))
		} else {
			args = append(args, v.Name+"."+f.Name)
		}
	}
	formatStr := `"` + strings.Join(format, ",") + `"`
	if len(args) <= 3 {
		return formatStr + ", " + strings.Join(args, ",")
	}
	args = append(args, "")
	return formatStr + ",\n" + strings.Join(args, ",\n")
}

// A struct used to generate methods and fields on the Queries struct
type Query struct {
	Cmd          string
	Comments     []string
	Pkg          string
	MethodName   string
	FieldName    string
	ConstantName string
	SQL          string
	SourceName   string
	Ret          QueryValue
	Arg          QueryValue
	Option       WPgxOption
	Invalidates  []InvalidateParam
	// Used for :copyfrom
	Table *plugin.Identifier
}

func (q Query) hasRetType() bool {
	scanned := q.Cmd == metadata.CmdOne || q.Cmd == metadata.CmdMany ||
		q.Cmd == metadata.CmdBatchMany || q.Cmd == metadata.CmdBatchOne
	return scanned && !q.Ret.isEmpty()
}

func (q Query) TableIdentifierAsGoSlice() string {
	escapedNames := make([]string, 0, 3)
	for _, p := range []string{q.Table.Catalog, q.Table.Schema, q.Table.Name} {
		if p != "" {
			escapedNames = append(escapedNames, fmt.Sprintf("%q", p))
		}
	}
	return "[]string{" + strings.Join(escapedNames, ", ") + "}"
}

func (q Query) TableIdentifierForMySQL() string {
	escapedNames := make([]string, 0, 3)
	for _, p := range []string{q.Table.Catalog, q.Table.Schema, q.Table.Name} {
		if p != "" {
			escapedNames = append(escapedNames, fmt.Sprintf("`%s`", p))
		}
	}
	return strings.Join(escapedNames, ".")
}

// CountIntent is used by WPgx only.
func (q Query) CountIntent() bool {
	return q.Option.CountIntent
}

// AllowReplica is used by WPgx only.
func (q Query) AllowReplica() bool {
	return q.Option.AllowReplica
}

// CacheKey is used by WPgx only.
func (q Query) CacheKey() string {
	return genCacheKeyWithArgName(q, q.Arg.Name)
}

// InvalidateArgs is used by WPgx only.
func (q Query) InvalidateArgs() string {
	rv := ""
	// pretty hacky, but works...
	if !q.Arg.isEmpty() {
		rv = ","
	}
	for _, inv := range q.Invalidates {
		if inv.NoArg {
			continue
		}
		t := "*" + inv.Q.Arg.Type()
		rv += fmt.Sprintf("%s %s,", inv.ArgName, t)
	}
	return strings.TrimRight(rv, ",")
}

// InvalidateArgsNames is used by WPgx only.
func (q Query) InvalidateArgsNames() string {
	rv := ""
	// pretty hacky, but works...
	if !q.Arg.isEmpty() {
		rv = ", "
	}
	for _, inv := range q.Invalidates {
		if inv.NoArg {
			continue
		}
		rv += inv.ArgName + ","
	}
	return strings.TrimRight(rv, ",")
}

// UniqueLabel is used by WPgx only.
func (q Query) UniqueLabel() string {
	return fmt.Sprintf("%s.%s", q.Pkg, q.MethodName)
}

// CacheUniqueLabel is used by WPgx only.
func (q Query) CacheUniqueLabel() string {
	return fmt.Sprintf("%s:%s:", q.Pkg, q.MethodName)
}

// ConnType is used by WPgx only.
// Returns the interface type that the query should be called on, either CacheWGConn or CacheQuerierConn.
// NOTE: because we have check the mutually exclusiveness, that invalidates can only happen on
// queries that are not read-only, we can safely assume that if the query does not have invalidates,
// CacheQuerierConn is enough.
func (q Query) ConnType() string {
	if len(q.Invalidates) > 0 {
		return "CacheWGConn"
	} else {
		return "CacheQuerierConn"
	}
}

// IsConnTypeQuerier is used by WPgx only.
// Returns true if the query should be called on CacheQuerierConn.
func (q Query) IsConnTypeQuerier() bool {
	return len(q.Invalidates) == 0
}

func genCacheKeyWithArgName(q Query, argName string) string {
	if len(q.Pkg) == 0 {
		panic("empty pkg name is invalid")
	}
	prefix := q.CacheUniqueLabel()
	if q.Arg.isEmpty() {
		return `"` + prefix + `"`
	}
	// when it's non-struct parameter, generate inline fmt.Sprintf.
	if q.Arg.Struct == nil {
		if q.Arg.IsTypePointer() {
			argName = wrapPtrStr(argName)
		}
		fmtStr := `hashIfLong(fmt.Sprintf("%+v",` + argName + `))`
		return fmt.Sprintf("\"%s\" + %s", prefix, fmtStr)
	} else {
		return argName + `.CacheKey()`
	}
}

func wrapPtrStr(v string) string {
	return fmt.Sprintf("ptrStr(%s)", v)
}

type InvalidateParam struct {
	Q        *Query
	NoArg    bool
	ArgName  string
	CacheKey string
}

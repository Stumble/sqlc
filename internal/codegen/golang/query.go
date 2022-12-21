package golang

import (
	"fmt"
	"strings"

	"github.com/kyleconroy/sqlc/internal/metadata"
	"github.com/kyleconroy/sqlc/internal/plugin"
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

func (v QueryValue) Pair() string {
	if v.isEmpty() {
		return ""
	}

	var out []string
	if !v.EmitStruct() && v.IsStruct() {
		for _, f := range v.Struct.Fields {
			out = append(out, toLowerCase(f.Name)+" "+f.Type)
		}

		return strings.Join(out, ",")
	}

	return v.Name + " " + v.DefineType()
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
		return strings.Join(out, ",")
	}
	out = append(out, "")
	return "\n" + strings.Join(out, ",\n")
}

func (v QueryValue) ColumnNames() string {
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
					out = append(out, "&"+v.Name+"."+f.Name+"."+embed)
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

func (q Query) TableIdentifier() string {
	escapedNames := make([]string, 0, 3)
	for _, p := range []string{q.Table.Catalog, q.Table.Schema, q.Table.Name} {
		if p != "" {
			escapedNames = append(escapedNames, fmt.Sprintf("%q", p))
		}
	}
	return "[]string{" + strings.Join(escapedNames, ", ") + "}"
}

// CacheKey is used by WPgx only.
func (q Query) CacheKey() string {
	return genCacheKeyWithArgName(q, q.Arg.Name)
}

// InvalidateArgs is used by WPgx only.
func (q Query) InvalidateArgs() string {
	rv := ""
	if !q.Arg.isEmpty() {
		rv = ", "
	}
	for _, inv := range q.Invalidates {
		if inv.NoArg {
			continue
		}
		t := "*" + inv.Q.Arg.Type()
		rv += fmt.Sprintf("%s %s,", inv.ArgName, t)
	}
	return rv
}

// UniqueLabel is used by WPgx only.
func (q Query) UniqueLabel() string {
	return fmt.Sprintf("%s.%s", q.Pkg, q.MethodName)
}

// CacheUniqueLabel is used by WPgx only.
func (q Query) CacheUniqueLabel() string {
	return fmt.Sprintf("%s:%s:", q.Pkg, q.MethodName)
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

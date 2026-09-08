package wicked

import (
	"testing"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

func TestPostgreSQLTypeCompatibility(t *testing.T) {
	req := &plugin.GenerateRequest{Catalog: &plugin.Catalog{DefaultSchema: "public"}}
	for _, tc := range []struct {
		name, required, nullable string
	}{
		{"int2", "int16", "*int16"}, {"int4", "int32", "*int32"}, {"int8", "int64", "*int64"},
		{"serial2", "int16", "*int16"}, {"serial4", "int32", "*int32"}, {"serial8", "int64", "*int64"},
		{"float4", "float32", "*float32"}, {"float8", "float64", "*float64"},
		{"numeric", "pgtype.Numeric", "pgtype.Numeric"}, {"money", "pgtype.Numeric", "pgtype.Numeric"},
		{"bool", "bool", "*bool"}, {"bytea", "[]byte", "[]byte"},
		{"json", "json.RawMessage", "json.RawMessage"}, {"jsonb", "json.RawMessage", "json.RawMessage"},
		{"text", "string", "*string"}, {"varchar", "string", "*string"}, {"bpchar", "string", "*string"},
		{"uuid", "uuid.UUID", "*uuid.UUID"},
		{"date", "pgtype.Date", "pgtype.Date"}, {"time", "pgtype.Time", "pgtype.Time"},
		{"timetz", "time.Time", "*time.Time"}, {"timestamp", "time.Time", "*time.Time"}, {"timestamptz", "time.Time", "*time.Time"},
		{"interval", "int64", "*int64"},
		{"inet", "netip.Addr", "*netip.Addr"}, {"cidr", "netip.Prefix", "*netip.Prefix"},
		{"macaddr", "net.HardwareAddr", "*net.HardwareAddr"}, {"macaddr8", "net.HardwareAddr", "*net.HardwareAddr"},
		{"ltree", "string", "*string"}, {"lquery", "string", "*string"}, {"ltxtquery", "string", "*string"},
		{"hstore", "pgtype.Hstore", "pgtype.Hstore"},
		{"bit", "pgtype.Bits", "pgtype.Bits"}, {"varbit", "pgtype.Bits", "pgtype.Bits"},
		{"cid", "pgtype.Uint32", "pgtype.Uint32"}, {"oid", "pgtype.Uint32", "pgtype.Uint32"}, {"tid", "pgtype.TID", "pgtype.TID"},
		{"box", "pgtype.Box", "pgtype.Box"}, {"circle", "pgtype.Circle", "pgtype.Circle"}, {"line", "pgtype.Line", "pgtype.Line"},
		{"lseg", "pgtype.Lseg", "pgtype.Lseg"}, {"path", "pgtype.Path", "pgtype.Path"}, {"point", "pgtype.Point", "pgtype.Point"}, {"polygon", "pgtype.Polygon", "pgtype.Polygon"},
		{"daterange", "pgtype.Range[pgtype.Date]", "pgtype.Range[pgtype.Date]"},
		{"tsrange", "pgtype.Range[pgtype.Timestamp]", "pgtype.Range[pgtype.Timestamp]"},
		{"tstzrange", "pgtype.Range[pgtype.Timestamptz]", "pgtype.Range[pgtype.Timestamptz]"},
		{"numrange", "pgtype.Range[pgtype.Numeric]", "pgtype.Range[pgtype.Numeric]"},
		{"int4range", "pgtype.Range[pgtype.Int4]", "pgtype.Range[pgtype.Int4]"},
		{"int8range", "pgtype.Range[pgtype.Int8]", "pgtype.Range[pgtype.Int8]"},
		{"datemultirange", "pgtype.Multirange[pgtype.Range[pgtype.Date]]", "pgtype.Multirange[pgtype.Range[pgtype.Date]]"},
		{"tsmultirange", "pgtype.Multirange[pgtype.Range[pgtype.Timestamp]]", "pgtype.Multirange[pgtype.Range[pgtype.Timestamp]]"},
		{"tstzmultirange", "pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]", "pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]"},
		{"nummultirange", "pgtype.Multirange[pgtype.Range[pgtype.Numeric]]", "pgtype.Multirange[pgtype.Range[pgtype.Numeric]]"},
		{"int4multirange", "pgtype.Multirange[pgtype.Range[pgtype.Int4]]", "pgtype.Multirange[pgtype.Range[pgtype.Int4]]"},
		{"int8multirange", "pgtype.Multirange[pgtype.Range[pgtype.Int8]]", "pgtype.Multirange[pgtype.Range[pgtype.Int8]]"},
		{"name", "interface{}", "interface{}"}, {"xid", "interface{}", "interface{}"},
	} {
		for _, schema := range []string{"", "pg_catalog"} {
			t.Run(schema+"/"+tc.name, func(t *testing.T) {
				for _, notNull := range []bool{false, true} {
					col := &plugin.Column{Type: &plugin.Identifier{Schema: schema, Name: tc.name}, NotNull: notNull}
					want := tc.nullable
					if notNull {
						want = tc.required
					}
					if got := goType(req, &opts.Options{}, col); got != want {
						t.Fatalf("not-null=%v: type = %s, want %s", notNull, got, want)
					}
					col.IsArray, col.ArrayDims = true, 2
					if got := goType(req, &opts.Options{}, col); got != "[][]"+tc.required {
						t.Fatalf("array type = %s, want [][]%s", got, tc.required)
					}
				}
			})
		}
	}
}

func TestUUIDImportCompatibility(t *testing.T) {
	for _, typ := range []string{"uuid.UUID", "*uuid.UUID", "[]uuid.UUID"} {
		i := importer{Options: &opts.Options{SqlPackage: "pgx/v5"}, Structs: []Struct{{Fields: []Field{{Type: typ}}}}}
		imports := i.modelImports()
		if len(imports.Dep) != 1 || imports.Dep[0].Path != "github.com/google/uuid" {
			t.Errorf("%s imports = %+v, want google/uuid", typ, imports.Dep)
		}
	}
}

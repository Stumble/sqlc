package wicked

import (
	"fmt"
	"log"
	"strings"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/codegen/sdk"
	"github.com/sqlc-dev/sqlc/internal/debug"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

func parseIdentifierString(name string) (*plugin.Identifier, error) {
	parts := strings.Split(name, ".")
	switch len(parts) {
	case 1:
		return &plugin.Identifier{
			Name: parts[0],
		}, nil
	case 2:
		return &plugin.Identifier{
			Schema: parts[0],
			Name:   parts[1],
		}, nil
	case 3:
		return &plugin.Identifier{
			Catalog: parts[0],
			Schema:  parts[1],
			Name:    parts[2],
		}, nil
	default:
		return nil, fmt.Errorf("invalid name: %s", name)
	}
}

func postgresType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	columnType := strings.TrimPrefix(sdk.DataType(col.Type), "pg_catalog.")
	notNull := col.NotNull || col.IsArray
	switch columnType {
	case "serial", "serial4", "pg_catalog.serial4":
		if notNull {
			return "int32"
		} else {
			return "*int32"
		}
	case "bigserial", "serial8", "pg_catalog.serial8":
		if notNull {
			return "int64"
		} else {
			return "*int64"
		}
	case "smallserial", "serial2", "pg_catalog.serial2":
		if notNull {
			return "int16"
		} else {
			return "*int16"
		}
	case "integer", "int", "int4", "pg_catalog.int4":
		if notNull {
			return "int32"
		} else {
			return "*int32"
		}
	case "bigint", "int8", "pg_catalog.int8":
		if notNull {
			return "int64"
		} else {
			return "*int64"
		}
	case "smallint", "int2", "pg_catalog.int2":
		if notNull {
			return "int16"
		} else {
			return "*int16"
		}
	case "float", "double precision", "float8", "pg_catalog.float8":
		if notNull {
			return "float64"
		} else {
			return "*float64"
		}
	case "real", "float4", "pg_catalog.float4":
		if notNull {
			return "float32"
		} else {
			return "*float32"
		}
	case "numeric", "pg_catalog.numeric", "money":
		return "pgtype.Numeric"
	case "boolean", "bool", "pg_catalog.bool":
		if notNull {
			return "bool"
		} else {
			return "*bool"
		}
	case "json":
		return "json.RawMessage"
	case "jsonb":
		return "json.RawMessage"
	case "bytea", "blob", "pg_catalog.bytea":
		return "[]byte"
	case "date":
		return "pgtype.Date"
	case "time", "pg_catalog.time":
		return "pgtype.Time"
	case "timetz", "pg_catalog.timetz":
		// TODO(yumin): strange what timetz	are not scanned to pgtype.Time
		if notNull {
			return "time.Time"
		} else {
			return "*time.Time"
		}
	case "timestamp", "pg_catalog.timestamp":
		if notNull {
			return "time.Time"
		} else {
			return "*time.Time"
		}
	case "pg_catalog.timestamptz", "timestamptz":
		if notNull {
			return "time.Time"
		} else {
			return "*time.Time"
		}
	case "text", "varchar", "bpchar", "pg_catalog.varchar", "pg_catalog.bpchar", "string":
		if notNull {
			return "string"
		} else {
			return "*string"
		}
	case "uuid":
		if notNull {
			return "uuid.UUID"
		} else {
			return "*uuid.UUID"
		}
	case "inet":
		if notNull {
			return "netip.Addr"
		} else {
			return "*netip.Addr"

		}
	case "cidr":
		if notNull {
			return "netip.Prefix"
		} else {
			return "*netip.Prefix"
		}
	case "macaddr", "macaddr8":
		if notNull {
			return "net.HardwareAddr"
		} else {
			return "*net.HardwareAddr"
		}
	case "ltree", "lquery", "ltxtquery":
		// This module implements a data type ltree for representing labels
		// of data stored in a hierarchical tree-like structure. Extensive
		// facilities for searching through label trees are provided.
		//
		// https://www.postgresql.org/docs/current/ltree.html
		if notNull {
			return "string"
		} else {
			return "*string"
		}
	case "interval", "pg_catalog.interval":
		if notNull {
			return "int64"
		} else {
			return "*int64"
		}
	case "daterange":
		return "pgtype.Range[pgtype.Date]"
	case "datemultirange":
		return "pgtype.Multirange[pgtype.Range[pgtype.Date]]"
	case "tsrange":
		return "pgtype.Range[pgtype.Timestamp]"
	case "tsmultirange":
		return "pgtype.Multirange[pgtype.Range[pgtype.Timestamp]]"
	case "tstzrange":
		return "pgtype.Range[pgtype.Timestamptz]"
	case "tstzmultirange":
		return "pgtype.Multirange[pgtype.Range[pgtype.Timestamptz]]"
	case "numrange":
		return "pgtype.Range[pgtype.Numeric]"
	case "nummultirange":
		return "pgtype.Multirange[pgtype.Range[pgtype.Numeric]]"
	case "int4range":
		return "pgtype.Range[pgtype.Int4]"
	case "int4multirange":
		return "pgtype.Multirange[pgtype.Range[pgtype.Int4]]"
	case "int8range":
		return "pgtype.Range[pgtype.Int8]"
	case "int8multirange":
		return "pgtype.Multirange[pgtype.Range[pgtype.Int8]]"
	case "hstore":
		return "pgtype.Hstore"
	case "bit", "varbit", "pg_catalog.bit", "pg_catalog.varbit":
		return "pgtype.Bits"
	case "box":
		return "pgtype.Box"
	case "cid", "oid":
		return "pgtype.Uint32"
	case "tid":
		return "pgtype.TID"
	case "circle":
		return "pgtype.Circle"
	case "line":
		return "pgtype.Line"
	case "lseg":
		return "pgtype.Lseg"
	case "path":
		return "pgtype.Path"
	case "point":
		return "pgtype.Point"
	case "polygon":
		return "pgtype.Polygon"
	case "void":
		return "interface{}"
	case "any":
		return "interface{}"

	default:
		rel, err := parseIdentifierString(columnType)
		if err != nil {
			panic(fmt.Errorf("cannot parse identifier string: %s", columnType))
		}
		if rel.Schema == "" {
			rel.Schema = req.Catalog.DefaultSchema
		}

		for _, schema := range req.Catalog.Schemas {
			if schema.Name == "pg_catalog" || schema.Name == "information_schema" {
				continue
			}

			for _, enum := range schema.Enums {
				if rel.Name == enum.Name && rel.Schema == schema.Name {
					if notNull {
						if schema.Name == req.Catalog.DefaultSchema {
							return StructName(enum.Name, options)
						}
						return StructName(schema.Name+"_"+enum.Name, options)
					} else {
						if schema.Name == req.Catalog.DefaultSchema {
							return "Null" + StructName(enum.Name, options)
						}
						return "Null" + StructName(schema.Name+"_"+enum.Name, options)
					}
				}
			}

			for _, ct := range schema.CompositeTypes {
				if rel.Name == ct.Name && rel.Schema == schema.Name {
					if notNull {
						return "string"
					} else {
						return "*string"
					}
				}
			}
		}
	}

	if debug.Active {
		log.Printf("unknown PostgreSQL type: %s\n", columnType)
	}
	return "interface{}"
}

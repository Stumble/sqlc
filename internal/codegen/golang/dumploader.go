package golang

import (
	"fmt"
	"strings"
)

type DumpLoader struct {
	MainStruct *Struct
}

func (d DumpLoader) MainStructName() string {
	return d.MainStruct.Name
}

func (d DumpLoader) Fields(prefix string) string {
	if d.MainStruct == nil {
		panic("no MainStruct in DumpLoader")
	}
	var fields []string
	for _, f := range d.MainStruct.Fields {
		fields = append(fields, prefix+f.Name)
	}
	return strings.Join(fields, ",")
}

func (d DumpLoader) FieldDBNames() string {
	if d.MainStruct == nil {
		panic("no MainStruct in DumpLoader")
	}
	var fields []string
	for _, f := range d.MainStruct.Fields {
		fields = append(fields, f.DBName)
	}
	return strings.Join(fields, ",")
}

func (d DumpLoader) DumpSortByFields() string {
	if d.MainStruct == nil {
		panic("no MainStruct in DumpLoader")
	}
	var fields []string
	for _, f := range d.MainStruct.Fields {
		switch f.Type {
		// TODO(yumin):
		// best-effort sorting for now. Once we pass table indexes to codegen
		// we can just use index information.
		case "int", "int16", "int32", "int64", "float32", "float64", "string", "bool", "time.Time":
			fields = append(fields, f.DBName)
		case "*int", "*int16", "*int32", "*int64", "*float32", "*float64", "*string", "*bool", "*time.Time":
			fields = append(fields, f.DBName)
		default:
			continue
		}
	}
	return strings.Join(fields, ",")
}

func (d DumpLoader) ParamList() string {
	if d.MainStruct == nil {
		panic("no MainStruct in DumpLoader")
	}
	var vals []string
	for i := range d.MainStruct.Fields {
		vals = append(vals, fmt.Sprintf("$%d", i+1))
	}
	return strings.Join(vals, ",")
}

func (d DumpLoader) DumpSQL() string {
	return fmt.Sprintf(`SELECT %s FROM \"%s\" ORDER BY %s ASC;`,
		d.FieldDBNames(), d.MainStruct.Table.Name, d.DumpSortByFields())
}

func (d DumpLoader) LoadSQL() string {
	return fmt.Sprintf(`INSERT INTO \"%s\" (%s) VALUES (%s);`,
		d.MainStruct.Table.Name, d.FieldDBNames(), d.ParamList())
}

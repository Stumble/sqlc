package wicked

import (
	"testing"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

func TestPostgreSQLTypeCompatibility(t *testing.T) {
	req := &plugin.GenerateRequest{Catalog: &plugin.Catalog{DefaultSchema: "public"}}
	for _, tc := range []struct {
		name, want string
		nullable   bool
	}{
		{"json", "json.RawMessage", true}, {"jsonb", "json.RawMessage", true},
		{"int8", "*int64", true}, {"int8", "int64", false},
		{"varchar", "*string", true}, {"uuid", "uuid.UUID", false},
		{"numeric", "pgtype.Numeric", true}, {"timestamptz", "*time.Time", true},
		{"timestamp", "time.Time", false}, {"date", "pgtype.Date", true},
	} {
		for _, schema := range []string{"", "pg_catalog"} {
			t.Run(schema+"/"+tc.name+"/"+tc.want, func(t *testing.T) {
				col := &plugin.Column{Type: &plugin.Identifier{Schema: schema, Name: tc.name}, NotNull: !tc.nullable}
				if got := postgresType(req, &opts.Options{}, col); got != tc.want {
					t.Fatalf("type = %s, want %s", got, tc.want)
				}
			})
		}
	}
}

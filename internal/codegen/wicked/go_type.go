package wicked

import (
	"strings"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/codegen/sdk"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

func addExtraGoStructTags(tags map[string]string, req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) {
	for _, override := range options.Overrides {
		oride := override.ShimOverride
		if oride.GoType.StructTags == nil {
			continue
		}
		// Preserve column-scoped tags. Applying previously ignored db_type
		// tags here would silently change JSON and shared cache payloads.
		if !override.Matches(col.Table, req.Catalog.DefaultSchema) {
			// Different table.
			continue
		}
		cname := col.Name
		if col.OriginalName != "" {
			cname = col.OriginalName
		}
		if !sdk.MatchString(oride.ColumnName, cname) {
			// Different column.
			continue
		}
		// Add the extra tags.
		for k, v := range oride.GoType.StructTags {
			tags[k] = v
		}
	}
}

func goType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	// Check if the column's type has been overridden
	for _, override := range options.Overrides {
		oride := override.ShimOverride

		if oride.GoType.TypeName == "" {
			continue
		}
		cname := col.Name
		if col.OriginalName != "" {
			cname = col.OriginalName
		}
		sameTable := override.Matches(col.Table, req.Catalog.DefaultSchema)
		if oride.Column != "" && sdk.MatchString(oride.ColumnName, cname) && sameTable {
			if col.IsSqlcSlice {
				return "[]" + oride.GoType.TypeName
			}
			return oride.GoType.TypeName
		}
	}
	typ := goInnerType(req, options, col)
	if col.IsSqlcSlice {
		return "[]" + typ
	}
	if col.IsArray {
		return strings.Repeat("[]", int(col.ArrayDims)) + typ
	}
	return typ
}

func goInnerType(req *plugin.GenerateRequest, options *opts.Options, col *plugin.Column) string {
	// Preserve the legacy combined override order (global entries first).
	for _, override := range options.Overrides {
		oride := override.ShimOverride
		if oride.GoType.TypeName == "" {
			continue
		}
		if overrideMatchesColumn(override, col) {
			return oride.GoType.TypeName
		}
	}

	return postgresType(req, options, col)
}

// PostgreSQL catalog updates may qualify built-in types which were previously
// unqualified (notably json/jsonb). Preserve configured wicked type overrides
// across that representation change, without conflating user-defined schemas.
func overrideMatchesColumn(override opts.Override, col *plugin.Column) bool {
	if override.DBType == "" {
		return false
	}
	want := strings.TrimPrefix(override.DBType, "pg_catalog.")
	got := strings.TrimPrefix(sdk.DataType(col.Type), "pg_catalog.")
	notNull := col.NotNull || col.IsArray
	return want == got && override.Nullable != notNull && override.Unsigned == col.Unsigned
}

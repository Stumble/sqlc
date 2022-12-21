package config

import "fmt"

func Validate(c *Config) error {
	seen := make(map[string]struct{})
	for _, sql := range c.SQL {
		sqlGo := sql.Gen.Go
		if sqlGo == nil {
			continue
		}
		if sqlGo.EmitMethodsWithDBArgument && sqlGo.EmitPreparedQueries {
			return fmt.Errorf("invalid config: emit_methods_with_db_argument and emit_prepared_queries settings are mutually exclusive")
		}
		if sql.Database != nil {
			if sql.Database.URI == "" {
				return fmt.Errorf("invalid config: database must have a non-empty URI")
			}
		}
		if _, ok := seen[sql.Gen.Go.Package]; ok {
			return fmt.Errorf("duplicated package name is not allowed: %s", sql.Gen.Go.Package)
		}
		seen[sql.Gen.Go.Package] = struct{}{}
	}
	return nil
}

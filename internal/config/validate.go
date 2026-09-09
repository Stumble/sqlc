package config

import (
	"fmt"
	"path/filepath"
)

func Validate(c *Config) error {
	packages := make(map[string]bool)
	for _, sql := range c.SQL {
		if sql.IsWicked() {
			if sql.Engine != EnginePostgreSQL {
				return fmt.Errorf("wpgx requires the postgresql engine")
			}
			name := sql.Gen.Go.Package
			if name == "" {
				name = filepath.Base(sql.Gen.Go.Out)
			}
			if packages[name] {
				return fmt.Errorf("duplicated package name is not allowed: %s", name)
			}
			packages[name] = true
		}
		if sql.Database != nil {
			if sql.Database.URI == "" && !sql.Database.Managed {
				return ErrInvalidDatabase
			}
		}
	}
	return nil
}

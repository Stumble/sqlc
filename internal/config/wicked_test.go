package config

import (
	"testing"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
)

func TestWickedPackageUniqueness(t *testing.T) {
	for _, driver := range []string{"wpgx", "pgx/v5"} {
		c := Config{SQL: []SQL{
			{Engine: EnginePostgreSQL, Gen: SQLGen{Go: &opts.Options{Package: "books", Out: "one", SqlPackage: driver}}},
			{Engine: EnginePostgreSQL, Gen: SQLGen{Go: &opts.Options{Package: "books", Out: "two", SqlPackage: driver}}},
		}}
		err := Validate(&c)
		if (err != nil) != (driver == "wpgx") {
			t.Fatalf("driver=%s validation=%v", driver, err)
		}
	}
}

func TestWickedEffectivePackageUniqueness(t *testing.T) {
	c := Config{SQL: []SQL{
		{Engine: EnginePostgreSQL, Gen: SQLGen{Go: &opts.Options{Out: "one/books", SqlPackage: "wpgx"}}},
		{Engine: EnginePostgreSQL, Gen: SQLGen{Go: &opts.Options{Out: "two/books", SqlPackage: "wpgx"}}},
	}}
	if err := Validate(&c); err == nil {
		t.Fatal("duplicate effective package names must not share cache keys")
	}
}

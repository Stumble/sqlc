package metadata

import (
	"errors"
	"testing"
)

func TestParseMetadata(t *testing.T) {
	for _, query := range []string{
		`-- name: CreateFoo, :one`,
		`-- name: 9Foo_, :one`,
		`-- name: CreateFoo :two`,
		`-- name: CreateFoo`,
		`-- name: CreateFoo :one something`,
		`-- name: `,
		`--name: CreateFoo :one`,
		`--name CreateFoo :one`,
		`--name: CreateFoo :two`,
		"-- name:CreateFoo",
		`--name:CreateFoo :two`,
	} {
		if _, err := Parse(query, CommentSyntax{Dash: true}); err == nil {
			t.Errorf("expected invalid metadata: %q", query)
		}
	}

	for _, query := range []string{
		`-- some comment`,
		`-- name comment`,
		`--name comment`,
	} {
		if _, err := Parse(query, CommentSyntax{Dash: true}); err != nil {
			t.Errorf("expected valid comment: %q", query)
		}
	}

	query := `-- name: CreateFoo :one`
	config, err := Parse(query, CommentSyntax{Dash: true})
	if err != nil {
		t.Errorf("expected valid metadata: %q", query)
	}
	if config.Name != "CreateFoo" {
		t.Errorf("incorrect queryName parsed: %q", query)
	}
	if config.Cmd != CmdOne {
		t.Errorf("incorrect queryType parsed: %q", query)
	}

}

func TestParseMetadataWithOptions(t *testing.T) {
	for _, tc := range []struct {
		query   string
		options [][2]string
		err     error
	}{
		{
			"-- name: q :one\n-- -- key:value",
			[][2]string{{"key", "value"}},
			nil,
		},
		{
			"-- name: q :one\n-- -- no key value pairs",
			nil,
			errors.New("no key value pair"),
		},
		{
			"-- name: q :one\n-- --    key   :    value   ",
			[][2]string{{"key", "value"}},
			nil,
		},
	} {
		config, err := Parse(tc.query, CommentSyntax{Dash: true})
		if tc.err == nil {
			if err != nil {
				t.Errorf("unexpected err: %s %s", tc.query, err)
			}
			if len(config.Options) != len(tc.options) {
				t.Errorf("option parse error, query: %s, expected: %+v, got: %+v", tc.query, tc.options, config.Options)
			}
			for _, opt := range tc.options {
				val, ok := config.Options[opt[0]]
				if !ok || val != opt[1] {
					t.Errorf(
						"option parse error, query: %s, expect: %+v, got: %+v",
						tc.query, tc.options, config.Options)

				}
			}
		}
	}
}

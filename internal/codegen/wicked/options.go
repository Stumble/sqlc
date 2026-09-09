package wicked

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sqlc-dev/sqlc/internal/codegen/golang/opts"
	"github.com/sqlc-dev/sqlc/internal/plugin"
)

var wickedReservedNames = map[string]bool{"load": true, "dump": true, "check": true}

// Reuse upstream's override and rename parsing without teaching the standard Go
// generator about wpgx. Only this private copy is normalized to its pgx driver.
func parseOptions(req *plugin.GenerateRequest) (*opts.Options, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(req.PluginOptions, &raw); err != nil {
		return nil, fmt.Errorf("wicked options: %w", err)
	}
	if raw == nil {
		return nil, fmt.Errorf("wicked: missing Go options")
	}
	raw["sql_package"] = json.RawMessage(`"pgx/v5"`)
	blob, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	copy := &plugin.GenerateRequest{Settings: req.Settings, Catalog: req.Catalog, PluginOptions: blob, GlobalOptions: req.GlobalOptions}
	options, err := opts.Parse(copy)
	if err != nil {
		return nil, err
	}
	// Legacy wicked configurations use local rename entries over global ones.
	// Upstream opts.Parse currently applies them in the opposite order.
	if rename, present := raw["rename"]; present {
		var local map[string]string
		if err := json.Unmarshal(rename, &local); err != nil {
			return nil, err
		}
		if options.Rename == nil {
			options.Rename = make(map[string]string)
		}
		for key, value := range local {
			options.Rename[key] = value
		}
	}
	return options, nil
}

func parseQueryOptions(comments []string, isSelect bool, names map[string]bool) (WPgxOption, error) {
	values := make(map[string]string)
	for _, comment := range comments {
		body := strings.TrimSpace(comment)
		if !strings.HasPrefix(body, "--") {
			continue
		}
		key, value, ok := strings.Cut(body[2:], ":")
		if !ok {
			return WPgxOption{}, fmt.Errorf("invalid query option: %s", comment)
		}
		values[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	for _, key := range []string{WpgxOptionKeyCountIntent, WpgxOptionKeyAllowReplica} {
		value, present := values[key]
		if !present {
			if isSelect {
				values[key] = "true"
			} else {
				values[key] = "false"
			}
		} else if value != "true" && value != "false" {
			return WPgxOption{}, fmt.Errorf("invalid %s value: %s", key, value)
		} else if !isSelect && value == "true" {
			return WPgxOption{}, fmt.Errorf("%s requires a SELECT statement", key)
		}
	}
	if _, present := values[WPgxOptionKeyInvalidate]; isSelect && present {
		return WPgxOption{}, fmt.Errorf("invalidate cannot be used on a SELECT statement")
	}
	return parseOption(values, names)
}

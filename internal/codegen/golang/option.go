package golang

import (
	"fmt"
	"strings"
	"time"
)

const (
	WPgxOptionKeyCache        = "cache"
	WPgxOptionKeyInvalidate   = "invalidate"
	WpgxOptionKeyCountIntent  = "count_intent"
	WpgxOptionKeyTimeout      = "timeout"
	WpgxOptionKeyAllowReplica = "allow_replica"
)

type WPgxOption struct {
	Cache        time.Duration
	Invalidates  []string
	CountIntent  bool
	Timeout      time.Duration
	AllowReplica bool
}

func parseOption(options map[string]string, queryNames map[string]bool) (rv WPgxOption, err error) {
	for k, v := range options {
		switch k {
		case WPgxOptionKeyCache:
			rv.Cache, err = time.ParseDuration(v)
			if err != nil {
				return
			}
			if rv.Cache < 1*time.Millisecond {
				return rv, fmt.Errorf("cache duration too short: %s", v)
			}
		case WPgxOptionKeyInvalidate:
			trimed := strings.Trim(v, " []")
			fnNames := strings.Split(trimed, ",")
			for _, rawFnName := range fnNames {
				queryName := strings.TrimSpace(rawFnName)
				if !queryNames[queryName] {
					return rv, fmt.Errorf("Unknown to invalidate query: %s", queryName)
				}
				rv.Invalidates = append(rv.Invalidates, queryName)
			}
		case WpgxOptionKeyCountIntent:
			if v == "true" {
				rv.CountIntent = true
			} else if v == "false" {
				rv.CountIntent = false
			} else {
				return rv, fmt.Errorf("Unknown count_intent value: %s", v)
			}
		case WpgxOptionKeyTimeout:
			rv.Timeout, err = time.ParseDuration(v)
			if err != nil {
				return
			}
			if rv.Timeout < 1*time.Millisecond {
				return rv, fmt.Errorf("timeout duration too short: %s", v)
			}
		case WpgxOptionKeyAllowReplica:
			if v == "true" {
				rv.AllowReplica = true
			} else if v == "false" {
				rv.AllowReplica = false
			} else {
				return rv, fmt.Errorf("Unknown allow_replica value: %s", v)
			}
		default:
			return rv, fmt.Errorf("Unknown option: %s", k)
		}
	}
	return
}

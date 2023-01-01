package golang

import (
	"fmt"
	"strings"
	"time"
)

const (
	WPgxOptionKeyCache      = "cache"
	WPgxOptionKeyInvalidate = "invalidate"
)

type WPgxOption struct {
	Cache       time.Duration
	Invalidates []string
}

func parseOption(options map[string]string, queryNames map[string]bool) (rv WPgxOption, err error) {
	for k, v := range options {
		switch k {
		case WPgxOptionKeyCache:
			rv.Cache, err = time.ParseDuration(v)
			if err != nil {
				return
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
		}
	}
	return
}

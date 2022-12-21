package metadata

import (
	"fmt"
	"strings"
	"unicode"
)

type CommentSyntax struct {
	Dash      bool
	Hash      bool
	SlashStar bool
}

const (
	CmdExec       = ":exec"
	CmdExecResult = ":execresult"
	CmdExecRows   = ":execrows"
	CmdExecLastId = ":execlastid"
	CmdMany       = ":many"
	CmdOne        = ":one"
	CmdCopyFrom   = ":copyfrom"
	CmdBatchExec  = ":batchexec"
	CmdBatchMany  = ":batchmany"
	CmdBatchOne   = ":batchone"
)

// A query name must be a valid Go identifier
//
// https://golang.org/ref/spec#Identifiers
func validateQueryName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("invalid query name: %q", name)
	}
	for i, c := range name {
		isLetter := unicode.IsLetter(c) || c == '_'
		isDigit := unicode.IsDigit(c)
		if i == 0 && !isLetter {
			return fmt.Errorf("invalid query name %q", name)
		} else if !(isLetter || isDigit) {
			return fmt.Errorf("invalid query name %q", name)
		}
	}
	return nil
}

type QueryConfig struct {
	Name    string
	Cmd     string
	Options map[string]string
}

// Parse returns query name and the specified return type.
func ParseQueryNameAndType(t string, commentStyle CommentSyntax) (*QueryConfig, error) {
	config := &QueryConfig{
		Options: make(map[string]string),
	}
	for _, line := range strings.Split(t, "\n") {
		var prefix string
		if strings.HasPrefix(line, "--") {
			if !commentStyle.Dash {
				continue
			}
			prefix = "--"
		}
		if strings.HasPrefix(line, "/*") {
			if !commentStyle.SlashStar {
				continue
			}
			prefix = "/*"
		}
		if strings.HasPrefix(line, "#") {
			if !commentStyle.Hash {
				continue
			}
			prefix = "#"
		}
		if prefix == "" {
			continue
		}

		// comments body
		body := line[len(prefix):]
		if prefix == "/*" {
			body = body[:len(body)-1] // removes the trailing "*/" element
		}
		body = strings.TrimSpace(body)

		if strings.HasPrefix(body, "name:") {
			// original	query comments.
			part := strings.Split(strings.TrimSpace(line), " ")
			if len(part) == 2 {
				return nil, fmt.Errorf("missing query type [':one', ':many', ':exec', ':execrows', ':execlastid', ':execresult', ':copyfrom', 'batchexec', 'batchmany', 'batchone']: %s", line)
			}
			if len(part) != 4 {
				return nil, fmt.Errorf("invalid query comment: %s", line)
			}
			queryName := part[2]
			queryType := strings.TrimSpace(part[3])
			switch queryType {
			case CmdOne, CmdMany, CmdExec, CmdExecResult, CmdExecRows, CmdExecLastId, CmdCopyFrom, CmdBatchExec, CmdBatchMany, CmdBatchOne:
			default:
				return nil, fmt.Errorf("invalid query type: %s", queryType)
			}
			if err := validateQueryName(queryName); err != nil {
				return nil, err
			}
			config.Name = queryName
			config.Cmd = queryType
		} else if strings.HasPrefix(body, "--") {
			body = body[2:] // trim "--"
			// expecting a key value pair of this format: "key:value"
			sepIndex := strings.Index(body, ":")
			if sepIndex == -1 {
				return nil, fmt.Errorf("invalid query option string: %s", line)
			}
			key := strings.TrimSpace(body[:sepIndex])
			val := strings.TrimSpace(body[sepIndex+1:])
			config.Options[key] = val
		} else {
			// to be consistent with previous logic: if comments start with name
			// and have ":", it must start with "name:".
			// TODO(yumin): is this necessary?
			if strings.HasPrefix(body, "name") && strings.Contains(body, ":") {
				return nil, fmt.Errorf("invalid metadata: %s", line)
			}
			continue
		}

	}
	return config, nil
}

func ParseQueryFlags(comments []string) (map[string]bool, error) {
	flags := make(map[string]bool)
	for _, line := range comments {
		cleanLine := strings.TrimPrefix(line, "--")
		cleanLine = strings.TrimPrefix(cleanLine, "/*")
		cleanLine = strings.TrimPrefix(cleanLine, "#")
		cleanLine = strings.TrimSuffix(cleanLine, "*/")
		cleanLine = strings.TrimSpace(cleanLine)
		if strings.HasPrefix(cleanLine, "@") {
			flagName := strings.SplitN(cleanLine, " ", 2)[0]
			flags[flagName] = true
			continue
		}
	}
	return flags, nil
}

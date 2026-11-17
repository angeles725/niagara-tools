package runbooks

import (
	"errors"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
)

// parse reads one runbook file: a front matter between "---" lines, then the body. Keys:
//
//	title: <text>
//	description: <text>
//	arg <name>[!]: <description> [= <default>]   ("!" marks a required argument)
//
// The runbook's name is the file name without ".md".
func parse(file, text string) (jsonrpc.Runbook, error) {
	b := jsonrpc.Runbook{Name: strings.TrimSuffix(file, ".md")}
	rest, ok := strings.CutPrefix(text, "---\n")
	if !ok {
		return b, errors.New(file + ": missing front matter")
	}
	head, body, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		return b, errors.New(file + ": unterminated front matter")
	}
	b.Body = body
	for _, line := range strings.Split(head, "\n") {
		key, val, ok := strings.Cut(line, ": ")
		if !ok {
			return b, fmt.Errorf("%s: front matter line %q is not key: value", file, line)
		}
		switch {
		case key == "title":
			b.Title = val
		case key == "description":
			b.Description = val
		case strings.HasPrefix(key, "arg "):
			b.Args = append(b.Args, arg(strings.TrimPrefix(key, "arg "), val))
		default:
			return b, fmt.Errorf("%s: unknown front matter key %q", file, key)
		}
	}
	if b.Title == "" || b.Description == "" {
		return b, errors.New(file + ": needs title and description")
	}
	return b, nil
}

func arg(name, val string) jsonrpc.RunbookArg {
	a := jsonrpc.RunbookArg{Name: strings.TrimSuffix(name, "!"), Required: strings.HasSuffix(name, "!")}
	desc, def, _ := strings.Cut(val, " = ")
	a.Description, a.Default = desc, def
	return a
}

// Package runbooks holds the recurring station procedures (post-deploy check, alarm triage,
// commissioning close-out) the server offers as MCP prompts and resources. Each runbook is a
// Markdown file under files/ with a small front matter (parse.go); the text is data, this
// package only loads it.
package runbooks

import (
	"embed"
	"fmt"
	"sort"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/jsonrpc"
)

//go:embed files/*.md
var files embed.FS

// All returns the shipped runbooks sorted by name. A malformed file is a build defect the
// package tests catch, so it panics rather than serving a partial list.
func All() []jsonrpc.Runbook {
	entries, err := files.ReadDir("files")
	if err != nil {
		panic(fmt.Sprintf("runbooks: %v", err))
	}
	out := make([]jsonrpc.Runbook, 0, len(entries))
	for _, e := range entries {
		raw, err := files.ReadFile("files/" + e.Name())
		if err != nil {
			panic(fmt.Sprintf("runbooks: %v", err))
		}
		b, err := parse(e.Name(), string(raw))
		if err != nil {
			panic("runbooks: " + err.Error())
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

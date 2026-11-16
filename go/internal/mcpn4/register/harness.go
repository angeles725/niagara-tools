package register

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Harnesses are the agent harnesses register writes an MCP server entry for.
var Harnesses = []string{"claude", "pi", "gentle-shell"}

// target is where one harness keeps its MCP server entry, and the entry itself.
type target struct {
	file  string   // the harness config file
	path  []string // the JSON path of the entry in it
	entry []byte   // the entry, compact JSON
}

func jsonString(s string) []byte { return encodeKey(s) }

// harnessTarget is the entry for server name running launcher in the given harness. Claude Code
// keeps a project's (local scope) servers in ~/.claude.json under projects[<dir>].mcpServers;
// pi and gentle-shell read <agent-dir>/mcp.json, where "exposure": "direct" makes every tool a
// first-class tool.
func harnessTarget(harness, home, project, name, launcher string) (target, error) {
	cmd := jsonString(launcher)
	switch harness {
	case "claude":
		return target{filepath.Join(home, ".claude.json"), []string{"projects", project, "mcpServers", name},
			[]byte(`{"args":[],"command":` + string(cmd) + `,"env":{},"type":"stdio"}`)}, nil
	case "pi", "gentle-shell":
		dir := filepath.Join(home, ".pi", "agent")
		if harness == "gentle-shell" {
			dir = filepath.Join(home, ".gentle-shell", "agent")
		}
		return target{filepath.Join(dir, "mcp.json"), []string{"mcpServers", name},
			[]byte(`{"command":` + string(cmd) + `,"args":[],"exposure":"direct"}`)}, nil
	}
	return target{}, usageErrorf("--harness must be one of %v, got %q", Harnesses, harness)
}

// plan reads the target file and returns the new file bytes, or nil when the entry is already
// there byte-for-byte equal in value. A differing entry is a conflict unless force is set.
func (t target) plan(force bool) (doc, next []byte, err error) {
	doc, err = os.ReadFile(t.file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, err
	}
	if t.path[0] == "projects" {
		if _, ok, perr := getPath(doc, t.path[:2]); err != nil || perr != nil || !ok {
			return nil, nil, fmt.Errorf("no Claude Code project %s in %s: start Claude Code in that directory once, or pass --project", t.path[1], t.file)
		}
	}
	old, found, err := getPath(doc, t.path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", t.file, err)
	}
	if found && compactString(old) == string(t.entry) {
		return doc, nil, nil
	}
	if found && !force {
		return nil, nil, fmt.Errorf("%s already holds %s with a different %v; rerun with --force to replace it",
			t.file, t.path[len(t.path)-1], changedFields(old, t.entry))
	}
	next, err = setPath(doc, t.path, t.entry)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %w", t.file, err)
	}
	return doc, next, nil
}

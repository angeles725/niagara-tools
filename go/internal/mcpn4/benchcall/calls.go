package benchcall

import (
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// call is one line of the calls file.
type call struct {
	tool  string
	args  *pyjson.Object
	write bool
	// execute is the line's own "execute": true, the explicit step a destructive write needs to
	// execute (retro 2026-11-16 row M17): a run-wide flag alone never executes one.
	execute bool
	line    int // 1-based line in the calls file
}

// loadCalls reads the calls file: one JSON object per non-blank line. Any line that is not
// {"tool": string, "args": object, "write": bool, "execute": bool} is an error naming file and line.
func loadCalls(path string) ([]call, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var calls []call
	for i, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		c, err := parseCall(line)
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %s", path, i+1, err)
		}
		c.line = i + 1
		calls = append(calls, c)
	}
	if len(calls) == 0 {
		return nil, fmt.Errorf("%s: no calls", path)
	}
	return calls, nil
}

func parseCall(line string) (call, error) {
	v, err := pyjson.Unmarshal([]byte(line))
	if err != nil {
		return call{}, fmt.Errorf("not JSON: %v", err)
	}
	obj, ok := v.(*pyjson.Object)
	if !ok {
		return call{}, fmt.Errorf("not a JSON object")
	}
	c := call{args: pyjson.NewObject()}
	for _, key := range obj.Keys() {
		val, _ := obj.Get(key)
		switch key {
		case "tool":
			c.tool, ok = val.(string)
		case "args":
			c.args, ok = val.(*pyjson.Object)
		case "write":
			c.write, ok = val.(bool)
		case "execute":
			c.execute, ok = val.(bool)
		default:
			return call{}, fmt.Errorf("unknown key %q", key)
		}
		if !ok {
			return call{}, fmt.Errorf("key %q has the wrong type", key)
		}
	}
	if c.tool == "" {
		return call{}, fmt.Errorf(`no "tool"`)
	}
	if c.execute && !c.write {
		return call{}, fmt.Errorf(`"execute" needs "write": true`)
	}
	return c, nil
}

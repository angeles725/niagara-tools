package livesmoke

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// stubWrites are the tools the stub plans before it executes.
var stubWrites = map[string]bool{"n4_create_component": true, "n4_create_link": true, "n4_invoke_action": true,
	"n4_save_station": true, "n4_remove_component": true, "n4_rollback": true}

type stubCall struct {
	tool      string
	executing bool
	args      *pyjson.Object
}

// stubHook overrides one call: handled false lets the stub answer.
type stubHook func(c *stubClient, tool string, executing bool, args *pyjson.Object) (handled, isErr bool, payload any, err error)

// stubClient is a scripted in-memory stand-in for the MCP client (Scenario-level tests, no
// subprocess): it models whether the scratch folder exists and the thermostat values.
type stubClient struct {
	present bool
	calls   []stubCall
	vals    map[string]float64
	counts  map[string]int // "<tool>/<executing>" -> calls
	hook    stubHook
}

func countKey(tool string, executing bool) string {
	if executing {
		return tool + "/exec"
	}
	return tool + "/dry"
}

func (c *stubClient) Call(tool string, args *pyjson.Object) (bool, any, error) {
	_, executing := args.Get("confirmation_token")
	c.counts[countKey(tool, executing)]++
	c.calls = append(c.calls, stubCall{tool, executing, args})
	if c.hook != nil {
		if handled, isErr, payload, err := c.hook(c, tool, executing, args); handled {
			return isErr, payload, err
		}
	}
	if stubWrites[tool] && !executing {
		return false, kwargs("confirmation_token", "t", "plan_hash", "h"), nil
	}
	ok := kwargs("batch_id", "b-"+tool, "verdict", "verified", "observed", pyjson.NewObject())
	str := func(k string) string { v, _ := args.Get(k); s, _ := v.(string); return s }
	switch tool {
	case "n4_connect":
		return false, kwargs("station_name", str("station")), nil
	case "n4_navigate":
		name := "Services"
		if c.present {
			name = Scratch
		}
		return false, kwargs("children", []any{kwargs("name", name)}), nil
	case "n4_create_component":
		c.present = c.present || str("name") == Scratch
	case "n4_remove_component":
		c.present = false
	case "n4_rollback":
		c.present = true
	case "n4_invoke_action":
		v, _ := args.Get("arg")
		n, _ := v.(int)
		ord := str("ord")
		c.vals[ord[strings.LastIndex(ord, "/")+1:]] = float64(n)
	case "n4_find_dangling_outputs":
		return false, kwargs("dangling", []any{kwargs("path", FolderOrd+"/Cooling")}), nil
	case "n4_read_slots":
		return false, kwargs("slots", []any{kwargs("name", "out", "value", c.vals["Temp"] > c.vals["Setpoint"])}), nil
	case "n4_save_station":
		ok.Set("persisted", true)
		ok.Set("evidence", "stub")
	}
	return false, ok, nil
}

func stubScenario(hook stubHook) (*stubClient, *Scenario) {
	c := &stubClient{vals: map[string]float64{}, counts: map[string]int{}, hook: hook}
	return c, NewScenario(c, "SmokeStation", 0.2, func(string) {})
}

func names(sc *Scenario) []string {
	var out []string
	for _, s := range sc.Steps {
		n, _ := s.Get("name")
		out = append(out, n.(string))
	}
	return out
}

func verdicts(sc *Scenario) map[string]any {
	out := map[string]any{}
	for _, s := range sc.Steps {
		n, _ := s.Get("name")
		v, _ := s.Get("verdict")
		out[n.(string)] = v
	}
	return out
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

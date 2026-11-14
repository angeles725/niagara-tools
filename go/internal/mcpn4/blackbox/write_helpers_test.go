package blackbox

// Helpers of the write-tool tests (port of test_tools_write.WriteTestCase).

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/journal"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/readtools/readtoolstest/boxstation"
)

var (
	writeTools       = []string{"n4_create_component", "n4_set_slot", "n4_invoke_action", "n4_create_link"}
	destructiveTools = []string{"n4_remove_component", "n4_rollback", "n4_save_station"}
	batchIDPattern   = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

// writeRig is WriteTestCase: a writes-allowed rig (Rig.WriteOptions), then configure.
func writeRig(t *testing.T, configure func(o *Options)) *Rig {
	return NewRig(t, func(r *Rig, o *Options) {
		r.WriteOptions(o)
		if configure != nil {
			configure(o)
		}
	})
}

// verifiedRig is writeRig connected with a verified identity (connect_verified in setUp).
func verifiedRig(t *testing.T) *Rig {
	r := writeRig(t, nil)
	r.ConnectVerified()
	return r
}

// with is args with the extra keys set (dict(args, **extra)).
func withArgs(args Args, extra Args) Args {
	out := Args{}
	for k, v := range args {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// listedTools are the server's tools/list entries by name.
func listedTools(s *Server) map[string]map[string]any {
	out := map[string]map[string]any{}
	for _, tool := range objects(Dig(s.Request("tools/list", nil), "result", "tools")) {
		name, _ := tool["name"].(string)
		out[name] = tool
	}
	return out
}

// batchView is state_files.Journal.read: the merged journal view of a batch, nil when it was
// never journaled.
func batchView(t *testing.T, r *Rig, id any) map[string]any {
	t.Helper()
	s, _ := id.(string)
	view, err := journal.New(r.StateDir).Read(s)
	if err != nil {
		t.Fatal(err)
	}
	if view == nil {
		return nil
	}
	raw, err := pyjson.Marshal(view, pyjson.Options{})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// phases are the phase fields of the journal lines.
func phases(r *Rig) []any { return field(anySlice(r.Lines("journal.jsonl")), "phase") }

// keys are the sorted keys of m.
func keys(m map[string]any) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// nodeValue is the value of the node under handle h at path, "<absent>" when it is missing.
func nodeValue(r *Rig, h string, path ...string) (v any) {
	v = "<absent>"
	r.Station.Do(func(st *boxstation.Station) {
		n := st.ByHandle(h)
		for _, name := range path {
			if n == nil {
				return
			}
			n = n.Child(name)
		}
		if n != nil {
			v = n.Value
		}
	})
	return v
}

// hasChild reports whether the node under handle h has a direct child named name.
func hasChild(r *Rig, h, name string) (ok bool) {
	r.Station.Do(func(st *boxstation.Station) { ok = st.ByHandle(h).Child(name) != nil })
	return ok
}

// invocations records the accepted actions other than save, as (action, handle) pairs: the
// fake_station.invoked list. extra, when not nil, runs after each one.
type invocations struct {
	r    *Rig
	list [][2]string
}

func recordInvocations(r *Rig, extra func(st *boxstation.Station, action string, n *boxstation.Node)) *invocations {
	inv := &invocations{r: r}
	r.Station.Do(func(st *boxstation.Station) {
		st.AfterInvoke = func(st *boxstation.Station, action string, n *boxstation.Node) {
			if action != "save" {
				inv.list = append(inv.list, [2]string{action, n.Handle})
			}
			if extra != nil {
				extra(st, action, n)
			}
		}
	})
	return inv
}

func (inv *invocations) get() (out [][2]string) {
	inv.r.Station.Do(func(*boxstation.Station) { out = append(out, inv.list...) })
	return out
}

// journalPhasesOnDisk reads journal.jsonl directly, for a fault running outside the test
// goroutine (where Rig.Lines cannot fail the test); an unreadable line is "<bad>".
func journalPhasesOnDisk(stateDir string) []any {
	out := []any{}
	data, err := os.ReadFile(filepath.Join(stateDir, "journal.jsonl"))
	if err != nil {
		return out
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var m map[string]any
		if line == "" {
			continue
		}
		if json.Unmarshal([]byte(line), &m) != nil {
			out = append(out, "<bad>")
			continue
		}
		out = append(out, m["phase"])
	}
	return out
}

// callNaN is Call with the argument nanKey sent as the bare JSON token NaN (Python's json.dumps
// writes float("nan") so; encoding/json refuses it); it returns the tool result.
func callNaN(t *testing.T, s *Server, tool string, args Args, nanKey string) map[string]any {
	t.Helper()
	const mark = "__nan_placeholder__"
	line := RPC("tools/call", Args{"name": tool, "arguments": withArgs(args, Args{nanKey: mark})}, 1)
	line = strings.Replace(line, `"`+mark+`"`, "NaN", 1)
	if !strings.Contains(line, "NaN") {
		t.Fatalf("no NaN in %s", line)
	}
	reply := s.decode(s.Line(line))
	result, _ := reply["result"].(map[string]any)
	if _, bad := reply["error"]; bad || result == nil {
		t.Fatalf("%s: not a tool result: %v", tool, reply)
	}
	return result
}

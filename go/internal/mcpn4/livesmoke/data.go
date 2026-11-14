// Package livesmoke is the live smoke runner of mcp-n4 (formerly mcp-n4-kit/tools/live_smoke.py):
// it drives the REAL MCP server over stdio.
//
// Plan only by default (prints the scenario, contacts nothing). With --apply it starts the Go
// server `mcp-n4 --allow-writes ...` as a subprocess and runs the kitControl thermostat scenario
// (niagara-research B1199) through the public tool interface only: every write is dry run ->
// confirmation token -> execute -> verdict.
//
// Credentials reach the server only through the environment variables it reads
// (<PREFIX>_USER / <PREFIX>_PASSWORD); the runner scrubs their values from everything it prints
// or writes. Exit 0 only when every required step is verified.
package livesmoke

import (
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// Scratch is the folder the scenario creates under the station root, and the only place it
// writes to.
const Scratch = "McpSmoke"

// RootOrd is the station root.
const RootOrd = "station:|slot:/"

// FolderOrd is the scratch folder's ORD.
var FolderOrd = mustChild(RootOrd, Scratch)

// ReadTools are the tools the scenario calls without the dry run -> token -> execute cycle.
var ReadTools = map[string]bool{"n4_connect": true, "n4_describe_session": true, "n4_navigate": true,
	"n4_read_slots": true, "n4_list_links": true, "n4_find_dangling_outputs": true}

// The component types of the thermostat.
const (
	numeric = "control:NumericWritable"
	boolean = "control:BooleanWritable"
	compare = "kitControl:GreaterThan"
)

// component is one component the scenario creates in the scratch folder. X and Y (and the fixed
// width in create) are cosmetic wire-sheet positions only; no step checks them.
type component struct {
	Name, Type string
	X, Y       int
}

// link is one link the scenario creates: Src.SrcSlot -> Dst.DstSlot.
type link struct{ Src, SrcSlot, Dst, DstSlot string }

var components = []component{{"Temp", numeric, 1, 1}, {"Setpoint", numeric, 1, 6},
	{"Compare", compare, 16, 3}, {"Cooling", boolean, 32, 3}}

var links = []link{{"Temp", "out", "Compare", "inA"}, {"Setpoint", "out", "Compare", "inB"},
	{"Compare", "out", "Cooling", "in10"}}

// ProbeOnly lists the steps whose verdict is recorded but does not gate the exit code.
var ProbeOnly = []string{"n4_rollback of the folder removal re-creates a nested subtree: the T4 claim " +
	"not yet proven live; its verdict is recorded but does not gate the exit code"}

// WriteScopes is the scratch folder plus the root.
//
// The root scope is unavoidable: creating or removing the folder itself writes to its parent
// (the station root) and the server matches scopes by ORD prefix, so it cannot express "root, but
// only this child". The runner enforces the narrowing itself: AssertInBounds refuses any write
// that is not inside McpSmoke or the root create/remove of McpSmoke.
func WriteScopes() []string { return []string{FolderOrd, RootOrd} }

// ChildOrd is the ORD of child name under parentOrd, joined as the server joins it:
// `station:|slot:/` -> `station:|slot:/X`; `station:|slot:/A` -> `station:|slot:/A/X`. Only one
// plain slot name is accepted, so the join never produces `//`.
func ChildOrd(parentOrd, name string) (string, error) {
	if name == "" || name == ".." || strings.ContainsAny(name, "/|") {
		return "", fmt.Errorf("ORD child name must be one plain slot name, got %s", pyjson.StrRepr(name))
	}
	return strings.TrimRight(parentOrd, "/") + "/" + name, nil
}

func mustChild(parentOrd, name string) string {
	o, err := ChildOrd(parentOrd, name)
	if err != nil {
		panic(err)
	}
	return o
}

// inFolder is ChildOrd under the scratch folder, for the scenario's fixed plain names.
func inFolder(name string) string { return mustChild(FolderOrd, name) }

package readtools

import (
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/safety"
)

// The local file of an inventory snapshot (station-audits S8): its name, layout, writer and reader.
// It lives under <state-dir>/exports/ beside the other exports, with a name the server makes.

// inventoryFormat names the layout of an inventory snapshot file.
const inventoryFormat = "mcp-n4.inventory-snapshot/v1"

// inventoryFileName is what a snapshot name must look like: one plain file name ending .json.
var inventoryFileName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._$-]{0,159}\.json$`)

// invComponent is one component of a snapshot: its slot path, type, handle and recorded slots.
type invComponent struct {
	path, typ, handle string
	slots             *pyjson.Object // slot name -> value text (nil value: not sent, at its default)
}

// invSnapshot is one inventory snapshot as stored.
type invSnapshot struct {
	name, station, base, takenAt string
	depth                        int
	truncated, contracts         bool
	chosen                       []string // the slots argument (nil: configured properties)
	components                   []invComponent
}

func (s *invSnapshot) json() *pyjson.Object {
	comps := make([]any, len(s.components))
	for i, c := range s.components {
		comps[i] = obj("path", c.path, "type", c.typ, "handle", c.handle, "slots", c.slots)
	}
	var chosen any
	if s.chosen != nil {
		chosen = strs(s.chosen)
	}
	return obj("format", inventoryFormat, "station", s.station, "base", s.base, "depth", s.depth,
		"taken_at", s.takenAt, "truncated", s.truncated, "contracts", s.contracts, "slots", chosen,
		"components", comps)
}

// inventoryName is inventory-<station>-<UTC time to the millisecond>.json (exportFileName's rule).
func inventoryName(station string, at time.Time) string {
	return "inventory-" + strings.TrimSuffix(exportFileName(station, at), ".csv") + ".json"
}

// saveInventory writes s as a new 0600 file under <state-dir>/exports/ and returns its path and size.
func (t *Tools) saveInventory(s *invSnapshot) (string, int, error) {
	path, err := t.outputPath(s.name, exportSubdir, inventoryFileName, "(the snapshot name) ")
	if err != nil {
		return "", 0, err
	}
	data, err := pyjson.Marshal(s.json(), pyjson.Options{})
	if err != nil {
		return "", 0, err
	}
	if err := writePrivate(path, func(f *os.File) error { _, err := f.Write(data); return err }); err != nil {
		return "", 0, writeFailed(path, err)
	}
	return path, len(data), nil
}

// loadInventory reads the snapshot called name from <state-dir>/exports/. A name with a path
// separator or "..", a name that is no plain file there, or a foreign file is a ToolError.
func (t *Tools) loadInventory(arg, name string) (*invSnapshot, error) {
	if !strings.HasSuffix(name, ".json") {
		name += ".json"
	}
	if strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") || !inventoryFileName.MatchString(name) {
		return nil, toolError("%s must be the name of a snapshot n4_inventory_snapshot wrote under "+
			"<state-dir>/%s/ (a plain file name, no '..' and no path separator), got %s",
			arg, exportSubdir, pyjson.StrRepr(name))
	}
	path := safety.Join(t.stateDir(), exportSubdir, name)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, toolError("%s: no inventory snapshot %s under <state-dir>/%s/: take it with n4_inventory_snapshot",
			arg, pyjson.StrRepr(name), exportSubdir)
	}
	data, err := os.ReadFile(path)
	v, _ := pyjson.Unmarshal(data)
	o, ok := v.(*pyjson.Object)
	if err != nil || !ok || lookup(o, "format") != inventoryFormat {
		return nil, toolError("%s: %s is not a %s file", arg, path, inventoryFormat)
	}
	return parseInventory(name, o), nil
}

// parseInventory is the snapshot o holds; a field of the wrong kind reads as empty.
func parseInventory(name string, o *pyjson.Object) *invSnapshot {
	s := &invSnapshot{name: name, truncated: lookup(o, "truncated") == true, contracts: lookup(o, "contracts") == true}
	s.station, _ = lookup(o, "station").(string)
	s.base, _ = lookup(o, "base").(string)
	s.takenAt, _ = lookup(o, "taken_at").(string)
	if chosen, ok := lookup(o, "slots").([]any); ok {
		s.chosen = []string{}
		for _, c := range chosen {
			if n, ok := c.(string); ok {
				s.chosen = append(s.chosen, n)
			}
		}
	}
	raw, _ := lookup(o, "components").([]any)
	for _, r := range raw {
		c, ok := r.(*pyjson.Object)
		if !ok {
			continue
		}
		var ic invComponent
		ic.path, _ = lookup(c, "path").(string)
		ic.typ, _ = lookup(c, "type").(string)
		ic.handle, _ = lookup(c, "handle").(string)
		if ic.slots, _ = lookup(c, "slots").(*pyjson.Object); ic.slots == nil {
			ic.slots = pyjson.NewObject()
		}
		s.components = append(s.components, ic)
	}
	return s
}

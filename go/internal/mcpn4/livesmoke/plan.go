package livesmoke

import (
	"fmt"
	"strings"
)

// row is one scenario step: Run executes it, PlanRows prints it (the single source of both).
type row struct {
	name, tool string
	fn         stepFn
	optional   bool // not required: its verdict does not gate the exit code
	probe      bool
}

// plan is the ordered scenario; the step builders are lazy, so building it contacts nothing.
func (s *Scenario) plan() []row {
	rows := []row{{name: "connect", tool: "n4_connect", fn: s.connect},
		{name: "folder absent before", tool: "n4_navigate", fn: s.absentBefore},
		{name: "create Folder " + Scratch, tool: "n4_create_component", fn: s.create(Scratch, "baja:Folder", nil, RootOrd)}}
	for i := range components {
		c := &components[i]
		rows = append(rows, row{name: "create " + c.Name, tool: "n4_create_component",
			fn: s.create(c.Name, c.Type, c, FolderOrd)})
	}
	for _, l := range links {
		rows = append(rows, row{name: fmt.Sprintf("link %s.%s->%s.%s", l.Src, l.SrcSlot, l.Dst, l.DstSlot),
			tool: "n4_create_link", fn: s.link(l)})
	}
	return append(rows,
		row{name: "find_dangling_outputs == [Cooling]", tool: "n4_find_dangling_outputs", fn: s.dangling},
		row{name: "set Temp=30", tool: "n4_invoke_action", fn: s.setValue("Temp", 30)},
		row{name: "set Setpoint=25", tool: "n4_invoke_action", fn: s.setValue("Setpoint", 25)},
		row{name: "read outputs == true", tool: "n4_read_slots", fn: s.expectOutputs(true)},
		row{name: "set Temp=20", tool: "n4_invoke_action", fn: s.setValue("Temp", 20)},
		row{name: "read outputs == false", tool: "n4_read_slots", fn: s.expectOutputs(false)},
		row{name: "save (folder present)", tool: "n4_save_station", fn: s.Save(true)},
		row{name: "remove folder", tool: "n4_remove_component", fn: s.remove(true)},
		row{name: "rollback folder removal", tool: "n4_rollback", fn: s.rollback, optional: true, probe: true},
		row{name: "remove folder (again)", tool: "n4_remove_component", fn: s.removeAgain, optional: true},
		row{name: "save (folder removed)", tool: "n4_save_station", fn: s.Save(false)},
		row{name: "folder absent", tool: "n4_navigate", fn: s.folderAbsent})
}

// PlanRows is the (name, tool) of every step Run executes, from the same scenario definition.
func PlanRows() [][2]string {
	var out [][2]string
	for _, r := range NewScenario(nil, "NAME", 0, nil).plan() {
		out = append(out, [2]string{r.name, r.tool})
	}
	return out
}

// PlanText is what plan-only mode prints.
func PlanText() string {
	comps := make([]string, len(components))
	for i, c := range components {
		comps[i] = fmt.Sprintf("%s (%s)", c.Name, c.Type)
	}
	lines := []string{"live-smoke: plan only (nothing is sent; pass --apply to execute)",
		fmt.Sprintf("scratch folder: %s, write scopes: %s", FolderOrd, strings.Join(WriteScopes(), ", ")),
		"components: " + strings.Join(comps, ", "),
		"every write: dry run -> confirmation token -> execute -> verdict", ""}
	for i, r := range PlanRows() {
		lines = append(lines, fmt.Sprintf("%2d. %-42s %s", i+1, r[0], r[1]))
	}
	lines = append(lines, "", "probe-only (not required): "+strings.Join(ProbeOnly, "; "))
	return strings.Join(lines, "\n") + "\n"
}

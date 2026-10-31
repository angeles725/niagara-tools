package portparity

import (
	"fmt"
	"path"
)

// Severities of a finding.
const (
	Fail = "FAIL"
	Warn = "WARN"
	Info = "INFO"
)

// Finding is one row of the report: a missing item (Site in the reference tree) or a waiver problem (Site
// in the parity file).
type Finding struct {
	Severity string `json:"severity"`
	Site     string `json:"site"`
	Kind     Kind   `json:"kind,omitempty"`
	Name     string `json:"name,omitempty"`
	Message  string `json:"message"`
}

// Counts summarises a check.
type Counts struct {
	RefFiles int `json:"ref_files"`
	RefIDs   int `json:"ref_ids"`
	RefFuncs int `json:"ref_functions"`
	Missing  int `json:"missing"`
	Waived   int `json:"waived"`   // explicit and cascaded
	Cascaded int `json:"cascaded"` // waived via a file waiver (INFO rows)
	Fail     int `json:"fail"`
	Warn     int `json:"warn"`
	Info     int `json:"info"`
}

// Result is a check's findings, missing items first (Missing order), then waiver rows in line order.
type Result struct {
	Findings []Finding `json:"findings"`
	Counts   Counts    `json:"counts"`
}

// Failed reports whether any finding is a FAIL.
func (r Result) Failed() bool { return r.Counts.Fail > 0 }

// Check judges every reference item the port lacks against wf. A missing item a waiver names is waived; an
// id or function that only file-waived reference files define is waived via that file (an INFO row); any
// other is a FAIL. Each waiver adds at most one row: malformed (FAIL), no decision reference (FAIL:
// dropping a reference feature needs a recorded user decision), a rename whose target the port lacks
// (FAIL), or stale, matching no missing item (WARN). waiverPath labels the waiver rows.
func Check(ref, port Tree, wf WaiverFile, waiverPath string) Result {
	res := Result{Counts: Counts{RefFiles: len(ref.Files), RefIDs: len(ref.IDs), RefFuncs: len(ref.Funcs)}}
	byKey := map[string]*Waiver{}
	used := map[int]bool{}
	for i := range wf.Waivers {
		w := &wf.Waivers[i]
		if w.Problem != "" {
			continue
		}
		k := w.Kind
		if k == KindRename {
			k = KindFile
		}
		if _, dup := byKey[key(k, w.Name)]; !dup {
			byKey[key(k, w.Name)] = w
		}
	}
	// File waivers with a decision cascade to the ids and functions only their file defines.
	waivedFiles := map[string]bool{}
	for _, w := range wf.Waivers {
		if w.Problem == "" && w.Kind == KindFile && w.Decision != "" {
			waivedFiles[w.Name] = true
		}
	}
	for _, it := range Missing(ref, port) {
		res.Counts.Missing++
		// Mutation: PP-waiver-match -- ignoring waivers FAILs every waived absence (the clean fixture exits 1)
		if w := byKey[key(it.Kind, it.Name)]; w != nil {
			used[w.Line] = true
			res.Counts.Waived++
			continue
		}
		// Mutation: PP-cascade-shared -- cascading a name another reference file also defines hides that file's drop
		if f := onlyIn(ref.Defs[it.Kind][it.Name], waivedFiles); f != "" {
			res.Counts.Waived++
			res.Counts.Cascaded++
			res.add(Finding{Info, it.Site(), it.Kind, it.Name,
				fmt.Sprintf("%s %s waived via file %s (defined only in that reference file)", it.Kind, it.Name, f)})
			continue
		}
		res.add(Finding{Fail, it.Site(), it.Kind, it.Name,
			fmt.Sprintf("missing %s %s (in the reference, not in the port, no waiver)", it.Kind, it.Name)})
	}
	for _, w := range wf.Waivers {
		site := fmt.Sprintf("%s:%d", waiverPath, w.Line)
		switch {
		case w.Problem != "":
			res.add(Finding{Fail, site, "", "", w.Problem})
		case w.Kind == KindRename && port.Files[w.To].Name == "":
			res.add(Finding{Fail, site, w.Kind, w.Name, fmt.Sprintf("rename target %s is not a file of the port", w.To)})
		// Mutation: PP-no-decision -- accepting a waiver without [decision: ...] lets a feature drop silently
		case w.Kind != KindRename && w.Decision == "":
			res.add(Finding{Fail, site, w.Kind, w.Name, fmt.Sprintf(
				"waiver %s %s has no [decision: <ref>] (dropping a reference feature needs a recorded user decision)", w.Kind, w.Name)})
		case !used[w.Line]:
			res.add(Finding{Warn, site, w.Kind, w.Name, fmt.Sprintf("stale waiver: %s %s matches no missing item (remove the line)", w.Kind, w.Name)})
		}
	}
	return res
}

// onlyIn returns the waived file basename when every defining file of a name has that one basename, else "".
func onlyIn(defs []string, waived map[string]bool) string {
	base := ""
	for _, d := range defs {
		b := path.Base(d)
		if !waived[b] || base != "" && b != base {
			return ""
		}
		base = b
	}
	return base
}

func key(k Kind, name string) string { return string(k) + "\x00" + name }

func (r *Result) add(f Finding) {
	r.Findings = append(r.Findings, f)
	switch f.Severity {
	case Fail:
		r.Counts.Fail++
	case Warn:
		r.Counts.Warn++
	default:
		r.Counts.Info++
	}
}

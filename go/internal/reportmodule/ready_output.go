package reportmodule

import (
	"encoding/json"
	"fmt"
	"io"
)

// The output of report-module --ready: the rows of each built jar, then its verdict line (READY TO LOAD or NOT
// READY with the failing rows) and, when ready, the read-back to run after loading it; or all of it as one JSON
// object with --json.

const (
	verdictReady    = "READY"
	verdictNotReady = "NOT READY"
)

// readyRow is one check of a jar.
type readyRow struct {
	Status string `json:"status"`
	Check  string `json:"check"`
	Detail string `json:"detail"`
}

// readyJar is the assessment of one artifact's built jar.
type readyJar struct {
	Artifact string     `json:"artifact"`
	Jar      string     `json:"jar"`
	Module   string     `json:"module"`  // the module part name of META-INF/module.xml ("" unreadable)
	Version  string     `json:"version"` // its vendorVersion
	Niagara  string     `json:"niagara"` // the Niagara the line names: the target, else the compile home, else ">= floor"
	SHA256   string     `json:"sha256"`
	Rows     []readyRow `json:"rows"`
	Verdict  string     `json:"verdict"`
	Line     string     `json:"line"`
	Readback string     `json:"readback,omitempty"` // only when ready
	notBuilt bool       // an artifact with no built jar: its verdict line says so instead of a version
}

// readyJSON is the --json document.
type readyJSON struct {
	Root    string     `json:"root"`
	Verdict string     `json:"verdict"` // READY only when every jar is
	Jars    []readyJar `json:"jars"`
}

// readback is what the operator or the coordinator runs on the station after loading the jar (this tool never
// contacts a station): the loaded module list of mcp-n4, which must name the part at this version.
const readback = `after loading, read back (mcp-n4, this tool never contacts the station): n4_station_runtime -> ` +
	`loaded_modules has name %q version %q and mismatches is empty (restart_needed false); ` +
	`or n4_spy_page(path="sysManagers/moduleManager") lists it`

// conclude sets the verdict, the line and the read-back of j from its rows.
func (j *readyJar) conclude() {
	var bad []readyRow
	for _, r := range j.Rows {
		if r.Status == failed || r.Status == errored {
			bad = append(bad, r)
		}
	}
	if len(bad) == 0 {
		j.Verdict = verdictReady
		j.Line = fmt.Sprintf("READY TO LOAD: %s %s for Niagara %s (sha256 %s)", j.Module, j.Version, j.Niagara, short(j.SHA256))
		j.Readback = fmt.Sprintf(readback, j.Module, j.Version)
		return
	}
	plural := "s"
	if len(bad) == 1 {
		plural = ""
	}
	j.Verdict = verdictNotReady
	j.Line = fmt.Sprintf("NOT READY: %s %s — %d failing row%s:", shownOr(j.Module, j.Artifact), j.versionLabel(), len(bad), plural)
	for _, r := range bad {
		j.Line += fmt.Sprintf("\n  %s  %s  %s  %s", j.Artifact, r.Status, r.Check, r.Detail)
	}
}

// versionLabel is the version the NOT READY line names: "(not built)" for an artifact with no jar, else the
// vendorVersion or "?" when it is unreadable.
func (j *readyJar) versionLabel() string {
	if j.notBuilt {
		return "(not built)"
	}
	return shownOr(j.Version, "?")
}

// writeText prints each jar's rows, its verdict line and, when ready, its read-back.
func writeText(w io.Writer, jars []readyJar) {
	for _, j := range jars {
		for _, r := range j.Rows {
			fmt.Fprintf(w, "%s  %s  %s  %s\n", j.Artifact, r.Status, r.Check, r.Detail)
		}
		fmt.Fprintln(w, j.Line)
		if j.Readback != "" {
			fmt.Fprintln(w, j.Readback)
		}
	}
}

// writeJSON prints the --json document.
func writeJSON(w io.Writer, root string, jars []readyJar) error {
	doc := readyJSON{Root: root, Verdict: verdictReady, Jars: jars}
	for _, j := range jars {
		if j.Verdict != verdictReady {
			doc.Verdict = verdictNotReady
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(doc)
}

// short is the first 12 hex digits of a sha256 (all of it when shorter).
func short(sum string) string {
	if len(sum) > 12 {
		return sum[:12]
	}
	return sum
}

func shownOr(s, alt string) string {
	if s == "" {
		return alt
	}
	return s
}

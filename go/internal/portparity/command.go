package portparity

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// Exit codes of lint-port-parity.
const (
	ExitOK   = 0 // every missing item waived with a decision (WARN rows allowed)
	ExitFail = 1 // any FAIL row
	ExitEnv  = 3 // usage, IO, or a module that declares no reference port
)

// ParityFile is the module-side declaration: a module that ports a reference frontend keeps it at its root.
const ParityFile = ".port-parity"

const usage = "usage: lint-port-parity.sh --ref <dir> --port <dir> [--waivers <file>] [--format text|json]\n" +
	"       lint-port-parity.sh [--format text|json] <module-root>   (reads <module-root>/" + ParityFile + ")\n"

// Main implements `port-parity`. With --ref/--port it diffs the two trees (waivers from --waivers, whose
// ref:/port: headers fill an absent flag, relative to the file's directory). With one <module-root> it reads
// <module-root>/.port-parity: its ref: header is required, its port: header defaults to the module root.
func Main(args []string, s cli.Stdio) int {
	var ref, port, waivers, format, root string
	format = "text"
	flags := map[string]*string{"--ref": &ref, "--port": &port, "--waivers": &waivers, "--format": &format}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if dest, ok := flags[a]; ok {
			if i+1 >= len(args) {
				fmt.Fprint(s.Err, usage)
				return ExitEnv
			}
			i++
			*dest = args[i]
			continue
		}
		if root != "" || len(a) > 1 && a[0] == '-' {
			fmt.Fprint(s.Err, usage)
			return ExitEnv
		}
		root = a
	}
	if format != "text" && format != "json" {
		fmt.Fprintf(s.Err, "port-parity: --format must be text or json, not %s\n", format)
		return ExitEnv
	}
	label := waivers
	if root != "" {
		if ref != "" || port != "" || waivers != "" {
			fmt.Fprint(s.Err, usage)
			return ExitEnv
		}
		waivers, label = filepath.Join(root, ParityFile), ParityFile
		if _, err := os.Stat(waivers); err != nil {
			fmt.Fprintf(s.Err, "port-parity: no %s in %s (the module declares no reference port)\n", ParityFile, root)
			return ExitEnv
		}
	}
	var wf WaiverFile
	if waivers != "" {
		b, err := os.ReadFile(waivers)
		if err != nil {
			fmt.Fprintf(s.Err, "port-parity: cannot read %v\n", err)
			return ExitEnv
		}
		wf = ParseWaivers(string(b))
		dir := filepath.Dir(waivers)
		if ref == "" && wf.Ref != "" {
			ref = resolve(dir, wf.Ref)
		}
		if port == "" {
			if wf.Port != "" {
				port = resolve(dir, wf.Port)
			} else if root != "" {
				port = root
			}
		}
		if root != "" && ref == "" {
			fmt.Fprintf(s.Err, "port-parity: %s: no ref: header\n", waivers)
			return ExitEnv
		}
	}
	if ref == "" || port == "" {
		fmt.Fprint(s.Err, usage)
		return ExitEnv
	}
	rt, err := Scan(ref)
	if err != nil {
		fmt.Fprintf(s.Err, "port-parity: cannot read the reference: %v\n", err)
		return ExitEnv
	}
	pt, err := Scan(port)
	if err != nil {
		fmt.Fprintf(s.Err, "port-parity: cannot read the port: %v\n", err)
		return ExitEnv
	}
	res := Check(rt, pt, wf, label)
	if format == "json" {
		writeJSON(s.Out, ref, port, waivers, res)
	} else {
		writeText(s.Out, res)
	}
	if res.Failed() {
		return ExitFail
	}
	return ExitOK
}

func resolve(dir, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(dir, p)
}

// writeText prints one `<SEV>  lint-port-parity  <site>  <message>` row per finding (the toolbelt row shape
// report-module relays), then a summary line that is not a row.
func writeText(w io.Writer, res Result) {
	for _, f := range res.Findings {
		fmt.Fprintf(w, "%s  lint-port-parity  %s  %s\n", f.Severity, f.Site, f.Message)
	}
	c := res.Counts
	fmt.Fprintf(w, "port-parity: %d files, %d ids, %d functions in the reference; %d missing, %d waived, %d FAIL, %d WARN\n",
		c.RefFiles, c.RefIDs, c.RefFuncs, c.Missing, c.Waived, c.Fail, c.Warn)
}

func writeJSON(w io.Writer, ref, port, waivers string, res Result) {
	if res.Findings == nil {
		res.Findings = []Finding{}
	}
	doc := struct {
		Ref      string    `json:"ref"`
		Port     string    `json:"port"`
		Waivers  string    `json:"waivers"`
		Counts   Counts    `json:"counts"`
		Findings []Finding `json:"findings"`
	}{ref, port, waivers, res.Counts, res.Findings}
	b, _ := json.MarshalIndent(doc, "", "  ")
	fmt.Fprintf(w, "%s\n", b)
}

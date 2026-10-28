// Package rcscan implements rc-scan, the browser-resource lint of a module's rc/ web assets (*.html, *.js,
// *.css): the defects the Java-side lints never see, each row naming a stable check id. It is the port of the
// awk program toolbelt/rc-scan.sh ran, read line by line as gawk read it under LC_ALL=C (bytes, POSIX classes,
// leftmost-longest matches), with the same walk, rows, order and exit codes; the check list and the escape
// hatch are documented in that script's header.
package rcscan

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

const usage = "usage: rc-scan.sh <artifact-dir> [--strict] [--profile hmi|lan|both|unknown] [--legacy]\n"

// sel is the walk of rc-scan.sh: **/rc/** *.html *.js *.css, minus *.min.js, srcTest/ and rc/ext/ (dot and
// *-worktrees sub-directories are never entered).
var sel = lint.Selection{Names: []string{"*.html", "*.js", "*.css"}, Except: []string{"*.min.js"},
	Prune: []string{"srcTest"}, Paths: []string{"*/rc/*"}, PrunePaths: []string{"*/rc/ext"}}

// vendored reports a vendored browser library, <rc>/vendor/**/*.js: lint-vendor-floor.sh owns it (an
// acorn parse at the Chromium 83 floor, its verdict recorded in rc/vendor/THIRD-PARTY.md, types/frontend-standard.md
// § Vendored libraries), and rc-scan's line heuristics only report third-party code the module cannot change
// (issue #459: three.js RenderPass.js / SSAOPass.js). Vendored CSS and HTML stay scanned: no other lint reads them.
func vendored(p string) bool {
	return strings.HasSuffix(p, ".js") && strings.Contains(p, "/rc/vendor/")
}

// options is a parsed command line.
type options struct {
	root           string
	strict, legacy bool
	profile        string
	mode           lint.Modularity // the module map mode above root (lint.ModuleMode); none runs no map check
}

// sevW is the severity of a WARN check (FAIL under --strict); sevFloor that of browser-floor (FAIL under
// --strict or --profile hmi|both); sevData that of datauri-budget (WARN only with --legacy and no --strict).
func (o options) sevW() lint.Severity { return pick(o.strict) }
func (o options) sevFloor() lint.Severity {
	return pick(o.strict || o.profile == "hmi" || o.profile == "both")
}
func (o options) sevData() lint.Severity { return pick(!o.legacy || o.strict) }

// sevMap is the severity of a module-map check (fetch-outside-services, js-file-size): FAIL in a strict module,
// else the WARN check's (a legacy map: WARN, FAIL under --strict).
func (o options) sevMap() lint.Severity {
	if o.mode == lint.ModularityStrict {
		return lint.Fail
	}
	return o.sevW()
}

// mapped reports a module map above the artifact dir (strict or legacy). With none, neither module-map check
// runs, so a map-less module's rows stay what they were before the checks existed.
func (o options) mapped() bool { return o.mode != lint.ModularityNone }

func pick(fail bool) lint.Severity {
	if fail {
		return lint.Fail
	}
	return lint.Warn
}

// Main is the rc-scan command: exit 0 with no FAIL row, 1 with one, 3 on a usage or environment error.
func Main(args []string, s cli.Stdio) int {
	o, msg := parse(args)
	if msg != "" {
		fmt.Fprint(s.Err, msg)
		return 3
	}
	o.mode = lint.ModuleMode(o.root)
	files, err := lint.Walk(o.root, sel)
	if err != nil {
		fmt.Fprintf(s.Err, "rc-scan: cannot list every file under %s: %s\n", o.root, err)
		return 3
	}
	var rows []lint.Finding
	var pages []*source
	for _, p := range files {
		if vendored(p) {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil { // gawk printed its fatal error and the script went on: fail closed instead
			fmt.Fprintf(s.Err, "rc-scan: cannot scan %s: %s\n", p, strerror.Text(err))
			return 3
		}
		src := &source{rel: strings.TrimPrefix(p, o.root+"/"), path: p, ext: p[strings.LastIndexByte(p, '.')+1:], lines: records(data)}
		rows = append(rows, scanFile(src, o)...)
		if src.ext != "css" {
			pages = append(pages, src)
		}
	}
	rows = append(rows, orphanPages(pages, o.sevW())...)
	code := 0
	for _, f := range rows {
		f.Rule = "rc-scan"
		fmt.Fprintln(s.Out, f.Row())
		if f.Severity == lint.Fail {
			code = 1
		}
	}
	return code
}

// parse reads `<artifact-dir> [--strict] [--profile P] [--legacy]` (the directory always first) and returns
// the stderr text of a usage or environment error.
func parse(args []string) (options, string) {
	if len(args) < 1 {
		return options{}, usage
	}
	o := options{root: args[0], profile: "unknown"}
	for i := 1; i < len(args); i++ {
		switch a := args[i]; a {
		case "--strict":
			o.strict = true
		case "--legacy":
			o.legacy = true
		case "--profile":
			if i+1 >= len(args) {
				return options{}, usage
			}
			i++
			switch args[i] {
			case "hmi", "lan", "both", "unknown":
				o.profile = args[i]
			default:
				return options{}, "rc-scan: unknown profile: " + args[i] + "\n"
			}
		default:
			return options{}, "rc-scan: unknown option: " + a + "\n"
		}
	}
	if st, err := os.Stat(o.root); err != nil || !st.IsDir() {
		return options{}, "rc-scan: not a directory: " + o.root + "\n"
	}
	return o, ""
}

// records splits data into awk records: one per "\n", the last one kept when it has no "\n", none for an
// empty file; a "\r" stays part of its record.
func records(data []byte) []string {
	if len(data) == 0 {
		return nil
	}
	lines := strings.Split(string(bytes.TrimSuffix(data, []byte("\n"))), "\n")
	return lines
}

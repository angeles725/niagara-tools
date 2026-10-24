// Package bogaudit is bog-audit: a station config.bog audited against the deployed module source
// (toolbelt/bog-audit.sh, the wrapper, documents CHECK1-20). The bog is read once through xmlsrc (a zipped
// .bog through its file.xml member) and replayed through the state machine of the python line scanner it
// replaces; --source-dir is read as that scanner read it, with javasrc's paren counting.
package bogaudit

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/htmltext"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

const usage = "usage: bog-audit.sh <config.bog|file.xml> --module <MOD>... [--source-dir <DIR>] [--strict]\n"

type options struct {
	bog, srcDir string
	modules     []string
	strict      bool
}

// parseArgs is the bash argument loop: a value option takes the next argument whatever it is.
func parseArgs(args []string) (options, bool) {
	var o options
	var mods []string
	for i := 0; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--module" || a == "--source-dir":
			if i++; i == len(args) {
				return o, false
			}
			if a == "--module" {
				mods = append(mods, args[i])
			} else {
				o.srcDir = args[i]
			}
		case a == "--strict":
			o.strict = true
		case strings.HasPrefix(a, "--") || o.bog != "":
			return o, false
		default:
			o.bog = a
		}
	}
	// "${MODULES[*]}" then python's str.split(): the values joined by spaces, split on whitespace.
	o.modules = strings.FieldsFunc(strings.Join(mods, " "), htmltext.IsSpace)
	return o, o.bog != "" && len(mods) > 0
}

// Main runs bog-audit: exit 0 clean, 1 on a FAIL row, 3 on usage, an unreadable or malformed bog, or no
// component of the named modules.
func Main(args []string, s cli.Stdio) int {
	start := time.Now()
	o, ok := parseArgs(args)
	if !ok {
		fmt.Fprint(s.Err, usage)
		return 3
	}
	doc, code := readBog(o.bog, s.Err)
	if doc == nil {
		return code
	}
	a := &audit{out: s.Out, g: build(doc), own: map[string]bool{}, strict: o.strict}
	for _, m := range o.modules {
		a.own[m] = true
	}
	if o.srcDir != "" {
		a.src = scanSources(o.srcDir)
	}
	if !a.run() {
		mods := append([]string(nil), o.modules...)
		sort.Strings(mods)
		fmt.Fprintf(s.Err, "bog-audit: no components found for modules: %s\n", strings.Join(dedupe(mods), ", "))
		return 3
	}
	fmt.Fprintf(s.Out, "# bog-audit: parse time %.3fs\n", time.Since(start).Seconds())
	if a.failed {
		return 1
	}
	return 0
}

func dedupe(sorted []string) []string {
	out := sorted[:0]
	for i, m := range sorted {
		if i == 0 || m != sorted[i-1] {
			out = append(out, m)
		}
	}
	return out
}

// readBog reads a bare *.xml or a zipped bog (its file.xml), with the bash script's messages.
func readBog(path string, stderr io.Writer) (*xmlsrc.Doc, int) {
	fail := func(format string, args ...any) (*xmlsrc.Doc, int) {
		fmt.Fprintf(stderr, "bog-audit: "+format+"\n", args...)
		return nil, 3
	}
	st, err := os.Stat(path)
	if err != nil || !st.Mode().IsRegular() {
		return fail("not a file: %s", path)
	}
	bare := strings.HasSuffix(path, ".xml")
	if bare && st.Size() == 0 {
		return fail("empty file.xml in %s", path)
	}
	if !bare && !isZip(path) {
		return fail("cannot extract file.xml from %s", path)
	}
	doc, err := xmlsrc.Open(path)
	var xe *xmlsrc.Error
	switch {
	case err == nil:
		return doc, 0
	case errors.As(err, &xe) && xe.Line == 1 && xe.Col == 1 && xe.Msg == "no root element":
		return fail("empty file.xml in %s", path) // a zero-byte file.xml member
	case xe != nil:
		return fail("%s", xe)
	case bare:
		return fail("cannot read %s: %v", path, err)
	}
	return fail("cannot extract file.xml from %s", path)
}

// isZip reports whether path starts with a zip local-file header; any other content is no bog.
func isZip(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return true // xmlsrc.Open names the read error
	}
	defer f.Close()
	head := make([]byte, 4)
	n, _ := io.ReadFull(f, head)
	return string(head[:n]) == "PK\x03\x04"
}

// audit runs the checks over one graph and prints their rows.
type audit struct {
	out    io.Writer
	g      *graph
	src    *sources // nil without --source-dir
	own    map[string]bool
	strict bool
	failed bool
}

func (a *audit) emit(check, status, path, detail string) {
	fmt.Fprintf(a.out, "%s  %s  %s  %s\n", check, status, path, detail)
	if status == "FAIL" {
		a.failed = true
	}
}

// run prints every check in the bash order; false when no component belongs to an own module.
func (a *audit) run() bool {
	a.duplicates()
	a.orphans()
	if !a.inventory() {
		return false
	}
	a.hoaLeftover()
	a.proxyLinkSafety()
	a.sourceChecks()
	a.stationLogic()
	return true
}

func (a *audit) ownComp(h string) *comp {
	if c := a.g.comp(h); c != nil && a.own[c.module] {
		return c
	}
	return nil
}

// byPath is the components sorted by path, ties in dict order (Python's stable sorted()).
func (a *audit) byPath() []*comp {
	cs := make([]*comp, 0, len(a.g.comps.keys))
	for _, h := range a.g.comps.keys {
		cs = append(cs, a.g.comp(h))
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].path < cs[j].path })
	return cs
}

// isOff is Python's `v.lower() in (”, 'false', '0', 'null')`.
func isOff(v string) bool {
	switch strings.ToLower(v) {
	case "", "false", "0", "null":
		return true
	}
	return false
}

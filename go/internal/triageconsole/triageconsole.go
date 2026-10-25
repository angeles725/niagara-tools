// Package triageconsole is the Go port of toolbelt/triage-console.sh: own-module exception triage over station
// console logs. Three attribution channels make an own row (C1 an own stack frame, C2 an own logger tag or the
// own-package path in the message, C3 a [sys] fatal / [sys.xml] slot drift / [sys.registry] missing own class);
// with --site a foreign WARNING/SEVERE block of a known site-fault shape becomes an informational SITE row.
// Output and exit are the bash script's byte for byte (goldens in testdata/), apart from the truth-* cases.
package triageconsole

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/awkorder"
	"github.com/angeles725/niagara-tools/go/internal/strerror"
)

// The defaults of --package, --tag and --module-prefix come from the wrapper toolbelt/triage-console.sh, which
// keeps the kit's lists; run directly, the binary defaults to empty lists.
const (
	DefaultPackageEnv      = "TRIAGE_CONSOLE_DEFAULT_PACKAGE"
	DefaultTagsEnv         = "TRIAGE_CONSOLE_DEFAULT_TAGS"
	DefaultModulePrefixEnv = "TRIAGE_CONSOLE_DEFAULT_MODULE_PREFIX"
)

// options are the parsed arguments; files are the operands in order, --console-dir appended last.
type options struct {
	pkg, tags, prefixes string
	site                bool
	files               []string
}

// Main runs triage-console. Exit: 0 no own row, 1 any own row, 3 usage/env.
func Main(args []string, s cli.Stdio) int {
	o, err := parse(args, s)
	if err != "" {
		fmt.Fprint(s.Err, err)
		return 3
	}
	files, dirArg := list(o.files, s)
	if len(files) == 0 {
		fmt.Fprint(s.Err, "usage: triage-console.sh [...] <console.txt>...\n")
		return 3
	}
	subject := basename(dirname(files[0]))
	if dirArg != "" {
		subject = basename(dirArg)
	} else if len(files) == 1 {
		subject = basename(files[0])
	}
	// The bash handed these to awk with -v, which processes backslash escapes in that order.
	esc := &escaper{warn: func(w string) { fmt.Fprint(s.Err, w) }}
	pkg, tags, prefixes, subject := esc.value(o.pkg), esc.value(o.tags), esc.value(o.prefixes), esc.value(subject)
	contents := make([]string, len(files))
	for i, f := range files {
		b, err := os.ReadFile(f)
		if err != nil { // the bash failed open here (awk's fatal open skipped every row)
			fmt.Fprintf(s.Err, "triage-console: cannot read %s: %s\n", f, strerror.Text(err))
			return 3
		}
		contents[i] = string(b)
	}
	sc := newScanner(pkg, tags, prefixes, o.site)
	for _, c := range contents {
		sc.file(c)
	}
	sc.flush()
	own, site := sc.rows(subject)
	for _, r := range append(own, site...) {
		fmt.Fprintln(s.Out, r)
	}
	if len(own) > 0 {
		return 1
	}
	return 0
}

func parse(args []string, s cli.Stdio) (options, string) {
	o := options{pkg: s.Env(DefaultPackageEnv), tags: s.Env(DefaultTagsEnv), prefixes: s.Env(DefaultModulePrefixEnv)}
	consoleDir := ""
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--package", "--tag", "--module-prefix", "--console-dir":
			if i+1 == len(args) {
				return o, fmt.Sprintf("triage-console: %s requires a value\n", a)
			}
			i++
			switch a {
			case "--package":
				o.pkg = args[i]
			case "--tag":
				o.tags = args[i]
			case "--module-prefix":
				o.prefixes = args[i]
			default:
				consoleDir = args[i]
			}
		case "--site":
			o.site = true
		case "--":
			i = len(args) // the bash broke out of its loop here: later operands are dropped
		default:
			if strings.HasPrefix(a, "-") {
				return o, fmt.Sprintf("triage-console: unknown option: %s\n", a)
			}
			o.files = append(o.files, a)
		}
	}
	if consoleDir != "" {
		o.files = append(o.files, consoleDir)
	}
	return o, ""
}

// list expands the operands as the bash did: a directory (symlink followed) contributes its own regular *.txt
// files outside dot paths, sorted (find -H DIR -maxdepth 1 -not -path '*/.*' -type f -name '*.txt' | sort -z),
// a regular file itself, anything else nothing. dirArg is the last directory operand.
func list(operands []string, s cli.Stdio) (files []string, dirArg string) {
	for _, arg := range operands {
		st, err := os.Stat(arg)
		switch {
		case err != nil:
		case st.IsDir():
			dirArg = arg
			entries, err := os.ReadDir(arg)
			if err != nil {
				fmt.Fprintf(s.Err, "find: %s: %s\n", strerror.FindQuote(arg), strerror.Text(err))
			}
			var found []string
			for _, e := range entries {
				p := arg + "/" + e.Name()
				if strings.HasSuffix(arg, "/") {
					p = arg + e.Name()
				}
				if !strings.Contains(p, "/.") && e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".txt") {
					found = append(found, p)
				}
			}
			sort.Strings(found)
			files = append(files, found...)
		case st.Mode().IsRegular():
			files = append(files, arg)
		}
	}
	return files, dirArg
}

// basename and dirname are the coreutils commands (no path cleaning: x/y/.. names "..").
func basename(p string) string {
	if t := strings.TrimRight(p, "/"); t != "" {
		return t[strings.LastIndexByte(t, '/')+1:]
	}
	if p == "" {
		return ""
	}
	return "/"
}

func dirname(p string) string {
	t := strings.TrimRight(p, "/")
	i := strings.LastIndexByte(t, '/')
	if t == "" || i < 0 {
		if strings.HasPrefix(p, "/") {
			return "/"
		}
		return "."
	}
	if d := strings.TrimRight(t[:i], "/"); d != "" {
		return d
	}
	return "/"
}

// rows renders the own rows in gawk for-in order and the SITE rows count descending, then by their text up
// to a tab, then by the whole line (the bash's sort -t TAB -k2,2nr -k3,3 under LC_ALL=C). A NUL byte stays in
// the grouping key but is dropped from the printed row, as the bash's command substitution dropped it.
func (sc *scanner) rows(subject string) (own, site []string) {
	for _, k := range awkorder.Order(sc.order) {
		g := sc.groups[k]
		own = append(own, noNUL(fmt.Sprintf("FAIL  triage-console  %s  %dx %s -> %s  %s %s @ %s",
			subject, g.cnt, g.first, g.last, g.level, g.msg, g.frame)))
	}
	type siteRow struct {
		cnt  int
		line string
	}
	var rs []siteRow
	for _, g := range sc.sites {
		rs = append(rs, siteRow{g.cnt, noNUL(fmt.Sprintf("SITE  triage-console  %s  %s  %dx %s -> %s  %s [%s] %s (distinct %d)",
			subject, g.cls, g.cnt, g.first, g.last, g.level, g.tag, g.msg, g.dist))})
	}
	sort.Slice(rs, func(i, j int) bool {
		a, b := rs[i], rs[j]
		if a.cnt != b.cnt {
			return a.cnt > b.cnt
		}
		if ka, kb := upToTab(a.line), upToTab(b.line); ka != kb {
			return ka < kb
		}
		return a.line < b.line
	})
	for _, r := range rs {
		site = append(site, r.line)
	}
	return own, site
}

func upToTab(s string) string {
	if i := strings.IndexByte(s, '\t'); i >= 0 {
		return s[:i]
	}
	return s
}

func noNUL(s string) string { return strings.ReplaceAll(s, "\x00", "") }

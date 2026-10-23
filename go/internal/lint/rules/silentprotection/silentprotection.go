// Package silentprotection is lint-silent-protection: a protection trip that forces an output off or
// sheds a stage with no operator surface (no SUMMARY/OPERATOR status-or-reason slot written on its path,
// no console alarm). Port of the awk in toolbelt/lint-silent-protection.sh; the wrapper keeps the
// user-facing contract and its header documents the rule and its mutation pins.
//
// It is a lint.TreeRule although its rows are ordered by file: every file's check reads four indexes
// built from the whole tree first (surface slots, effect slots, fields written to a surface slot, alarm
// classes), so no file can be reported before every file was read. Matching keeps gawk's POSIX
// leftmost-longest semantics (regexp Longest) and C-locale bytes.
package silentprotection

import (
	"regexp"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name printed in every row.
const ID = "lint-silent-protection"

// Spec registers the rule with the argument contract of toolbelt/lint-silent-protection.sh: flags, then
// one root (extra arguments ignored), a usage error prints the usage line alone and exits 3, and a root
// with no .java file is the K20 ERROR row. The walk stops on an unreadable directory (scan_files_or_exit).
func Spec() lint.Spec {
	return lint.Spec{
		Tree:       rule{},
		Mode:       lint.First,
		Usage:      "usage: lint-silent-protection.sh [--strict] <java-src-dir>",
		Select:     lint.Selection{Names: []string{"*.java"}},
		EmptyError: lint.EmptyRow{Msg: "no Java sources found (K20: wrong path or empty scaffold)"},
		BareUsage:  true,
	}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

func rx(s string) *regexp.Regexp { re := regexp.MustCompile(s); re.Longest(); return re }

// space is awk's [[:space:]] in the C locale.
const space = " \t\n\v\f\r"

var (
	ident     = rx(`^[A-Za-z_][A-Za-z0-9_]*`)
	identOnly = rx(`^[A-Za-z_][A-Za-z0-9_]*$`)
	dotIdent  = rx(`\.[A-Za-z_][A-Za-z0-9_]*$`)
	propName  = rx(`name[[:space:]]*=[[:space:]]*"([^"]*)"`)
	propFlags = rx(`flags[[:space:]]*=[[:space:]]*`)
	flagsVal  = rx(`^[^,)]*`)
	className = rx(`class[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)`)
	wsRun     = rx(`[[:space:]]+`)
	rowFields = rx(`[[:space:]]{2,}`)
)

// CheckTree builds the tree-wide indexes, then checks every file in walk order and keeps the first row
// of each dedupe key.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	ix := buildIndex(files)
	var out []lint.Finding
	seen := map[string]bool{}
	for _, f := range files {
		for _, fd := range ix.check(f) {
			fd.Path = f.Path
			if k := dedupeKey(fd); !seen[k] {
				seen[k] = true
				out = append(out, fd)
			}
		}
	}
	return out, nil
}

// dedupeKey is the bash dedupe key: the third field of the printed row split on runs of two or more
// blanks (the "<file>:<line>" column when the path holds no such run), or the whole row.
func dedupeKey(fd lint.Finding) string {
	fd.Rule = ID
	if fd.Severity == lint.Unset {
		fd.Severity = lint.Warn
	}
	row := fd.Row()
	if a := rowFields.Split(row, -1); len(a) >= 3 {
		return a[2]
	}
	return row
}

// index is what the bash passes 0, 0b and 1 computed over every file before the per-file check.
type index struct {
	surfSlots   []slot   // awk split of the sorted unique surface slot names
	effectSlots []slot   // awk split of the sorted unique effect slot names
	surfFields  []string // fields written to a surface slot, sorted unique
	surfList    string   // surfFields joined as the bash SURF_WRITE_FIELDS (in_list)
	alarmList   string   // alarm class names joined as the bash ALARM_CLASSES (in_list)
}

func buildIndex(files []*javasrc.File) *index {
	var surf, effect []string
	for _, f := range files {
		properties(f.Raw, func(name, flags string) {
			// bash: IFS='|' read -r pname pflags splits "<name>|<flags>" at its first '|'
			if i := strings.IndexByte(name, '|'); i >= 0 {
				name, flags = name[:i], name[i+1:]+"|"+flags
			}
			if name == "" {
				return
			}
			if allowlisted(name) && (strings.Contains(flags, "SUMMARY") || strings.Contains(flags, "OPERATOR")) {
				surf = append(surf, name)
			} else {
				effect = append(effect, name)
			}
		})
	}
	ix := &index{surfSlots: slots(awkSplit(joinSorted(surf))), effectSlots: slots(awkSplit(joinSorted(effect)))}
	fields := map[string]bool{}
	for _, f := range files {
		ix.surfaceWrites(f.Raw, fields)
	}
	for fld := range fields {
		ix.surfFields = append(ix.surfFields, fld)
	}
	ix.surfList = joinSorted(ix.surfFields)
	sort.Strings(ix.surfFields)
	var alarms []string
	for _, f := range files {
		if cn := alarmClass(f.Stripped()); cn != "" {
			alarms = append(alarms, cn)
		}
	}
	ix.alarmList = joinSorted(alarms)
	return ix
}

// properties is pass 0: every @NiagaraProperty annotation of the raw lines (comments included), single-
// or multi-line, closed when its parentheses balance; it calls emit with the name and the flags text.
// One annotation is read at a time, so a second one starting on the line that closes the first is missed.
func properties(raw []string, emit func(name, flags string)) {
	in, first := false, false
	var buf, name string
	depth, paren := 0, false
	count := func(s string) {
		depth += strings.Count(s, "(") - strings.Count(s, ")")
		paren = paren || strings.Contains(s, "(")
	}
	for _, ln := range raw {
		if !in && strings.Contains(ln, "@NiagaraProperty") {
			in, buf, name, first = true, ln, "", true
			depth, paren = 0, false
			count(ln)
		}
		if in && !first {
			buf += " " + ln
			count(" " + ln)
		}
		if !in {
			continue
		}
		first = false
		if name == "" {
			if m := propName.FindStringSubmatch(buf); m != nil {
				name = m[1]
			}
		}
		if depth <= 0 && paren {
			flags := ""
			if loc := propFlags.FindStringIndex(buf); loc != nil {
				flags = flagsVal.FindString(buf[loc[1]:])
			}
			if name != "" {
				emit(name, flags)
			}
			in, buf, name = false, "", ""
		}
	}
}

// allowlisted is the bash surface-name allowlist, matched on the lower-cased name.
func allowlisted(name string) bool {
	n := asciiLower(name)
	if strings.Contains(n, "skip") {
		return true
	}
	for _, s := range []string{"alarm", "fault", "reason", "status", "mismatch", "stuck", "available", "fallback"} {
		if strings.HasSuffix(n, s) {
			return true
		}
	}
	return false
}

// surfaceWrites is pass 1 over raw lines (comments included): the field named by the argument of every
// get<Slot>().setValue(<arg>) and set<Slot>(<arg>) of a surface slot.
func (ix *index) surfaceWrites(raw []string, fields map[string]bool) {
	for _, ln := range raw {
		for _, s := range ix.surfSlots {
			pos := strings.Index(ln, s.setValueCall)
			if pos >= 0 {
				arg := stripBlanks(cutAt(ln[pos+len(s.setValueCall):], ")"))
				if m := dotIdent.FindString(arg); m != "" {
					fields[m[1:]] = true
				} else if identOnly.MatchString(arg) {
					fields[arg] = true
				}
			}
			if pos2 := strings.Index(ln, s.setCall); pos2 >= 0 && pos < 0 {
				arg2 := stripBlanks(cutAt(ln[pos2+len(s.setCall):], ",)"))
				if identOnly.MatchString(arg2) {
					fields[arg2] = true
				}
			}
		}
	}
}

// alarmClass is pass 0b: the first declared class name of a file that carries a console alarm, Pattern A
// (BAlarmSourceExt or BAlarmRecord) or Pattern B (implements BIAlarmSource on one line, plus
// newOffnormalAlarm or new AlarmSupport(), or "" otherwise. It reads the comment-stripped lines.
func alarmClass(stripped []string) string {
	cn := ""
	var ha, bi, bn, bc bool
	for _, s := range stripped {
		if cn == "" {
			if m := className.FindStringSubmatch(s); m != nil {
				cn = m[1]
			}
		}
		ha = ha || strings.Contains(s, "BAlarmSourceExt") || strings.Contains(s, "BAlarmRecord")
		bi = bi || (strings.Contains(s, "implements") && strings.Contains(s, "BIAlarmSource"))
		bn = bn || strings.Contains(s, "newOffnormalAlarm")
		bc = bc || strings.Contains(s, "new AlarmSupport(")
	}
	if cn != "" && (ha || (bi && (bn || bc))) {
		return cn
	}
	return ""
}

// joinSorted is `sort -u | tr '\n' ' '` in the C locale: the unique names in byte order, each followed
// by one space.
func joinSorted(names []string) string {
	sort.Strings(names)
	var b strings.Builder
	for i, n := range names {
		if i > 0 && n == names[i-1] {
			continue
		}
		b.WriteString(n)
		b.WriteByte(' ')
	}
	return b.String()
}

// awkSplit is awk's split(s, a, " "): the fields between runs of blanks.
func awkSplit(s string) []string {
	var out []string
	for _, w := range wsRun.Split(strings.Trim(s, space), -1) {
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

// inList is the bash in_list(): word between single spaces in " " list " ".
func inList(word, list string) bool { return strings.Contains(" "+list+" ", " "+word+" ") }

func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

// cap1 is toupper(substr(s,1,1)) substr(s,2) in the C locale.
func cap1(s string) string {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return s
	}
	return string(s[0]-'a'+'A') + s[1:]
}

// slot is a slot name and the accessor texts the checks look for, built once per run.
type slot struct {
	name         string
	setValue     string // get<Name>().setValue
	setValueCall string // get<Name>().setValue(
	setCall      string // set<Name>(
	setFalse     string // get<Name>().setValue(false)
	setFalseSp   string // get<Name>().setValue( false )
}

func slots(names []string) []slot {
	out := make([]slot, len(names))
	for i, n := range names {
		g := "get" + cap1(n) + "().setValue"
		out[i] = slot{name: n, setValue: g, setValueCall: g + "(", setCall: "set" + cap1(n) + "(",
			setFalse: g + "(false)", setFalseSp: g + "( false )"}
	}
	return out
}

// cutAt is awk's sub(/[<chars>].*/, "", s): s up to its first byte in chars.
func cutAt(s, chars string) string {
	if i := strings.IndexAny(s, chars); i >= 0 {
		return s[:i]
	}
	return s
}

// stripBlanks is awk's gsub(/[[:space:]]/, "", s).
func stripBlanks(s string) string {
	if !strings.ContainsAny(s, space) {
		return s
	}
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(space, s[i]) < 0 {
			b = append(b, s[i])
		}
	}
	return string(b)
}

// cutComment is awk's sub(/\/\/.*$/, "", s): s up to its first "//".
func cutComment(s string) string {
	if i := strings.Index(s, "//"); i >= 0 {
		return s[:i]
	}
	return s
}

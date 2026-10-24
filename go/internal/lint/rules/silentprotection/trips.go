package silentprotection

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

var (
	privBool  = rx(`private[[:space:]]+(final[[:space:]]+)?boolean[[:space:]]+`)
	ifOpen    = rx(`(^|[^A-Za-z0-9_])if[[:space:]]*\(`)
	tripName  = rx(`([Ii]nhibited|[Tt]rip)$`)
	orReturn  = rx(`(^|[[:space:]])return[[:space:]]+[A-Za-z_][A-Za-z0-9_. ]*\|\|[[:space:]]*[A-Za-z_][A-Za-z0-9_]*`)
	returnKw  = rx(`(^|[[:space:]])return[[:space:]]+`)
	upToLast  = rx(`.*\|\|[[:space:]]*`)
	thisWrite = rx(`this\.[A-Za-z_][A-Za-z0-9_]*[[:space:]]*=[[:space:]]*(true|false)`)
)

const (
	warnMsg    = " -- no status/reason/alarm surface in scope; add a *Alarm/*Reason SUMMARY slot or a BAlarmSourceExt"
	consoleMsg = " -- surfaced only by a console alarm; add a local *Status/*Reason SUMMARY slot (advisory, unattended console)"
	// guardLines is how many lines above a trip action the if( guard search looks, within the method.
	guardLines = 10
	// followLines is how many lines from the trip the assignment of a surfaced field is looked for.
	followLines = 8
)

// trip is one protection trip: the reported line, its method (name and 1-based line span), the output
// it forces and the identifier its if() condition tests ("" when compound or absent).
type trip struct {
	line   int
	method javasrc.Method
	out    string
	cond   string
}

// check reports every trip of f with no local surface: an ADVISORY row when the file's class (or its
// B<Pure> adapter) carries a console alarm, a WARN row otherwise.
func (ix *index) check(f *javasrc.File) []lint.Finding {
	lines := f.Stripped()
	priv := privateBooleans(lines)
	trips := ix.trips(lines, newCode(lines), f.Methods(), priv)
	if len(trips) == 0 {
		return nil
	}
	surfLine := ix.surfaceWriteLines(lines)
	class := strings.TrimSuffix(f.Path[strings.LastIndexByte(f.Path, '/')+1:], ".java")
	alarmed := inList(class, ix.alarmList) || inList("B"+class, ix.alarmList)
	var out []lint.Finding
	for _, t := range trips {
		if ix.surfaced(t, lines, surfLine, priv) {
			continue
		}
		if alarmed {
			out = append(out, lint.Finding{Severity: lint.Advisory, Line: t.line,
				Msg: "console-only: " + t.method.Name + " forces " + t.out + "/sheds stage" + consoleMsg})
		} else {
			out = append(out, lint.Finding{Line: t.line,
				Msg: t.method.Name + " forces " + t.out + "/sheds stage on condition" + warnMsg})
		}
	}
	return out
}

// privateBooleans maps each private boolean field (or method: the pattern does not tell them apart) to
// its last declaration line.
func privateBooleans(lines []string) map[string]int {
	priv := map[string]int{}
	for i, ln := range lines {
		if loc := privBool.FindStringIndex(ln); loc != nil {
			if name := ident.FindString(ln[loc[1]:]); name != "" {
				priv[name] = i + 1
			}
		}
	}
	return priv
}

// surfaceWriteLines marks the lines that write a surface slot (get<Slot>().setValue or set<Slot>( ).
func (ix *index) surfaceWriteLines(lines []string) []bool {
	marks := make([]bool, len(lines)+1)
	for i, ln := range lines {
		for _, s := range ix.surfSlots {
			if strings.Contains(ln, s.setValue) || strings.Contains(ln, s.setCall) {
				marks[i+1] = true
				break
			}
		}
	}
	return marks
}

// trips finds the trips of every method, in line order (B824 §824.2): Pattern 1 Math.min(target, ...),
// Pattern 3 an effect slot's setValue(false) under an inline if(<identifier>), Pattern 4 this.<private
// boolean> = true/false, each guarded by an if( at most guardLines above it in the method; and Pattern 7,
// an *Inhibited/*Trip method returning x || <private field>, reported at the field's declaration line.
func (ix *index) trips(lines []string, src code, methods []javasrc.Method, priv map[string]int) []trip {
	var trips []trip
	for _, m := range methods {
		p7 := tripName.MatchString(m.Name)
		for i := m.Start; i <= m.End; i++ {
			st := cutComment(lines[i-1])
			if p7 {
				if loc := orReturn.FindStringIndex(st); loc != nil {
					seg := subFirst(upToLast, subFirst(returnKw, st[loc[0]:]))
					if line, ok := priv[ident.FindString(seg)]; ok {
						trips = append(trips, trip{line: line, method: m, out: ident.FindString(seg)})
					}
					continue
				}
			}
			out, col := ix.tripAction(st, priv)
			if out == "" || !guarded(lines, src, m.Start, i, col) {
				continue
			}
			trips = append(trips, trip{line: i, method: m, out: out, cond: condField(st)})
		}
	}
	return trips
}

// tripAction is the output a Pattern 1, 3 or 4 action on the line forces and the column the action
// starts at, or "".
func (ix *index) tripAction(st string, priv map[string]int) (string, int) {
	for _, m := range []string{"Math.min(target,", "Math.min(target ,"} {
		if k := strings.Index(st, m); k >= 0 {
			return "stage", k
		}
	}
	for _, e := range ix.effectSlots {
		k := strings.Index(st, e.setFalse)
		if k < 0 {
			k = strings.Index(st, e.setFalseSp)
		}
		if k >= 0 && condField(st) != "" {
			return e.name, k
		}
	}
	if loc := thisWrite.FindStringIndex(st); loc != nil {
		name := ident.FindString(strings.Replace(st[loc[0]:], "this.", "", 1))
		if _, ok := priv[name]; ok {
			return name, loc[0]
		}
	}
	return "", 0
}

// guarded reports an if( on line i or on one of the guardLines lines above it, inside the method, whose if
// statement holds the action at column col of line i (code.governs): an if that ended above the trip does
// not guard it.
func guarded(lines []string, src code, start, i, col int) bool {
	for j := i; j >= start && j >= i-guardLines; j-- {
		if ifOpen.MatchString(cutComment(lines[j-1])) && src.governs(j-1, i-1, col) {
			return true
		}
	}
	return false
}

// condField is the identifier an if(...) on the line tests, a leading ! ignored, or "" when the
// condition is compound or there is no if(.
func condField(st string) string {
	loc := ifOpen.FindStringIndex(st)
	if loc == nil {
		return ""
	}
	var cond []byte
	depth := 1
	for _, c := range []byte(st[loc[1]:]) {
		if c == ')' {
			if depth--; depth == 0 {
				break
			}
		} else if c == '(' {
			depth++
		}
		cond = append(cond, c)
	}
	s := strings.TrimRight(strings.TrimLeft(string(cond), space+"!"), space)
	if !identOnly.MatchString(s) {
		return ""
	}
	return s
}

// surfaced reports a local surface for the trip: its condition field is written to a surface slot
// somewhere in the tree (ii-B), its method writes a surface slot (2), or a non-private field written to a
// surface slot is assigned within followLines lines from the trip, inside the method (3).
func (ix *index) surfaced(t trip, lines []string, surfLine []bool, priv map[string]int) bool {
	if t.cond != "" && inList(t.cond, ix.surfList) {
		return true
	}
	for i := t.method.Start; i <= t.method.End; i++ {
		if surfLine[i] {
			return true
		}
	}
	for j := t.line; j <= t.line+followLines && j <= t.method.End; j++ {
		ln := cutComment(lines[j-1])
		for _, fld := range ix.surfFields {
			if _, p := priv[fld]; p {
				continue
			}
			// only the first occurrence of the field on the line is tested, as awk's index() finds it
			if pos := strings.Index(ln, fld); pos >= 0 {
				rest := strings.TrimLeft(ln[pos+len(fld):], space)
				if strings.HasPrefix(rest, "=") && !strings.HasPrefix(rest, "==") {
					return true
				}
			}
		}
	}
	return false
}

// subFirst is awk's sub(re, "", s): the first match removed.
func subFirst(re interface{ FindStringIndex(string) []int }, s string) string {
	if loc := re.FindStringIndex(s); loc != nil {
		return s[:loc[0]] + s[loc[1]:]
	}
	return s
}

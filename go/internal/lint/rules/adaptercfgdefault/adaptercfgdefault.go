// Package adaptercfgdefault is lint-adapter-cfg-default (audit 2026-10-03 A7-D1): a pure-model Cfg field whose default
// differs from the default of the @NiagaraProperty slot the Baja adapter fills it from. The station starts with the
// slot default; a pure JUnit test that builds `new Cfg()` runs with the field's default instead, so the two
// configurations the module is tested and deployed with drift apart (types/logic.md § stateful controller: one
// `step(now, inputs, Cfg)`, the BComponent a thin slot adapter). toolbelt/lint-adapter-cfg-default.sh is the wrapper
// and its header is the user-facing contract.
//
// It is a lint.TreeRule over each module root: every *.java under a -rt part's src/ (srcTest/ and a part's own test/
// and build/ not entered), read as javasrc tokens. A Cfg is any class whose simple name ends in "Cfg". The adapter's
// link is an assignment statement `<x>.<field> = get<Slot>();`, optionally `get<Slot>().getMillis()` or `.getSeconds()`
// for a BRelTime slot, where <field> is a field of a Cfg of the same part and <slot> a @NiagaraProperty of the part.
// Fields and slots are keyed by (class, name): the field is the one of the Cfg type <x> is declared with in the link's
// top-level class (or the part's only field of that name when <x> is declared elsewhere), the slot the one declared on
// a class enclosing the link (or the part's only slot of that name, an inherited one); a link these cannot resolve to
// exactly one field and one slot is not judged. The field's default is its literal initializer (a number, optionally
// negative, or true/false) or Java's 0/false when it has none; the slot's is its defaultValue when that is such a
// literal or BRelTime.make/makeSeconds/makeMinutes/makeHours(<n>). Anything else (a computed initializer, a field with
// several declarators, any other right-hand side, a default it cannot evaluate) is not judged.
package adaptercfgdefault

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ID is the lint name n4lint accepts and every row prints in its check column.
const ID = "lint-adapter-cfg-default"

// Spec registers the rule: [--strict] [--] <module-root>... (each root on its own), WARN rows (exit 1 under
// --strict), left out of n4lint --all (a module root).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-adapter-cfg-default.sh [--strict] <module-root>...",
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest"}, PrunePaths: lint.PartOutputs()},
		EmptyError: lint.EmptyRow{Msg: "no *.java under a -rt part's src/ (wrong path: pass the module root)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var rtPart = regexp.MustCompile(`^(.*/)?[^/]+-rt/src/`)

// value is an evaluated default: a number or a boolean.
type value struct {
	num    float64
	isBool bool
	b      bool
}

func (v value) String() string {
	if v.isBool {
		return strconv.FormatBool(v.b)
	}
	return strconv.FormatFloat(v.num, 'f', -1, 64)
}

// field is one Cfg field with a judged default; qual is its class's dotted name with every enclosing class
// (PanControl.Cfg), class the simple name the row prints.
type field struct {
	class, qual, path string
	line              int // 0-based
	def               value
}

// slot is one @NiagaraProperty with an evaluated default; rel marks a BRelTime default (num in milliseconds). owner
// is the dotted name of the class the annotation is declared on.
type slot struct {
	owner, file string
	line        int // 1-based
	def         value
	rel         bool
}

// part keys fields and slots by bare name to every declaration of that name: two Cfg classes (or two components)
// of a part may share one, so a link is resolved through its receiver's type and its enclosing class.
type part struct {
	fields map[string][]field
	slots  map[string][]slot
	links  []link
}

// link is one `<x>.<field> = get<Slot>()[.getMillis()|.getSeconds()];`. recv is the dotted Cfg type <x> is
// declared with in the link's top-level class ("" when no Cfg-typed declaration of <x> is there, ambiguous when
// several types are); chain the dotted names of the classes enclosing the statement, innermost first.
type link struct {
	field, slot, conv, recv string
	chain                   []string
}

// ambiguous marks a receiver declared with several Cfg types: such a link is not judged.
const ambiguous = "\x00ambiguous"

// CheckTree reads each -rt part's Cfg fields, slots and adapter assignments, then compares the linked defaults.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var order []string
	parts := map[string]*part{}
	for _, f := range files {
		m := rtPart.FindString(f.Path)
		if m == "" {
			continue
		}
		p := parts[m]
		if p == nil {
			p = &part{fields: map[string][]field{}, slots: map[string][]slot{}}
			parts[m] = p
			order = append(order, m)
		}
		toks := javasrc.Tokens(f.Stripped())
		classes := javasrc.Classes(toks)
		quals := qualified(classes)
		for i, c := range classes {
			if c.Kind == "class" && strings.HasSuffix(c.Name, "Cfg") {
				for _, fd := range javasrc.Fields(toks, c.Body) {
					if v, ok := fieldDefault(toks, fd); ok && !fd.Static {
						p.fields[fd.Name] = append(p.fields[fd.Name], field{c.Name, quals[i], f.Path, fd.Line, v})
					}
				}
			}
		}
		base := f.Path[strings.LastIndexByte(f.Path, '/')+1:]
		javasrc.NiagaraProperties(f.Stripped(), func(text string, line int) {
			if s, name, ok := property(text); ok {
				s.owner, s.file, s.line = annotated(classes, quals, line), base, line
				p.slots[name] = append(p.slots[name], s)
			}
		})
		p.links = append(p.links, links(toks, classes, quals)...)
	}
	if len(order) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	var out []lint.Finding
	for _, k := range order {
		p := parts[k]
		seen := map[string]bool{}
		for _, l := range p.links {
			fd, okF := p.field(l)
			s, okS := p.slot(l)
			key := fd.qual + "." + l.field + "\x00" + s.owner + "." + l.slot
			if !okF || !okS || seen[key] {
				continue
			}
			seen[key] = true
			want, ok := convert(s, l.conv)
			if !ok || want.isBool != fd.def.isBool || want == fd.def {
				continue
			}
			out = append(out, lint.Finding{Path: fd.path, Line: fd.line + 1, Msg: fd.class + "." + l.field + " defaults to " +
				fd.def.String() + " but slot " + l.slot + " (" + s.file + ":" + strconv.Itoa(s.line) + ") defaults to " + want.String() +
				": a pure test built from new Cfg() runs a configuration the station never starts with -- initialize the field to the slot default"})
		}
	}
	sortFindings(out)
	return out, nil
}

// field resolves a link's Cfg field: the field of that name in the class the receiver is declared with, or, when
// the receiver's declaration is not in the link's class, the part's only field of that name. Two candidates the
// receiver cannot tell apart are not judged (no row rather than a row against the wrong class).
func (p *part) field(l link) (field, bool) {
	cands := p.fields[l.field]
	if l.recv == ambiguous {
		return field{}, false
	}
	if l.recv != "" {
		var hit []field
		for _, f := range cands {
			// the type as written may be shorter (Cfg inside PanControl) or package-qualified (com.a.PanCfg)
			if f.qual == l.recv || strings.HasSuffix(f.qual, "."+l.recv) || strings.HasSuffix(l.recv, "."+f.qual) {
				hit = append(hit, f)
			}
		}
		cands = hit
	}
	if len(cands) != 1 {
		return field{}, false
	}
	return cands[0], true
}

// slot resolves a link's slot: the slot of that name declared on a class enclosing the link (innermost first), or,
// when none is (an inherited slot), the part's only slot of that name; several are not judged.
func (p *part) slot(l link) (slot, bool) {
	cands := p.slots[l.slot]
	for _, c := range l.chain {
		for _, s := range cands {
			if s.owner == c {
				return s, true
			}
		}
	}
	if len(cands) != 1 {
		return slot{}, false
	}
	return cands[0], true
}

// qualified is each class's dotted name with its enclosing classes (Outer.Inner), in Classes order.
func qualified(classes []javasrc.Class) []string {
	out := make([]string, len(classes))
	for i, c := range classes {
		out[i] = c.Name
		for j := i - 1; j >= 0; j-- { // the innermost enclosing class is the nearest earlier one whose body holds c
			if o := classes[j]; o.Body[0] <= c.At && c.At < o.Body[1] {
				out[i] = out[j] + "." + c.Name
				break
			}
		}
	}
	return out
}

// enclosing is the dotted names of the classes whose body holds token k, innermost first.
func enclosing(classes []javasrc.Class, quals []string, k int) []string {
	var out []string
	for i := len(classes) - 1; i >= 0; i-- {
		if c := classes[i]; c.Body[0] <= k && k < c.Body[1] {
			out = append(out, quals[i])
		}
	}
	return out
}

// annotated is the class an annotation at 1-based line belongs to: the first class declared on or after it.
func annotated(classes []javasrc.Class, quals []string, line int) string {
	for i, c := range classes {
		if c.Line+1 >= line {
			return quals[i]
		}
	}
	return ""
}

// receiverType is the dotted Cfg type <x> is declared with (`T x` followed by =, ;, , or ) -- a field, a local or
// a parameter) inside the top-level class of the link: "" with no such declaration, ambiguous with several types.
func receiverType(toks []javasrc.Token, body [2]int, x string) string {
	found := ""
	for j := body[0] + 1; j+1 < body[1]; j++ {
		if toks[j].Text != x || toks[j-1].Kind != javasrc.TokIdent || !strings.HasSuffix(toks[j-1].Text, "Cfg") {
			continue
		}
		switch toks[j+1].Text {
		case "=", ";", ",", ")":
		default:
			continue
		}
		if toks[j+1].Text == "=" && j+2 < body[1] && toks[j+2].Text == "=" {
			continue // x == ...
		}
		typ, i := toks[j-1].Text, j-1
		for i >= 2 && toks[i-1].Text == "." && toks[i-2].Kind == javasrc.TokIdent {
			typ, i = toks[i-2].Text+"."+typ, i-2
		}
		if found != "" && found != typ {
			return ambiguous
		}
		found = typ
	}
	return found
}

// sortFindings orders rows by file, then line (parts are read in walk order; links in source order).
func sortFindings(out []lint.Finding) {
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && (out[j].Path < out[j-1].Path || out[j].Path == out[j-1].Path && out[j].Line < out[j-1].Line); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
}

// fieldDefault is a field's default: its literal initializer, or 0/false for a numeric/boolean field with none.
func fieldDefault(toks []javasrc.Token, fd javasrc.Field) (value, bool) {
	if fd.Init[0] >= fd.Init[1] {
		switch fd.Type {
		case "boolean":
			return value{isBool: true}, true
		case "int", "long", "double", "float", "short", "byte":
			return value{}, true
		}
		return value{}, false
	}
	init := toks[fd.Init[0]:fd.Init[1]]
	var b strings.Builder
	for _, t := range init {
		if t.Text == "," {
			return value{}, false // several declarators: the range is every declarator's
		}
		b.WriteString(t.Text)
	}
	return literal(b.String())
}

var (
	propName = regexp.MustCompile(`\bname\s*=\s*"([^"]+)"`)
	propDef  = regexp.MustCompile(`\bdefaultValue\s*=\s*"((?:[^"\\]|\\.)*)"`)
	relTime  = regexp.MustCompile(`^BRelTime\.make(|Seconds|Minutes|Hours)\((-?[0-9]+)L?\)$`)
)

// relScale is the milliseconds per unit of each BRelTime factory.
var relScale = map[string]float64{"": 1, "Seconds": 1000, "Minutes": 60000, "Hours": 3600000}

// property is the name and evaluated default of one @NiagaraProperty annotation text.
func property(text string) (slot, string, bool) {
	n, d := propName.FindStringSubmatch(text), propDef.FindStringSubmatch(text)
	if n == nil || d == nil {
		return slot{}, "", false
	}
	def := strings.TrimSpace(d[1])
	if m := relTime.FindStringSubmatch(def); m != nil {
		x, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			return slot{}, "", false
		}
		return slot{def: value{num: x * relScale[m[1]]}, rel: true}, n[1], true
	}
	v, ok := literal(def)
	return slot{def: v}, n[1], ok
}

// literal evaluates a Java number literal (sign, suffix and underscores allowed) or true/false.
func literal(s string) (value, bool) {
	switch s {
	case "true", "false":
		return value{isBool: true, b: s == "true"}, true
	}
	t := strings.ReplaceAll(s, "_", "")
	if strings.HasPrefix(t, "0x") || strings.HasPrefix(t, "-0x") {
		n, err := strconv.ParseInt(strings.TrimRight(t, "lL"), 0, 64)
		return value{num: float64(n)}, err == nil
	}
	t = strings.TrimRight(t, "lLdDfF")
	n, err := strconv.ParseFloat(t, 64)
	return value{num: n}, err == nil && t != ""
}

// convert is the slot default as the adapter hands it to the field: a BRelTime only through getMillis (ms) or
// getSeconds (s), any other slot only as read.
func convert(s slot, conv string) (value, bool) {
	switch {
	case s.rel && conv == "getMillis":
		return s.def, true
	case s.rel && conv == "getSeconds":
		return value{num: float64(int64(s.def.num) / 1000)}, true
	case !s.rel && conv == "":
		return s.def, true
	}
	return value{}, false
}

// links finds every `<x>.<field> = get<Slot>()[.getMillis()|.getSeconds()];` statement, with its receiver's
// declared Cfg type and its enclosing classes.
func links(toks []javasrc.Token, classes []javasrc.Class, quals []string) []link {
	var out []link
	for k := 0; k+7 < len(toks); k++ {
		if toks[k].Kind != javasrc.TokIdent || toks[k+1].Text != "." || toks[k+2].Kind != javasrc.TokIdent || toks[k+3].Text != "=" ||
			toks[k+4].Text == "=" || k > 0 && toks[k-1].Text == "." {
			continue
		}
		g := toks[k+4].Text
		if !strings.HasPrefix(g, "get") || len(g) < 4 || toks[k+5].Text != "(" || toks[k+6].Text != ")" {
			continue
		}
		j, conv := k+7, ""
		if j+3 < len(toks) && toks[j].Text == "." && (toks[j+1].Text == "getMillis" || toks[j+1].Text == "getSeconds") &&
			toks[j+2].Text == "(" && toks[j+3].Text == ")" {
			conv, j = toks[j+1].Text, j+4
		}
		if j >= len(toks) || toks[j].Text != ";" {
			continue
		}
		l := link{field: toks[k+2].Text, slot: strings.ToLower(g[3:4]) + g[4:], conv: conv, chain: enclosing(classes, quals, k)}
		for i := range classes { // the outermost class holding the statement bounds the receiver's declaration
			if c := classes[i]; c.Body[0] <= k && k < c.Body[1] {
				l.recv = receiverType(toks, c.Body, toks[k].Text)
				break
			}
		}
		out = append(out, l)
	}
	return out
}

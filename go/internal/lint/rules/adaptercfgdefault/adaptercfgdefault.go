// Package adaptercfgdefault is lint-adapter-cfg-default (audit 2026-10-03 A7-D1): a pure-model Cfg field whose
// default differs from the default of the @NiagaraProperty slot the Baja adapter fills it from. The station starts
// with the slot default; a pure JUnit test that builds `new Cfg()` runs with the field's default instead, so the
// two configurations the module is tested and deployed with drift apart (types/logic.md § stateful controller:
// one `step(now, inputs, Cfg)`, the BComponent a thin slot adapter). toolbelt/lint-adapter-cfg-default.sh is the
// wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule over each module root: every *.java under a -rt part's src/ (srcTest/, test/ and build/
// not entered), read as javasrc tokens. A Cfg is any class whose simple name ends in "Cfg". The adapter's link is an
// assignment statement `<x>.<field> = get<Slot>();`, optionally `get<Slot>().getMillis()` or `.getSeconds()` for a
// BRelTime slot, where <field> is a field of a Cfg of the same part and <slot> a @NiagaraProperty of the part. The
// field's default is its literal initializer (a number, optionally negative, or true/false) or Java's 0/false when
// it has none; the slot's is its defaultValue when that is such a literal or BRelTime.make/makeSeconds/makeMinutes/
// makeHours(<n>). Anything else (a computed initializer, a field with several declarators, any other right-hand
// side, a default it cannot evaluate) is not judged.
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
		Select:     lint.Selection{Names: []string{"*.java"}, Prune: []string{"srcTest", "test", "build"}},
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

// field is one Cfg field with a judged default.
type field struct {
	class, path string
	line        int // 0-based
	def         value
}

// slot is one @NiagaraProperty with an evaluated default; rel marks a BRelTime default (num in milliseconds).
type slot struct {
	file string
	line int // 1-based
	def  value
	rel  bool
}

type part struct {
	fields map[string]field // Cfg field name -> its default
	slots  map[string]slot
	links  []link
}

// link is one `<x>.<field> = get<Slot>()[.getMillis()|.getSeconds()];`.
type link struct {
	field, slot, conv string
}

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
			p = &part{fields: map[string]field{}, slots: map[string]slot{}}
			parts[m] = p
			order = append(order, m)
		}
		toks := javasrc.Tokens(f.Stripped())
		for _, c := range javasrc.Classes(toks) {
			if c.Kind == "class" && strings.HasSuffix(c.Name, "Cfg") {
				for _, fd := range javasrc.Fields(toks, c.Body) {
					if v, ok := fieldDefault(toks, fd); ok && !fd.Static {
						p.fields[fd.Name] = field{c.Name, f.Path, fd.Line, v}
					}
				}
			}
		}
		base := f.Path[strings.LastIndexByte(f.Path, '/')+1:]
		javasrc.NiagaraProperties(f.Stripped(), func(text string, line int) {
			if s, name, ok := property(text); ok {
				s.file, s.line = base, line
				p.slots[name] = s
			}
		})
		p.links = append(p.links, links(toks)...)
	}
	if len(order) == 0 {
		return nil, lint.ErrEmptyRoot
	}
	var out []lint.Finding
	for _, k := range order {
		p := parts[k]
		seen := map[string]bool{}
		for _, l := range p.links {
			fd, okF := p.fields[l.field]
			s, okS := p.slots[l.slot]
			if !okF || !okS || seen[l.field+"\x00"+l.slot] {
				continue
			}
			seen[l.field+"\x00"+l.slot] = true
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

// links finds every `<x>.<field> = get<Slot>()[.getMillis()|.getSeconds()];` statement.
func links(toks []javasrc.Token) []link {
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
		out = append(out, link{field: toks[k+2].Text, slot: strings.ToLower(g[3:4]) + g[4:], conv: conv})
	}
	return out
}

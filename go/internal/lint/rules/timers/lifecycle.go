package timers

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// lifecyclePass is orphan-flag and release-gate-parity (WARN, advisory), per file on comment-stripped code.
// Rows come in the order the awk's `for (m in body)` visited methods: gawk's hash order (gawkOrder).
func (r *run) lifecyclePass(srcs []*source) {
	for _, s := range srcs {
		lc := r.newLifecycle(s)
		for _, o := range lc.orphans() {
			r.row(lint.Warn, "orphan-flag", s, o[0]+"() cancels "+o[1]+" but never clears its companion flag "+o[2]+" (set true beside the schedule) -- the expiry handler that clears it will not run")
		}
		for _, g := range lc.gates() {
			r.row(lint.Warn, "release-gate-parity", s, g[0]+"() re-applies "+g[1]+"() without the "+g[2]+"() gate that started()/atSteadyState() uses (types/logic.md release-point gate rule 2)")
		}
	}
}

// lifecycle is lifecycle.awk's state for one file.
type lifecycle struct {
	r      *run
	all    string            // every stripped line + "\n"
	body   map[string]string // method name -> its bodies' lines, each + "\n"
	order  []string          // the method names in gawk for-in order
	bfield map[string]bool   // class-scope boolean fields
	reach  map[string]string // memoised reach(m)
}

func (r *run) newLifecycle(s *source) *lifecycle {
	st, ms := s.Stripped(), s.Methods()
	lc := &lifecycle{r: r, all: joinLines(st, 1, len(st), "", "\n"), body: map[string]string{}, bfield: map[string]bool{}, reach: map[string]string{}}
	var inserted []string
	for _, m := range ms {
		if _, ok := lc.body[m.Name]; !ok {
			inserted = append(inserted, m.Name)
		}
		lc.body[m.Name] += joinLines(st, m.Start, m.End, "", "\n")
	}
	lc.order = gawkOrder(inserted)
	in := inMethod(len(st), ms)
	for i, ln := range st {
		if in[i+1] || !strings.Contains(ln, "boolean") {
			continue
		}
		if d, ok := r.rx.find(ln, `boolean`+sp+`+`+ident+sp+`*[;=]`); ok {
			lc.bfield[r.rx.sub(r.rx.sub(d, `^boolean`+sp+`+`), sp+`*[;=]$`)] = true
		}
	}
	return lc
}

// reachOf is reach(m): "\n" + m's body plus, breadth first, every same-file method it reaches through
// unqualified calls, each "\n" + body.
func (lc *lifecycle) reachOf(m string) string {
	if t, ok := lc.reach[m]; ok {
		return t
	}
	t := lc.r.reachText(lc.body, m, "\n"+lc.body[m])
	lc.reach[m] = t
	return t
}

// pair is a ticket T, its companion flag F (set true where T is armed) and T's expiry handler H.
type pair struct{ t, f, h string }

// pairs: F set true in a method that arms `T = Clock.schedule*(recv, delay, act, …)` and cleared in T's
// handler do<Act>(); only such a flag is T's companion.
func (lc *lifecycle) pairs() []pair {
	rx := lc.r.rx
	var out []pair
	seen := map[[2]string]bool{}
	for _, m := range lc.order {
		rx.each(lc.body[m], ident+sp+`*=`+sp+`*Clock`+sp+`*\.`+sp+`*schedule[A-Za-z]*`+sp+`*\(`, func(_, arm, rest string) bool {
			t := rx.sub(arm, sp+`*=.*`)
			act := thirdArg(rest)
			if !identOnly.MatchString(act) {
				return true
			}
			h := "do" + strings.ToUpper(act[:1]) + act[1:]
			handler, ok := lc.body[h]
			if !ok {
				return true
			}
			rx.each(lc.body[m], ident+sp+`*=`+sp+`*true`+sp+`*;`, func(_, set, _ string) bool {
				f := rx.sub(set, sp+`*=.*`)
				if lc.bfield[f] && rx.has(handler, notRef+f+sp+`*=`+sp+`*false`) && !seen[[2]string{t, f}] {
					seen[[2]string{t, f}] = true
					out = append(out, pair{t, f, h})
				}
				return true
			})
			return true
		})
	}
	return out
}

var identOnly = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// thirdArg is third_arg(): the third depth-0 argument of the call whose "(" ends just before s, blanks removed.
func thirdArg(s string) string {
	var a strings.Builder
	d, k := 0, 1
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == '(':
			d++
		case c == ')':
			if d == 0 {
				return dropSpace(a.String())
			}
			d--
		case c == ',' && d == 0:
			k++
			continue
		}
		if k == 3 {
			a.WriteByte(s[i])
		}
	}
	return dropSpace(a.String())
}

func dropSpace(s string) string {
	return strings.Map(func(c rune) rune {
		if strings.ContainsRune(" \t\n\v\f\r", c) {
			return -1
		}
		return c
	}, s)
}

// orphans lists [method, T, F]: a method other than stopped()/started()/H that cancels T directly but
// neither clears F nor re-arms T in its reach, with no class-wide self-heal and no caller owning F.
func (lc *lifecycle) orphans() [][3]string {
	ps := lc.pairs()
	if len(ps) == 0 {
		return nil
	}
	rx := lc.r.rx
	var out [][3]string
	for _, m := range lc.order {
		if m == "stopped" || m == "started" {
			continue
		}
		for _, p := range ps {
			if m == p.h || !rx.has(lc.body[m], notRef+p.t+sp+`*\.`+sp+`*cancel`+sp+`*\(`) {
				continue
			}
			if rx.has(lc.all, p.t+sp+`*==`+sp+`*null[^;]*`+p.f+sp+`*=`+sp+`*false`) || lc.owns(lc.reachOf(m), p) || lc.callersHandle(m, p) {
				continue
			}
			out = append(out, [3]string{m, p.t, p.f})
		}
	}
	return out
}

// owns: the text re-arms T or assigns F.
func (lc *lifecycle) owns(text string, p pair) bool {
	return lc.r.rx.has(text, notRef+p.t+sp+`*=`+sp+`*Clock`+sp+`*\.`+sp+`*schedule`) || lc.r.rx.has(text, notRef+p.f+sp+`*=[^=]`)
}

// callersHandle: m has a same-file caller, and every caller is stopped()/started()/H or owns F in its reach.
func (lc *lifecycle) callersHandle(m string, p pair) bool {
	call := notRef + regexp.QuoteMeta(m) + sp + `*\(`
	any := false
	for _, c := range lc.order {
		if c == m || !lc.r.rx.has(lc.body[c], call) {
			continue
		}
		any = true
		if c == "stopped" || c == "started" || c == p.h || lc.owns(lc.reachOf(c), p) {
			continue
		}
		return false
	}
	return any
}

// gates lists [method, apply, gate]: started()/atSteadyState() call begin*() gates and apply*() methods,
// and a release point (exit*/end*/finish*/leave*/on*Exit|Expired|Expiry|End*) calls one of those apply*()
// methods with no gate in its reach.
func (lc *lifecycle) gates() [][3]string {
	boot := ""
	for _, b := range []string{"started", "atSteadyState"} {
		if _, ok := lc.body[b]; ok {
			boot += lc.reachOf(b)
		}
	}
	var gs, as []string
	seen := map[string]bool{}
	lc.r.rx.calls(boot, func(id string) {
		if _, ok := lc.body[id]; !ok || seen[id] {
			return
		}
		switch {
		case beginName.MatchString(id):
			seen[id] = true
			gs = append(gs, id)
		case applyName.MatchString(id):
			seen[id] = true
			as = append(as, id)
		}
	})
	if len(gs) == 0 || len(as) == 0 {
		return nil
	}
	var out [][3]string
	for _, m := range lc.order {
		if !releaseName.MatchString(m) && !onEndName.MatchString(m) {
			continue
		}
		hit := ""
		for _, a := range as {
			if lc.r.rx.has(lc.body[m], notRef+a+sp+`*\(`) {
				hit = a
				break
			}
		}
		if hit == "" {
			continue
		}
		gated := false
		for _, g := range gs {
			gated = gated || lc.r.rx.has(lc.reachOf(m), notRef+g+sp+`*\(`)
		}
		if !gated {
			out = append(out, [3]string{m, hit, gs[0]})
		}
	}
	return out
}

var (
	beginName   = regexp.MustCompile(`^begin[A-Z]`)
	applyName   = regexp.MustCompile(`^apply[A-Z]`)
	releaseName = regexp.MustCompile(`^(exit|end|finish|leave)[A-Z]`)
	onEndName   = regexp.MustCompile(`^on[A-Za-z0-9_]*(Exit|Expired|Expiry|End)[A-Za-z0-9_]*$`)
)

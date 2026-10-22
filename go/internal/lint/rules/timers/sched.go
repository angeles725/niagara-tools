package timers

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// changedSchedPass is changed-sched: a Clock.schedule reachable from changed() or started() (directly or
// through one same-file callee) with no isRunning()/atSteadyState() guard in the scheduling body. It reads
// raw lines (comments included), as the bash awk did.
func (r *run) changedSchedPass(srcs []*source) {
	for _, s := range srcs {
		if r.unguardedSchedule(s.Raw) {
			r.row(lint.Fail, "changed-sched", s, "Clock.schedule reachable from changed()/started() without isRunning()/atSteadyState() guard in scheduling body")
		}
	}
}

// schedKeywords are the identifiers never taken as a method or a callee.
var schedKeywords = func() map[string]bool {
	kw := map[string]bool{}
	for _, k := range strings.Fields("if for while switch catch try finally return new assert synchronized throw else do instanceof super this void boolean int long double float String abstract final static public private protected class interface enum Clock Sys BRelTime BComponent BAbstractService") {
		kw[k] = true
	}
	return kw
}()

// unguardedSchedule is the changed-sched awk: it maps each name to the body of the first block that
// follows its first "name(" on a line, then judges changed() and started().
func (r *run) unguardedSchedule(lines []string) bool {
	meth := r.blocks(lines)
	unguarded := func(b string) bool {
		return strings.Contains(b, "Clock.schedule") && !strings.Contains(b, "isRunning") && !strings.Contains(b, "atSteadyState")
	}
	for _, lc := range []string{"changed", "started"} {
		body, ok := meth[lc]
		if !ok {
			continue
		}
		if unguarded(body) {
			return true
		}
		hit := false
		r.rx.each(body, `[a-z][a-zA-Z0-9_]*`+sp+`*\(`, func(_, m, _ string) bool {
			callee := strings.TrimRight(m[:len(m)-1], " \t\n\v\f\r")
			if cb, ok := meth[callee]; ok && !schedKeywords[callee] && unguarded(cb) {
				hit = true
				return false
			}
			return true
		})
		if hit {
			return true
		}
	}
	return false
}

// blocks is the awk's method map: for a line whose first "name(" is not a keyword, scan from that line's
// start; a ";" before any "{" ends the try, otherwise the text inside the balanced braces (each line end
// adding a space) is the body of name, and scanning resumes after the closing line.
func (r *run) blocks(lines []string) map[string]string {
	meth := map[string]string{}
	for i := 0; i < len(lines); i++ {
		if !strings.Contains(lines[i], "(") {
			continue
		}
		m, ok := r.rx.find(lines[i], `[a-zA-Z_][a-zA-Z0-9_]*`+sp+`*\(`)
		if !ok {
			continue
		}
		cand := strings.TrimRight(m[:len(m)-1], " \t\n\v\f\r")
		if cand == "" || schedKeywords[cand] {
			continue
		}
		if body, end, ok := blockAfter(lines, i); ok {
			if _, seen := meth[cand]; !seen {
				meth[cand] = body
			}
			i = end
		}
	}
	return meth
}

// blockAfter scans from line i for the first "{" (a ";" first means no block) and returns the text up to
// its matching "}" and the line of that brace.
func blockAfter(lines []string, i int) (string, int, bool) {
	var body strings.Builder
	inBlock, depth := false, 0
	for j := i; j < len(lines); j++ {
		ln := lines[j]
		for c := 0; c < len(ln); c++ {
			ch := ln[c]
			if !inBlock {
				if ch == '{' {
					inBlock, depth = true, 1
				} else if ch == ';' {
					return "", 0, false
				}
				continue
			}
			if ch == '{' {
				depth++
			} else if ch == '}' {
				if depth--; depth == 0 {
					return body.String(), j, true
				}
			}
			body.WriteByte(ch)
		}
		body.WriteByte(' ')
	}
	return "", 0, false
}

// steadyStatePass is atSteadyState-only-timer (WARN): a Clock.schedule inside atSteadyState() in a file
// with no started() override never arms on a component mounted after station bootstrap.
func (r *run) steadyStatePass(srcs []*source) {
	for _, s := range srcs {
		if r.steadyStateOnly(s.Raw) {
			r.row(lint.Warn, "atSteadyState-only-timer", s, "timer armed only in atSteadyState() — add started() override with Sys.atSteadyState() guard or commissioning-time mounts will never arm the timer")
		}
	}
}

func (r *run) steadyStateOnly(lines []string) bool {
	inBody, depth, sched, started := false, 0, false, false
	for _, ln := range lines {
		if !strings.Contains(ln, "started") && !strings.Contains(ln, "atSteadyState") && !inBody {
			continue
		}
		if r.rx.has(ln, `void`+sp+`+started`+sp+`*\(`) {
			started = true
		}
		if !inBody && r.rx.has(ln, `void`+sp+`+atSteadyState`+sp+`*\(`) {
			inBody, depth = true, 0
		}
		if !inBody {
			continue
		}
		if strings.Contains(ln, "Clock.schedule") {
			sched = true
		}
		for c := 0; c < len(ln); c++ {
			switch ln[c] {
			case '{':
				depth++
			case '}':
				if depth--; depth == 0 {
					inBody = false
				}
			}
		}
	}
	return sched && !started
}

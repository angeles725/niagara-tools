package timers

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// ticketPass is the first loop of the bash lint: per file, every discarded-ticket row, then the
// timer-ticket verdict of a file that owns a timer. Both read code (comments and literal contents
// blanked): a Clock.schedule( or Clock.Ticket in a comment or a string owns no timer.
func (r *run) ticketPass(srcs []*source) {
	for _, s := range srcs {
		r.discarded(s)
		at := timerLine(s)
		if at == 0 {
			continue // a timerless class is safe
		}
		const check = "timer-ticket"
		switch verdict, missing := r.ticketVerdict(s); verdict {
		case "OK":
			r.row(lint.Pass, check, s, at, "timer cancelled in stopped()")
		case "NOSTOP":
			r.row(lint.Fail, check, s, at, "schedules a Clock ticket but has no stopped() override to cancel it")
		case "MISSANY":
			r.row(lint.Fail, check, s, at, "stopped() cancels no ticket (the Clock ticket is not kept in a named Clock.Ticket field, and no .cancel( call is reachable from stopped())")
		default:
			for _, f := range missing {
				r.row(lint.Fail, check, s, f.line, "stopped() does not cancel ticket "+f.name+" (a cancel in a comment, a string or of another ticket does not count)")
			}
		}
	}
}

// discarded is discarded-ticket: one FAIL per code line holding a Clock.schedule(Periodically)( call
// whose result is neither assigned nor returned by its own statement.
func (r *run) discarded(s *source) {
	code, _ := s.blanked()
	for i, ln := range code {
		for _, loc := range scheduleCall.FindAllStringIndex(ln, -1) {
			if !kept(code, i, loc[0]) {
				r.row(lint.Fail, "discarded-ticket", s, i+1, "Clock.schedule result is discarded — assign it to a Clock.Ticket field")
				break
			}
		}
	}
}

// kept reports whether the statement holding the call at code[i][at:] assigns or returns its result:
// an assignment "=" or a return between the statement start (the last ";", "{" or "}" before the call, read back
// across lines) and the call. An "=" of an earlier statement on the same line does not count.
func kept(code []string, i, at int) bool {
	head := code[i][:at]
	for {
		if j := strings.LastIndexAny(head, ";{}"); j >= 0 {
			head = head[j+1:]
			break
		}
		if i == 0 {
			break
		}
		i--
		head = code[i] + " " + head
	}
	return assign.MatchString(head) || returned.MatchString(head)
}

var (
	assign   = regexp.MustCompile(`(^|[^=!<>])=($|[^=])`) // an assignment "=", not == != <= >=
	returned = regexp.MustCompile(`(^|[^A-Za-z0-9_])return([^A-Za-z0-9_]|$)`)
)

// timerLine is the first code line holding Clock.Ticket or a Clock.schedule(Periodically)( call: the
// line a timer-ticket row points at (0 when the file owns no timer).
func timerLine(s *source) int {
	code, _ := s.blanked()
	for i, ln := range code {
		if strings.Contains(ln, "Clock.Ticket") || scheduleCall.MatchString(ln) {
			return i + 1
		}
	}
	return 0
}

// ticketField is a Clock.Ticket / Ticket field declared outside every method, and its line.
type ticketField struct {
	name string
	line int
}

// ticketVerdict is ticket.awk: "OK", "NOSTOP", "MISSANY", or "MISS" with the ticket fields stopped()
// does not cancel (see the lint-timers.sh header for the rules).
func (r *run) ticketVerdict(s *source) (string, []ticketField) {
	code, ms := s.blanked()
	body, params := map[string]string{}, map[string]string{}
	for _, m := range ms {
		body[m.Name] += "\n" + joinLines(code, m.Start, m.End, "", "\n")
		params[m.Name] += " " + r.paramsOf(code, m.Name, m.Start)
	}
	fields := r.ticketFields(code, inMethod(len(code), ms))
	stopped, ok := body["stopped"]
	if !ok {
		return "NOSTOP", nil
	}
	scope := r.reachText(body, "stopped", stopped)
	if len(fields) == 0 {
		if r.rx.has(scope, `\.`+sp+`*cancel`+sp+`*\(`) {
			return "OK", nil
		}
		return "MISSANY", nil
	}
	var missing []ticketField
	for _, f := range fields {
		if !r.cancels(body, params, scope, f.name) {
			missing = append(missing, f)
		}
	}
	if missing == nil {
		return "OK", nil
	}
	return "MISS", missing
}

// ticketFields lists the Clock.Ticket / Ticket fields declared outside every method, first seen first.
func (r *run) ticketFields(code []string, in []bool) []ticketField {
	var fields []ticketField
	seen := map[string]bool{}
	for i, ln := range code {
		if in[i+1] || !strings.Contains(ln, "Ticket") {
			continue
		}
		r.rx.each(ln, notWord+`(Clock`+sp+`*\.`+sp+`*)?Ticket`+sp+`*(\[`+sp+`*\])?`+sp+`+`+ident, func(_, d, _ string) bool {
			if d = r.rx.sub(d, `.*`+sp); !seen[d] {
				seen[d] = true
				fields = append(fields, ticketField{d, i + 1})
			}
			return true
		})
	}
	return fields
}

// reachText is text plus, joined by "\n", the body of every same-file method reachable from start
// through unqualified calls, each added once (breadth first, as ticket.awk walks them).
func (r *run) reachText(body map[string]string, start, text string) string {
	var b strings.Builder
	b.WriteString(text)
	hop := map[string]bool{start: true}
	queue := []string{start}
	for qi := 0; qi < len(queue); qi++ {
		r.rx.calls(body[queue[qi]], func(id string) {
			if _, ok := body[id]; ok && !hop[id] {
				hop[id] = true
				queue = append(queue, id)
				b.WriteString("\n")
				b.WriteString(body[id])
			}
		})
	}
	return b.String()
}

// paramsOf is params_of(): the parameter names of method nm, read from the signature text from the line
// naming it (up to 4 lines above the brace line l) to the first ")".
func (r *run) paramsOf(code []string, nm string, l int) string {
	name := notRef + regexp.QuoteMeta(nm) + sp + `*\(`
	sig := ""
	for j := l; j >= 1 && j >= l-4; j-- {
		if r.rx.has(code[j-1], name) {
			sig = joinLines(code, j, l, "", "\n")
			break
		}
	}
	if sig == "" {
		return ""
	}
	sig = r.rx.sub(sig, `^.*`+name)
	sig = strings.ReplaceAll(r.rx.sub(sig, `\).*`), "\n", " ")
	var out strings.Builder
	for _, p := range strings.Split(sig, ",") {
		p = r.rx.gsub(p, `\[`+sp+`*\]`)
		p = r.rx.sub(p, sp+`+$`)
		if p = r.rx.sub(p, `.*[^A-Za-z0-9_]`); p != "" {
			out.WriteString(" " + p)
		}
	}
	return out.String()
}

// cancels is cancels(): f.cancel( / f[...].cancel( / this.f.cancel( in scope, or f passed whole (bare or
// this.) to a same-file method whose body cancels one of its own parameters.
func (r *run) cancels(body, params map[string]string, scope, f string) bool {
	cancelOf := func(v string) string {
		return v + sp + `*(\[[^\]]*\]` + sp + `*)?\.` + sp + `*cancel` + sp + `*\(`
	}
	if r.rx.has(scope, notRef+`(this`+sp+`*\.`+sp+`*)?`+cancelOf(f)) {
		return true
	}
	found := false
	// the separator before f excludes ( and ) so the match binds to the innermost call, not an enclosing if (
	r.rx.each(scope, ident+sp+`*\(([^()]*[^A-Za-z0-9_.()])?`+f+sp+`*[,)]`, func(pre, m, _ string) bool {
		nm := r.rx.sub(m, sp+`*\(.*`)
		if r.rx.has(pre, `\.`+sp+`*$`) && !r.rx.has(pre, notWord+`this`+sp+`*\.`+sp+`*$`) {
			return true
		}
		helper, ok := body[nm]
		if !ok {
			return true
		}
		for _, a := range strings.Fields(params[nm]) {
			if r.rx.has(helper, notRef+cancelOf(a)) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

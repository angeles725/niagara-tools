package timers

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// companionPass is companion-flag: a boolean/int/long class FIELD (declared at brace depth 1) assigned
// true in the same method body as a Clock.schedule* call must be assigned false in stopped() or
// started(); a clear only in the expiry handler does not survive a stop/restart cycle.
func (r *run) companionPass(srcs []*source) {
	for _, s := range srcs {
		if flag := r.stuckFlag(s.Stripped(), s); flag != "" {
			r.row(lint.Fail, "companion-flag", s, "flag '"+flag+"' set beside Clock.schedule* not cleared in stopped()/started()")
		}
	}
}

// stuckFlag returns the first field flag set true beside a schedule and never cleared in a lifecycle
// callback, or "".
func (r *run) stuckFlag(st []string, s *source) string {
	fields := r.depthOneFields(st)
	if len(fields) == 0 {
		return ""
	}
	flag := ""
	for _, m := range s.Methods() {
		body := joinLines(st, m.Start, m.End, " ", "")
		if !strings.Contains(body, "Clock.schedule") {
			continue
		}
		r.rx.each(body, `[a-zA-Z_][a-zA-Z0-9_]*`+sp+`*=`+sp+`*true`+sp+`*;`, func(_, frag, _ string) bool {
			id := r.rx.gsub(r.rx.sub(frag, sp+`*=.*`), sp)
			if id != "" && fields[id] {
				flag = id
				return false
			}
			return true
		})
		if flag != "" {
			break
		}
	}
	if flag == "" || r.clearedInLifecycle(st, flag) {
		return ""
	}
	return flag
}

// depthOneFields is Phase 1: boolean/int/long names declared on a line that starts at brace depth 1
// (braces in literals counted too, as the awk did).
func (r *run) depthOneFields(st []string) map[string]bool {
	fields := map[string]bool{}
	depth := 0
	for _, ln := range st {
		prev := depth
		depth += strings.Count(ln, "{") - strings.Count(ln, "}")
		if prev != 1 || !r.rx.has(ln, sp+`(boolean|int|long)`+sp+`[a-zA-Z_][a-zA-Z0-9_]*`+sp+`*[;=]`) {
			continue
		}
		t := r.rx.sub(ln, `^`+sp+`+`)
		for r.rx.has(t, `^(private|protected|public|static|final|volatile|transient)`+sp) {
			t = r.rx.sub(t, `^[a-z]+`+sp+`+`)
		}
		t = r.rx.sub(t, `^(boolean|int|long)`+sp+`+`)
		if id, ok := r.rx.find(t, `^[a-zA-Z_][a-zA-Z0-9_]*`); ok {
			fields[id] = true
		}
	}
	return fields
}

// clearedInLifecycle is Pass 2: some line of a stopped()/started() body (from its signature line to the
// brace that returns its depth to 0) assigns flag = false.
func (r *run) clearedInLifecycle(st []string, flag string) bool {
	inLC, depth := false, 0
	for _, ln := range st {
		if !inLC && r.rx.has(ln, `void`+sp+`+(stopped|started)`+sp+`*\(`) {
			inLC, depth = true, 0
		}
		if !inLC {
			continue
		}
		if r.rx.has(ln, flag+sp+`*=`+sp+`*false`) {
			return true
		}
		for j := 0; j < len(ln); j++ {
			if ln[j] == '{' {
				depth++
			}
			if ln[j] == '}' && depth > 0 {
				if depth--; depth == 0 {
					inLC = false
					break
				}
			}
		}
	}
	return false
}

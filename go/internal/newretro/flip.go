package newretro

import "strings"

const (
	sectionOpen  = "<!-- build-state.v1 -->"
	sectionClose = "<!-- /build-state.v1 -->"
	space        = " \t\n\v\f\r" // awk's [[:space:]] in the C locale
)

// Flip turns the first `retro_pending: false` of the build-state.v1 section whose `module:` value (or, before
// any module: line, whose `## <name>` heading) is module into `true`. Every other line is kept byte for byte;
// the text comes back with a final newline, as awk prints it.
func Flip(text, module string) string {
	lines := strings.Split(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var b strings.Builder
	in, target, gotModule, flipped := false, false, false, false
	for _, ln := range lines {
		switch {
		case ln == sectionOpen:
			in, target, gotModule, flipped = true, false, false, false
		case ln == sectionClose:
			in, target, gotModule = false, false, false
		case in && strings.HasPrefix(ln, "module:"):
			gotModule = true
			target = target || moduleValue(ln) == module
		case in && !gotModule && strings.HasPrefix(ln, "## "):
			target = target || strings.TrimRight(ln[3:], space) == module
		case in && target && !flipped && pendingFalse(ln):
			ln, flipped = strings.Replace(ln, "false", "true", 1), true
		}
		b.WriteString(ln + "\n")
	}
	return b.String()
}

// moduleValue is a `module:` line's value: leading space, a `#` comment with the space before it and
// trailing space removed.
func moduleValue(ln string) string {
	v := strings.TrimLeft(strings.TrimPrefix(ln, "module:"), space)
	if i := strings.IndexByte(v, '#'); i >= 0 {
		v = v[:i]
	}
	return strings.TrimRight(v, space)
}

// pendingFalse matches awk's /^retro_pending:[[:space:]]*false/.
func pendingFalse(ln string) bool {
	rest, ok := strings.CutPrefix(ln, "retro_pending:")
	return ok && strings.HasPrefix(strings.TrimLeft(rest, space), "false")
}

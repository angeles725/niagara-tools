package newretro

import "strings"

const (
	sectionOpen  = "<!-- build-state.v1 -->"
	sectionClose = "<!-- /build-state.v1 -->"
	space        = " \t\n\v\f\r" // awk's [[:space:]] in the C locale
)

// Outcome is what Flip found in the module's envelope.
type Outcome int

const (
	NoSection   Outcome = iota // no build-state.v1 section names the module
	NoPending                  // a section names it but holds no retro_pending: false|true line
	AlreadyTrue                // its retro_pending is already true; the text is unchanged
	Flipped                    // a retro_pending: false became true
)

// Flip turns the first `retro_pending: false` of each build-state.v1 section whose `module:` value (or, before
// any module: line, whose `## <name>` heading) is module into `true`, and says what it found. Every other line
// is kept byte for byte; the text comes back with a final newline, as awk prints it.
func Flip(text, module string) (string, Outcome) {
	lines := strings.Split(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	var b strings.Builder
	in, target, gotModule, flipped := false, false, false, false
	out := NoSection
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
		case in && target && !flipped && pending(ln, "false"):
			ln, flipped, out = strings.Replace(ln, "false", "true", 1), true, Flipped
		case in && target && pending(ln, "true"):
			out = max(out, AlreadyTrue)
		}
		if in && target {
			out = max(out, NoPending)
		}
		b.WriteString(ln + "\n")
	}
	return b.String(), out
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

// pending matches /^retro_pending:[[:space:]]*<value>/ (the awk matched "false" so).
func pending(ln, value string) bool {
	rest, ok := strings.CutPrefix(ln, "retro_pending:")
	return ok && strings.HasPrefix(strings.TrimLeft(rest, space), value)
}

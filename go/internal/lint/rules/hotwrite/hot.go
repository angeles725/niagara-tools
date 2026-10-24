package hotwrite

import (
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint/rules/linescan"
)

// Whether a setter call marks the station dirty on every cycle. Baja's ComplexSlotMap skips a write whose new
// value equals the current one when the value is simple: setDouble (`!BDouble.equals(oldValue, newValue)`,
// organized/baja/baja/vineflower/com/tridium/sys/schema/ComplexSlotMap.java:1239), setBoolean
// (`oldValue != newValue`, :860), set(Property, BValue) (`!oldValue.isSimple() || !oldValue.equals(newValue)`,
// :637), and setInt/setLong/setFloat/setString alike. So a simple-typed slot is hot only when its value changes
// every cycle, and a non-simple one (a BStatusValue, a BComponent) on every write. What can be read
// statically: the slot's type (its newProperty default), an enclosing if that tests the slot's own getter or a
// change flag, and whether the written value is computed fresh: a clock read, the slot's own getter in an
// arithmetic expression, or a field the tree accumulates by a non-literal delta (`hours[k] += dtH`). What
// cannot: that a mirrored model field really changes on every call, or that a helper method returns a new
// value; a field updated only by event counters (`++`, `+= 1`) or plain assignments counts as changing on
// events, not every cycle.

var (
	clockRead   = rx(`Clock\.(millis|ticks|nanoTime)[[:space:]]*\(|System\.(currentTimeMillis|nanoTime)[[:space:]]*\(|BAbsTime\.now[[:space:]]*\(`)
	arithmetic  = rx(`[-+*/%]`)
	identifier  = rx(`[A-Za-z_][A-Za-z0-9_]*`)
	ifHead      = rx(`(^|[^A-Za-z0-9_])if[[:space:]]*\(`)
	changeFlag  = rx(`(^|[^A-Za-z0-9_])[A-Za-z_][A-Za-z0-9_]*([Cc]hanged|[Dd]irty)[A-Za-z0-9_]*`)
	compound    = rx(`([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*(\[[^]]*\])?[[:space:]]*[-+]=[[:space:]]*([^;]*)`)
	selfArith   = rx(`([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*(\[[^]]*\])?[[:space:]]*=[[:space:]]*\(?[[:space:]]*([A-Za-z_][A-Za-z0-9_]*)[[:space:]]*(\[[^]]*\])?[[:space:]]*[-+][[:space:]]*([^;]*)`)
	literalHead = rx(`^[[:space:]]*\(?[[:space:]]*[0-9][0-9.]*[dDfFlL]?([^A-Za-z0-9_.]|$)`)
)

// Accumulators are the fields the files grow by a non-literal delta each time a statement runs:
// `f += <expr>`, `f[i] -= <expr>` or `f = f + <expr>` with <expr> not starting with a number (an event counter
// such as `n++`, `n += 1` or `idx = (idx + 1) % N` is not one).
func Accumulators(files []*javasrc.File) map[string]bool {
	acc := map[string]bool{}
	for _, f := range files {
		for _, ln := range linescan.Code(f) {
			for _, m := range compound.FindAllStringSubmatch(ln, -1) {
				if !literalHead.MatchString(m[3]) {
					acc[m[1]] = true
				}
			}
			for _, m := range selfArith.FindAllStringSubmatch(ln, -1) {
				if m[1] == m[3] && !literalHead.MatchString(m[5]) {
					acc[m[1]] = true
				}
			}
		}
	}
	return acc
}

// Hot reports whether the setter call of p at lines[line], byte at (just after its "("), inside body b, can
// mark the station dirty on every cycle: false under an enclosing if that tests p's getter or a change flag,
// false for a simple-typed p unless the value is fresh (fresh), true otherwise.
func Hot(lines []string, b Body, line, at int, p Prop, acc map[string]bool) bool {
	getter := "get" + Setter(p.Name)[len("set"):] + "("
	if guardedBy(lines, b, line, at, getter) {
		return false
	}
	if !p.Simple {
		return true
	}
	args := Args(joined(lines, line, at, b.End-line))
	value := strings.Join(args, ",")
	if clockRead.MatchString(value) || strings.Contains(value, getter) && arithmetic.MatchString(value) {
		return true
	}
	for _, id := range identifier.FindAllString(value, -1) {
		if acc[id] {
			return true
		}
	}
	return false
}

// guardedBy reports an if around the call whose condition calls getter or reads a change flag (a name holding
// Changed or Dirty): the call's own statement (`if (...) setX(...)`) or the header of a block that encloses it
// within the body (`if (...) { ...; setX(...)`).
func guardedBy(lines []string, b Body, line, at int, getter string) bool {
	text := strings.Join(append(lines[b.Start:line:line], lines[line][:at]), "\n")
	guards := func(head string) bool {
		return ifHead.MatchString(head) && (strings.Contains(head, getter) || changeFlag.MatchString(head))
	}
	if guards(text[strings.LastIndexAny(text, ";{}")+1:]) {
		return true
	}
	depth := 0
	for i := len(text) - 1; i >= 0; i-- {
		switch text[i] {
		case '}':
			depth++
		case '{':
			if depth > 0 {
				depth--
			} else if guards(text[strings.LastIndexAny(text[:i], ";{}")+1 : i]) {
				return true
			}
		}
	}
	return false
}

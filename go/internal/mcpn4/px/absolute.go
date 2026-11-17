package px

import (
	"strings"
	"unicode"
)

// Target is where a binding ORD points once read against the view's base: an absolute slot path
// (Path, a trailing |slot: hop dropped so the component is what is checked), a component handle
// (Handle), or neither with Why naming the reason it cannot be checked.
type Target struct {
	Path   string
	Handle string
	Why    string
}

// Absolutize reads ord against base, the slot path of the component the view is on ("" when
// unknown). Forms: station:|slot:/a, local:|slot:/a, slot:/a (absolute); slot:a, slot:../a and
// ../a (relative to base); h:<handle>. Any other scheme, a chained ORD and a relative ORD with no
// base are named in Why, never guessed.
func Absolutize(ord, base string) Target {
	s := strip(ord)
	for _, host := range []string{"station:|", "local:|"} {
		s = strings.TrimPrefix(s, host)
	}
	if loc := slotHop.FindStringIndex(s); loc != nil {
		s = s[:loc[0]]
	}
	if strings.Contains(s, "|") {
		return Target{Why: "a chained ORD (" + ord + ") is not checked"}
	}
	var rel string
	switch {
	case strings.HasPrefix(s, "h:"):
		if h := s[2:]; h != "" && plain(h) && !strings.Contains(h, "/") {
			return Target{Handle: h}
		}
		return Target{Why: "not a plain slot path or handle: " + ord}
	case strings.HasPrefix(s, "slot:/"):
		return slotTarget("/", s[len("slot:"):], ord)
	case strings.HasPrefix(s, "slot:"):
		rel = s[len("slot:"):]
	case strings.HasPrefix(s, "../") || strings.HasPrefix(s, "./"):
		rel = s
	default:
		if i := strings.IndexByte(s, ':'); i > 0 {
			return Target{Why: "scheme " + s[:i+1] + " is not checked (not a station slot)"}
		}
		return Target{Why: "not a station slot ORD: " + ord}
	}
	if base == "" {
		return Target{Why: "a relative ORD needs the view's base (the component the view is on), unknown here"}
	}
	return slotTarget(base, rel, ord)
}

// slotTarget joins rel to base ("/" for an absolute rel), resolving . and .. segments.
func slotTarget(base, rel, ord string) Target {
	if strings.Contains(rel, "//") || !plain(rel) {
		return Target{Why: "not a plain slot path: " + ord}
	}
	segs := []string{}
	for _, p := range strings.Split(base, "/") {
		if p != "" {
			segs = append(segs, p)
		}
	}
	for _, p := range strings.Split(rel, "/") {
		switch p {
		case "", ".":
		case "..":
			if len(segs) == 0 {
				return Target{Why: "the ORD goes above the station root: " + ord}
			}
			segs = segs[:len(segs)-1]
		default:
			segs = append(segs, p)
		}
	}
	return Target{Path: "/" + strings.Join(segs, "/")}
}

// plain reports a non-empty text with no control or space character.
func plain(s string) bool {
	return s != "" && strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) < 0
}

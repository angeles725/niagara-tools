package citeparse

import (
	"regexp"
	"strings"
)

var sectionSpan = regexp.MustCompile(`^(\d+(?:\.\d+)*)[–-]§?(\d+(?:\.\d+)*)$`)

// corpusClause reads `B<a>[/B<b>…][ locator][, B<c>[ locator]…]`; a part that opens with § C row rule
// adds to the last block named.
func corpusClause(c, rest string) []Pointer {
	var out []Pointer
	for _, part := range strings.Split(rest, ", ") {
		part = strings.TrimSpace(part)
		if m := blockList.FindStringSubmatch(part); m != nil {
			for _, b := range strings.Split(m[1], "/") {
				out = append(out, Pointer{Kind: Corpus, Target: b, Raw: c})
			}
			corpusLocator(&out[len(out)-1], m[2])
			continue
		}
		if len(out) == 0 {
			if bareID.MatchString(part) {
				return []Pointer{{Kind: Bare, Target: part, Raw: c}}
			}
			return unparsed(c, "no B<n> block named")
		}
		corpusLocator(&out[len(out)-1], part)
	}
	return out
}

func corpusLocator(p *Pointer, s string) {
	f := strings.Fields(s)
	for i := 0; i < len(f); i++ {
		w := f[i]
		switch {
		case strings.HasPrefix(w, "§") && len(w) > len("§"):
			p.Sections = append(p.Sections, sectionRange(strings.TrimRight(strings.TrimPrefix(w, "§"), ".:)"))...)
		case claimRe.MatchString(w):
			p.Claims = append(p.Claims, w)
		case (w == "row" || w == "rows") && i+1 < len(f):
			if rs, ok := ParseRanges(f[i+1]); ok {
				p.Rows, i = append(p.Rows, rs...), i+1
			} else {
				p.Notes = append(p.Notes, w)
			}
		case w == "rule" && i+1 < len(f):
			p.Rules, i = append(p.Rules, f[i+1]), i+1
		case w == "—" || w == "--":
			p.Notes = append(p.Notes, strings.Join(f[i:], " "))
			return
		default:
			p.Notes = append(p.Notes, w)
		}
	}
}

// sectionRange expands `958.3–958.4`, `867.2-3` and `12.3-§12.5` to both ends (a short end keeps the
// leading numbers of the first); anything else is one section.
func sectionRange(s string) []string {
	m := sectionSpan.FindStringSubmatch(s)
	if m == nil {
		return []string{s}
	}
	from, to := strings.Split(m[1], "."), strings.Split(m[2], ".")
	if len(to) < len(from) {
		to = append(append([]string(nil), from[:len(from)-len(to)]...), to...)
	}
	return []string{m[1], strings.Join(to, ".")}
}

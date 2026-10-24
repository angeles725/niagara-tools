package citeparse

import "strings"

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
			p.Sections = append(p.Sections, strings.TrimRight(strings.TrimPrefix(w, "§"), ".:)"))
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

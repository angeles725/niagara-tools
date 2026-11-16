package delays

import "strings"

// declarations is pass 1: BRelTime constants, slot MIN facets (slotomatic and @NiagaraProperty) and the
// slot each BRelTime getter reads.
func (r *file) declarations() {
	inProp, buf, name := false, "", ""
	for i, ln := range r.lines {
		if loc := constDecl.FindStringIndex(ln); loc != nil {
			if c := ident.FindString(ln[loc[1]:]); c != "" {
				if v := literalMs(ln); v >= 0 {
					r.constMs[c] = v
				}
			}
		}
		if loc := propDecl.FindStringIndex(ln); loc != nil {
			if p := slotName.FindString(ln[loc[1]:]); p != "" {
				if v := facetMinMs(ln); v >= 0 {
					r.propMin[p] = v
				}
			}
		}
		if !inProp {
			if strings.Contains(ln, "@NiagaraProperty") {
				inProp, buf, name = true, ln, ""
			}
		} else {
			buf += " " + ln
		}
		if inProp {
			if name == "" {
				if loc := propName.FindStringIndex(buf); loc != nil {
					seg := buf[loc[0]:]
					seg = seg[strings.IndexByte(seg, '"')+1:]
					name = seg[:strings.IndexByte(seg, '"')]
				}
			}
			if strings.Count(buf, "(")-strings.Count(buf, ")") <= 0 {
				if name != "" {
					if v := facetMinMs(buf); v != -2 {
						r.propMin[name] = v
					}
				}
				inProp, buf, name = false, "", ""
			}
		}
		if !inProp {
			r.getter(i, ln)
		}
	}
}

// getter maps `public BRelTime getXxx(` on line i to the slot a get("slot") or get(slot) names on that
// line or one of the next three.
func (r *file) getter(i int, ln string) {
	loc := getterDecl.FindStringIndex(ln)
	if loc == nil {
		return
	}
	seg := ln[loc[0]:]
	seg = seg[getterMods.FindStringIndex(seg)[1]:]
	g := getterName.FindString(seg)
	if g == "" {
		return
	}
	for j := i; j <= i+3 && j < len(r.lines); j++ {
		if m := getQuoted.FindString(r.lines[j]); m != "" {
			r.getterSlot[g] = m[len(`get("`) : len(m)-2]
			return
		}
		if m := getField.FindString(r.lines[j]); m != "" {
			r.getterSlot[g] = m[len("get(") : len(m)-1]
			return
		}
	}
}

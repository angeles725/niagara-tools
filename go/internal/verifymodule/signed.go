package verifymodule

import "strings"

var sigRe = rx(`^META-INF/[^/]+\.(SF|RSA|DSA|EC)$`)

func (r *run) signed() bool {
	var sigs, ok, orphan []string
	for _, n := range r.jar.list.lines {
		if sigRe.MatchString(n) && r.jar.list.shown(n) {
			sigs = append(sigs, n)
		}
	}
next:
	for _, sf := range sigs {
		if !strings.HasSuffix(sf, ".SF") {
			continue
		}
		alias := strings.TrimSuffix(strings.TrimPrefix(sf, "META-INF/"), ".SF")
		for _, ext := range []string{"RSA", "DSA", "EC"} {
			if contains(sigs, "META-INF/"+alias+"."+ext) {
				ok = append(ok, alias)
				continue next
			}
		}
		orphan = append(orphan, sf)
	}
	switch {
	case len(ok) > 0:
		r.row("PASS", "signed", "signed by alias "+strings.Join(ok, ", ")+" (META-INF/<ALIAS>.SF + signature block)")
		return true
	case len(orphan) > 0:
		r.row("FAIL", "signed", strings.Join(orphan, ", ")+" without its .RSA/.DSA/.EC signature block (jar not signed)")
	default:
		r.row("FAIL", "signed", "no META-INF/*.SF (jar not signed)")
	}
	return false
}

package verifymodule

import (
	"archive/zip"
	"fmt"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/jarversion"
)

// classes is what grep '\.class$' prints of LIST.
func (j *jarFile) classes() []string {
	var out []string
	for _, n := range j.list.lines {
		if strings.HasSuffix(n, ".class") && j.list.shown(n) {
			out = append(out, n)
		}
	}
	return out
}

// bytecode reads each class entry's CAFEBABE magic and two-byte major (jarversion.ClassMajor). The bash
// read byte 7 alone with od, so a major of 308 passed as 52, and so did an entry without the magic; it also
// named a class shorter than 6 bytes on stderr (od's error), which this does not (declared difference,
// truth-bytecode-magic and edge-bytecode-mixed).
func (r *run) bytecode() bool {
	n, bad, mr, first, ign := 0, 0, 0, "", ""
	for _, c := range r.jar.classes() {
		if strings.HasPrefix(c, "META-INF/versions/") {
			mr++
			continue
		}
		n++
		h, err := jarversion.Head(r.jar.first[c])
		major, isClass := jarversion.ClassMajor(h)
		if err == nil && isClass && major == jarversion.Java8Major {
			continue
		}
		bad++
		if first != "" {
			continue
		}
		switch {
		case err != nil:
			first = c + " unreadable (" + jarversion.Reason(err) + ")"
		case !isClass:
			first = c + " not a class file (no CAFEBABE header)"
		default:
			first = c + " major=" + itoa(major)
		}
	}
	if mr != 0 {
		ign = fmt.Sprintf("; %d multi-release classes ignored", mr)
	}
	switch {
	case n == 0 && ign != "":
		r.row("FAIL", "bytecode", "no .class entries outside META-INF/versions/"+ign)
	case n == 0:
		r.row("FAIL", "bytecode", "no .class entries")
	case bad == 0:
		r.row("PASS", "bytecode", fmt.Sprintf("%d classes, all major 52%s", n, ign))
		return true
	default:
		r.row("FAIL", "bytecode", fmt.Sprintf("%d of %d classes not major 52 (expected Java 8); first: %s%s", bad, n, first, ign))
	}
	return false
}

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

func (r *run) types() bool {
	if r.jar.mx == "" {
		r.row("FAIL", "types", "META-INF/module.xml missing (not an N4 module jar)")
		return false
	}
	missing, firstMiss := 0, ""
	for _, cls := range r.jar.types {
		// The exact entry; the bash's grep -qx read it as a basic regex, its "." matching any byte
		// (declared difference, truth-types-exact).
		if !r.jar.has(strings.ReplaceAll(cls, ".", "/") + ".class") {
			missing++
			if firstMiss == "" {
				firstMiss = cls
			}
		}
	}
	if missing == 0 {
		r.row("PASS", "types", fmt.Sprintf("%d declared types resolve to classes", len(r.jar.types)))
		return true
	}
	r.row("FAIL", "types", fmt.Sprintf("%d of %d declared types have no class; first: %s", missing, len(r.jar.types), firstMiss))
	return false
}

func (r *run) baja() bool {
	if r.target == "" {
		r.row("SKIP", "baja", "no --target-version")
		return true
	}
	if r.jar.mx == "" {
		r.row("FAIL", "baja", "META-INF/module.xml missing")
		return false
	}
	bv := ""
	for _, i := range r.jar.doc.Find("dependency") {
		e := r.jar.doc.Elements()[i]
		if name, _ := e.Attr("name"); name == "baja" {
			if bv, _ = e.Attr("vendorVersion"); bv != "" {
				break
			}
		}
	}
	if bv == "" {
		r.row("FAIL", "baja", "no baja dependency in module.xml")
		return false
	}
	t := r.target
	if verLE(bv, t) {
		r.row("PASS", "baja", fmt.Sprintf("stamped baja %s <= target %s", bv, t))
		return true
	}
	r.row("FAIL", "baja", fmt.Sprintf("stamped baja %s > target %s (a %s station rejects it; build against the %s niagara_home)", bv, t, t, t))
	return false
}

// storedCheck counts the entries whose zip method is not STORED: deflated, and any other compression
// (bzip2), which Workbench cannot re-sign either. The kit calls them all "deflated". The bash counted the
// `unzip -v` lines holding the text "Defl:", so an entry name or jar path holding it was miscounted
// (declared difference, truth-stored-method).
func (r *run) storedCheck() bool {
	if !r.stored {
		r.row("SKIP", "stored", "no --stored")
		return true
	}
	d := 0
	for _, f := range r.jar.zr.File {
		if f.Method != zip.Store {
			d++
		}
	}
	if d == 0 {
		r.row("PASS", "stored", "0 deflated entries (Workbench re-sign safe)")
		return true
	}
	r.row("FAIL", "stored", fmt.Sprintf("%d deflated entries (run stored-repack.sh before re-signing in Workbench)", d))
	return false
}

// rcBackup prints no row for a clean rc/.
func (r *run) rcBackup() bool {
	var names []string
	for _, n := range r.jar.list.lines {
		b := n[strings.LastIndexByte(n, '/')+1:]
		if strings.HasPrefix(n, "rc/") && (strings.HasSuffix(b, "~") || strings.HasSuffix(b, ".orig") || strings.Contains(b, ".bak")) {
			names = append(names, n)
		}
	}
	joined := strings.TrimRight(strings.Join(names, " "), " ")
	switch {
	case joined == "":
		return true
	case r.strict:
		r.row("FAIL", "rcbackup", "editor/backup files under rc/ (drop them): "+joined)
		return false
	}
	r.row("WARN", "rcbackup", "editor/backup files under rc/ (drop them): "+joined)
	return true
}

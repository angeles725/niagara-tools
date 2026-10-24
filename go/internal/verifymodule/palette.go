package verifymodule

import (
	"fmt"
	"regexp"
	"strings"
)

// paletteScan is the bash palette_scan awk over the module.palette text: one finding per line, ENTRY (a
// component entry), FOLDER <t> (a baja Folder under any alias), BADTYPE <t> <why> (plus BADENTRY when that
// element carries n=), UNTERM (a <!-- with no -->, which ends the document there; printed first).
func paletteScan(pal string) []string {
	var out []string
	doc := " " + strings.Join(here(pal), " ") // { doc = doc " " $0 } over the here-string's records
	for {
		i := strings.Index(doc, "<!--")
		if i < 0 {
			break
		}
		j := strings.Index(doc[i+4:], "-->")
		if j < 0 {
			out = append(out, "UNTERM")
			doc = doc[:i]
			break
		}
		doc = doc[:i] + doc[i+4+j+3:]
	}
	mod := map[string]string{}
	for _, e := range strings.Split(doc, "<")[1:] {
		if !paletteElemRe.MatchString(e) {
			continue
		}
		if k := strings.IndexByte(e, '>'); k >= 0 {
			e = " " + e[1:k]
		} else {
			e = " " + e[1:]
		}
		if m, ok := attr(e, "m"); ok {
			if k := strings.IndexByte(m, '='); k >= 0 {
				mod[m[:k]] = m[k+1:]
			}
		}
		t, ok := attr(e, "t")
		if !ok {
			continue
		}
		sym, ty := "", t
		if c := strings.IndexByte(t, ':'); c >= 0 {
			sym, ty = t[:c], t[c+1:]
		}
		_, declared := mod[sym]
		bad := ""
		switch {
		case strings.Contains(t, "."):
			bad = "is a Java class name, not <symbol>:<Type>"
		case sym == "":
			bad = "has no <symbol>: prefix"
		case !declared:
			bad = "uses symbol " + sym + " that no m=\"" + sym + "=<module>\" declared"
		}
		_, named := attr(e, "n")
		if bad != "" {
			out = append(out, "BADTYPE "+t+" "+bad)
			if named {
				out = append(out, "BADENTRY")
			}
			continue
		}
		folder := mod[sym] == "baja" && (ty == "Folder" || ty == "UnrestrictedFolder")
		if mod[sym] == "baja" && ty == "Folder" {
			out = append(out, "FOLDER "+t)
		}
		if _, valued := attr(e, "v"); !folder && named && !valued {
			out = append(out, "ENTRY")
		}
	}
	return out
}

var paletteElemRe = regexp.MustCompile(`^p[[:space:]/>]`)

// attr is the value of attribute a in element text e (the first ` a="..."`), and whether e holds one.
func attr(e, a string) (string, bool) {
	m := regexp.MustCompile(`[[:space:]]` + a + `="[^"]*"`).FindString(e)
	if m == "" {
		return "", false
	}
	return m[len(a)+3 : len(m)-1], true
}

func countOf(rows []string, want ...string) int {
	n := 0
	for _, x := range rows {
		if contains(want, x) {
			n++
		}
	}
	return n
}

func (r *run) palette() bool {
	if !r.jar.has("module.palette") {
		r.row("SKIP", "palette", "no module.palette")
		return true
	}
	scan, ok, folders := paletteScan(r.jar.text("module.palette")), true, []string{}
	for _, f := range scan {
		if bad, isBad := strings.CutPrefix(f, "BADTYPE "); isBad {
			r.row("FAIL", "palette-type", "module.palette t=\""+bad+"\" — write t=\"<symbol>:<Type>\" with m=\"<symbol>=<module>\" declared once [ev: retro n4-course-authoring-files-and-libraries Δ3]")
			ok = false
		} else if t, isFolder := strings.CutPrefix(f, "FOLDER "); isFolder {
			folders = append(folders, t)
		}
	}
	if len(folders) > 0 {
		folders = sortU(folders)
		for i, t := range folders {
			folders[i] = `t="` + t + `"`
		}
		r.row("WARN", "palette-root", "module.palette uses a baja Folder ("+strings.Join(folders, " ")+"), root or nested: access-controlled — use the baja UnrestrictedFolder (t=\"b:UnrestrictedFolder\") for ungated palette expansion [ev: corpus B746 §746.1]")
	}
	if contains(scan, "UNTERM") {
		r.row("FAIL", "palette-parse", "module.palette has an unterminated comment (<!-- with no -->): the rest of the document is not read")
		return false
	}
	entries, types := countOf(scan, "ENTRY"), newGrepText(r.jar.mx).count(typeElemRe)
	if entries == 0 && types >= 1 {
		msg := fmt.Sprintf("empty palette but module declares %d type(s) — nothing to drag in Workbench (add one <p n=.../> per exposed type)", types)
		if r.strict {
			r.row("FAIL", "palette", msg)
			return false
		}
		r.row("WARN", "palette", msg)
		return ok
	}
	r.row("PASS", "palette", fmt.Sprintf("module.palette has %d component entries", entries))
	return ok
}

var typeElemRe = regexp.MustCompile(`<type `)

func (r *run) wbScaffold() bool {
	if !strings.HasSuffix(basename(r.jar.path, ".jar"), "-wb") {
		r.row("SKIP", "wb-scaffold", "not a -wb jar")
		return true
	}
	if !r.jar.has("module.palette") {
		r.row("SKIP", "wb-scaffold", "no module.palette")
		return true
	}
	classes := r.jar.list.count(classEntryRe)
	entries := countOf(paletteScan(r.jar.text("module.palette")), "ENTRY", "BADENTRY")
	if classes == 0 && entries == 0 {
		msg := "empty -wb jar: 0 classes, 0 palette entries — nothing packaged (add src or a palette entry)"
		if r.strict {
			r.row("FAIL", "wb-scaffold", msg)
			return false
		}
		r.row("WARN", "wb-scaffold", msg)
		return true
	}
	r.row("PASS", "wb-scaffold", fmt.Sprintf("ok (%d classes, %d palette entries)", classes, entries))
	return true
}

var classEntryRe = regexp.MustCompile(`\.class$`)

package ntplmanifest

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// moduleORD finds a module:// ORD in an entry's bytes (a value of the bog, an image or a binding of a Px
// file): the module is the name after the scheme.
var moduleORD = regexp.MustCompile(`module://([A-Za-z0-9_]+)`)

// modulesUsed are the modules an entry of the archive uses, each once, in first-use order: in a bog the
// module of each `m="<alias>=<module>"` declaration (several are separated by blanks), in a Px file each
// <import><module name>, and in either the module of each module:// ORD. why is the ntpl-uses message of
// an entry that is not well-formed XML: its modules are then not known, never "none".
func modulesUsed(name string, data []byte) (mods []string, why string) {
	doc, err := xmlsrc.Parse(name, data)
	if err != nil {
		var pos *xmlsrc.Error
		if errors.As(err, &pos) {
			return nil, fmt.Sprintf(entryBadXML, name, fmt.Sprintf("%d:%d: %s", pos.Line, pos.Col, pos.Msg))
		}
		return nil, fmt.Sprintf(entryBadXML, name, err)
	}
	seen := map[string]bool{}
	add := func(m string) {
		if m != "" && !seen[m] {
			seen[m] = true
			mods = append(mods, m)
		}
	}
	elems := doc.Elements()
	for _, e := range elems {
		if v, ok := e.Attr("m"); ok {
			for _, decl := range strings.Fields(v) {
				if _, module, found := strings.Cut(decl, "="); found {
					add(module)
				}
			}
		}
		if e.Name == "module" && e.Parent >= 0 && elems[e.Parent].Name == "import" {
			v, _ := e.Attr("name")
			add(v)
		}
	}
	for _, m := range moduleORD.FindAllSubmatch(data, -1) {
		add(string(m[1]))
	}
	return mods, ""
}

// moduleOf is the module a part name belongs to: fox-rt is fox; a name with no part suffix is its own.
func moduleOf(part string) string {
	for _, s := range []string{"-rt", "-ux", "-wb", "-se", "-doc"} {
		if strings.HasSuffix(part, s) {
			return strings.TrimSuffix(part, s)
		}
	}
	return part
}

// moduleSet is the set of modules the part names belong to, each name also kept as written.
func moduleSet(parts map[string]bool, names ...[]string) map[string]bool {
	out := map[string]bool{}
	for p := range parts {
		out[p], out[moduleOf(p)] = true, true
	}
	for _, list := range names {
		for _, n := range list {
			out[n], out[moduleOf(n)] = true, true
		}
	}
	return out
}

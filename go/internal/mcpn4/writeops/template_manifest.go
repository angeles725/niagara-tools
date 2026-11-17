package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/plan"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// tplManifest is what a deploy reads of template-manifest.xml (template-rt ManifestXMLReader):
// the root's identity and kind, the settings (TemplateManifest.Value: n, typ (absent is num),
// req, units, min, max, slotPath) and the module dependencies.
type tplManifest struct {
	title, version, vendor, id string
	isStation, isApplication   bool
	settings                   []tplSetting
	deps                       []string // "name vendorVersion rel", as written
}

// tplSetting is one <settings><p> entry.
type tplSetting struct {
	name, typ, units, slotPath string
	minText, maxText           string // as written; "" when absent
	min, max                   float64
	required                   bool
}

// parseTemplateManifest reads the manifest; an XML error or a root other than <template> is a
// refusal.
func parseTemplateManifest(data []byte) (*tplManifest, error) {
	doc, err := xmlsrc.Parse(tplManifestEntry, data)
	if err != nil {
		return nil, plan.ToolError("template manifest is not well-formed XML: " + err.Error())
	}
	elems := doc.Elements()
	if len(elems) == 0 || elems[0].Name != "template" {
		return nil, plan.ToolError("template manifest has no <template> root")
	}
	root := elems[0]
	m := &tplManifest{}
	m.title, _ = root.Attr("title")
	m.version, _ = root.Attr("version")
	m.vendor, _ = root.Attr("vendor")
	m.id, _ = root.Attr("id")
	m.isStation = attrTrue(root, "isStation")
	m.isApplication = attrTrue(root, "isApplication")
	for _, i := range doc.Children(0) {
		switch elems[i].Name {
		case "settings":
			for _, j := range doc.Children(i) {
				s, err := parseSetting(elems[j])
				if err != nil {
					return nil, err
				}
				m.settings = append(m.settings, s)
			}
		case "dependencies":
			for _, j := range doc.Children(i) {
				e := elems[j]
				name, _ := e.Attr("name")
				ver, _ := e.Attr("vendorVersion")
				rel, _ := e.Attr("rel")
				m.deps = append(m.deps, strings.TrimSpace(name+" "+ver+" "+rel))
			}
		}
	}
	return m, nil
}

func parseSetting(e xmlsrc.Element) (tplSetting, error) {
	s := tplSetting{typ: "num"}
	s.name, _ = e.Attr("n")
	if t, ok := e.Attr("typ"); ok {
		s.typ = t
	}
	s.units, _ = e.Attr("units")
	s.slotPath, _ = e.Attr("slotPath")
	s.required = attrTrue(e, "req")
	for _, b := range []struct {
		attr string
		text *string
		val  *float64
	}{{"min", &s.minText, &s.min}, {"max", &s.maxText, &s.max}} {
		v, ok := e.Attr(b.attr)
		if !ok {
			continue
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return s, plan.ToolError("template manifest: setting " + s.name + " has " + b.attr + "=" + v +
				", not a number")
		}
		*b.text, *b.val = v, f
	}
	return s, nil
}

func attrTrue(e xmlsrc.Element, name string) bool {
	v, _ := e.Attr(name)
	return strings.EqualFold(v, "true")
}

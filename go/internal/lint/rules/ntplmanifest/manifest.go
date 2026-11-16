package ntplmanifest

import (
	"errors"
	"fmt"

	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// manifest is what the lint reads of a well-formed template-manifest.xml.
type manifest struct {
	root     string   // the root element's name
	station  bool     // the root carries isStation="true"
	deps     []string // the name of each <dependencies><dependency>, in document order
	nameless []int    // the line of each <dependencies><dependency> with no name
	stray    []int    // the line of each <dependency> whose parent is not <dependencies>
}

// parseManifest reads data, the manifest entry called name, as XML through the shared scanner; why is the
// ntpl-manifest message of a document that is not well-formed (its position is inside the manifest, so the
// path is left out).
func parseManifest(name string, data []byte) (m manifest, why string) {
	doc, err := xmlsrc.Parse(name, data)
	if err != nil {
		var pos *xmlsrc.Error
		if errors.As(err, &pos) {
			return m, fmt.Sprintf(manifestBadXML, name, fmt.Sprintf("%d:%d: %s", pos.Line, pos.Col, pos.Msg))
		}
		return m, fmt.Sprintf(manifestBadXML, name, err)
	}
	elems := doc.Elements()
	for i, e := range elems {
		if i == 0 {
			v, _ := e.Attr(stationAttr)
			m.root, m.station = e.Name, v == "true"
		}
		if e.Name != "dependency" {
			continue
		}
		name, _ := e.Attr("name")
		switch {
		case e.Parent < 0 || elems[e.Parent].Name != "dependencies":
			// Mutation: NTPL-stray -- skipping a <dependency> outside <dependencies> passes a manifest whose needs no reader sees
			m.stray = append(m.stray, e.Line)
		case name == "":
			m.nameless = append(m.nameless, e.Line)
		default:
			m.deps = append(m.deps, name)
		}
	}
	return m, ""
}

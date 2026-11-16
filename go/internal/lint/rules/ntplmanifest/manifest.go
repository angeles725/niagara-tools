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
	nameless []int    // the line of each <dependency> with no name
}

// parseManifest reads data as XML through the shared scanner; why is the ntpl-manifest message of a document
// that is not well-formed (its position is inside the manifest, so the path is left out).
func parseManifest(data []byte) (m manifest, why string) {
	doc, err := xmlsrc.Parse(manifestEntry, data)
	if err != nil {
		var pos *xmlsrc.Error
		if errors.As(err, &pos) {
			return m, fmt.Sprintf(manifestBadXML, fmt.Sprintf("%d:%d: %s", pos.Line, pos.Col, pos.Msg))
		}
		return m, fmt.Sprintf(manifestBadXML, err)
	}
	elems := doc.Elements()
	for i, e := range elems {
		if i == 0 {
			v, _ := e.Attr(stationAttr)
			m.root, m.station = e.Name, v == "true"
		}
		if e.Name != "dependency" || e.Parent < 0 || elems[e.Parent].Name != "dependencies" {
			continue
		}
		if name, _ := e.Attr("name"); name != "" {
			m.deps = append(m.deps, name)
		} else {
			m.nameless = append(m.nameless, e.Line)
		}
	}
	return m, ""
}

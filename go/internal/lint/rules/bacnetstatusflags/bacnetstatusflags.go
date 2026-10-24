// Package bacnetstatusflags is lint-bacnet-status-flags (G36): a BACnet Present_Value proxy point whose facets lack
// statusFlags=true. BBacnetProxyExt.useStatusFlags() is true only for a Present_Value proxy whose point facets carry
// statusFlags=true (vineflower bacnet-rt 4.14.0.162 javax/baja/bacnet/point/BBacnetProxyExt.java:808-822), so without
// the facet a point in alarm, fault or override in the controller reads {ok} in Niagara (types/driver-authoring.md
// § 2). The contract is the deferred G36 spec (build-n4-module-kit/ledger.d/2026-10-23-r8-p1-drivers.md);
// toolbelt/lint-bacnet-status-flags.sh is the wrapper and its header is the user-facing contract.
//
// It is a lint.TreeRule over each station directory: every *.bog under it, read through xmlsrc (a zipped bog through
// its file.xml member). A proxy is a <p> whose t names a Bacnet…ProxyExt type of the bacnet module (its prefix
// resolved through the m="prefix=module" declarations); it reads Present_Value when its propertyId slot is absent
// (the default) or holds ordinal 85; its point is the parent <p>. A proxy with a useStatusFlags slot set true is
// clean. Rows are aggregated per BACnet device, in document order: the parent of the nearest bacnet …PointDeviceExt
// ancestor (so a vendor device type of another module counts), else the nearest ancestor of a bacnet …Device type.
package bacnetstatusflags

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/javasrc"
	"github.com/angeles725/niagara-tools/go/internal/lint"
	"github.com/angeles725/niagara-tools/go/internal/xmlsrc"
)

// ID is the lint name n4lint accepts and every row prints in its check column (the kit's lint-<name> convention).
const ID = "lint-bacnet-status-flags"

// Spec registers the rule: [--strict] [--] <station-dir>... (each on its own), WARN rows (exit 1 under --strict),
// left out of n4lint --all (its roots are station directories, not source roots).
func Spec() lint.Spec {
	return lint.Spec{Tree: rule{}, Mode: lint.Each, ModuleRoot: true, Usage: "usage: lint-bacnet-status-flags.sh [--strict] <station-dir>...",
		Select:     lint.Selection{Names: []string{"*.bog"}},
		EmptyError: lint.EmptyRow{Msg: "no *.bog under the root (wrong path: pass a station directory)"}}
}

type rule struct{}

func (rule) ID() string              { return ID }
func (rule) Severity() lint.Severity { return lint.Warn }

var proxyType = regexp.MustCompile(`^Bacnet[A-Za-z]*ProxyExt$`)

// CheckTree reads each bog; a malformed or unreadable one fails the scan (exit 3) after the other bogs' rows.
func (rule) CheckTree(files []*javasrc.File) ([]lint.Finding, error) {
	var out []lint.Finding
	var bad error
	for _, f := range files {
		doc, err := xmlsrc.Open(f.Path)
		if err != nil {
			bad = err
			continue
		}
		out = append(out, check(f.Path, doc)...)
	}
	return out, bad
}

type tally struct {
	device, first string
	n             int
}

func check(path string, doc *xmlsrc.Doc) []lint.Finding {
	elems := doc.Elements()
	modules := map[string]string{} // prefix -> module
	for _, e := range elems {
		if m, ok := e.Attr("m"); ok {
			for _, decl := range strings.Fields(m) {
				if p, mod, ok := strings.Cut(decl, "="); ok {
					modules[p] = mod
				}
			}
		}
	}
	bacnet := func(i int) (string, bool) { // the bacnet type name of element i
		t, _ := elems[i].Attr("t")
		p, name, ok := strings.Cut(t, ":")
		return name, ok && modules[p] == "bacnet"
	}
	var order []*tally
	byDevice := map[int]*tally{}
	for i, e := range elems {
		if name, ok := bacnet(i); !ok || e.Name != "p" || !proxyType.MatchString(name) || !presentValue(doc, i) ||
			slot(doc, i, "useStatusFlags") == "true" || e.Parent < 0 || statusFlags(slot(doc, e.Parent, "facets")) {
			continue
		}
		dev := -1 // the parent of the point's bacnet …PointDeviceExt (a vendor device subtype too), else a bacnet …Device
		for a := e.Parent; a >= 0; a = elems[a].Parent {
			if name, ok := bacnet(a); ok && strings.HasSuffix(name, "PointDeviceExt") && elems[a].Parent >= 0 {
				dev = elems[a].Parent
				break
			} else if ok && strings.HasSuffix(name, "Device") {
				dev = a
				break
			}
		}
		t := byDevice[dev]
		if t == nil {
			t = &tally{device: "(no BACnet device)", first: slotPath(elems, e.Parent)}
			if dev >= 0 {
				t.device = slotPath(elems, dev)
			}
			byDevice[dev] = t
			order = append(order, t)
		}
		t.n++
	}
	var out []lint.Finding
	for _, t := range order {
		out = append(out, lint.Finding{Path: path, Msg: fmt.Sprintf("%s: %d Present_Value proxy point(s) without facets statusFlags=true (first: %s) -- a field alarm, fault or override reads {ok}", t.device, t.n, t.first)})
	}
	return out
}

// slot is the v attribute of the child <p n="name"> of element i, "" when there is none.
func slot(doc *xmlsrc.Doc, i int, name string) string {
	elems := doc.Elements()
	for _, c := range doc.Children(i) {
		if n, _ := elems[c].Attr("n"); n == name && elems[c].Name == "p" {
			v, _ := elems[c].Attr("v")
			return v
		}
	}
	return ""
}

// presentValue reports a proxy whose propertyId is absent (Present_Value, 85, by default) or names ordinal 85,
// written "85", "85@bacnet:BacnetPropertyIdentifier" or as the tag presentValue.
func presentValue(doc *xmlsrc.Doc, i int) bool {
	v := slot(doc, i, "propertyId")
	ord, _, _ := strings.Cut(v, "@")
	return v == "" || ord == "85" || ord == "presentValue"
}

// statusFlags reports a facets value ("key=type:value|…") holding statusFlags=b:true.
func statusFlags(facets string) bool {
	for _, kv := range strings.Split(facets, "|") {
		if kv == "statusFlags=b:true" {
			return true
		}
	}
	return false
}

// slotPath is the slot path of element i: the decoded n of each <p> from below the root down to it.
func slotPath(elems []xmlsrc.Element, i int) string {
	var parts []string
	for ; i >= 0; i = elems[i].Parent {
		if n, ok := elems[i].Attr("n"); ok && elems[i].Name == "p" {
			parts = append(parts, xmlsrc.UnescapeName(n))
		}
	}
	var b strings.Builder
	for k := len(parts) - 1; k >= 0; k-- {
		b.WriteString("/" + parts[k])
	}
	return b.String()
}

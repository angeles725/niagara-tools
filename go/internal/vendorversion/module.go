package vendorversion

import (
	"os/exec"
	"regexp"
	"strings"
)

// moduleTag is a <module ...> root tag carrying its own vendorVersion on one line: two or more numeric
// components (Niagara accepts an X.Y module version) plus any suffix. The greedy [^>]* reaches the last
// vendorVersion attribute of the tag, as the bash reader does.
var moduleTag = regexp.MustCompile(`<module [^>]*vendorVersion="([0-9]+(\.[0-9]+)+[^"]*)"`)

// ModuleXMLVersion is the MODULE's own vendorVersion in a META-INF/module.xml text: the one on the
// <module ...> root tag, not a <dependency name="baja" vendorVersion="..."/> floor (METHODOLOGY.md "two
// version concepts"). The first line holding such a tag wins; "" when there is none (callers fail closed).
func ModuleXMLVersion(xml string) string {
	for _, line := range strings.Split(xml, "\n") {
		if m := moduleTag.FindStringSubmatch(line); m != nil {
			return m[1]
		}
	}
	return ""
}

// ModuleVersion is ModuleXMLVersion of the jar's META-INF/module.xml, read with `unzip -p` as the bash
// reader does (a host without unzip reads nothing, and the drift gate then fails closed). "" when the jar
// or the version cannot be read.
func ModuleVersion(jar string) string {
	xml, _ := exec.Command("unzip", "-p", jar, "META-INF/module.xml").Output() // a failed unzip keeps what it printed
	return ModuleXMLVersion(string(xml))
}

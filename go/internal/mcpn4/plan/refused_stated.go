package plan

import "strings"

// statedRefusals are the texts the station's checkLinks handler throws while it still only checks (box-rt
// BComponentSpaceSessionHandler.checkLinks, :609-641): doLinkCheck refuses a missing source or target
// (:622, :627), a missing slot (:647, :652) and an invalid link (:659, "Could not create link: <the
// reason the link check gives>", e.g. a readonly target property) before any link is added; the links
// are added afterwards, one by one, in CheckLinksRecord.respond (:1033-1043). The text itself says that
// nothing was applied, so no read-back is needed to believe it. Each is a specific prefix of the
// station's own words; a bare "station error" or a transport failure is never in this list.
var statedRefusals = []string{
	"Could not create link: ", "Could not find Source Slot: ", "Could not find Target Slot: ",
	"Could not find Source: ", "Could not find Target: ",
}

// statesRefusal is true when op is a checkLinks call and the station answered with one of the texts
// it throws before it adds anything.
func statesRefusal(op any, text string) bool {
	o, _ := op.(interface{ Get(string) (any, bool) })
	if o == nil {
		return false
	}
	if key, _ := o.Get("ssc"); key != "checkLinks" {
		return false
	}
	for _, p := range statedRefusals {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

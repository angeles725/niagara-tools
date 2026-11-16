package writeops

import (
	"strconv"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// The wording of the remove's link scan (bench retro 2026-11-15 delta 3), apart from its logic.

// scanHead names the root the remove scanned and why that root.
func scanHead(root string, narrowed bool) string {
	why := " (the write-scope prefix that holds the parent; link_scan_ord narrows it)"
	if narrowed {
		why = " (link_scan_ord)"
	}
	return "scanned " + root + why + " to depth " + strconv.Itoa(LinkScanDepth)
}

// scanBlind is what the scan cannot see; the stronger form is for a scan that found nothing in a
// subtree with Status outputs.
func scanBlind(root string, outputs bool) string {
	if outputs {
		return "; outgoing links to components outside the removed subtree that are stored outside " + root +
			" cannot be detected and will be broken by the remove (link_scan_ord narrows the scan inside the " +
			"write scope, it never widens it)"
	}
	return "; links stored outside " + root + " are not detected"
}

const scanNone = ": no outgoing links from this subtree found (components deeper than that were not " +
	"scanned, and a link whose source sits more than 3 levels below the removed component is not matched)"

// scanBroken is the note of an acknowledged remove that breaks n links.
func scanBroken(n int) string {
	return ": the remove breaks " + strconv.Itoa(n) + " link(s) from this subtree into components outside it " +
		"(see outgoing_links_broken), acknowledged with acknowledge_links; " + linksFate
}

// linksFate is what happens to the broken links.
const linksFate = "the remove records them in its inverse and n4_rollback re-creates them"

// linksRefusal is the dry run's refusal when the scan found links and acknowledge_links is not true.
func linksRefusal(head, root string, found []any) string {
	var each []string
	for _, f := range found {
		o := f.(*pyjson.Object)
		source := pyjson.Str(get(o, "source_slot"))
		if p := pyjson.Str(get(o, "source_path")); p != "" {
			source = p + "/" + source
		}
		each = append(each, pyjson.Str(get(o, "target"))+" ("+pyjson.Str(get(o, "target_slot"))+" <- "+source+")")
	}
	return head + ": the remove would break " + strconv.Itoa(len(found)) + " link(s) from this subtree into " +
		"components outside it: " + strings.Join(each, ", ") + "; links stored outside " + root + " are not " +
		"detected. " + strings.ToUpper(linksFate[:1]) + linksFate[1:] + ". Repeat the dry run with " +
		"acknowledge_links: true to remove anyway: refusing"
}

// scanOrdOutside is the dry run's refusal of a link_scan_ord that is not root or under it.
func scanOrdOutside(scan, root string) string {
	return "link_scan_ord " + scan + " is not inside " + root + ", the write-scope prefix that holds the parent: " +
		"link_scan_ord only narrows the link scan to a root equal to or under that prefix, it never widens or " +
		"moves it: refusing"
}

// scanTooLarge is the dry run's refusal when the scan root loads more than LinkScanNodeBudget nodes.
func scanTooLarge(root string, n int, narrowed bool) string {
	what, ask := "the scope root "+root, "pass link_scan_ord, a narrower root inside it that holds the components "+
		"the removed one feeds"
	if narrowed {
		what, ask = "the link_scan_ord root "+root, "pass a narrower link_scan_ord inside it"
	}
	return what + " is too large to scan for outgoing links: it holds " + strconv.Itoa(n) + " node(s) to depth " +
		strconv.Itoa(LinkScanDepth) + ", more than " + strconv.Itoa(LinkScanNodeBudget) + "; " + ask +
		" (the scan is never skipped): refusing"
}

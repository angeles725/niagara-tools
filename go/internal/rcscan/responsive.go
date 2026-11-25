package rcscan

import (
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/lint"
)

// The responsive check (issue #1087 R1) runs for --profile lan|both, the profiles whose module serves a browser or
// phone view; hmi (a fixed 1280x800 kiosk, types/frontend-standard.md) and unknown (not enforced) give no row.
// Over the module's own rc HTML and CSS (vendor/, ext/ and *.min.js are third-party) it reports an HTML entry page
// with no viewport meta, a module with no @media width query, and a width / min-width above 360px outside @media.
// Its WARN is FAIL under --strict. The `rc-scan: allow responsive` marker waives a declaration on its own line, an
// entry page when it is anywhere in that page, and the module-level row when any scanned file carries it.
const (
	phoneWidth = 360
	respAllow  = "responsive"
	respRef    = " (types/frontend-standard.md § Enforcement)"
)

var (
	entryPage  = regexp.MustCompile(`(?i)<(head|html)([\s>/]|$)`)
	htmlRemark = regexp.MustCompile(`(?s)<!--.*?-->`)
	metaTag    = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	metaView   = regexp.MustCompile(`(?is)\bname\s*=\s*["']?viewport\b`)
	metaDevice = regexp.MustCompile(`(?is)\bwidth\s*=\s*device-width\b`)
)

// responsiveRows is the responsive check over every scanned file (src), in walk order.
func responsiveRows(all []*source, o options) []lint.Finding {
	if o.profile != "lan" && o.profile != "both" {
		return nil
	}
	var rows []lint.Finding
	var seen, media, waived bool
	for _, src := range all {
		if thirdParty(src.rc) || src.ext == "js" {
			continue
		}
		seen = true
		text := strings.Join(src.lines, "\n")
		waived = waived || strings.Contains(text, "rc-scan: allow "+respAllow)
		if src.ext == "html" && entryPage.MatchString(text) && !hasViewport(text) && !strings.Contains(text, "rc-scan: allow "+respAllow) {
			rows = append(rows, lint.Finding{Severity: o.sevW(), Path: src.rel, Msg: "responsive: no <meta name=\"viewport\" " +
				"content=\"width=device-width\"> in the HTML entry page -- a phone renders it at desktop width" + respRef})
		}
		wide, hasMedia := scanCSS(src)
		media = media || hasMedia
		for _, w := range wide {
			rows = append(rows, lint.Finding{Severity: o.sevW(), Path: src.rel, Line: w.line, Msg: "responsive: " + w.decl +
				" outside @media is wider than 360px -- the layout cannot shrink to a phone; use max-width, % or an @media rule" + respRef})
		}
	}
	if seen && !media && !waived {
		rows = append(rows, lint.Finding{Severity: o.sevW(), Msg: "responsive: no @media width query (min-width/max-width/width) in any " +
			"rc CSS or <style> block -- the view cannot adapt to 360 / 768 / 1280 px" + respRef})
	}
	return rows
}

// hasViewport reports a <meta name="viewport"> whose content holds width=device-width, outside HTML comments.
func hasViewport(text string) bool {
	for _, m := range metaTag.FindAllString(htmlRemark.ReplaceAllString(text, ""), -1) {
		if metaView.MatchString(m) && metaDevice.MatchString(m) {
			return true
		}
	}
	return false
}

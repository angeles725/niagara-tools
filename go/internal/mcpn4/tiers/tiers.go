// Package tiers is mcp-n4-kit/mcp_n4/tiers.py: the version tiers of METHODOLOGY section 5 (B1197
// section 1197.6), a pure classification and policy.
//
//   - A: 4.13 and 4.14 (BOX 2.3 with checkLinks): full support.
//   - B: 4.15 and 4.3: read-only until a PoC matches this build; writes need operator opt-in.
//   - C: any other or undetected build: writes only after reg.loadContract, loadRoot and a
//     harmless scratch write succeeded, recorded by the operator with an opt-in.
//
// An absent version (Python's None) is the empty string here: tiers.py never yields an empty one.
package tiers

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// AboutPath is where the version comes from: oBIX productVersion (live-certified, B457/B1199).
const AboutPath = "/obix/about/"

// ReasonTier is safety.REASON_TIER, the start of every tier refusal (the session retro keys on it).
const ReasonTier = "version tier refuses writes"

// OptInFlags is tiers.OPT_IN_FLAGS: the start flag that opts a station in, per tier.
var OptInFlags = map[string]string{"B": "--allow-tier-b", "C": "--allow-tier-c"}

// version is _VERSION under fullmatch; [0-9] is ASCII in Python too.
var version = regexp.MustCompile(`^([0-9]+)\.([0-9]+)(?:\.[0-9]+)*$`)

// parseVersion is parse_version: major and minor as Python's int() would print them, or ok false.
func parseVersion(text string) (major, minor string, ok bool) {
	m := version.FindStringSubmatch(strings.TrimFunc(text, pyjson.IsSpace))
	if m == nil {
		return "", "", false
	}
	digits := func(s string) string {
		if s = strings.TrimLeft(s, "0"); s == "" {
			return "0"
		}
		return s
	}
	return digits(m[1]), digits(m[2]), true
}

// TierOf is tier_of: "A", "B" or "C"; an unknown or unparseable version is "C".
func TierOf(v string) string {
	major, minor, ok := parseVersion(v)
	switch {
	case ok && major == "4" && (minor == "13" || minor == "14"):
		return "A"
	case ok && major == "4" && (minor == "15" || minor == "3"):
		return "B"
	}
	return "C"
}

// isWord is a Python str pattern's \w: str.isalnum() or "_".
func isWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// ProductVersion is product_version: the productVersion of an oBIX About document, else "".
// It scans as re.findall does for _STR_TAG (`<str\b[^>]*>`) and _ATTR
// (`\b(name|val)\s*=\s*"([^"]*)"`), with Python's Unicode \b and \s; a later attribute wins.
func ProductVersion(about string) string {
	for pos := 0; ; {
		i := strings.Index(about[pos:], "<str")
		if i < 0 {
			return ""
		}
		start := pos + i
		end := strings.IndexByte(about[start:], '>')
		if end < 0 { // no later <str can close either
			return ""
		}
		if r, _ := utf8.DecodeRuneInString(about[start+4:]); isWord(r) {
			pos = start + 1
			continue
		}
		tag := about[start : start+end+1]
		attrs := map[string]string{}
		for j := 0; j < len(tag); {
			if key, val, next, ok := attrAt(tag, j); ok {
				attrs[key], j = val, next
				continue
			}
			_, size := utf8.DecodeRuneInString(tag[j:])
			j += size
		}
		if attrs["name"] == "productVersion" && attrs["val"] != "" {
			return attrs["val"]
		}
		pos = start + end + 1
	}
}

// attrAt matches _ATTR at byte offset i of s: the key, the value and the offset after it.
func attrAt(s string, i int) (key, val string, next int, ok bool) {
	if r, _ := utf8.DecodeLastRuneInString(s[:i]); i > 0 && isWord(r) {
		return "", "", 0, false
	}
	switch {
	case strings.HasPrefix(s[i:], "name"):
		key = "name"
	case strings.HasPrefix(s[i:], "val"):
		key = "val"
	default:
		return "", "", 0, false
	}
	j := skipSpace(s, i+len(key))
	if j >= len(s) || s[j] != '=' {
		return "", "", 0, false
	}
	if j = skipSpace(s, j+1); j >= len(s) || s[j] != '"' {
		return "", "", 0, false
	}
	n := strings.IndexByte(s[j+1:], '"')
	if n < 0 {
		return "", "", 0, false
	}
	return key, s[j+1 : j+1+n], j + n + 2, true
}

func skipSpace(s string, i int) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !pyjson.IsSpace(r) {
			break
		}
		i += size
	}
	return i
}

// WriteBlock is write_block: why tier refuses writes on station, or "" when they are allowed.
// optIn maps a tier ("B"/"C") to the station names the operator opted in at start. The text is
// possibility-first: it names every route to the write.
func WriteBlock(tier, version, station string, optIn map[string]map[string]bool) string {
	if tier == "A" || optIn[tier][station] {
		return ""
	}
	shown := version
	if shown == "" {
		shown = "unknown (no oBIX productVersion at " + AboutPath + ")"
	}
	routes := "Routes: (1) dry runs stay allowed here, so the plan can be reviewed now; " +
		"(2) make the change in Workbench; "
	if tier == "B" {
		return fmt.Sprintf("%s: station %s is tier B (version %s): it stays read-only until a PoC on a "+
			"scratch station matches this build (METHODOLOGY section 5). %s(3) once the "+
			"PoC matches, restart the server with `%s %s`.",
			ReasonTier, station, shown, routes, OptInFlags["B"], station)
	}
	return fmt.Sprintf("%s: station %s is tier C (version %s): writes need the probe sequence first "+
		"(METHODOLOGY section 5): `reg.loadContract` and `loadRoot` succeed and one "+
		"harmless scratch write (create, read back, remove in a scratch folder) is "+
		"verified on this build. The server has no probe tool. %s(3) run that probe "+
		"(e.g. in Workbench or a PoC script), then restart the server with `%s %s`.",
		ReasonTier, station, shown, routes, OptInFlags["C"], station)
}

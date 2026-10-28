// Package buildstatefields reads the hand-checked parts of the kit ledger for sweep-build-state
// (port of the former toolbelt/sweep-build-state.sh): the build-state.v1 envelopes of
// BUILD-STATE.md and their key: value fields, the data rows of retros/INDEX.md, a retro's
// review-status marker and an INDEX row's disposition comment. The generated parts (the kit block,
// the INDEX table) are checked by go/internal/buildstate and go/internal/retroindex.
//
// Every reader takes the whole text and reads an unterminated last line like any other (the bash
// `while read` loops dropped it). Whitespace is ASCII ([[:space:]] in the C locale).
package buildstatefields

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/retrocells"
)

// Envelope markers: a line equal to one of these, at column 0, opens or closes an envelope.
const (
	OpenMarker  = "<!-- build-state.v1 -->"
	CloseMarker = "<!-- /build-state.v1 -->"
)

// Lines splits text into lines; a final newline ends the last line rather than starting one.
func Lines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\v' || c == '\f' || c == '\r'
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && isSpace(s[i]) {
		i++
	}
	for j > i && isSpace(s[j-1]) {
		j--
	}
	return s[i:j]
}

// listItem is the shell pattern `[[:space:]]*-[[:space:]]*`: the line starts with whitespace and
// holds a '-' followed by whitespace after that first character (an open_issues list item).
func listItem(line string) bool {
	if line == "" || !isSpace(line[0]) {
		return false
	}
	for i := 1; i+1 < len(line); i++ {
		if line[i] == '-' && isSpace(line[i+1]) {
			return true
		}
	}
	return false
}

// CheckEnvelopes validates every build-state.v1 envelope of a BUILD-STATE.md text and returns the
// first violation ("" when clean): markers balanced; each envelope carries a non-empty module and
// retro_required + retro_pending, both true|false. A field line is `key: value  # comment`; blank
// lines and list items are skipped; an indented key is not a field.
func CheckEnvelopes(text string) string {
	lines := Lines(text)
	opens, closes := 0, 0
	for _, l := range lines {
		switch l {
		case OpenMarker:
			opens++
		case CloseMarker:
			closes++
		}
	}
	if opens != closes {
		return fmt.Sprintf("unbalanced build-state.v1 markers (%d open / %d close)", opens, closes)
	}
	inside := false
	var module, req, pend string
	var haveMod, haveReq, havePend bool
	for _, line := range lines {
		switch line {
		case OpenMarker:
			inside = true
			module, req, pend, haveMod, haveReq, havePend = "", "", "", false, false, false
			continue
		case CloseMarker:
			// A close is checked whether or not an envelope is open; the fields are not reset, as the
			// bash tool did (balanced markers make a stray close re-check the envelope before it).
			// Mutation: BSF-pending -- dropping a required-field check lets an envelope without retro_pending pass
			// (Go TestCheckEnvelopes; bats S4 alone does not flip: the boolean check also rejects the empty value)
			switch {
			case !haveMod:
				return "envelope missing required field: module"
			case !haveReq:
				return fmt.Sprintf("envelope (%s) missing required field: retro_required", module)
			case !havePend:
				return fmt.Sprintf("envelope (%s) missing required field: retro_pending", module)
			case req != "true" && req != "false":
				return fmt.Sprintf("envelope (%s) retro_required not boolean: '%s'", module, req)
			case pend != "true" && pend != "false":
				return fmt.Sprintf("envelope (%s) retro_pending not boolean: '%s'", module, pend)
			}
			inside = false
			continue
		}
		if !inside || trimSpace(line) == "" || listItem(line) {
			continue
		}
		key, val := line, line
		if i := strings.IndexByte(line, ':'); i >= 0 {
			key, val = line[:i], line[i+1:]
		}
		if i := strings.IndexByte(val, '#'); i >= 0 {
			val = val[:i]
		}
		val = trimSpace(val)
		switch key {
		case "module":
			module = val
			if val != "" {
				haveMod = true
			}
		case "retro_required":
			req, haveReq = val, true
		case "retro_pending":
			pend, havePend = val, true
		}
	}
	return ""
}

// moduleFields are the DECLARED fields every module envelope carries (BUILD-STATE.src.md § How to read), with
// the shape named in the WARN. The kit's own envelope (module: kit) has no module tree and is exempt.
var moduleFields = []struct{ key, shape string }{
	{"module_root", "the module's absolute root"},
	{"source_of_truth", "<path>@<branch>@<commit> or unknown"},
}

// EnvelopeWarnings returns one WARN text per module envelope field of moduleFields that is missing or empty
// (`unknown` is a value), in envelope order (audit A4). Envelopes are read as CheckEnvelopes reads them;
// call it on a text CheckEnvelopes accepted.
func EnvelopeWarnings(text string) []string {
	var warns []string
	inside := false
	var fields map[string]string
	for _, line := range Lines(text) {
		switch line {
		case OpenMarker:
			inside, fields = true, map[string]string{}
			continue
		case CloseMarker:
			if inside && fields["module"] != "kit" {
				for _, f := range moduleFields {
					// Mutation: SOT1-warn -- dropping this check lets an envelope lose its source_of_truth silently
					if fields[f.key] == "" {
						warns = append(warns, fmt.Sprintf("envelope (%s) has no %s (required for every module: %s)", fields["module"], f.key, f.shape))
					}
				}
			}
			inside = false
			continue
		}
		if !inside || trimSpace(line) == "" || listItem(line) {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if i := strings.IndexByte(val, '#'); i >= 0 {
			val = val[:i]
		}
		fields[key] = trimSpace(val)
	}
	return warns
}

// IndexRow is one data row of retros/INDEX.md.
type IndexRow struct {
	File   string // first `*.md` name of the row other than INDEX.md
	Status string // first cell equal to pending or folded ("" when none)
	Deltas string // the cell after the status cell, padding spaces trimmed ("" when none); read it with retrocells.ParseDeltasCell
}

var (
	datedName = regexp.MustCompile(`[0-9]{4}-[0-9]{2}-[0-9]{2}[^|]*\.md`)
	mdName    = regexp.MustCompile(`[A-Za-z0-9._-]+\.md`)
)

// ParseIndexRow reads one INDEX.md line. It is a data row when it starts with '|', holds a dated
// `YYYY-MM-DD…​.md` name within one cell, and names a `*.md` file other than INDEX.md; the header
// and the |---| separator are not.
func ParseIndexRow(line string) (IndexRow, bool) {
	if !strings.HasPrefix(line, "|") || !datedName.MatchString(line) {
		return IndexRow{}, false
	}
	var r IndexRow
	for _, name := range mdName.FindAllString(line, -1) {
		if name != "INDEX.md" {
			r.File = name
			break
		}
	}
	if r.File == "" {
		return IndexRow{}, false
	}
	cells := strings.Split(line, "|")
	for i, c := range cells {
		if c = strings.Trim(c, " "); c == "pending" || c == "folded" {
			r.Status = c
			if i+1 < len(cells) {
				r.Deltas = strings.Trim(cells[i+1], " ")
			}
			break
		}
	}
	return r, true
}

var markerLine = regexp.MustCompile(`^<!--[[:space:]]*review-status:[[:space:]]*`)

// MarkerLines is how many lines of a retro hold its review-status marker.
const MarkerLines = 5

// Marker is the first blank-delimited word after `review-status:` on the first column-0
// `<!-- review-status:` line within the first MarkerLines lines ("" when there is none). Words are
// split on spaces and tabs only, as awk's default field splitting did.
func Marker(text string) string {
	for i, line := range Lines(text) {
		if i >= MarkerLines {
			break
		}
		if m := markerLine.FindStringIndex(line); m != nil {
			if f := strings.FieldsFunc(line[m[1]:], func(r rune) bool { return r == ' ' || r == '\t' }); len(f) > 0 {
				return f[0]
			}
			return ""
		}
	}
	return ""
}

// allowedDisposition matches an allowed value as a whole token (R3-002): it ends at whitespace,
// the comment close or the line end, so `worklist-completed` or `tracked-by-issue #12abc` is not one.
var allowedDisposition = regexp.MustCompile(`<!-- disposition: (tracked-by-issue #[0-9]+|worklist-complete)([[:space:]]|-->|$)`)

// DispositionSkips reports whether an INDEX row is a parked non-retro doc that never ages: its
// deltas cell reads as 0 (retrocells.ParseDeltasCell) and it carries `<!-- disposition: tracked-by-issue #<n>` or
// `<!-- disposition: worklist-complete`, each a whole token. Any other disposition, or one on a row
// with deltas, ages.
func DispositionSkips(line string) bool {
	// Mutation: D9 -- accepting any disposition value hides a row parked with no tracked reason.
	if !allowedDisposition.MatchString(line) {
		return false
	}
	r, ok := ParseIndexRow(line)
	if !ok || r.Status == "" {
		return false
	}
	// The deltas cell is read by the shared go/internal/retrocells parser (decimal, fail closed): an
	// unreadable cell is never a zero count, so the row ages.
	deltas, err := retrocells.ParseDeltasCell(r.Deltas)
	// Mutation: D8 -- dropping the 0-delta check hides a real retro's debt behind a comment.
	return err == nil && len(deltas) == 0
}

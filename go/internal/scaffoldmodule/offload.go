package scaffoldmodule

import (
	"fmt"
	"strings"
)

// The logic fixture carries an opt-in async-offload component (BPanOffload, its two station tests and the rows
// that register them): the fixture stays the scaffold source (types/actions.md § 2), and `--with-offload` is what
// emits it. Without the flag the emitted tree is the one the fixture had before the component was added.
//
// What belongs to the component is declared in the fixture itself:
//   - offloadOnly: files emitted only with the flag;
//   - marker lines in a shared file: `offload:begin` ... `offload:end` as whole lines in the file's comment syntax
//     (offloadBegins, offloadEnds) bracket the lines the flag adds, and a row ending in offloadLine is one such line.
//     The marker lines are never emitted; the bracketed lines are dropped without the flag; a block never closed,
//     nested, or an end without a begin is an error;
//   - offloadFallback: a shared file whose offload form is not a dropped block (the permissions root is
//     `<permissions/>` without the request), emitted as this text without the flag.
const (
	offloadLine = " <!-- offload:line -->" // the end of a row that is an offload line (a table row takes no own-line marker)
)

// offloadBlockMarkers are the whole trimmed lines that open and close a block, one per comment syntax of the
// fixture's shared files (XML, and `#` for the lexicon). Anything else, the token inside another line included, is
// ordinary text.
var (
	offloadBegins = map[string]bool{"<!-- offload:begin -->": true, "# offload:begin": true}
	offloadEnds   = map[string]bool{"<!-- offload:end -->": true, "# offload:end": true}
)

// offloadOnly are the fixture files (below the fixture root) only --with-offload emits.
var offloadOnly = map[string]bool{
	"MinimalPan/MinimalPan-rt/src/com/angeles/MinimalPan/BPanOffload.java":                         true,
	"MinimalPan/MinimalPan-rt/srcTest/test/com/angeles/MinimalPan/BMinimalPanOffloadTest.java":     true,
	"MinimalPan/MinimalPan-rt/srcTest/test/com/angeles/MinimalPan/BMinimalPanPermissionsTest.java": true,
}

// offloadFallback is the text of a shared file without --with-offload.
var offloadFallback = map[string]string{
	"MinimalPan/MinimalPan-rt/module-permissions.xml": "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<permissions/>\n",
}

// offloadShape shapes the fixture copy of a type that carries the component (typ.offload); another type's copy is
// untouched.
func offloadShape(typ moduleType, withOffload bool) fileShape {
	if !typ.offload {
		return nil
	}
	return func(rel, text string) (string, bool, error) {
		if !withOffload {
			if offloadOnly[rel] {
				return "", false, nil
			}
			if fallback, ok := offloadFallback[rel]; ok {
				return fallback, true, nil
			}
		}
		out, err := filterOffload(text, withOffload)
		if err != nil {
			return "", false, fmt.Errorf("%s: %w", rel, err)
		}
		return out, true, nil
	}
}

// filterOffload drops the marker lines, and with them the bracketed and the offloadLine lines unless keep. A block
// that is never closed, one opened inside another, and an end with no begin are errors (the text is not shaped).
func filterOffload(text string, keep bool) (string, error) {
	var b strings.Builder
	opened := 0 // the line a block was opened at, 0 outside a block
	for i, line := range strings.SplitAfter(text, "\n") {
		n, bare := i+1, strings.TrimSpace(line)
		switch {
		case offloadBegins[bare]:
			if opened != 0 {
				return "", fmt.Errorf("line %d: offload:begin inside the block opened at line %d", n, opened)
			}
			opened = n
		case offloadEnds[bare]:
			if opened == 0 {
				return "", fmt.Errorf("line %d: offload:end without an offload:begin", n)
			}
			opened = 0
		case strings.HasSuffix(strings.TrimRight(line, "\r\n"), offloadLine):
			if keep {
				b.WriteString(strings.Replace(line, offloadLine, "", 1))
			}
		case opened == 0 || keep:
			b.WriteString(line)
		}
	}
	if opened != 0 {
		return "", fmt.Errorf("line %d: offload:begin is never closed", opened)
	}
	return b.String(), nil
}

// skipTestIncludeOverlay stops the station-test overlay from writing moduleTest-include.xml over the fixture's own
// copy when the component is emitted: the fixture's lists the offload tests, the template's does not.
func skipTestIncludeOverlay(withOffload bool) fileShape {
	if !withOffload {
		return nil
	}
	return func(rel, text string) (string, bool, error) { return text, rel != stationTestProbe, nil }
}

package scaffoldmodule

import "strings"

// The logic fixture carries an opt-in async-offload component (BPanOffload, its two station tests and the rows
// that register them): the fixture stays the scaffold source (types/actions.md § 2), and `--with-offload` is what
// emits it. Without the flag the emitted tree is the one the fixture had before the component was added.
//
// What belongs to the component is declared in the fixture itself:
//   - offloadOnly: files emitted only with the flag;
//   - marker lines in a shared file: `offload:begin` ... `offload:end` on their own lines (an XML or `#` comment)
//     bracket the lines the flag adds, and a row ending in offloadLine is one such line. The marker lines are never
//     emitted; the bracketed lines are dropped without the flag;
//   - offloadFallback: a shared file whose offload form is not a dropped block (the permissions root is
//     `<permissions/>` without the request), emitted as this text without the flag.
const (
	offloadBegin = "offload:begin"
	offloadEnd   = "offload:end"
	offloadLine  = " <!-- offload:line -->"
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
func offloadShape(typ moduleType, with bool) fileShape {
	if !typ.offload {
		return nil
	}
	return func(rel, text string) (string, bool) {
		if !with {
			if offloadOnly[rel] {
				return "", false
			}
			if fallback, ok := offloadFallback[rel]; ok {
				return fallback, true
			}
		}
		return filterOffload(text, with), true
	}
}

// filterOffload drops the marker lines, and with them the bracketed and the offloadLine lines unless keep.
func filterOffload(text string, keep bool) string {
	var b strings.Builder
	inBlock := false
	for _, line := range strings.SplitAfter(text, "\n") {
		switch {
		case strings.Contains(line, offloadBegin):
			inBlock = true
		case strings.Contains(line, offloadEnd):
			inBlock = false
		case strings.HasSuffix(strings.TrimRight(line, "\r\n"), offloadLine):
			if keep {
				b.WriteString(strings.Replace(line, offloadLine, "", 1))
			}
		case !inBlock || keep:
			b.WriteString(line)
		}
	}
	return b.String()
}

// keepFixtureTestList stops the station-test overlay from replacing the fixture's moduleTest-include.xml, whose
// offload entries the template does not have, when the component is emitted.
func keepFixtureTestList(with bool) fileShape {
	if !with {
		return nil
	}
	return func(rel, text string) (string, bool) { return text, rel != stationTestProbe }
}

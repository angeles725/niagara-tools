package checkbatsskips

import (
	"regexp"
	"strings"
)

// TAP shapes the check reads.
const skipMarker = " # skip" // followed by a blank or the line end; " # skipped" is test-name text

var (
	planLine = regexp.MustCompile(`^1\.\.[0-9]+$`) // the plan bats prints once it ran
	okLine   = regexp.MustCompile(`^ok [0-9]+ `)   // a passed test: the only line that can be a skip
)

// Skip is one skipped test of a TAP: the `ok N <name>` text and the reason ("" when none was given).
type Skip struct{ Test, Reason string }

// lines splits text at newlines; a final newline ends the last line rather than starting one. A
// carriage return stays in its line, as it did for grep and awk.
func lines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

// HasPlan reports whether the TAP holds a `1..N` plan line.
func HasPlan(tap string) bool {
	for _, line := range lines(tap) {
		if planLine.MatchString(line) {
			return true
		}
	}
	return false
}

// Skips lists the skipped tests of a TAP in order.
func Skips(tap string) []Skip {
	var skips []Skip
	for _, line := range lines(tap) {
		if !okLine.MatchString(line) {
			continue
		}
		at := lastMarker(line)
		if at < 0 {
			continue
		}
		skips = append(skips, fields(line[:at], strings.TrimPrefix(line[at+len(skipMarker):], " ")))
	}
	return skips
}

// lastMarker is the offset of the last skipMarker that is a marker (a blank or the line end follows),
// or -1. A reason may quote a later " # skipped" and keep its line.
func lastMarker(line string) int {
	at := -1
	for from := 0; ; {
		i := strings.Index(line[from:], skipMarker)
		if i < 0 {
			return at
		}
		from += i
		// Mutation: SK8-last-marker -- taking the last " # skip" even when a letter follows drops the skip line
		if end := from + len(skipMarker); end == len(line) || line[end] == ' ' {
			at = from
		}
		from++
	}
}

// fields splits name and reason as the bash tool's `IFS=$'\t' read -r test reason` over
// "<name>\t<reason>" did: a tab inside the name ends the test and starts the reason, and the tabs
// around the reason are dropped.
func fields(name, reason string) Skip {
	test, rest, _ := strings.Cut(name+"\t"+reason, "\t")
	return Skip{Test: test, Reason: strings.Trim(rest, "\t")}
}

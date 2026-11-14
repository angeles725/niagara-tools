// Package slotomaticskip is the one Go detector of a source file Slot-o-Matic skipped (METHODOLOGY K15):
// the detector build-module and ng-deploy read. It replaced the bash lib
// toolbelt/lib/slotomatic-skip.sh (slotomatic_skip_scan), whose cases its tests hold.
//
// Slot-o-Matic logs "Could not process <path>.java: <msg>" and, at the end, "The following files failed to
// compile:" with one "  <path>.java[: <msg>]" line per file, all at WARNING level, and the gradle task still
// ends BUILD SUCCESSFUL: a jar built after a skip carries stale (or no) generated code.
// [ev: retro r10-slotomatic-skip-fail-open Δ1]
package slotomaticskip

import (
	"os"
	"regexp"
	"strings"
)

// Status is how a scan ended; its values are the statuses of slotomatic_skip_scan.
type Status int

// The statuses of ScanFile.
const (
	Clean      Status = 0 // no file was skipped
	Skipped    Status = 1 // at least one row
	Unreadable Status = 2 // the log cannot be read: one FAIL row naming it (fail closed, never a pass)
)

const (
	rowPrefix   = "FAIL slotomatic-skip "
	couldNot    = "Could not process "
	blockHeader = "The following files failed to compile:"
	// The reasons of a row whose log line gives none.
	reasonCouldNot = "Could not process"
	reasonBlock    = "listed under: The following files failed to compile"
	reasonNone     = "skipped by Slot-o-Matic"
)

var (
	// blockEnd is a line that closes the "failed to compile" block.
	blockEnd = regexp.MustCompile(`^> Task |^BUILD (SUCCESSFUL|FAILED)`)
	// blockEntry is the indent of a block entry (two or more blanks, after any logger prefix) and its first byte.
	blockEntry = regexp.MustCompile(`[[:space:]][[:space:]]+([^[:space:]])`)
	// testSource is a path under srcTest/: Slot-o-Matic walks test sources too, but they never reach the
	// module jar. [ev: retro r10-slotomatic-skip-srctest Δ1]
	testSource = regexp.MustCompile(`[/\\]srcTest[/\\]`)
)

// Scan returns one row per skipped file of a gradle log text, in log order, each file once:
// `FAIL slotomatic-skip <path> (<reason>)`. A file named by both forms is one row; nil when none was skipped.
func Scan(log string) []string {
	var rows []string
	seen := map[string]bool{}
	add := func(path, reason string) {
		if seen[path] || testSource.MatchString(path) {
			return
		}
		seen[path] = true
		if reason == "" {
			reason = reasonNone
		}
		rows = append(rows, rowPrefix+path+" ("+reason+")")
	}
	inBlock := false
	for _, line := range strings.Split(log, "\n") {
		line = strings.TrimSuffix(line, "\r")
		if i := strings.Index(line, couldNot); i >= 0 {
			if path, reason, ok := splitEntry(line[i+len(couldNot):], reasonCouldNot); ok {
				add(path, reason)
			}
			continue
		}
		if strings.Contains(line, blockHeader) {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if blockEnd.MatchString(line) {
			inBlock = false
			continue
		}
		if m := blockEntry.FindStringSubmatchIndex(line); m != nil {
			entry := strings.TrimRight(line[m[2]:], " \t\n\v\f\r")
			if path, reason, ok := splitEntry(entry, reasonBlock); ok {
				add(path, reason)
			}
		}
	}
	return rows
}

// splitEntry reads "<path>.java[: <msg>]": the path ends at ".java" (so a Windows drive letter does not cut
// it short) and the reason is the message, or dflt when there is none. A javac "X.java:12: error" line has
// no ".java: " and does not end in ".java": ok is false.
func splitEntry(s, dflt string) (path, reason string, ok bool) {
	if j := strings.Index(s, ".java: "); j >= 0 {
		return s[:j+len(".java")], s[j+len(".java: "):], true
	}
	if strings.HasSuffix(s, ".java") {
		return s, dflt, true
	}
	return "", "", false
}

// ScanFile is Scan over the gradle log file path. A log that cannot be read (missing, a directory) is
// Unreadable with one FAIL row naming it.
func ScanFile(path string) ([]string, Status) {
	st, err := os.Stat(path)
	var text []byte
	if err == nil && !st.IsDir() {
		text, err = os.ReadFile(path)
	}
	if err != nil || st.IsDir() {
		return []string{rowPrefix + path + " (gradle log unreadable)"}, Unreadable
	}
	if rows := Scan(string(text)); rows != nil {
		return rows, Skipped
	}
	return nil, Clean
}

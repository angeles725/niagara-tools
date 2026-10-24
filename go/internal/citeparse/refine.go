package citeparse

import (
	"regexp"
	"strings"
)

// placeholder matches the template text of a pointer: a `<word>` slot (`retro <stem> Δn`, `corpus B<n>`)
// or a lone ellipsis.
var placeholder = regexp.MustCompile(`<[A-Za-z][\w-]*>|^(?:…|\.\.\.)$`)

// The classes of unverifiable evidence: each names a kind of evidence no source tree holds, so the tool
// can only report it, never check it.
const (
	memoryTool = "memory-tool observation"
	liveObs    = "live observation"
	vcsRef     = "VCS or tracker reference"
	selfMarked = "self-marked unverifiable"
	commandRun = "command run"
	decision   = "decision or plan ID"
)

// unverifiableWords maps the first word of an unparsed clause to its class. The words are the leading
// tokens of the unverifiable rows of the 2026-10-24 triage (kit issue #416); a word joins only when no
// checkable clause starts with it.
var unverifiableWords = map[string]string{
	"engram": memoryTool,
	"live":   liveObs, "CERT-live": liveObs, "QA": liveObs,
	"commit": vcsRef, "issue": vcsRef, "PR": vcsRef, "apply": vcsRef,
	"unverifiable": selfMarked,
	"bats":         commandRun, "RED": commandRun, "GREEN": commandRun, "find": commandRun, "gradle": commandRun,
	"run-pure-test": commandRun, "smoke": commandRun, "dmesg": commandRun, "ls": commandRun, "ps": commandRun,
	"shellcheck": commandRun, "mutation": commandRun, "mutations": commandRun, "test": commandRun,
	"tests": commandRun, "git": commandRun, "grep": commandRun, "real-tree": commandRun,
	"lint-structure": commandRun, "bog-nav": commandRun, "module-navigator": commandRun,
	"design": decision, "transcript": decision, "user": decision, "user-observed": decision,
	"operator": decision, "seed": decision, "host": decision, "landing": decision,
}

var (
	shortSHA     = regexp.MustCompile(`^[0-9a-f]{7,40}$`)
	campaignUnit = regexp.MustCompile(`^r\d+$`)
	planID       = regexp.MustCompile(`^(?:[CDT]\d+[a-z]?(?:-\d+)?|audit-\d{4}-\d{2}-\d{2})$`)
)

// refine marks a placeholder pointer Template and an unparsed clause of an unverifiable kind Unverifiable.
func refine(p *Pointer) {
	if placeholder.MatchString(p.Target) || placeholder.MatchString(p.File) {
		p.Kind, p.Notes = Template, nil
		return
	}
	if p.Kind != Unparsed {
		return
	}
	if class := unverifiableClass(firstWord(p.Target)); class != "" {
		p.Kind, p.Notes = Unverifiable, []string{class}
	}
}

func unverifiableClass(word string) string {
	switch {
	case unverifiableWords[word] != "":
		return unverifiableWords[word]
	case shortSHA.MatchString(word) && strings.ContainsAny(word, "0123456789"), campaignUnit.MatchString(word):
		return vcsRef
	case planID.MatchString(word):
		return decision
	}
	return ""
}

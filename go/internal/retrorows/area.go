package retrorows

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Areas a delta can belong to.
const (
	AreaMCP   = "mcp-n4"
	AreaBuild = "build-n4-module"
)

var (
	strongMCP   = regexp.MustCompile(`(?i)mcp-?n4|\bn4_[a-z0-9_]+`)
	strongBuild = regexp.MustCompile(`(?i)build-n4-module`)
	weakKit     = regexp.MustCompile(`(?i)\bkit\b|toolbelt|BUILD-LOOP|go/internal`)
	foreign     = regexp.MustCompile(`(?i)research-sdd|rsdd|PROMPT-LOOP`)
	boldLead    = regexp.MustCompile(`^\*\*([^*]+?)\.?\*\*`)
	openLead    = regexp.MustCompile(`^(?i)(OPEN|PENDING)[.:]\s*`)
)

func strong(s string) string {
	switch {
	case strongMCP.MatchString(s):
		return AreaMCP
	case strongBuild.MatchString(s):
		return AreaBuild
	}
	return ""
}

// Area says which area a row belongs to: "mcp-n4", "build-n4-module" or "" (not ours). An explicit name (mcp-n4,
// n4_* tool names, build-n4-module) in the target, the heading, the text, then the retro's file name decides first;
// then the kit words (kit, toolbelt, BUILD-LOOP, go/internal) in the target, then in the text; the evidence column,
// which only cites where a delta was seen, decides last. Rows about the research kit (research-sdd, rsdd,
// PROMPT-LOOP) get no kit area. def is what a row with no signal gets (a retro inside this kit passes
// build-n4-module; others pass "").
func Area(r Row, file, def string) string {
	body := r.Text + " " + r.Evidence
	for _, s := range []string{r.Target, r.Heading, r.Text, file} {
		if a := strong(s); a != "" {
			return a
		}
	}
	for _, s := range []string{r.Target, r.Text} {
		if weakKit.MatchString(s) && !foreign.MatchString(r.Target+" "+body) {
			return AreaBuild
		}
	}
	// the evidence only cites where a delta was seen ("~40 mcp-n4 deltas"): it decides only when nothing above did
	if a := strong(r.Evidence); a != "" {
		return a
	}
	if weakKit.MatchString(r.Evidence) && !foreign.MatchString(r.Target+" "+body) {
		return AreaBuild
	}
	return def
}

// Summary is the one-line gist of a delta: the bold lead sentence, else the first sentence, cut to max characters
// with an ellipsis.
func Summary(text string, max int) string {
	text = strings.TrimSpace(openLead.ReplaceAllString(strings.TrimSpace(text), ""))
	if m := boldLead.FindStringSubmatch(text); m != nil {
		text = m[1]
	} else if i := strings.Index(text, ". "); i > 0 {
		text = text[:i]
	}
	text = strings.TrimRight(strings.Join(strings.Fields(strings.ReplaceAll(text, "**", "")), " "), ".")
	if utf8.RuneCountInString(text) <= max {
		return text
	}
	return strings.TrimRight(string([]rune(text)[:max-1]), " ,;:") + "…"
}

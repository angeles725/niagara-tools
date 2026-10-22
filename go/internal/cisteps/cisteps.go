// Package cisteps lists the steps of a GitHub workflow for scripts/ci-local.sh. It is the Go port
// of scripts/lib/ci_steps.py and keeps its output byte for byte. [ev: retro go-ci-steps]
//
// It reads the subset of YAML that ci.yml uses and refuses the rest: a folded `>` block, an
// unterminated quote, a `run:`/`uses:` outside a step item or a workflow/job-level `env:`/`defaults:`
// is an error naming its line, so no step can vanish from the replay. A step key other than
// name/run/uses/with/id (env:, shell:, if:, working-directory:, ...) changes step semantics, so the
// step comes back as kind "bad" for ci-local.sh to report as UNCOVERED.
package cisteps

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Step is one `steps:` item: Kind "run" (Body = the script), "uses" (Body = the action ref) or
// "bad" (Body = why the local replay cannot reproduce it).
type Step struct {
	Kind, Name, Body string
}

// key matches a `key:` line; Python's re `\w` is Unicode-aware, hence \p{L}\p{N}.
var key = regexp.MustCompile(`^( *)(- +)?([A-Za-z_][\p{L}\p{N}_-]*):(?: (.*))?$`)

// stepKeys are the keys that do not change what a step does; any other key makes it "bad".
var stepKeys = map[string]bool{"name": true, "run": true, "uses": true, "with": true, "id": true}

// isSpace is Python's str.isspace: Go's unicode.IsSpace plus the separators U+001C..U+001F.
func isSpace(r rune) bool { return unicode.IsSpace(r) || (r >= 0x1c && r <= 0x1f) }

func strip(s string) string { return strings.TrimFunc(s, isSpace) }

// indentOf counts the leading spaces (tabs do not indent).
func indentOf(s string) int { return len(s) - len(strings.TrimLeft(s, " ")) }

type item struct {
	line            int
	bad             []string
	name, run, uses string
	hasRun, hasUses bool
	hasName         bool
}

// unquote strips one pair of matching quotes, with no escape processing.
func unquote(val, where string) (string, error) {
	val = strip(val)
	if val != "" && (val[0] == '"' || val[0] == '\'') {
		if len(val) < 2 || val[len(val)-1] != val[0] {
			return "", fmt.Errorf("%s: unterminated quoted scalar", where)
		}
		return val[1 : len(val)-1], nil
	}
	return val, nil
}

// block collects the `|` block after lines[i]: every following line that is blank or indented
// deeper than keyIndent, trailing blank lines dropped, dedented by the smallest indent.
func block(lines []string, i, keyIndent int) (string, int) {
	j := i + 1
	for j < len(lines) && (strip(lines[j]) == "" || indentOf(lines[j]) > keyIndent) {
		j++
	}
	body := lines[i+1 : j]
	for len(body) > 0 && strip(body[len(body)-1]) == "" {
		body = body[:len(body)-1]
	}
	cut := -1
	for _, b := range body {
		if strip(b) != "" && (cut < 0 || indentOf(b) < cut) {
			cut = indentOf(b)
		}
	}
	out := make([]string, len(body))
	for k, b := range body {
		out[k] = dropRunes(b, cut)
	}
	return strings.Join(out, "\n"), j
}

// dropRunes is Python's s[n:] (a blank line may hold multi-byte whitespace).
func dropRunes(s string, n int) string {
	for ; n > 0 && s != ""; n-- {
		_, size := utf8.DecodeRuneInString(s)
		s = s[size:]
	}
	return s
}

// Parse returns every `steps:` item of the workflow text data read from path (path only names
// the errors, as `<path>:<line>: <reason>`).
func Parse(path string, data []byte) ([]Step, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s: not valid UTF-8", path)
	}
	// Python's text mode reads \r\n and \r as \n.
	text := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(text, "\n")
	var items []*item
	var cur *item
	inSteps, stepsIndent, itemKeys := false, 0, 0
	for i := 0; i < len(lines); {
		line, where := lines[i], fmt.Sprintf("%s:%d", path, i+1)
		m := key.FindStringSubmatch(line)
		if m == nil { // a comment line never matches: its first non-space is '#'
			i++
			continue
		}
		indent, dash, k, val := len(m[1]), m[2], m[3], m[4]
		if inSteps && indent <= stepsIndent && dash == "" {
			inSteps, cur = false, nil // left the steps: list
		}
		if dash != "" && inSteps && indent >= stepsIndent {
			cur = &item{line: i + 1}
			items = append(items, cur)
			itemKeys = indent + len(dash)
		} else if cur != nil && indent != itemKeys {
			i++ // nested mapping (with:, env: values)
			continue
		}
		if cur == nil {
			switch k {
			case "run", "uses":
				return nil, fmt.Errorf("%s: %s: outside a steps: item", where, k)
			case "env", "defaults":
				return nil, fmt.Errorf("%s: workflow/job-level %s: changes every step; not replayed", where, k)
			case "steps":
				inSteps, stepsIndent = true, indent
			}
			i++
			continue
		}
		if !stepKeys[k] {
			cur.bad = append(cur.bad, k+":")
		}
		if k == "run" {
			if v := strip(val); v == "|" || v == "|-" {
				cur.run, i = block(lines, i, itemKeys)
				cur.hasRun = true
				continue
			} else if strings.HasPrefix(v, "|") || strings.HasPrefix(v, ">") {
				return nil, fmt.Errorf("%s: run: %s block style not supported (use | or |-)", where, v)
			}
		}
		if k == "name" || k == "run" || k == "uses" {
			v, err := unquote(val, where)
			if err != nil {
				return nil, err
			}
			switch k {
			case "name":
				cur.name, cur.hasName = v, true
			case "run":
				cur.run, cur.hasRun = v, true
			default:
				cur.uses, cur.hasUses = v, true
			}
		}
		i++
	}
	steps := make([]Step, 0, len(items))
	for _, it := range items {
		steps = append(steps, it.step())
	}
	if len(steps) == 0 {
		return nil, nil
	}
	return steps, nil
}

// step names the item (name, else its first run line, else its action ref, else its line) and
// classifies it.
func (it *item) step() Step {
	name := it.name
	if name == "" {
		if it.run != "" {
			name, _, _ = strings.Cut(it.run, "\n")
		} else {
			name = it.uses
		}
	}
	if name == "" {
		name = fmt.Sprintf("step at line %d", it.line)
	}
	switch {
	case len(it.bad) > 0 || it.hasRun == it.hasUses:
		why := strings.Join(it.bad, " ")
		if why == "" {
			why = "needs exactly one of run:/uses:"
		}
		return Step{"bad", name, "step-level " + why + " not replayed"}
	case it.hasRun:
		return Step{"run", name, it.run}
	default:
		return Step{"uses", name, it.uses}
	}
}

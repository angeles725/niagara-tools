package retro

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

// template is mcp_n4/templates/retro.template.md, byte for byte (TestTheTemplateShipsInsideThePackage).
//
//go:embed retro.template.md
var template string

// Honesty replaces an empty delta table and the verdict when nothing was found.
const Honesty = "no new deltas; the kit already covers this session."

const (
	deltaHeader = "| # | Proposed change | Target (file · section) | " +
		"Evidence (batch_id / tool call / audit line) | Type | Priority |"
	maxEvidence = 5
)

// cell is one markdown table cell: no pipes, no line breaks.
func cell(text string) string {
	return strings.NewReplacer("|", `\|`, "\n", " ", "\r", " ").Replace(text)
}

func evidenceCell(evidence []any) string {
	shown := make([]string, 0, maxEvidence)
	for _, e := range evidence[:min(len(evidence), maxEvidence)] {
		shown = append(shown, e.(string))
	}
	text := strings.Join(shown, "; ")
	if extra := len(evidence) - len(shown); extra > 0 {
		text += fmt.Sprintf("; +%d more", extra)
	}
	return text
}

func deltaTable(candidates []*pyjson.Object) string {
	rows := []string{deltaHeader, "|---|---|---|---|---|---|"}
	for i, c := range candidates {
		rows = append(rows, fmt.Sprintf("| %d | %s | `%s` | %s | %s | %s |", i+1, cell(get(c, "change").(string)),
			cell(get(c, "target").(string)), cell(evidenceCell(get(c, "evidence").([]any))), get(c, "type"), get(c, "priority")))
	}
	return strings.Join(rows, "\n")
}

// tally counts values as the keys of a Python dict: the first of equal keys is kept.
type tally struct {
	keys   []any
	counts map[string]int
	index  map[string]int
}

func (t *tally) add(v any) error {
	k, err := pyjson.DictKey(v)
	if err != nil {
		return err
	}
	if _, seen := t.index[k]; !seen {
		t.index[k] = len(t.keys)
		t.keys = append(t.keys, v)
	}
	t.counts[k]++
	return nil
}

// String is ", ".join("%s %d" % (_safe(k), n) for k, n in sorted(items, key=str(k))) or "none".
func (t *tally) String() string {
	keys := append([]any(nil), t.keys...)
	sort.SliceStable(keys, func(i, j int) bool { return pyjson.Str(keys[i]) < pyjson.Str(keys[j]) })
	parts := make([]string, len(keys))
	for i, k := range keys {
		dk, _ := pyjson.DictKey(k)
		parts[i] = fmt.Sprintf("%s %d", safe(k), t.counts[dk])
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func newTally() *tally { return &tally{counts: map[string]int{}, index: map[string]int{}} }

func summary(audit, views []*pyjson.Object, since string) (string, error) {
	outcomes, verdicts := newTally(), newTally()
	for _, e := range audit {
		if err := outcomes.add(get(e, "outcome")); err != nil {
			return "", err
		}
	}
	for _, v := range views {
		key := get(v, "verdict")
		if get(v, "state") == "in-doubt" {
			key = "in-doubt"
		}
		if err := verdicts.add(key); err != nil {
			return "", err
		}
	}
	window := "whole state directory"
	if since != "" {
		window = "since " + since
	}
	return fmt.Sprintf("- Window: %s\n- Audit entries: %d (%s)\n- Journaled batches: %d (%s)",
		window, len(audit), outcomes, len(views), verdicts), nil
}

// render is retro.render on the shipped template; the honesty line replaces an empty table.
func render(candidates []*pyjson.Object, station, date string, audit, views []*pyjson.Object, since string) (string, error) {
	deltas, verdict := Honesty, Honesty
	covered := "- none recorded: no friction was found in the audit or the journal."
	if len(candidates) > 0 {
		deltas = deltaTable(candidates)
		verdict = fmt.Sprintf("Draft with %d candidate delta(s) derived mechanically from the audit and "+
			"journal. The reviewer must confirm each is a genuine kit gap rather than operator error "+
			"before keeping it; drop the rest. A retro that always finds something is noise (honesty "+
			"clause).", len(candidates))
		covered = "- (reviewer: list the guards that worked as designed and the rules the kit already encodes)"
	}
	sum, err := summary(audit, views, since)
	if err != nil {
		return "", err
	}
	text := template
	for _, kv := range [][2]string{{"STATION", station}, {"DATE", date}, {"DELTAS", deltas},
		{"VERDICT", verdict}, {"COVERED", covered}, {"SUMMARY", sum}} {
		text = strings.ReplaceAll(text, "{{"+kv[0]+"}}", kv[1]) // in order, as str.replace runs
	}
	return text, nil
}

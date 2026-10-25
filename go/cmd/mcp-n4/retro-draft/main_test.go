package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
	"github.com/angeles725/niagara-tools/go/internal/mcpn4/golden"
)

// draftScript is what tools/new_retro.py computed in Python before S17: retro.draft for the
// station, date and window it was given, and count_deltas of the markdown.
const draftScript = `import json, sys
from mcp_n4 import retro
def run(c):
    try:
        d = retro.draft(c["dir"], station=c["station"], date=c["date"], since=c["since"] or None)
    except ValueError as exc:
        return "exit 3: %s" % exc
    return json.dumps({"markdown": d["markdown"], "deltas": retro.count_deltas(d["markdown"])})
print(json.dumps([run(c) for c in json.load(sys.stdin)]))`

type draftCase struct {
	Dir     string `json:"dir"`
	Station string `json:"station"`
	Date    string `json:"date"`
	Since   string `json:"since"`
}

// goDraft is what the binary answers for c, in the script's form.
func goDraft(c draftCase) string {
	args := []string{"--state-dir", c.Dir, "--station", c.Station, "--date", c.Date}
	if c.Since != "" {
		args = append(args, "--since", c.Since)
	}
	var out, errOut bytes.Buffer
	switch code := run(args, cli.Stdio{Out: &out, Err: &errOut}); code {
	case 0:
		return strings.TrimSuffix(out.String(), "\n")
	case 3:
		return "exit 3: " + strings.TrimPrefix(strings.TrimSuffix(errOut.String(), "\n"), "retro-draft: ")
	default:
		return "exit " + strconv.Itoa(code) + ": " + errOut.String()
	}
}

func writeState(t *testing.T, audit, journal string) string {
	t.Helper()
	dir := t.TempDir()
	for name, body := range map[string]string{"audit.jsonl": audit, "journal.jsonl": journal} {
		if body != "" {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	return dir
}

// TestDraftIsNewRetroPythonDraft replays what the Python new_retro.py drafted for the same state
// dirs (testdata/draft.golden.jsonl): the markdown and the delta count, byte for byte.
func TestDraftIsNewRetroPythonDraft(t *testing.T) {
	g := golden.Open(t, "draft")
	fixture, err := filepath.Abs(filepath.Join("..", "..", "..", "..", "mcp-n4-kit", "tests", "fixtures", "journal", "python-written"))
	if err != nil {
		t.Fatal(err)
	}
	g.Alias(fixture, "$FIXTURE")
	missing := filepath.Join(t.TempDir(), "missing")
	g.Alias(filepath.Dir(filepath.Dir(missing)), "$TMP") // every t.TempDir of the test shares it
	friction := writeState(t, "", `{"batch_id": "b1", "ts": "2026-10-01T11:00:00+00:00", "tool": "n4_set_slot", "ops": [{"nm": "s"}], "phase": "intent", "station_name": "LLM"}
{"batch_id": "b1", "ts": "2026-10-01T11:00:01+00:00", "phase": "result", "accepted": null, "inverse": [], "verdict": "mismatch"}
`)
	refusals := writeState(t, `{"ts": "2026-10-01T10:00:00+00:00", "tool": "n4_set_slot", "batch_id": null, "dry_run": false, "outcome": "refused", "reason": "write budget exhausted: 2 writes", "args_redacted": {}}
{"ts": "2026-10-01T10:01:00+00:00", "tool": "n4_link", "batch_id": null, "dry_run": false, "outcome": "refused", "reason": "ord 'station:|slot:/X' is outside the write scope", "args_redacted": {}}
{"ts": "2026-10-01T10:02:00+00:00", "tool": "n4_set_slot", "batch_id": null, "outcome": "error", "reason": "BoxError: HTTP 500"}
`, "")
	cases := []draftCase{
		{Dir: fixture, Station: "LLM", Date: "2026-10-01"},
		{Dir: fixture, Station: "LLM", Date: "2026-10-01", Since: "2026-10-01T00:00:00Z"},
		{Dir: missing, Station: "LLM", Date: "2026-10-01"},
		{Dir: friction, Station: "LLM", Date: "2026-10-01"},
		{Dir: friction, Station: "LLM", Date: "2026-10-01", Since: "2026-10-02T00:00:00Z"},
		{Dir: refusals, Station: "S-1.x_y", Date: "2099-12-31"},
		{Dir: refusals, Station: "LLM", Date: "2026-10-01", Since: "nope"},
	}
	stdin, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	var want []string
	if err := json.Unmarshal(g.Run(draftScript, stdin), &want); err != nil || len(want) != len(cases) {
		t.Fatalf("golden answers %d drafts for %d cases (%v)", len(want), len(cases), err)
	}
	for i, c := range cases {
		if got := goDraft(c); got != want[i] {
			t.Errorf("case %d (%+v):\n got %q\nwant %q", i, c, got, want[i])
		}
	}
}

func TestUsageIsExitTwo(t *testing.T) {
	for _, args := range [][]string{nil, {"--state-dir", "d", "--station", "S"}, {"--bogus"},
		{"--state-dir", "d", "--station", "S", "--date", "2026-10-01", "extra"}} {
		var out, errOut bytes.Buffer
		if code := run(args, cli.Stdio{Out: &out, Err: &errOut}); code != 2 || out.Len() != 0 ||
			!strings.HasPrefix(errOut.String(), "usage: retro-draft ") {
			t.Errorf("run(%q) = %d, out %q, err %q; want 2 and the usage", args, code, out.String(), errOut.String())
		}
	}
}

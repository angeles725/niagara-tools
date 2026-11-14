package livesmoke

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/mcpn4/pyjson"
)

func TestScrubReplacesEverySecretValue(t *testing.T) {
	if got := Scrub("a SECRETX b", []string{"SECRETX"}); got != "a *** b" {
		t.Errorf("Scrub = %q", got)
	}
	if got := Scrub("plain", nil); got != "plain" {
		t.Errorf("Scrub with no secrets = %q", got)
	}
}

// nasty is a secret with a quote, a backslash, a newline, a control character and non-ASCII.
const nasty = "p\"w\\x\n\x01caf\u00e9-unique"

func TestRenderReportRemovesRawAndEscapedForms(t *testing.T) {
	report := kwargs("error", "bad "+nasty, "steps", []any{kwargs("detail", kwargs("k", []any{nasty}))},
		"server_stderr", []any{"login " + nasty}, nasty, 1)
	text, err := RenderReport(report, []string{nasty, ""})
	if err != nil {
		t.Fatal(err)
	}
	// json.dumps(nasty)[1:-1] and json.dumps(nasty, ensure_ascii=False)[1:-1], as Python spells them.
	for _, form := range []string{nasty, `p\"w\\x\n\u0001caf\u00e9-unique`, "p\\\"w\\\\x\\n\\u0001caf\u00e9-unique"} {
		if strings.Contains(text, form) {
			t.Errorf("report holds %q:\n%s", form, text)
		}
	}
	if !strings.Contains(text, Mask) {
		t.Error("report has no mask")
	}
	var back map[string]any
	if err := json.Unmarshal([]byte(text), &back); err != nil || back["error"] != "bad ***" {
		t.Errorf("error = %v (%v)", back["error"], err)
	}
}

func TestScrubValueWalksNestedStructures(t *testing.T) {
	in := kwargs("a", []any{"x S y", []any{"S"}}, "n", 3, "b", nil)
	got := pyjson.Repr(ScrubValue(in, []string{"S"}))
	if want := "{'a': ['x *** y', ['***']], 'n': 3, 'b': None}"; got != want {
		t.Errorf("ScrubValue = %s, want %s", got, want)
	}
}

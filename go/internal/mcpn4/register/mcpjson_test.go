package register

import (
	"strings"
	"testing"
)

func TestPiAndGentleShellEntriesGoToTheAgentMcpJSONWithDirectExposure(t *testing.T) {
	for harness, rel := range map[string]string{"pi": ".pi/agent/mcp.json", "gentle-shell": ".gentle-shell/agent/mcp.json"} {
		h := newHome(t)
		tgt, err := harnessTarget(harness, h.dir, "", "mcp-n4-lab", "/l/lab-run.sh")
		if err != nil || tgt.file != h.path(rel) {
			t.Fatalf("%s: file %s %v, want %s", harness, tgt.file, err, rel)
		}
		_, next, err := tgt.plan(false)
		want := "{\n  \"mcpServers\": {\n    \"mcp-n4-lab\": {\n      \"command\": \"/l/lab-run.sh\",\n" +
			"      \"args\": [],\n      \"exposure\": \"direct\"\n    }\n  }\n}\n"
		if err != nil || string(next) != want {
			t.Fatalf("%s: mcp.json =\n%s\n%v\nwant\n%s", harness, next, err, want)
		}
	}
}

func TestOtherServersInMcpJSONSurviveByteForByte(t *testing.T) {
	h := newHome(t)
	before := "{\n  \"mcpServers\": {\n    \"codegraph\": {\n      \"args\": [\"serve\", \"--mcp\"],\n      \"command\": \"codegraph\"\n    }\n  },\n  \"after\": 1\n}\n"
	h.write(t, ".gentle-shell/agent/mcp.json", before)
	tgt, _ := harnessTarget("gentle-shell", h.dir, "", "mcp-n4-lab", "/l/lab-run.sh")
	_, next, err := tgt.plan(false)
	want := strings.Replace(before, "\"codegraph\"\n    }\n", "\"codegraph\"\n    },\n"+
		"    \"mcp-n4-lab\": {\n      \"command\": \"/l/lab-run.sh\",\n      \"args\": [],\n      \"exposure\": \"direct\"\n    }\n", 1)
	if err != nil || string(next) != want {
		t.Fatalf("mcp.json =\n%s\n%v\nwant\n%s", next, err, want)
	}
}

func TestAnEntryWithAnotherCommandIsAConflictUnlessForced(t *testing.T) {
	h := newHome(t)
	before := "{\n  \"mcpServers\": {\n    \"mcp-n4-lab\": {\"command\": \"/old/run.sh\", \"args\": [], \"exposure\": \"direct\"}\n  }\n}\n"
	h.write(t, ".pi/agent/mcp.json", before)
	tgt, _ := harnessTarget("pi", h.dir, "", "mcp-n4-lab", "/l/lab-run.sh")
	_, _, err := tgt.plan(false)
	if err == nil || !strings.Contains(err.Error(), h.path(".pi/agent/mcp.json")) ||
		!strings.Contains(err.Error(), "mcp-n4-lab") || !strings.Contains(err.Error(), "[command]") {
		t.Fatalf("err = %v, want a refusal naming the file, the entry and the field", err)
	}
	if _, next, err := tgt.plan(true); err != nil || !strings.Contains(string(next), `"command": "/l/lab-run.sh"`) {
		t.Fatalf("forced plan = %s %v, want the entry replaced", next, err)
	}
}

func TestAnUnknownHarnessIsAUsageError(t *testing.T) {
	if _, err := harnessTarget("vim", "/h", "", "n", "/l"); err == nil || !strings.Contains(err.Error(), "vim") {
		t.Fatalf("err = %v, want a usage error naming the harness", err)
	}
}

func TestAnEntryDifferingOnlyInArgsNamesJustThatField(t *testing.T) {
	h := newHome(t)
	h.write(t, ".pi/agent/mcp.json", "{\"mcpServers\": {\"mcp-n4-lab\": {\"exposure\": \"direct\", \"args\": [\"-x\"], \"command\": \"/l/lab-run.sh\"}}}\n")
	tgt, _ := harnessTarget("pi", h.dir, "", "mcp-n4-lab", "/l/lab-run.sh")
	if _, _, err := tgt.plan(false); err == nil || !strings.Contains(err.Error(), "different [args]") {
		t.Fatalf("err = %v, want only args named", err)
	}
}

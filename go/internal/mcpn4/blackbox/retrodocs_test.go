package blackbox

// The closing retro step is documented (port of mcp-n4-kit/tests/test_new_retro.py; the write
// mode itself is tested in go/cmd/mcp-n4/retro-draft/write_test.go).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMethodologyAndSkillNameTheClosingStep(t *testing.T) {
	for _, doc := range []string{"METHODOLOGY.md", filepath.Join("skill", "SKILL.md")} {
		text := kitDoc(t, doc)
		for _, needle := range []string{"n4_session_retro_draft", "retro-draft --write", "never apply"} {
			if !strings.Contains(text, needle) {
				t.Errorf("%s does not name %q", doc, needle)
			}
		}
	}
}

func TestMethodologyNoLongerMarksTheRetroStepManualUntilTooling(t *testing.T) {
	if strings.Contains(kitDoc(t, "METHODOLOGY.md"), "tooling arrives in T7") {
		t.Error("METHODOLOGY.md still says the retro step waits for tooling (T7)")
	}
}

func TestTheTemplateAndIndexExistWithTheAgreedGrammar(t *testing.T) {
	if !strings.Contains(kitDoc(t, "retros", "INDEX.md"), `| file | Station | Date | pending\|folded | deltas |`) {
		t.Error("retros/INDEX.md lacks the agreed header row")
	}
	tmpl, err := os.ReadFile(filepath.Join(mcpn4Dir, "retro", "retro.template.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(tmpl), "## Proposed kit deltas") {
		t.Error("retro.template.md lacks ## Proposed kit deltas")
	}
}

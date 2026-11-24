package nretool

import (
	"strings"
	"testing"
)

func TestListNamesTheSupportedToolsWithAPurpose(t *testing.T) {
	code, out, errs := run("list")
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if code != 0 || errs != "" || len(lines) != 3 {
		t.Fatalf("exit %d, out %q, err %q", code, out, errs)
	}
	for i, name := range []string{"deps", "palette", "recompile"} {
		purpose := strings.TrimSpace(strings.TrimPrefix(lines[i], name))
		if !strings.HasPrefix(lines[i], name+" ") || purpose == "" {
			t.Errorf("line %d = %q, want %q then a purpose", i, lines[i], name)
		}
	}
}

func TestAnUnknownToolIsRefusedWithTheSupportedList(t *testing.T) {
	code, out, errs := run("depcheck", t.TempDir(), "ModuleA")
	if code != 3 || out != "" || !strings.Contains(errs, "nre-tool: unknown tool depcheck (known: deps, list, palette, recompile)") {
		t.Errorf("exit %d, out %q, err %q", code, out, errs)
	}
}

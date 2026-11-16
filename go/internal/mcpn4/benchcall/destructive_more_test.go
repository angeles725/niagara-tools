package benchcall

import (
	"strings"
	"testing"
)

func TestEveryDestructiveToolIsRefusedBeforeAnyCallRuns(t *testing.T) {
	b := newBench(t)
	early := `{"tool":"n4_create_component","write":true,"args":{"parent_ord":"station:|slot:/","name":"Early","type":"baja:Folder"}}` + "\n"
	for _, tool := range []string{"n4_remove_component", "n4_rollback", "n4_save_station"} {
		code, lines, errOut := b.run(connect+early+`{"tool":"`+tool+`","write":true,"execute":true,"args":{}}`+"\n", "--execute")
		if code != 2 || len(lines) != 0 || b.fake.Has("Early") || !strings.Contains(errOut, "line 3: "+tool+" is a destructive tool") {
			t.Errorf("%s: exit %d lines %d\n%s", tool, code, len(lines), errOut)
		}
	}
}

func TestDryRunOnlyWinsOverAllowDestructive(t *testing.T) {
	b := newBench(t)
	b.folder()
	code, _, errOut := b.run(connect+removeConfirmed, "--dry-run-only", "--allow-destructive")
	if code != 0 || !b.fake.Has("BenchFolder") || errOut != "n4call: "+modeDryRunOnly+"\n" {
		t.Errorf("exit %d stderr %q", code, errOut)
	}
}

func TestADestructiveToolWithoutWriteOnlyDryRuns(t *testing.T) {
	b := newBench(t)
	b.folder()
	call := `{"tool":"n4_remove_component","args":{"parent_ord":"station:|slot:/","name":"BenchFolder"}}` + "\n"
	if code, _, _ := b.run(connect+call, "--execute", "--allow-destructive"); code != 0 || !b.fake.Has("BenchFolder") {
		t.Errorf("exit %d present=%v", code, b.fake.Has("BenchFolder"))
	}
}

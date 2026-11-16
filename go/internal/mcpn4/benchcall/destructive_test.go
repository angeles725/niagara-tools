package benchcall

// A destructive tool (remove, rollback, save) only executes with --execute, --allow-destructive
// and its line's own "execute": true; the run says on stderr whether its writes execute (bench
// retro 2026-11-15 delta 19 and retro 2026-11-16 row M17: a wrapper dropped --dry-run-only and a
// destructive call planned as a dry run executed). execute_test.go holds the M17 cases.

import (
	"strings"
	"testing"
)

// folder creates BenchFolder through n4call itself (a write that is not destructive).
func (b *bench) folder() {
	b.t.Helper()
	create := `{"tool":"n4_create_component","write":true,"args":{"parent_ord":"station:|slot:/","name":"BenchFolder","type":"baja:Folder"}}` + "\n"
	if code, _, errOut := b.run(connect+create, "--execute"); code != 0 || !b.fake.Has("BenchFolder") {
		b.t.Fatalf("setup: exit %d\n%s", code, errOut)
	}
}

const removeCall = `{"tool":"n4_remove_component","write":true,"args":{"parent_ord":"station:|slot:/","name":"BenchFolder"}}` + "\n"

func TestADestructiveWriteIsRefusedWithoutTheFlagBeforeTheServerStarts(t *testing.T) {
	b := newBench(t)
	b.folder()
	code, lines, errOut := b.run(connect+removeConfirmed, "--execute")
	want := "line 2: n4_remove_component is a destructive tool: pass --allow-destructive to execute it, or run without --execute for a dry run"
	if code != 2 || len(lines) != 0 || !strings.Contains(errOut, want) || !b.fake.Has("BenchFolder") {
		t.Errorf("exit %d lines %v present=%v\n%s", code, lines, b.fake.Has("BenchFolder"), errOut)
	}
}

func TestADestructiveDryRunNeedsNoFlagAndSaysSo(t *testing.T) {
	b := newBench(t)
	b.folder()
	code, lines, errOut := b.run(connect+removeCall, "--dry-run-only")
	if code != 0 || len(lines) != 2 || !b.fake.Has("BenchFolder") || errOut != "n4call: "+modeDryRunOnly+"\n" {
		t.Errorf("exit %d lines %d\n%s", code, len(lines), errOut)
	}
}

// A run with no write prints no mode line (a destructive line without "write" is covered in
// destructive_more_test.go).
func TestARunWithoutWritesSaysNothingOnStderr(t *testing.T) {
	b := newBench(t)
	code, _, errOut := b.run(connect)
	if code != 0 || errOut != "" {
		t.Errorf("exit %d stderr %q", code, errOut)
	}
}

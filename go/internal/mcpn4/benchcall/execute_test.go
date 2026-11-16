package benchcall

// A write executes only with --execute, and a destructive one also needs its own line's
// "execute": true and --allow-destructive (retro 2026-11-16 row M17: a dry-run-only flag did not
// reach n4call and an n4_remove_component executed on the lab station). Losing any flag must
// leave a dry run, never an execution.

import (
	"strings"
	"testing"
)

const createCall = `{"tool":"n4_create_component","write":true,"args":{"parent_ord":"station:|slot:/","name":"BenchFolder","type":"baja:Folder"}}` + "\n"

const removeConfirmed = `{"tool":"n4_remove_component","write":true,"execute":true,"args":{"parent_ord":"station:|slot:/","name":"BenchFolder"}}` + "\n"

func TestAWriteWithoutExecuteIsADryRunAndSaysSo(t *testing.T) {
	b := newBench(t)
	code, lines, errOut := b.run(connect + createCall)
	if code != 0 || len(lines) != 2 || lines[1]["phase"] != "dry_run" || b.fake.Has("BenchFolder") ||
		errOut != "n4call: "+modeDryRunOnly+"\n" {
		t.Errorf("exit %d lines %d present=%v stderr %q", code, len(lines), b.fake.Has("BenchFolder"), errOut)
	}
}

// The incident's shape: the wrapper always passes --allow-destructive and the line is marked to
// execute, but the flag that chooses execution did not reach n4call.
func TestALostExecuteFlagLeavesAConfirmedDestructiveLineADryRun(t *testing.T) {
	b := newBench(t)
	b.folder()
	code, lines, errOut := b.run(connect+removeConfirmed, "--allow-destructive")
	if code != 0 || len(lines) != 2 || lines[1]["phase"] != "dry_run" || !b.fake.Has("BenchFolder") {
		t.Errorf("exit %d lines %d present=%v\n%s", code, len(lines), b.fake.Has("BenchFolder"), errOut)
	}
}

func TestADestructiveLineWithoutItsOwnExecuteIsRefusedEvenWithBothFlags(t *testing.T) {
	b := newBench(t)
	b.folder()
	code, lines, errOut := b.run(connect+removeCall, "--execute", "--allow-destructive")
	want := `line 2: n4_remove_component is a destructive tool: its line needs its own "execute": true to execute`
	if code != 2 || len(lines) != 0 || !strings.Contains(errOut, want) || !b.fake.Has("BenchFolder") {
		t.Errorf("exit %d lines %v present=%v\n%s", code, lines, b.fake.Has("BenchFolder"), errOut)
	}
}

func TestAConfirmedDestructiveLineNeedsAllowDestructive(t *testing.T) {
	b := newBench(t)
	b.folder()
	code, lines, errOut := b.run(connect+removeConfirmed, "--execute")
	want := "line 2: n4_remove_component is a destructive tool: pass --allow-destructive to execute it"
	if code != 2 || len(lines) != 0 || !strings.Contains(errOut, want) || !b.fake.Has("BenchFolder") {
		t.Errorf("exit %d lines %v present=%v\n%s", code, lines, b.fake.Has("BenchFolder"), errOut)
	}
}

func TestAConfirmedDestructiveLineExecutesWithBothFlags(t *testing.T) {
	b := newBench(t)
	b.folder()
	code, lines, errOut := b.run(connect+removeConfirmed, "--execute", "--allow-destructive")
	if code != 0 || len(lines) != 3 || lines[2]["phase"] != "execute" || b.fake.Has("BenchFolder") ||
		!strings.Contains(errOut, "n4call: "+modeExecute) {
		t.Errorf("exit %d lines %d present=%v\n%s", code, len(lines), b.fake.Has("BenchFolder"), errOut)
	}
}

func TestExecuteAndDryRunOnlyTogetherAreAUsageError(t *testing.T) {
	b := newBench(t)
	code, lines, errOut := b.run(connect+createCall, "--execute", "--dry-run-only")
	if code != 2 || len(lines) != 0 || !strings.Contains(errOut, "--execute and --dry-run-only exclude each other") ||
		b.fake.Requests() != 0 {
		t.Errorf("exit %d lines %d requests %d\n%s", code, len(lines), b.fake.Requests(), errOut)
	}
}

func TestExecuteOnALineThatIsNotAWriteIsRefused(t *testing.T) {
	b := newBench(t)
	read := `{"tool":"n4_read_slots","execute":true,"args":{"ord":"station:|slot:/"}}` + "\n"
	code, _, errOut := b.run(connect+read, "--execute")
	if code != 2 || !strings.Contains(errOut, `calls.jsonl:2: "execute" needs "write": true`) || b.fake.Requests() != 0 {
		t.Errorf("exit %d requests %d\n%s", code, b.fake.Requests(), errOut)
	}
}

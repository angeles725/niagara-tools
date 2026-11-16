package benchcall

// Independent probes for retro 2026-11-16 row M17 (O9): a lost flag must land on a dry run, and no
// calls-file shape executes a destructive call without --execute, --allow-destructive and the
// line's own "execute": true.

import (
	"strings"
	"testing"
)

const removeArgs = `"args":{"parent_ord":"station:|slot:/","name":"BenchFolder"}`

// A line's own "dry_run": false never survives into the dry run that n4call sends.
func TestALineDryRunFalseWithoutExecuteStaysADryRun(t *testing.T) {
	b := newBench(t)
	b.folder()
	line := `{"tool":"n4_remove_component","write":true,"execute":true,"args":{"parent_ord":"station:|slot:/","name":"BenchFolder","dry_run":false}}` + "\n"
	code, lines, errOut := b.run(connect+line, "--allow-destructive")
	if code != 0 || len(lines) != 2 || lines[1]["phase"] != "dry_run" || !b.fake.Has("BenchFolder") ||
		errOut != "n4call: "+modeDryRunOnly+"\n" {
		t.Errorf("exit %d lines %d present=%v %q", code, len(lines), b.fake.Has("BenchFolder"), errOut)
	}
}

// A destructive tool sent as a read with dry_run false and a forged token never executes, even
// with every run-wide flag: the server refuses the token.
func TestADestructiveReadLineWithAForgedTokenNeverExecutes(t *testing.T) {
	b := newBench(t)
	b.folder()
	forged := "9999999999.0011223344556677." + strings.Repeat("ab", 32)
	line := `{"tool":"n4_remove_component","args":{"parent_ord":"station:|slot:/","name":"BenchFolder","dry_run":false,"confirmation_token":"` + forged + `"}}` + "\n"
	code, lines, _ := b.run(connect+line, "--execute", "--allow-destructive")
	if code != 1 || len(lines) != 2 || lines[1]["error"] != true || !b.fake.Has("BenchFolder") {
		t.Errorf("exit %d lines %v present=%v", code, lines, b.fake.Has("BenchFolder"))
	}
}

// Flag order does not matter, and a flag after "--" belongs to the server, not to n4call.
func TestFlagOrderAndFlagsAfterTheSeparator(t *testing.T) {
	b := newBench(t)
	b.folder()
	line := `{"tool":"n4_remove_component","write":true,"execute":true,` + removeArgs + `}` + "\n"
	if code, lines, _ := b.run(connect+line, "--allow-destructive", "--call-timeout", "30", "--execute"); code != 0 ||
		len(lines) != 3 || b.fake.Has("BenchFolder") {
		t.Errorf("reordered flags: exit %d lines %d present=%v", code, len(lines), b.fake.Has("BenchFolder"))
	}
}

// Any spelling other than the exact flag is a usage error, never an execution.
func TestExecuteSpellingsAreUsageErrors(t *testing.T) {
	for _, flag := range []string{"--execute=true", "-execute", "--Execute", "--exec"} {
		b := newBench(t)
		b.folder()
		line := `{"tool":"n4_remove_component","write":true,"execute":true,` + removeArgs + `}` + "\n"
		code, _, errOut := b.run(connect+line, flag, "--allow-destructive")
		if code != 2 || !b.fake.Has("BenchFolder") || !strings.Contains(errOut, "unrecognized argument: "+flag) {
			t.Errorf("%s: exit %d present=%v %q", flag, code, b.fake.Has("BenchFolder"), errOut)
		}
	}
}

// One unconfirmed destructive line refuses the whole run before any call, including an earlier
// non-destructive write.
func TestAnUnconfirmedDestructiveLineStopsEveryEarlierWrite(t *testing.T) {
	b := newBench(t)
	b.folder()
	before := b.fake.Requests()
	line := `{"tool":"n4_remove_component","write":true,` + removeArgs + `}` + "\n"
	code, lines, _ := b.run(connect+createCall+line, "--execute", "--allow-destructive")
	if code != 2 || len(lines) != 0 || b.fake.Requests() != before {
		t.Errorf("exit %d lines %d requests %d->%d", code, len(lines), before, b.fake.Requests())
	}
}

// "execute": false on a destructive line is the same as leaving it out.
func TestExecuteFalseIsRefusedLikeAMissingExecute(t *testing.T) {
	b := newBench(t)
	b.folder()
	line := `{"tool":"n4_remove_component","write":true,"execute":false,` + removeArgs + `}` + "\n"
	code, _, errOut := b.run(connect+line, "--execute", "--allow-destructive")
	if code != 2 || !b.fake.Has("BenchFolder") || !strings.Contains(errOut, `its line needs its own "execute": true`) {
		t.Errorf("exit %d present=%v %q", code, b.fake.Has("BenchFolder"), errOut)
	}
}

// "execute" must be a JSON boolean: a string "true" is a usage error.
func TestExecuteAsAStringIsRefused(t *testing.T) {
	b := newBench(t)
	b.folder()
	line := `{"tool":"n4_remove_component","write":true,"execute":"true",` + removeArgs + `}` + "\n"
	code, _, errOut := b.run(connect+line, "--execute", "--allow-destructive")
	if code != 2 || !b.fake.Has("BenchFolder") || !strings.Contains(errOut, `key "execute" has the wrong type`) {
		t.Errorf("exit %d present=%v %q", code, b.fake.Has("BenchFolder"), errOut)
	}
}

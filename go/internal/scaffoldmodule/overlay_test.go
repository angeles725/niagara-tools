package scaffoldmodule

import (
	"path/filepath"
	"testing"
)

const ledgerWant = "# VERSION-LEDGER\n\n| Module | Version |\n|---|---|\n\n## Classes\n| <Module> | not at line start: kept\n"

// logic and dashboard get the station-test overlay in -rt with their own link-in slots; the driver gets none.
func TestStationTestOverlay(t *testing.T) {
	tb := newKit(t)
	for typ, slots := range map[string]string{"logic": `"setpoint", "interval"`, "dashboard": `"setpoint", "status"`, "driver": ""} {
		out := t.TempDir()
		r := run(t, tb, "ColdBox", out, "--type", typ, "--vendor", "Acme")
		if r.code != 0 || r.err != "" {
			t.Fatalf("%s: code=%d err=%q", typ, r.code, r.err)
		}
		got := tree(t, filepath.Join(out, "ColdBox"))
		include := "ColdBox/ColdBox-rt/moduleTest-include.xml"
		test := "ColdBox/ColdBox-rt/srcTest/test/com/acme/ColdBox/BColdBoxLinkTest.java"
		if slots == "" {
			if _, ok := got[include]; ok {
				t.Errorf("driver got a station-test overlay: %v", keys(got))
			}
			continue
		}
		wantFile(t, got, include, "<type class=\"test.com.acme.ColdBox.BColdBoxLinkTest\"/>\n")
		wantFile(t, got, test, "package test.com.acme.ColdBox;\n// Acme ColdBox\nString[] in = { "+slots+" };\n")
	}
}

// Every type gets odd/VERSION-LEDGER.md: the template minus its sample row, each line newline-terminated.
func TestVersionLedger(t *testing.T) {
	tb := newKit(t)
	for _, typ := range []string{"logic", "dashboard", "driver"} {
		out := t.TempDir()
		if r := run(t, tb, "Foo", out, "--type", typ); r.code != 0 {
			t.Fatalf("%s: code=%d err=%q", typ, r.code, r.err)
		}
		wantFile(t, tree(t, filepath.Join(out, "Foo")), "odd/VERSION-LEDGER.md", ledgerWant)
	}
}

// A missing template is exit 3 after the fixture copy, naming the template path the launcher resolved.
func TestMissingTemplates(t *testing.T) {
	t.Run("station-test", func(t *testing.T) {
		tb := newKit(t, "templates/station-test")
		out := t.TempDir()
		r := run(t, tb, "Foo", out)
		want := "scaffold-module: station-test template not found: " + tb + "/../templates/station-test\n"
		if r.code != 3 || r.out != "" || r.err != want {
			t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
		}
		if r := run(t, tb, "Bar", out, "--type", "driver"); r.code != 0 { // the driver never reads it
			t.Errorf("driver: code=%d err=%q", r.code, r.err)
		}
	})
	t.Run("version ledger", func(t *testing.T) {
		tb := newKit(t, "templates/VERSION-LEDGER.md")
		r := run(t, tb, "Foo", t.TempDir(), "--type", "driver")
		want := "scaffold-module: version-ledger template not found: " + tb + "/../templates/VERSION-LEDGER.md\n"
		if r.code != 3 || r.out != "" || r.err != want {
			t.Errorf("code=%d out=%q err=%q", r.code, r.out, r.err)
		}
	})
}

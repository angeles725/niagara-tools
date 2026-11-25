package scaffoldmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A fixture file with a malformed block is refused with its path and line, exit 3, and no module is left behind.
func TestWithOffloadMalformedFixture(t *testing.T) {
	for _, extra := range [][]string{nil, {"--with-offload"}} {
		tb := offloadKit(t)
		bad := filepath.Join(tb, "..", "fixtures", "MinimalPan", "MinimalPan", "MinimalPan-rt", "module-include.xml")
		if err := os.WriteFile(bad, []byte("<types>\n  <!-- offload:begin -->\n</types>\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		out := t.TempDir()
		r := run(t, tb, append([]string{"ColdBox", out, "--vendor", "Acme"}, extra...)...)
		want := "MinimalPan/MinimalPan-rt/module-include.xml: line 2: offload:begin is never closed"
		if r.code != ExitEnv || r.out != "" || !strings.Contains(r.err, want) {
			t.Errorf("%v: code=%d out=%q err=%q", extra, r.code, r.out, r.err)
		}
		if _, err := os.Stat(filepath.Join(out, "ColdBox")); err == nil {
			t.Errorf("%v: left the module", extra)
		}
	}
}

// The internal mode CI and the bats round trip share: markers removed in place, every other file untouched.
func TestStripOffloadMarkers(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"a/include.xml": "<types>\n  <!-- offload:begin -->\n  <t/>\n  <!-- offload:end -->\n</types>\n",
		"MODULE-MAP.md": "| a |\n| b | <!-- offload:line -->\n",
		"plain.txt":     "nothing here\n",
		"bin.jar":       "PK\x00offload:begin\xff",
	}
	for rel, content := range files {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if r := run(t, "", "--strip-offload-markers", dir); r.code != 0 || r.out != "" || r.err != "" {
		t.Fatalf("code=%d out=%q err=%q", r.code, r.out, r.err)
	}
	got := tree(t, dir)
	wantFile(t, got, "a/include.xml", "<types>\n  <t/>\n</types>\n")
	wantFile(t, got, "MODULE-MAP.md", "| a |\n| b |\n")
	wantFile(t, got, "plain.txt", "nothing here\n")
	wantFile(t, got, "bin.jar", files["bin.jar"])
	if st, err := os.Stat(filepath.Join(dir, "a/include.xml")); err != nil || st.Mode().Perm() != 0o640 {
		t.Errorf("mode changed: %v %v", st, err)
	}
}

func TestStripOffloadMarkersRefusals(t *testing.T) {
	if r := run(t, "", "--strip-offload-markers"); r.code != ExitUsage || r.err != stripUsage {
		t.Errorf("no dir: code=%d err=%q", r.code, r.err)
	}
	if r := run(t, "", "--strip-offload-markers", filepath.Join(t.TempDir(), "missing")); r.code != ExitEnv || !strings.Contains(r.err, "missing") {
		t.Errorf("missing dir: code=%d err=%q", r.code, r.err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x.xml"), []byte("<!-- offload:end -->\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	r := run(t, "", "--strip-offload-markers", dir)
	if r.code != ExitEnv || !strings.Contains(r.err, "x.xml: line 1: offload:end without an offload:begin") {
		t.Errorf("malformed: code=%d err=%q", r.code, r.err)
	}
	if got := tree(t, dir)["x.xml"]; got != "<!-- offload:end -->\n" {
		t.Errorf("a refused run rewrote the file: %q", got)
	}
}

// --with-offload and --with-doc together: the doc part, its map row before plumbing, and the offload rows.
func TestRealKitWithOffloadAndDoc(t *testing.T) {
	kit, err := filepath.Abs("../../../build-n4-module-kit")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if r := run(t, kit+"/toolbelt", "ColdBox", out, "--vendor", "Acme", "--with-offload", "--with-doc"); r.code != 0 || r.err != "" {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	got := tree(t, filepath.Join(out, "ColdBox"))
	if _, ok := got["ColdBox/ColdBox-doc/src/doc/ColdBox-ColdBox.html"]; !ok {
		t.Errorf("no doc page; emitted %v", keys(got))
	}
	rows := strings.Split(got["ColdBox/MODULE-MAP.md"], "\n")
	var names []string // concern and part of each row: the -doc part's plumbing row goes before the -rt one
	for _, row := range rows {
		if cells := strings.Split(row, "|"); len(cells) > 3 && strings.HasPrefix(row, "| ") && !strings.HasPrefix(row, "| Concern") {
			names = append(names, strings.TrimSpace(cells[1])+"/"+strings.TrimSpace(cells[2]))
		}
	}
	if want := "control/rt config-save/rt async-offload/rt permissions/rt station-test/rt plumbing/doc plumbing/rt"; strings.Join(names, " ") != want {
		t.Errorf("map rows %v, want %s", names, want)
	}
	if strings.Contains(got["ColdBox/MODULE-MAP.md"], "offload:") {
		t.Errorf("a marker survived in the map")
	}
	rt := "ColdBox/ColdBox-rt/"
	if n := strings.Count(got[rt+"moduleTest-include.xml"], "<type "); n != 4 {
		t.Errorf("moduleTest-include.xml has %d entries, want 4", n)
	}
	if !strings.Contains(got[rt+"module-permissions.xml"], "MANAGE_EXECUTION") {
		t.Errorf("no MANAGE_EXECUTION request")
	}
}

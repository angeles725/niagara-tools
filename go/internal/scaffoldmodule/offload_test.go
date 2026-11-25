package scaffoldmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFilterOffload(t *testing.T) {
	const text = "a\n<!-- offload:begin -->\nb\n  <!-- offload:end -->\nc\n\t# offload:begin\nd\n# offload:end\n| row | <!-- offload:line -->\ne"
	for _, tc := range []struct {
		name string
		keep bool
		want string
	}{
		{"flag: markers go, every line stays", true, "a\nb\nc\nd\n| row |\ne"},
		{"no flag: blocks and marked rows go", false, "a\nc\ne"},
	} {
		got, err := filterOffload(text, tc.keep)
		if err != nil || got != tc.want {
			t.Errorf("%s:\n got %q (%v)\nwant %q", tc.name, got, err, tc.want)
		}
	}
	if got, err := filterOffload("no marker\r\nat all", false); err != nil || got != "no marker\r\nat all" {
		t.Errorf("a file without markers changed: %q (%v)", got, err)
	}
}

// A marker is a whole trimmed line in the file's comment syntax, the row suffix sits at the end of the line; the
// token anywhere else is ordinary text and is kept, flag or not.
func TestFilterOffloadIgnoresLookalikes(t *testing.T) {
	for _, line := range []string{
		"x <!-- offload:begin -->", "<!-- offload:begin --> x", "// offload:begin", "offload:begin", "# offload:end now",
		"see offload:begin and offload:end", "<!-- offload:begin", "| a | <!-- offload:line --> more |", "offload:line",
	} {
		for _, keep := range []bool{true, false} {
			if got, err := filterOffload("a\n"+line+"\nb\n", keep); err != nil || got != "a\n"+line+"\nb\n" {
				t.Errorf("keep=%v %q: got %q (%v)", keep, line, got, err)
			}
		}
	}
}

func TestFilterOffloadRefusesMalformedBlocks(t *testing.T) {
	for _, tc := range []struct{ name, text, want string }{
		{"unclosed begin", "a\n<!-- offload:begin -->\nb\n", "line 2: offload:begin is never closed"},
		{"unclosed # begin", "# offload:begin\nb", "line 1: offload:begin is never closed"},
		{"nested begin", "<!-- offload:begin -->\na\n<!-- offload:begin -->\n", "line 3: offload:begin inside the block opened at line 1"},
		{"end without begin", "a\n# offload:end\n", "line 2: offload:end without an offload:begin"},
		{"second end", "# offload:begin\n# offload:end\n# offload:end\n", "line 3: offload:end without an offload:begin"},
	} {
		for _, keep := range []bool{true, false} {
			if got, err := filterOffload(tc.text, keep); err == nil || !strings.Contains(err.Error(), tc.want) || got != "" {
				t.Errorf("%s keep=%v: got %q err=%v, want error %q", tc.name, keep, got, err, tc.want)
			}
		}
	}
}

// offloadKit is the small kit plus the offload parts of the logic fixture: two offload-only files, a shared file
// with a marked block and the permissions file (a fallback without the flag).
func offloadKit(t *testing.T) string {
	t.Helper()
	tb := newKit(t)
	rt := filepath.Join(tb, "..", "fixtures", "MinimalPan", "MinimalPan", "MinimalPan-rt")
	for rel, content := range map[string]string{
		"src/com/angeles/MinimalPan/BPanOffload.java":                         "// Angeles MinimalPan-offload-\n",
		"srcTest/test/com/angeles/MinimalPan/BMinimalPanOffloadTest.java":     "// MinimalPan\n",
		"srcTest/test/com/angeles/MinimalPan/BMinimalPanPermissionsTest.java": "// MinimalPan\n",
		"moduleTest-include.xml":                                              "<types>\n  <type name=\"Base\"/>\n  <!-- offload:begin -->\n  <type name=\"OffloadTest\"/>\n  <!-- offload:end -->\n</types>\n",
		"module-include.xml":                                                  "<types>\n  <!-- offload:begin -->\n  <type name=\"PanOffload\"/>\n  <!-- offload:end -->\n</types>\n",
		"module-permissions.xml":                                              "<permissions>\n  <name>MANAGE_EXECUTION</name>\n</permissions>\n",
	} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(rt, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(rt, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return tb
}

func TestWithOffload(t *testing.T) {
	tb := offloadKit(t)
	without, with := t.TempDir(), t.TempDir()
	for _, c := range []struct {
		out   string
		extra []string
	}{{without, nil}, {with, []string{"--with-offload"}}} {
		if r := run(t, tb, append([]string{"ColdBox", c.out, "--vendor", "Acme"}, c.extra...)...); r.code != 0 || r.err != "" {
			t.Fatalf("%v: code=%d err=%q", c.extra, r.code, r.err)
		}
	}
	rt := "ColdBox/ColdBox-rt/"
	off, on := tree(t, filepath.Join(without, "ColdBox")), tree(t, filepath.Join(with, "ColdBox"))
	for _, rel := range []string{"src/com/acme/ColdBox/BPanOffload.java", "srcTest/test/com/acme/ColdBox/BColdBoxOffloadTest.java", "srcTest/test/com/acme/ColdBox/BColdBoxPermissionsTest.java"} {
		if _, ok := off[rt+rel]; ok {
			t.Errorf("without the flag: %s emitted", rel)
		}
		if _, ok := on[rt+rel]; !ok {
			t.Errorf("with the flag: %s missing; emitted %v", rel, keys(on))
		}
	}
	wantFile(t, on, rt+"src/com/acme/ColdBox/BPanOffload.java", "// Acme ColdBox-offload-\n")
	wantFile(t, on, rt+"module-include.xml", "<types>\n  <type name=\"PanOffload\"/>\n</types>\n")
	wantFile(t, on, rt+"module-permissions.xml", "<permissions>\n  <name>MANAGE_EXECUTION</name>\n</permissions>\n")
	wantFile(t, off, rt+"module-include.xml", "<types>\n</types>\n")
	// with the flag the fixture's own test list (markers gone) stays; without it the station-test overlay writes it
	wantFile(t, on, rt+"moduleTest-include.xml", "<types>\n  <type name=\"Base\"/>\n  <type name=\"OffloadTest\"/>\n</types>\n")
	wantFile(t, off, rt+"moduleTest-include.xml", "<type class=\"test.com.acme.ColdBox.BColdBoxLinkTest\"/>\n")
	wantFile(t, off, rt+"module-permissions.xml", "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<permissions/>\n")
}

// Another type has no component to emit: exit 2, the message, nothing written.
func TestWithOffloadRefusal(t *testing.T) {
	tb := newKit(t)
	for _, typ := range []string{"dashboard", "driver", "library", "service"} {
		out := t.TempDir()
		r := run(t, tb, "Foo", out, "--vendor", "Acme", "--type", typ, "--with-offload")
		want := "scaffold-module: --with-offload needs --type logic; " + typ + " has no async-offload component\n"
		if r.code != 2 || r.out != "" || r.err != want {
			t.Errorf("%s: code=%d out=%q err=%q", typ, r.code, r.out, r.err)
		}
		if _, err := os.Stat(filepath.Join(out, "Foo")); err == nil {
			t.Errorf("%s: wrote the module", typ)
		}
	}
}

// Without the flag the real logic fixture emits no offload file, no MANAGE_EXECUTION request, the empty
// <permissions/> root and only the base and link test entries.
func TestRealKitWithoutOffload(t *testing.T) {
	kit, err := filepath.Abs("../../../build-n4-module-kit")
	if err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if r := run(t, kit+"/toolbelt", "ColdBox", out, "--vendor", "Acme"); r.code != 0 || r.err != "" {
		t.Fatalf("code=%d err=%q", r.code, r.err)
	}
	got := tree(t, filepath.Join(out, "ColdBox"))
	for rel, content := range got {
		if strings.Contains(strings.ToLower(rel+content), "offload") || strings.Contains(content, "MANAGE_EXECUTION") {
			t.Errorf("%s mentions the offload component", rel)
		}
	}
	rt := "ColdBox/ColdBox-rt/"
	wantFile(t, got, rt+"module-permissions.xml", "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<permissions/>\n")
	if n := strings.Count(got[rt+"moduleTest-include.xml"], "<type "); n != 2 {
		t.Errorf("moduleTest-include.xml has %d entries, want 2", n)
	}
}

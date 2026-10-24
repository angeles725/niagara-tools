package newretro

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// fixture is a kit root under a fresh temp dir with the bats retro-loop fixture: an INDEX header and the
// minimal `## kit` envelope.
func fixture(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "kit")
	if err := os.MkdirAll(filepath.Join(root, "retros"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "retros", "INDEX.md"), "| Retro file | Module | Date | review-status | deltas |\n|---|---|---|---|---|\n")
	write(t, filepath.Join(root, "BUILD-STATE.src.md"), "<!-- build-state.v1 -->\n## kit\nretro_pending: false\n")
	return root
}

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runIn(root string, args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := Command(today)(args, cli.Stdio{Out: &o, Err: &e, Getenv: func(k string) string {
		if k == "KIT" {
			return root
		}
		return ""
	}})
	return code, o.String(), e.String()
}

// TestEnvelopeWriteFailureLandsNothing: when BUILD-STATE.src.md cannot be replaced (its directory and the file
// are read-only), the stub is removed and INDEX.md restored, so the run leaves the tree as it found it.
// Mutation: NR-rollback -- keeping the stub and the new INDEX.md on the failed envelope write fails this test.
func TestEnvelopeWriteFailureLandsNothing(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: read-only modes do not block writes")
	}
	root := fixture(t)
	before := tree(t, root)
	state := filepath.Join(root, "BUILD-STATE.src.md")
	if err := os.Chmod(state, 0o444); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	code, out, _ := runIn(root, "kit", "slugabc")
	if after := tree(t, root); code != ExitWrite || out != "" || len(after) != len(before) ||
		after["retros/INDEX.md"] != before["retros/INDEX.md"] || after["BUILD-STATE.src.md"] != before["BUILD-STATE.src.md"] {
		t.Errorf("exit %d, stdout %q, tree after %q; want exit 1, no stdout and the tree before %q", code, out, after, before)
	}
}

// TestDeltasRefusedBeforeRender: an over-cap or overflowing --deltas is refused while parsing, so nothing is
// rendered or written (the inputs stay small or are never rendered: an overflowing count must never size a loop).
func TestDeltasRefusedBeforeRender(t *testing.T) {
	for _, v := range []string{"100", "99999999999999999999", "18446744073709551616", "-1", "x", ""} {
		if _, msg := parse([]string{"kit", "slugabc", "--deltas", v}, today); msg != "new-retro: --deltas must be an integer from 0 to 99: "+v+"\n" {
			t.Errorf("parse --deltas %q: message %q, want the 0-99 refusal", v, msg)
		}
	}
	saved := render
	t.Cleanup(func() { render = saved })
	render = func(retro) string { t.Fatal("the stub was rendered for a refused --deltas"); return "" }
	root := fixture(t)
	if code, _, _ := runIn(root, "kit", "slugabc", "--deltas", "99999999999999999999"); code != ExitUsage {
		t.Errorf("exit %d, want %d", code, ExitUsage)
	}
	if _, err := os.Stat(filepath.Join(root, "retros", "2026-01-02-slugabc.md")); err == nil {
		t.Error("a stub was written for a refused --deltas")
	}
}

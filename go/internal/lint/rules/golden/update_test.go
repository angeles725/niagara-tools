package golden

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

// echoMain prints its first argument on stdout and "note" (no newline) on stderr, and leaves the tree alone.
func echoMain(_, args []string, s cli.Stdio) int {
	fmt.Fprintln(s.Out, args[0])
	fmt.Fprint(s.Err, "note")
	return 0
}

// stale is a case whose recorded stdout and stderr no longer match echoMain: only want and stderr may change.
const stale = "-- args --\nnew\n-- want --\nexit 0\nold\n-- stderr --\n-- a --\nx\n"

// fresh is stale re-recorded: the stderr without a final newline is spelled "(noeol)", the rest is kept.
const fresh = "-- args --\nnew\n-- want --\nexit 0\nnew\n-- stderr (noeol) --\nnote\n-- a --\nx\n"

// withUpdate runs f with -update set to on, as `go test ... -update` would set it.
func withUpdate(t *testing.T, on bool, f func()) {
	t.Helper()
	was := update
	update = on
	defer func() { update = was }()
	f()
}

// TestUpdateRewritesTheCase: with -update a stale case is rewritten in place, even when the runner was given a
// relative case dir and the case runs in its tree (CHDIR=1); a second run without -update is green and
// writes nothing.
func TestUpdateRewritesTheCase(t *testing.T) {
	parent := t.TempDir()
	t.Chdir(parent)
	if err := os.Mkdir("cases", 0o755); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"c.txt": stale, "record.env": "CHDIR=1\n"} {
		if err := os.WriteFile(filepath.Join("cases", name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	r := Runner{Dir: "cases", Cases: 1, Main: echoMain}
	withUpdate(t, true, func() { r.Run(t) })
	path := filepath.Join(parent, "cases", "c.txt")
	if got, _ := os.ReadFile(path); string(got) != fresh {
		t.Fatalf("after -update the case is\n%s\nwant\n%s", got, fresh)
	}
	fi, _ := os.Stat(path)
	withUpdate(t, false, func() { r.Run(t) })
	if got, _ := os.ReadFile(path); string(got) != fresh {
		t.Errorf("the second run changed the case:\n%s", got)
	}
	if fi2, _ := os.Stat(path); !fi2.ModTime().Equal(fi.ModTime()) {
		t.Errorf("the second run wrote the case (mtime %v, was %v)", fi2.ModTime(), fi.ModTime())
	}
}

// TestNoUpdateWritesNothing: without -update a stale case is a problem and its file is left byte for byte.
func TestNoUpdateWritesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.txt")
	if err := os.WriteFile(path, []byte(stale), 0o644); err != nil {
		t.Fatal(err)
	}
	var problems []string
	withUpdate(t, false, func() { problems, _ = Runner{Main: echoMain}.replay(t, path) })
	if len(problems) != 1 {
		t.Fatalf("problems %q, want one output difference", problems)
	}
	if got, _ := os.ReadFile(path); string(got) != stale {
		t.Errorf("without -update the case was rewritten:\n%s", got)
	}
}

// TestRewriteSpellsSections pins how a re-recorded section is spelled: a body ending in a newline (or empty)
// under the plain header, any other under "(noeol)"; a missing stderr section is added after want.
func TestRewriteSpellsSections(t *testing.T) {
	for _, c := range []struct{ text, want, stderr, out string }{
		{stale, "exit 0\nnew\n", "", "-- args --\nnew\n-- want --\nexit 0\nnew\n-- stderr --\n-- a --\nx\n"},
		{"-- want (noeol) --\nexit 1\n-- stderr (noeol) --\ne\n", "exit 1\nrow\n", "err\n", "-- want --\nexit 1\nrow\n-- stderr --\nerr\n"},
		{"-- args --\n-- want --\nexit 0\n-- a --\nx\n", "exit 2", "e", "-- args --\n-- want (noeol) --\nexit 2\n-- stderr (noeol) --\ne\n-- a --\nx\n"},
		{"-- args --\n-- want --\nexit 0\n", "exit 0\n", "", "-- args --\n-- want --\nexit 0\n"},
	} {
		if got := rewrite(c.text, c.want, c.stderr); got != c.out {
			t.Errorf("rewrite(%q, %q, %q)\n= %q\nwant %q", c.text, c.want, c.stderr, got, c.out)
		}
	}
}

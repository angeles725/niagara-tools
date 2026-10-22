package facetscard

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/angeles725/niagara-tools/go/internal/cli"
)

const usageLine = "usage: check-facets-card.sh <card.md> <BFacets.java | keys-file>\n"

// files writes name -> text into a temp dir and returns the dir.
func files(t *testing.T, m map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, text := range m {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func run(args ...string) (code int, out, errs string) {
	var o, e bytes.Buffer
	code = Main(args, cli.Stdio{Out: &o, Err: &e})
	return code, o.String(), e.String()
}

func TestMain(t *testing.T) {
	t.Parallel()
	d := files(t, map[string]string{
		"min.md":     cardDoc(row("MIN", "min")),
		"minnull.md": cardDoc(row("MIN", "min"), row("NULL_TEXT", "nullText")),
		"maxmin.md":  cardDoc(row("MAX", "max"), row("MIN", "min")),
		"minord.md":  cardDoc(row("MIN", "min"), row("ORD_RELATIVIZE", "ordRelativize")),
		"nocard.md":  cardDoc(),
		"empty.md":   "",
		"k1.keys":    "MIN=min\n",
		"k2.keys":    "MIN=min\nMAX=max\n",
		"empty.keys": "",
		"BFacets.java": "public final class BFacets {\n  public static final String MIN = \"min\";\n" +
			"  public static final String ORD_RELATIVIZE= \"ordRelativize\";\n  private static final String[] NUMERIC_KEYS = { MIN };\n}\n",
	})
	p := func(name string) string { return filepath.Join(d, name) }
	if err := os.Mkdir(p("adir"), 0o755); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name      string
		args      []string
		code      int
		out, errs string
	}{
		{"FC-missing", []string{p("min.md"), p("k2.keys")}, ExitDiff,
			"MISSING MAX=max\ncheck-facets-card: card " + p("min.md") + " differs from " + p("k2.keys") + " (1 card rows, 2 source keys)\n", ""},
		{"FC-extra", []string{p("minnull.md"), p("k1.keys")}, ExitDiff,
			"EXTRA NULL_TEXT=nullText\ncheck-facets-card: card " + p("minnull.md") + " differs from " + p("k1.keys") + " (2 card rows, 1 source keys)\n", ""},
		{"FC-order", []string{p("maxmin.md"), p("k2.keys")}, ExitDiff,
			"ORDER card #1 MAX=max != source MIN=min\nORDER card #2 MIN=min != source MAX=max\n" +
				"check-facets-card: card " + p("maxmin.md") + " differs from " + p("k2.keys") + " (2 card rows, 2 source keys)\n", ""},
		{"FC-java", []string{p("minord.md"), p("BFacets.java")}, ExitOK, "check-facets-card: 2 keys match " + p("BFacets.java") + "\n", ""},
		{"FC-empty: no card rows", []string{p("nocard.md"), p("k1.keys")}, ExitEnv, "", "check-facets-card: no card rows parsed from " + p("nocard.md") + "\n"},
		{"empty card file", []string{p("empty.md"), p("k1.keys")}, ExitEnv, "", "check-facets-card: no card rows parsed from " + p("empty.md") + "\n"},
		{"no source keys", []string{p("min.md"), p("empty.keys")}, ExitEnv, "", "check-facets-card: no key constants parsed from " + p("empty.keys") + "\n"},
		{"FC-empty: /dev/null is not a regular file", []string{p("min.md"), "/dev/null"}, ExitEnv, "", "check-facets-card: cannot read /dev/null\n"},
		{"FC-empty: one argument", []string{p("min.md")}, ExitEnv, "", usageLine},
		{"no arguments", nil, ExitEnv, "", usageLine},
		{"three arguments", []string{"a", "b", "c"}, ExitEnv, "", usageLine},
		{"missing card", []string{p("nope.md"), p("k1.keys")}, ExitEnv, "", "check-facets-card: cannot read " + p("nope.md") + "\n"},
		{"missing source", []string{p("min.md"), p("nope.keys")}, ExitEnv, "", "check-facets-card: cannot read " + p("nope.keys") + "\n"},
		{"card checked before source", []string{p("nope.md"), p("nope.keys")}, ExitEnv, "", "check-facets-card: cannot read " + p("nope.md") + "\n"},
		{"directory", []string{p("min.md"), p("adir")}, ExitEnv, "", "check-facets-card: cannot read " + p("adir") + "\n"},
		{"empty path", []string{"", p("k1.keys")}, ExitEnv, "", "check-facets-card: cannot read \n"},
		{"a flag is a path", []string{"--help", p("k1.keys")}, ExitEnv, "", "check-facets-card: cannot read --help\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			code, out, errs := run(tt.args...)
			if code != tt.code || out != tt.out || errs != tt.errs {
				t.Errorf("got (%d, %q, %q)\nwant (%d, %q, %q)", code, out, errs, tt.code, tt.out, tt.errs)
			}
		})
	}
}

func TestMainUnreadableFile(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode-000 file")
	}
	d := files(t, map[string]string{"min.md": cardDoc(row("MIN", "min")), "locked.keys": "MIN=min\n"})
	locked := filepath.Join(d, "locked.keys")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	code, out, errs := run(filepath.Join(d, "min.md"), locked)
	if code != ExitEnv || out != "" || errs != "check-facets-card: cannot read "+locked+"\n" {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}

// TestMainLiveCard is FC1: the kit card matches the pinned 4.15.6.20 constants, in order.
func TestMainLiveCard(t *testing.T) {
	t.Parallel()
	card := filepath.Join("..", "..", "..", "build-n4-module-kit", "types", "logic-authoring.md")
	keys := filepath.Join("..", "..", "..", "tests", "fixtures", "facets-card", "BFacets-4.15.6.20.keys")
	code, out, errs := run(card, keys)
	if code != ExitOK || out != "check-facets-card: 34 keys match "+keys+"\n" || errs != "" {
		t.Errorf("got (%d, %q, %q)", code, out, errs)
	}
}
